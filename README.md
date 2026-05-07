# 環境構築手順
## ソースをクローン
## .envをコピー
```bash
cp .devcontainer/.env.sample .devcontainer/.env
```
## AWSのクレデンシャルを設定
`.devcontainer/.env` に以下を設定してください。

```dotenv
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
AWS_DEFAULT_REGION=ap-northeast-1
```

## VsCodeでプロジェクトフォルダーを開く
## Reopen in Containerを押下
Ctrl Shift P → Reopen in containerと入力して実行

## PoC実行手順の方針

- Step Functions + ECS PoC の実行手順は README に集約する。
- このコンテナ内では Docker コマンドを利用しない前提とする。
- ECR へのイメージ push はローカル環境などから手動で実施する。
- Terraform の `plan` / `apply` / `destroy` と Step Functions の実行手順は、この README に追記して管理する。
