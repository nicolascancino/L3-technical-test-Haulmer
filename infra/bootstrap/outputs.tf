output "state_bucket" {
  description = "Bucket to reference in every backend.tf."
  value       = aws_s3_bucket.state.id
}
