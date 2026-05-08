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

## Docker Build / ECR Push（ローカル環境で実行）

前提:
- Docker と AWS CLI が利用可能
- AWS ログイン済み
- リージョンは `ap-northeast-1`（必要に応じて変更）

使用する ECR リポジトリ名（手動作成前提）:
- `sfn-ecs-poc-parent`
- `sfn-ecs-poc-worker`
- `sfn-ecs-poc-lambda-handler`

### 1. 変数を設定

```bash
export AWS_REGION=ap-northeast-1
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
export PARENT_REPO=${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/sfn-ecs-poc-parent
export WORKER_REPO=${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/sfn-ecs-poc-worker
export LAMBDA_REPO=${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/sfn-ecs-poc-lambda-handler
```

### 2. ECR リポジトリ作成（未作成の場合）

```bash
aws ecr create-repository --repository-name sfn-ecs-poc-parent --region ${AWS_REGION} || true
aws ecr create-repository --repository-name sfn-ecs-poc-worker --region ${AWS_REGION} || true
aws ecr create-repository --repository-name sfn-ecs-poc-lambda-handler --region ${AWS_REGION} || true
```

### 3. ECR ログイン

```bash
aws ecr get-login-password --region ${AWS_REGION} \
  | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com
```

### 4. イメージをビルド

```bash
docker build -t ${PARENT_REPO}:latest ./apps/parent-task
docker build -t ${WORKER_REPO}:latest ./apps/worker-task
docker build -t ${LAMBDA_REPO}:latest ./apps/lambda-handler
```

### 5. ECR に push

```bash
docker push ${PARENT_REPO}:latest
docker push ${WORKER_REPO}:latest
docker push ${LAMBDA_REPO}:latest
```

### 6. Terraform 変数に反映（envs/poc/terraform.tfvars）

```hcl
parent_image_uri = "<AWS_ACCOUNT_ID>.dkr.ecr.ap-northeast-1.amazonaws.com/sfn-ecs-poc-parent:latest"
worker_image_uri = "<AWS_ACCOUNT_ID>.dkr.ecr.ap-northeast-1.amazonaws.com/sfn-ecs-poc-worker:latest"
lambda_image_uri = "<AWS_ACCOUNT_ID>.dkr.ecr.ap-northeast-1.amazonaws.com/sfn-ecs-poc-lambda-handler:latest"
```
