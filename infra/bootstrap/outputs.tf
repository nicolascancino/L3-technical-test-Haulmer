output "state_bucket" {
  description = "Bucket to reference in every backend.tf."
  value       = aws_s3_bucket.state.id
}

output "plan_role_arn" {
  description = "Set as the AWS_PLAN_ROLE_ARN repository variable in GitHub."
  value       = aws_iam_role.plan.arn
}

output "apply_role_arn" {
  description = "Set as the AWS_APPLY_ROLE_ARN variable of the prod GitHub environment."
  value       = aws_iam_role.apply.arn
}
