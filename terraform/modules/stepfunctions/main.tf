variable "name" { type = string }
variable "state_machine_role_arn" { type = string }
variable "parent_cluster_arn" { type = string }
variable "worker_cluster_arn" { type = string }
variable "parent_task_definition_arn" { type = string }
variable "worker_task_definition_arn" { type = string }
variable "subnet_ids" { type = list(string) }
variable "security_group_id" { type = string }
variable "parent_container_name" { type = string }
variable "worker_container_name" { type = string }
variable "workers_s3_bucket" { type = string }
variable "result_aggregator_lambda_arn" { type = string }
variable "workflow_template_file" { type = string }

locals {
  # PoCなのでネストが深いのは一旦許容する
  definition = replace(
    replace(
      replace(
        replace(
          replace(
            replace(
              replace(
                replace(
                  replace(
                    templatefile(var.workflow_template_file, {}),
                    "__PARENT_ECS_CLUSTER_ARN__",
                    var.parent_cluster_arn
                  ),
                  "__WORKER_ECS_CLUSTER_ARN__",
                  var.worker_cluster_arn
                ),
                "__PARENT_TASK_DEFINITION_ARN__",
                var.parent_task_definition_arn
              ),
              "__WORKER_TASK_DEFINITION_ARN__",
              var.worker_task_definition_arn
            ),
            "__SUBNET_ID_1__",
            var.subnet_ids[0]
          ),
          "__SUBNET_ID_2__",
          var.subnet_ids[1]
        ),
        "__SECURITY_GROUP_ID__",
        var.security_group_id
      ),
      "__PARENT_CONTAINER_NAME__",
      var.parent_container_name
    ),
    "__WORKER_CONTAINER_NAME__",
    var.worker_container_name
  )

  with_workers_bucket = replace(local.definition, "__WORKERS_S3_BUCKET__", var.workers_s3_bucket)
  with_results_bucket = replace(local.with_workers_bucket, "__RESULTS_S3_BUCKET__", var.workers_s3_bucket)
  rendered_definition = replace(local.with_results_bucket, "__RESULT_AGGREGATOR_LAMBDA_ARN__", var.result_aggregator_lambda_arn)
}

resource "aws_sfn_state_machine" "this" {
  name       = "${var.name}-workflow"
  role_arn   = var.state_machine_role_arn
  definition = local.rendered_definition
  type       = "STANDARD"
}

output "state_machine_arn" { value = aws_sfn_state_machine.this.arn }
