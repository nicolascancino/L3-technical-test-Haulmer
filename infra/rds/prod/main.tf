data "terraform_remote_state" "network" {
  backend = "s3"

  config = {
    bucket = "haulmer-cards-tfstate"
    key    = "network/${var.environment}/terraform.tfstate"
    region = var.aws_region
  }
}

data "aws_secretsmanager_secret_version" "db_password" {
  secret_id = "haulmer-cards/${var.environment}/db-password"
}

resource "aws_db_subnet_group" "main" {
  name       = "haulmer-cards-${var.environment}"
  subnet_ids = data.terraform_remote_state.network.outputs.private_subnet_ids
}

resource "aws_db_instance" "main" {
  identifier = "haulmer-cards-${var.environment}"

  engine         = "postgres"
  engine_version = "16"
  instance_class = "db.t4g.micro"

  allocated_storage = 20
  storage_type      = "gp3"
  storage_encrypted = true

  db_name  = "cards"
  username = "cards_admin"
  password = data.aws_secretsmanager_secret_version.db_password.secret_string

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [data.terraform_remote_state.network.outputs.rds_security_group_id]
  publicly_accessible    = false
  multi_az               = false

  # Ephemeral test environment: no final snapshot so destroy leaves nothing behind.
  skip_final_snapshot     = true
  backup_retention_period = 0
}

output "db_host" {
  value = aws_db_instance.main.address
}

output "db_name" {
  value = aws_db_instance.main.db_name
}

output "db_user" {
  value = aws_db_instance.main.username
}

output "db_identifier" {
  value = aws_db_instance.main.identifier
}
