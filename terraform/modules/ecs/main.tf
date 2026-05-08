variable "name" { type = string }
variable "parent_image_uri" { type = string }
variable "worker_image_uri" { type = string }
variable "parent_container_name" { type = string }
variable "worker_container_name" { type = string }
variable "parent_task_role_arn" { type = string }
variable "worker_task_role_arn" { type = string }
variable "execution_role_arn" { type = string }
variable "parent_log_group_name" { type = string }
variable "worker_log_group_name" { type = string }
variable "aws_region" { type = string }
variable "sqs_queue_url" { type = string }
variable "workers_s3_bucket" { type = string }

resource "aws_ecs_cluster" "this" {
  name = "${var.name}-cluster"
}

resource "aws_ecs_task_definition" "parent" {
  family                   = "${var.name}-parent"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = var.execution_role_arn
  task_role_arn            = var.parent_task_role_arn

  container_definitions = jsonencode([
    {
      name      = var.parent_container_name
      image     = var.parent_image_uri
      essential = true
      environment = [
        {
          name  = "WORKERS_S3_BUCKET"
          value = var.workers_s3_bucket
        }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = var.parent_log_group_name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "ecs"
        }
      }
    }
  ])
}

resource "aws_ecs_task_definition" "worker" {
  family                   = "${var.name}-worker"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "256"
  memory                   = "512"
  execution_role_arn       = var.execution_role_arn
  task_role_arn            = var.worker_task_role_arn

  container_definitions = jsonencode([
    {
      name      = var.worker_container_name
      image     = var.worker_image_uri
      essential = true
      environment = [
        {
          name  = "SQS_QUEUE_URL"
          value = var.sqs_queue_url
        }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = var.worker_log_group_name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "ecs"
        }
      }
    }
  ])
}

output "cluster_arn" { value = aws_ecs_cluster.this.arn }
output "parent_task_definition_arn" { value = aws_ecs_task_definition.parent.arn }
output "worker_task_definition_arn" { value = aws_ecs_task_definition.worker.arn }
