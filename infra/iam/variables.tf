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

variable "github_repo" {
  description = "GitHub repository allowed to assume the roles, as owner/name."
  type        = string
  default     = "nicolascancino/L3-technical-test-Haulmer-"
}
