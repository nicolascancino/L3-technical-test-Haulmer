output "plan_role_arn" {
  description = "Set as the AWS_PLAN_ROLE_ARN repository variable in GitHub."
  value       = aws_iam_role.plan.arn
}

output "apply_role_arn" {
  description = "Set as the AWS_APPLY_ROLE_ARN variable of the prod GitHub environment."
  value       = aws_iam_role.apply.arn
}

output "ci_role_arn" {
  description = "Set as the AWS_CI_ROLE_ARN repository variable in GitHub."
  value       = aws_iam_role.ci.arn
}
