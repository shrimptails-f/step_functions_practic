# Step Functions + ECS PoC 概要設計

## 1. 目的

本PoCの目的は、以下の一連の処理を AWS Step Functions を中心にオーケストレーションできるかを確認することです。

1. Step Functions から ECS Run Task で親タスクを起動する
2. 親タスクの出力値をもとに、複数のワーカータスクを起動する
3. ワーカータスク内で SQS にメッセージを投入する
4. SQS をトリガーに Lambda を起動し、後続処理を実行する
5. 各段階でどこまでリトライ、タイムアウト、失敗制御を組み込めるか確認する
6. Step Functions の実行入力を親タスクへ受け渡し、その値をもとにワーカー起動内容を決められることを確認する

PoC では本番相当の業務処理は扱わず、状態遷移・データ連携・失敗時挙動の確認を優先します。

## 2. 想定アーキテクチャ

```text
Step Functions
  -> ECS 親タスク
      -> 実行対象ワーカー情報を JSON で出力
  -> Map State
      -> ECS ワーカータスクを並列起動
          -> SQS にメッセージ送信
  -> SQS
      -> Lambda
          -> メッセージ単位の後続処理
```

### サービス構成

- Step Functions
  - 全体フローの制御
  - 親タスク実行
  - Map によるワーカータスク並列実行
  - リトライ、Catch、タイムアウト制御
- ECS Fargate
  - 親タスクとワーカータスクの実行基盤
  - コンテナイメージは ECR から取得
- ECR
  - 親タスク用・ワーカータスク用イメージ保管
- SQS
  - ワーカータスクから Lambda への非同期受け渡し
- Lambda
  - SQS メッセージの処理
  - 失敗時の再試行、DLQ 連携の確認対象
  - 実行結果は CloudWatch Logs で確認
- CloudWatch Logs
  - Step Functions / ECS / Lambda のログ集約

## 3. 処理フロー

本 PoC では、コンソール上で状態遷移を追いやすくするため、親タスク・ワーカータスク・Lambda の各処理で 30 秒の遅延を入れます。

### 3.1 親タスク

Step Functions から `ecs:runTask.sync` 相当で親タスクを実行します。  
親タスクは Step Functions 実行時の入力パラメータを受け取り、後続で起動すべきワーカー一覧を JSON ファイルまたは標準出力経由で返却します。

Step Functions 実行入力イメージ:

```json
{
  "jobId": "job-001",
  "retryLimit": 2,
  "workerInputs": [
    { "workerId": "w1", "payload": { "value": 1, "shouldFail": false } },
    { "workerId": "w2", "payload": { "value": 2, "shouldFail": true } },
    { "workerId": "w3", "payload": { "value": 3, "shouldFail": false } }
  ]
}
```

出力イメージ:

```json
{
  "jobId": "job-001",
  "retryLimit": 2,
  "workers": [
    { "workerId": "w1", "payload": { "value": 1, "shouldFail": false } },
    { "workerId": "w2", "payload": { "value": 2, "shouldFail": true } },
    { "workerId": "w3", "payload": { "value": 3, "shouldFail": false } }
  ]
}
```

### 3.2 ワーカータスク群

親タスクの出力 `workers` を Step Functions の `Map` State に渡し、要素ごとに ECS ワーカータスクを起動します。
PoC では Step Functions の `MaxConcurrency` は `2` とし、同時起動数を抑制します。

ワーカータスクは次を実施します。

1. Step Functions から受け取った `workerId` と `payload` を読む
2. 30 秒の遅延を入れた上で疑似的な業務処理を行う
3. 処理結果を SQS に送信する
4. 失敗時は失敗理由と入力値を返し、後続の再実行判定対象にする

### 3.3 失敗ワーカーの再実行

ワーカータスク実行後、Step Functions で成功・失敗の結果を集約します。  
失敗したワーカーが存在する場合は、その失敗ワーカーのみを抽出して再度 `Map` State に渡します。

再実行は以下の制御とします。

1. 失敗ワーカー一覧を抽出する
2. `retryCount` を 1 加算する
3. `retryCount < retryLimit` の場合のみ失敗ワーカーを再実行する
4. `retryCount >= retryLimit` の場合は再実行を打ち切り、最終失敗一覧を出力して終了する

これにより、失敗したワーカーだけを対象に段階的に再試行しつつ、無限リトライを防止します。

### 3.4 Lambda

SQS イベントソースマッピングで Lambda を起動し、ワーカータスクから投入されたメッセージを処理します。
Lambda 側の同時実行数上限も `2` とし、PoC のコストと挙動観察を優先します。

PoC では以下を確認対象とします。

