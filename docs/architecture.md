# アーキテクチャ

## 目的

このドキュメントでは、Step Functions + ECS PoC のベースとなるディレクトリ構成を定義します。

## ディレクトリ構成

```text
infra/
├── terraform/
│   ├── envs/
│   │   └── poc/
│   │       ├── main.tf
│   │       ├── variables.tf
│   │       ├── terraform.tfvars.example
│   │       └── outputs.tf
│   ├── modules/
│   │   ├── network/
│   │   ├── ecs/
│   │   ├── s3/
│   │   ├── sqs/
│   │   ├── lambda/
│   │   ├── stepfunctions/
│   │   ├── iam/
│   │   └── logs/
│   └── versions.tf
├── apps/
│   ├── parent-task/
│   │   ├── main.go
│   │   ├── go.mod
│   │   ├── go.sum
│   │   └── Dockerfile
│   ├── worker-task/
│   │   ├── main.go
│   │   ├── go.mod
│   │   ├── go.sum
│   │   └── Dockerfile
│   └── lambda-handler/
│       ├── main.go
│       ├── go.mod
│       ├── go.sum
│       └── Dockerfile
└── statemachine/
    ├── workflow.asl.json
    └── input.example.json

```

## 補足

- `apps/` は親タスク、ワーカータスク、Lambda ハンドラーを Go で実装する前提です。
- `statemachine/workflow.asl.json` は ASL の定義本体とし、Terraform (`file` または `templatefile`) から読み込みます。
- ECR リポジトリは Terraform ではなく手動作成・手動 push 前提です。
- この PoC では、シェルスクリプト (`.sh`) は意図的に使用しません。
- 運用手順は `../README.md` に記載します。
- このコンテナ内では Docker コマンドを実行せず、ECR へのイメージ push はコンテナ外で手動対応します。
