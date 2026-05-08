# workflow.asl.json 解説

このドキュメントは [workflow.asl.json](/home/dev/infra/statemachine/workflow.asl.json) の意図と、State ごとの役割を整理したものです。

## 1. 全体像

このステートマシンは次を実現します。

1. 実行入力（`batchId`, `text`）を検証する
2. 親 ECS タスク（`RunParentTaskSync`）を実行し、`workers/<batchId>.json` を S3 に生成する
3. S3 の workers JSON を `Distributed Map + ItemReader` で直接読み、ワーカー ECS を並列実行する
4. 集約 Lambda（`AggregateWorkerTaskAttemptResults`）で部分失敗を抽出する
5. `RETRYABLE` な失敗だけ再実行し、上限到達時は `failedWorkersFinal` に確定して終了する

## 2. State ごとの意味

### `ValidateWorkflowInput`

- `batchId` と `text` の存在をチェック。
- 不足時は `FailInvalidWorkflowInput`。

### `InitializeWorkflowContext`

- 以降で使う共通データを初期化。
- `workersS3Key = workers/<batchId>.json`
- `retryCount = 0`
- `results = []`
- `allResults = []`
- `failedWorkersFinal = []`

### `RunParentTaskSync`

- `ecs:runTask.sync` で親タスクを実行。
- 親には以下を環境変数で渡す。
  - `STEP_FUNCTIONS_INPUT`
  - `WORKERS_S3_BUCKET`
  - `WORKERS_S3_KEY`
- 親タスクは workers JSON を S3 に書き込む。
- 技術失敗のみ `Retry`。

### `ValidateParentTaskCompletion`

- 親コンテナの `ExitCode == 0` を確認。
- 失敗時は `FailParentTaskExecution`。

### `RunInitialWorkerTasksFromS3`（初回Map）

- `Distributed Map`。
- `ItemReader` で S3 の workers JSON を直接読み取る。
- `ItemSelector` で `jobId/workerId/payload/retryCount` などを各 worker item に渡す。

Map 内部:
- `RunInitialWorkerTaskSync`:
  - `ecs:runTask.sync` でワーカーを実行
  - 技術失敗のみ `Retry`
  - 最終失敗は `Catch` で `HandleInitialWorkerTaskTechnicalFailure`
- `NormalizeInitialWorkerTaskSuccess`:
  - 成功結果を標準フォーマットで返す
- `HandleInitialWorkerTaskTechnicalFailure`:
  - 技術失敗を `status=FAILED, errorType=RETRYABLE` に正規化

### `AggregateWorkerTaskAttemptResults`

- 集約 Lambda を呼び出し、今回の Map 実行結果を整理する。
- 役割:
  - 今回結果の `failedWorkers` 抽出
  - `NON_RETRYABLE` を `failedWorkersFinal` へ追加
  - `allResults` に今回結果を累積
  - `retryCount >= retryLimit` のとき `failedWorkers` を `failedWorkersFinal` に確定移送

### `HasRetryableFailedWorkerTasks`

- `failedWorkers` があるか判定。
- なければ `Finish`。

### `CanRetryFailedWorkerTasks`

- `retryCount < retryLimit` か判定。
- 条件を満たさなければ `FinishWithFailedWorkers`。

### `PrepareNextRetryWorkerTasks`

- `retryCount` を `+1`。
- 再実行対象を `workers = failedWorkers` に差し替え。
- `allResults` と `failedWorkersFinal` は引き継ぐ。
- 次は `RunRetryWorkerTasks` へ。

### `RunRetryWorkerTasks`（再試行Map）

- 再試行対象だけを `Distributed Map` で実行。
- Map 内の構成は初回Mapと同じ責務。
  - `RunRetryWorkerTaskSync`
  - `NormalizeRetryWorkerTaskSuccess`
  - `HandleRetryWorkerTaskTechnicalFailure`
- 完了後は再び `AggregateWorkerTaskAttemptResults`。

### 終端 State

- `Finish`: 正常終了（`Succeed`）
- `FinishWithFailedWorkers`: 失敗残ありで終了（`Succeed` へ進む）
- `FailInvalidWorkflowInput`: 入力不正
- `FailParentTaskExecution`: 親タスク失敗

## 3. 重要なデータ項目

- `workersS3Key`: 親が workers を保存する S3 key
- `results`: その時点の Map 実行結果
- `allResults`: 全試行の累積結果
- `failedWorkers`: 次回再試行対象（RETRYABLE）
- `failedWorkersFinal`: 最終失敗として確定した一覧

## 4. リトライの考え方

- `Task.Retry`: ECS起動失敗やタイムアウトなど「技術失敗」の吸収
- 部分失敗の制御: `Map -> 集約 -> 失敗抽出 -> 再Map`

つまり、部分失敗は `Retry` だけでなく、Step Functions の状態遷移で扱っています。

## 5. 実行入力例

```json
{
  "batchId": "batch-20260508-001",
  "text": "hello!"
}
```

同一 `batchId` の再実行は、親タスク側の重複チェック（同一 S3 key 既存）で失敗させる設計です。
