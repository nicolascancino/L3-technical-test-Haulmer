resource "random_password" "db_password" {
  length  = 32
  special = false
}

resource "aws_secretsmanager_secret" "db_password" {
  name = "haulmer-cards/${var.environment}/db-password"

  # Delete immediately on destroy so the name can be reused by the next apply.
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "db_password" {
  secret_id     = aws_secretsmanager_secret.db_password.id
  secret_string = random_password.db_password.result
}

resource "random_password" "pan_salt" {
  length  = 32
  special = false
}

resource "aws_secretsmanager_secret" "pan_salt" {
  name = "haulmer-cards/${var.environment}/pan-salt"

  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "pan_salt" {
  secret_id     = aws_secretsmanager_secret.pan_salt.id
  secret_string = random_password.pan_salt.result
}

output "db_password_secret_arn" {
  value = aws_secretsmanager_secret.db_password.arn
}

output "pan_salt_secret_arn" {
  value = aws_secretsmanager_secret.pan_salt.arn
}
