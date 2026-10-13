variable "aws_region" {
  description = "AWS region where the resources are created."
  type        = string
  default     = "us-east-1"
}

variable "state_bucket_name" {
  description = "Name of the S3 bucket that stores Terraform state (created by bootstrap)."
  type        = string
  default     = "haulmer-cards-tfstate"
}

variable "github_subject" {
  description = "Repository part of the GitHub OIDC subject claim. This repository uses immutable subjects, so it is owner@owner_id/name@repo_id, not owner/name. Read it from the sub_claim_prefix of the repository OIDC customization API."
  type        = string
  default     = "nicolascancino@17253388/L3-technical-test-Haulmer-@1411043094"
}
