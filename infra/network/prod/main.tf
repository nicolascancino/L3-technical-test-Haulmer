# Connectivity check: confirms Terraform can reach AWS with the current credentials.
data "aws_caller_identity" "current" {}

output "account_id" {
  description = "AWS account this stack is connected to."
  value       = data.aws_caller_identity.current.account_id
}

# TODO: VPC, public/private subnets, optional NAT gateway and VPC endpoints.
