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
