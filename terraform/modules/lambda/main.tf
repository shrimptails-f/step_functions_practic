variable "name" { type = string }
variable "sqs_queue_arn" { type = string }
variable "lambda_role_arn" { type = string }
variable "aggregator_role_arn" { type = string }
variable "sqs_consumer_image_uri" { type = string }

locals {
  result_aggregator_dir = "${path.root}/../../lambda_src/result_aggregator"
}

data "archive_file" "result_aggregator_zip" {
  type        = "zip"
  source_dir  = local.result_aggregator_dir
  output_path = "${path.module}/result-aggregator.zip"
}

resource "aws_lambda_function" "sqs_consumer" {
  function_name = "${var.name}-sqs-consumer"
  role          = var.lambda_role_arn
  package_type  = "Image"
  image_uri     = var.sqs_consumer_image_uri
  timeout       = 30
}

resource "aws_lambda_function" "result_aggregator" {
  function_name    = "${var.name}-result-aggregator"
  role             = var.aggregator_role_arn
  runtime          = "python3.12"
  handler          = "main.handler"
  filename         = data.archive_file.result_aggregator_zip.output_path
  source_code_hash = data.archive_file.result_aggregator_zip.output_base64sha256
  timeout          = 30
}

resource "aws_lambda_event_source_mapping" "sqs_trigger" {
  event_source_arn                   = var.sqs_queue_arn
  function_name                      = aws_lambda_function.sqs_consumer.arn
  batch_size                         = 10
  maximum_batching_window_in_seconds = 1
  function_response_types            = ["ReportBatchItemFailures"]
}

output "result_aggregator_lambda_arn" { value = aws_lambda_function.result_aggregator.arn }
output "sqs_consumer_lambda_arn" { value = aws_lambda_function.sqs_consumer.arn }
