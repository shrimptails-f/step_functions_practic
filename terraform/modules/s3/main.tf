variable "name" { type = string }

resource "aws_s3_bucket" "workers" {
  bucket = "${var.name}-workers-${data.aws_caller_identity.current.account_id}"
}

resource "aws_s3_bucket_versioning" "workers" {
  bucket = aws_s3_bucket.workers.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "workers" {
  bucket = aws_s3_bucket.workers.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

data "aws_caller_identity" "current" {}

output "workers_bucket_name" { value = aws_s3_bucket.workers.bucket }
output "workers_bucket_arn" { value = aws_s3_bucket.workers.arn }
