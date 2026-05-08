output "vpc_id" {
  value = module.network.vpc_id
}

output "ecs_cluster_arn" {
  value = module.ecs.cluster_arn
}

output "stepfunctions_state_machine_arn" {
  value = module.stepfunctions.state_machine_arn
}

output "sqs_queue_url" {
  value = module.sqs.queue_url
}

output "workers_bucket_name" {
  value = module.s3.workers_bucket_name
}
