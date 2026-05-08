provider "aws" {
  region = var.aws_region
}

locals {
  name = var.project
}

module "network" {
  source   = "../../modules/network"
  name     = local.name
  az_count = var.az_count
}

module "logs" {
  source = "../../modules/logs"
  name   = local.name
}

module "sqs" {
  source = "../../modules/sqs"
  name   = local.name
}

module "s3" {
  source = "../../modules/s3"
  name   = local.name
}

module "iam" {
  source               = "../../modules/iam"
  name                 = local.name
  region               = var.aws_region
  account_id           = data.aws_caller_identity.current.account_id
  sqs_queue_arn        = module.sqs.queue_arn
  sqs_queue_url        = module.sqs.queue_url
  workers_bucket_arn   = module.s3.workers_bucket_arn
  parent_log_group_arn = module.logs.parent_log_group_arn
  worker_log_group_arn = module.logs.worker_log_group_arn
}

module "ecs" {
  source                = "../../modules/ecs"
  name                  = local.name
  parent_image_uri      = var.parent_image_uri
  worker_image_uri      = var.worker_image_uri
  parent_container_name = var.parent_container_name
  worker_container_name = var.worker_container_name
  parent_task_role_arn  = module.iam.parent_task_role_arn
  worker_task_role_arn  = module.iam.worker_task_role_arn
  execution_role_arn    = module.iam.ecs_task_execution_role_arn
  parent_log_group_name = module.logs.parent_log_group_name
  worker_log_group_name = module.logs.worker_log_group_name
  aws_region            = var.aws_region
  sqs_queue_url         = module.sqs.queue_url
  workers_s3_bucket     = module.s3.workers_bucket_name
}

module "lambda" {
  source                 = "../../modules/lambda"
  name                   = local.name
  sqs_queue_arn          = module.sqs.queue_arn
  lambda_role_arn        = module.iam.lambda_role_arn
  aggregator_role_arn    = module.iam.aggregator_lambda_role_arn
  sqs_consumer_image_uri = var.lambda_image_uri
}

module "stepfunctions" {
  source                       = "../../modules/stepfunctions"
  name                         = local.name
  state_machine_role_arn       = module.iam.stepfunctions_role_arn
  parent_cluster_arn           = module.ecs.cluster_arn
  worker_cluster_arn           = module.ecs.cluster_arn
  parent_task_definition_arn   = module.ecs.parent_task_definition_arn
  worker_task_definition_arn   = module.ecs.worker_task_definition_arn
  subnet_ids                   = module.network.private_subnet_ids
  security_group_id            = module.network.ecs_security_group_id
  parent_container_name        = var.parent_container_name
  worker_container_name        = var.worker_container_name
  workers_s3_bucket            = module.s3.workers_bucket_name
  result_aggregator_lambda_arn = module.lambda.result_aggregator_lambda_arn
  workflow_template_file       = "${path.root}/../../../statemachine/workflow.asl.json"
}

data "aws_caller_identity" "current" {}
