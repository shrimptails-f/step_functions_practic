variable "project" {
  type        = string
  description = "Project name prefix"
  default     = "sfn-ecs-poc"
}

variable "aws_region" {
  type        = string
  description = "AWS region"
  default     = "ap-northeast-1"
}

variable "az_count" {
  type        = number
  description = "Number of AZs/subnets"
  default     = 2
}

variable "parent_image_uri" {
  type        = string
  description = "ECR image URI for parent ECS task"
}

variable "worker_image_uri" {
  type        = string
  description = "ECR image URI for worker ECS task"
}

variable "lambda_image_uri" {
  type        = string
  description = "ECR image URI for lambda-handler"
}

variable "parent_container_name" {
  type    = string
  default = "parent-task"
}

variable "worker_container_name" {
  type    = string
  default = "worker-task"
}
