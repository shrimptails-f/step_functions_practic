variable "name" { type = string }

resource "aws_ecr_repository" "parent" {
  name                 = "${var.name}-parent"
  image_tag_mutability = "MUTABLE"
}

resource "aws_ecr_repository" "worker" {
  name                 = "${var.name}-worker"
  image_tag_mutability = "MUTABLE"
}

resource "aws_ecr_repository" "lambda_handler" {
  name                 = "${var.name}-lambda-handler"
  image_tag_mutability = "MUTABLE"
}

output "parent_repository_url" { value = aws_ecr_repository.parent.repository_url }
output "worker_repository_url" { value = aws_ecr_repository.worker.repository_url }
output "lambda_handler_repository_url" { value = aws_ecr_repository.lambda_handler.repository_url }