- 正常時に処理が完了すること
- 一定条件で失敗させた場合に再試行されること
- 最大受信回数超過後に DLQ に送られること
- 処理ログを CloudWatch Logs から追跡できること

## 4. Step Functions 設計方針

### 採用する基本パターン

- 親タスクは逐次 1 回実行
- ワーカータスクは `Map` で並列実行
- `Map` の並列度は `MaxConcurrency = 2`
- ワーカー実行結果から失敗分のみを抽出して再実行するループを設ける
- ECS 実行は Step Functions の service integration を利用
- ワーカー後続の非同期処理は SQS + Lambda に切り出す

### 状態遷移イメージ

```text
Start
  -> RunParentTask
  -> ValidateParentOutput
  -> InitializeRetryCount
  -> RunWorkersInParallel (Map)
  -> AggregateWorkerResults
  -> Choice: FailedWorkersExist?
  -> Choice: RetryCountUnderLimit?
  -> PrepareFailedWorkersForRetry
  -> RunWorkersInParallel (Map) ...
  -> Success or FinishWithFailedWorkers

Error:
  -> NotifyFailure or Fail
```

### リトライ設計

- 基本方針
  - 親タスクからワーカータスクへは、Step Functions の状態データとして値を受け渡す
  - 親タスクは `workers` 配列を出力し、各要素を `Map` でワーカータスクへ渡す
  - リトライ判定は `リトライしてよい失敗` と `リトライしてはいけない失敗` で分ける
  - ただし Step Functions の扱いとしては、内部的に `技術失敗` と `業務失敗` を分離する

- `RunParentTask`
  - ECS の一時失敗を対象に数回リトライ
  - 指数バックオフを設定
- `RunParentTask` の入出力
  - 入力は Step Functions 実行開始時の JSON をそのまま親タスクへ渡す
  - 出力は `jobId`、`retryLimit`、`workers` を含む JSON とする
- `RunWorkersInParallel`
  - ECS 起動の一時失敗は Step Functions の `Retry` で吸収
  - 親タスク出力の `workers[*]` を各ワーカータスクへそのまま渡す
  - 業務失敗は item 単位で結果化し、失敗ワーカーのみ再実行対象とする
- `RetryFailedWorkers`
  - 再実行対象は失敗ワーカーのみ
  - `retryLimit` を超えたら打ち切る
- Lambda
  - SQS の再試行ポリシーに従う
  - 一定回数超過で DLQ へ退避

### 4.1 失敗分類と再試行方針

本 PoC では、失敗を最終的に以下の 2 つに分類します。

- リトライしてよい失敗
  - Step Functions で再試行対象にする
- リトライしてはいけない失敗
  - その場で最終失敗として保持する

ただし実装上は、Step Functions への伝え方の違いにより、内部的に以下の 2 系統を使い分けます。

#### 技術失敗

- 例
  - ECS 起動失敗
  - タイムアウト
  - コンテナクラッシュ
  - イメージ取得失敗
- Step Functions への伝え方
  - `ecs:runTask.sync` の Task state 自体を失敗させる
  - Step Functions の `Retry` / `Catch` で扱う
- 判定
  - 原則として一時的なものはリトライ対象
  - 規定回数超過後は失敗として終了

#### 業務失敗

- 例
  - PoC 用に Go 実装で意図的に発生させる失敗
  - 特定 `workerId` を固定で失敗させるケース
- Step Functions への伝え方
  - ECS タスク自体は `exit 0` で終了する
  - ワーカータスク出力 JSON の `status` と `errorType` で結果を返す
- 判定
  - `errorType = RETRYABLE` は再実行対象
  - `errorType = NON_RETRYABLE` は即時に最終失敗対象

業務失敗の出力例:

```json
{
  "workerId": "w2",
  "payload": {
    "value": 2
  },
  "status": "failed",
  "errorType": "RETRYABLE",
  "message": "simulated retryable failure"
}
```

```json
{
  "workerId": "w3",
  "payload": {
    "value": 3
  },
  "status": "failed",
  "errorType": "NON_RETRYABLE",
  "message": "simulated non-retryable failure"
}
```

#### なぜ `NON_RETRYABLE` を `exit 1` にしないか

- `exit 1` にすると Step Functions 上では Task state failure として扱われる
- その場合、`Retry` / `Catch` の制御には乗せやすいが、失敗ワーカー一覧として集約しにくい
- 今回の PoC では、失敗したワーカー入力と失敗種別を保持しながら後続分岐したい
- そのため、業務失敗は `exit 0` とし、結果 JSON で `RETRYABLE` / `NON_RETRYABLE` を返す

#### PoC でのハードコード方針

