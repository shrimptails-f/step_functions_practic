variable "name" { type = string }

resource "aws_cloudwatch_log_group" "parent" {
  name              = "/ecs/${var.name}-parent"
  retention_in_days = 14
}

resource "aws_cloudwatch_log_group" "worker" {
  name              = "/ecs/${var.name}-worker"
  retention_in_days = 14
}

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${var.name}-sqs-consumer"
  retention_in_days = 14
}

resource "aws_cloudwatch_log_group" "aggregator" {
  name              = "/aws/lambda/${var.name}-result-aggregator"
  retention_in_days = 14
}

output "parent_log_group_name" { value = aws_cloudwatch_log_group.parent.name }
output "worker_log_group_name" { value = aws_cloudwatch_log_group.worker.name }
output "parent_log_group_arn" { value = aws_cloudwatch_log_group.parent.arn }
output "worker_log_group_arn" { value = aws_cloudwatch_log_group.worker.arn }
