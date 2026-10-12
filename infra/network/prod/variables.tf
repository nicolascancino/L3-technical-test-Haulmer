variable "aws_region" {
  description = "AWS region where the resources are created."
  type        = string
}

variable "environment" {
  description = "Environment name (dev or prod)."
  type        = string
}