- ワーカーの Go コード内で特定の `workerId` を固定で失敗させる
- 例
  - `w2` は `RETRYABLE`
  - `w3` は `NON_RETRYABLE`
  - それ以外は成功
- この判定は PoC 用ロジックとしてコード内に直接持たせる

### Catch 設計

- 親タスク失敗時
  - Step Functions 側で捕捉し、失敗終了
- ワーカータスク失敗時
  - ワーカー入力と失敗理由を結果として保持する
  - 失敗ワーカーのみを再実行対象として再度 Map に流す
  - `retryLimit` 到達後は最終失敗一覧を出力して終了する
- Lambda 失敗時
  - Step Functions からは直接見えないため、SQS / Lambda / DLQ 側で観測

## 5. PoC で確認したい論点

### オーケストレーション

- 親タスク出力を Step Functions が次段へ安全に渡せるか
- Step Functions 実行入力を親タスクへ安全に渡せるか
- `Map` で ECS タスクを複数起動する構成がシンプルに書けるか
- 並列数 `MaxConcurrency = 2` を期待通り制御できるか
- Lambda の同時実行上限 `2` が期待通り効くか
- 失敗ワーカーのみを抽出して再実行するループを素直に表現できるか

### リトライ

- ECS の `runTask` 失敗に対する Step Functions の `Retry`
- アプリ内失敗時に ECS タスク終了コードが Step Functions にどう反映されるか
- ワーカー失敗結果の集約と再実行回数上限の制御
- SQS -> Lambda の再試行と DLQ の挙動

### 実装責務

- Step Functions は「順序・分岐・リトライ・失敗制御」
- 親コンテナは「ワーカー一覧の決定」
- ワーカーコンテナは「1件単位の処理と SQS 送信」
- Lambda は「SQS 消費後の軽量処理」

## 6. Terraform 実装対象

PoC では以下を Terraform 管理対象とします。

- VPC
  - PoC 用に新規作成
  - ECS Fargate 実行用サブネットを含む最小構成
- ECS Cluster
- ECS Task Definition
  - 親タスク用
  - ワーカータスク用
- IAM Role
  - Step Functions 実行ロール
  - ECS Task Execution Role
  - ECS Task Role
  - Lambda 実行ロール
- ECR Repository
  - 親タスク用
  - ワーカータスク用
- Step Functions State Machine
- SQS Queue
  - メインキュー
  - DLQ
- Lambda Function
  - 予約同時実行数 `2`
- CloudWatch Log Group

## 7. アプリ実装対象

### 親タスクコンテナ

- Step Functions 実行入力を受け取る
- 30 秒の遅延を入れて実行中状態を観察しやすくする
- 入力値をもとにワーカー一覧 JSON を組み立てる
- `retryLimit` など再実行制御に必要な値も後続へ引き継ぐ
- Step Functions が参照しやすい形式で出力する

### ワーカータスクコンテナ

- 単一ワーカー入力を受け取る
- 30 秒の遅延を入れた疑似処理を実行する
- 成功時は SQS にメッセージ送信
- 失敗時は入力値と失敗理由を結果として返せるようにする
- 失敗試験のために、特定の入力値で意図的に異常終了できるようにする

### Lambda

- SQS メッセージを受け取る
- 30 秒の遅延を入れた疑似処理を行う
- 結果は CloudWatch Logs に出力する
- 失敗試験のために、特定条件で例外送出できるようにする

## 8. PoC の簡易方針

PoC では以下の割り切りを行います。

- 親タスク、ワーカータスク、Lambda は Go で簡素に実装する
- 業務データはダミー JSON を使用する
- 永続ストアは使わない
- Lambda の結果保存先は設けず、CloudWatch Logs のみで確認する
- 外部通知は入れない
- まずは同期的な ECS 完了待ちを採用する
- 画面から挙動を追いやすくするため、親タスク、ワーカータスク、Lambda の各処理で 30 秒の遅延を入れる
- ワーカー再実行は Step Functions 内で最大回数を管理し、無限ループを防ぐ

## 9. 事前に決めておきたい点

- VPC は PoC 用に新規作成する
- ECS は Fargate 固定
- Lambda の処理結果は CloudWatch Logs のみで確認する
- 親タスクへ渡す実行入力の項目
- ワーカー再実行回数の上限値
- 並列起動数の上限は Step Functions 側 2、Lambda 側 2

## 10. 次フェーズ

次に詳細化する内容は以下です。

1. Terraform ディレクトリ構成
2. Step Functions の state machine 定義
3. 親タスク・ワーカータスク・Lambda の入出力仕様
4. デプロイ手順
5. 実行確認シナリオ
6. 失敗注入シナリオ
