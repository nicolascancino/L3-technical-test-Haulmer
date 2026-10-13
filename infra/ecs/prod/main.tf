locals {
  name = "haulmer-cards-${var.environment}"
}

data "terraform_remote_state" "network" {
  backend = "s3"

  config = {
    bucket = "haulmer-cards-tfstate"
    key    = "network/${var.environment}/terraform.tfstate"
    region = var.aws_region
  }
}

data "terraform_remote_state" "secrets" {
  backend = "s3"

  config = {
    bucket = "haulmer-cards-tfstate"
    key    = "secrets/${var.environment}/terraform.tfstate"
    region = var.aws_region
  }
}

data "terraform_remote_state" "rds" {
  backend = "s3"

  config = {
    bucket = "haulmer-cards-tfstate"
    key    = "rds/${var.environment}/terraform.tfstate"
    region = var.aws_region
  }
}

data "terraform_remote_state" "alb" {
  backend = "s3"

  config = {
    bucket = "haulmer-cards-tfstate"
    key    = "alb/${var.environment}/terraform.tfstate"
    region = var.aws_region
  }
}

data "terraform_remote_state" "ecr" {
  backend = "s3"

  config = {
    bucket = "haulmer-cards-tfstate"
    key    = "ecr/${var.environment}/terraform.tfstate"
    region = var.aws_region
  }
}

resource "aws_cloudwatch_log_group" "api" {
  name              = "/ecs/${local.name}"
  retention_in_days = 7
}

# Execution role: used by ECS itself to pull the image and write logs.
resource "aws_iam_role" "execution" {
  name = "${local.name}-execution"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "execution" {
  role       = aws_iam_role.execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# Task role: used by the application code, only to read its two secrets.
resource "aws_iam_role" "task" {
  name = "${local.name}-task"

  assume_role_policy = aws_iam_role.execution.assume_role_policy
}

resource "aws_iam_role_policy" "read_secrets" {
  name = "read-secrets"
  role = aws_iam_role.task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = "secretsmanager:GetSecretValue"
      Resource = [
        data.terraform_remote_state.secrets.outputs.db_password_secret_arn,
        data.terraform_remote_state.secrets.outputs.pan_salt_secret_arn,
      ]
    }]
  })
}

resource "aws_ecs_cluster" "main" {
  name = local.name
}

resource "aws_ecs_task_definition" "api" {
  family                   = local.name
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.task.arn

  container_definitions = jsonencode([{
    name         = "api"
    image        = "${data.terraform_remote_state.ecr.outputs.repository_url}:latest"
    essential    = true
    portMappings = [{ containerPort = 8080 }]

    environment = [
      { name = "APP_ENV", value = "cloud" },
      { name = "AWS_REGION", value = var.aws_region },
      { name = "LOG_FORMAT", value = "json" },
      { name = "DB_HOST", value = data.terraform_remote_state.rds.outputs.db_host },
      { name = "DB_NAME", value = data.terraform_remote_state.rds.outputs.db_name },
      { name = "DB_USER", value = data.terraform_remote_state.rds.outputs.db_user },
      { name = "DB_PASSWORD_SECRET_ID", value = data.terraform_remote_state.secrets.outputs.db_password_secret_arn },
      { name = "PAN_SALT_SECRET_ID", value = data.terraform_remote_state.secrets.outputs.pan_salt_secret_arn },
    ]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.api.name
        "awslogs-region"        = var.aws_region
        "awslogs-stream-prefix" = "api"
      }
    }
  }])
}

resource "aws_ecs_service" "api" {
  name            = local.name
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.api.arn
  desired_count   = 3
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = data.terraform_remote_state.network.outputs.public_subnet_ids
    security_groups  = [data.terraform_remote_state.network.outputs.ecs_security_group_id]
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = data.terraform_remote_state.alb.outputs.target_group_arn
    container_name   = "api"
    container_port   = 8080
  }

  # CI registers new task definition revisions on every deploy; Terraform must not undo them.
  lifecycle {
    ignore_changes = [task_definition]
  }
}

output "cluster_name" {
  value = aws_ecs_cluster.main.name
}

output "service_name" {
  value = aws_ecs_service.api.name
}
