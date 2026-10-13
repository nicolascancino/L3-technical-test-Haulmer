data "terraform_remote_state" "alb" {
  backend = "s3"

  config = {
    bucket = "haulmer-cards-tfstate"
    key    = "alb/${var.environment}/terraform.tfstate"
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

resource "aws_cloudwatch_metric_alarm" "target_5xx" {
  alarm_name          = "haulmer-cards-${var.environment}-target-5xx"
  alarm_description   = "The API returned 5 or more 5xx responses in one minute"
  namespace           = "AWS/ApplicationELB"
  metric_name         = "HTTPCode_Target_5XX_Count"
  statistic           = "Sum"
  period              = 60
  evaluation_periods  = 1
  threshold           = 5
  comparison_operator = "GreaterThanOrEqualToThreshold"
  treat_missing_data  = "notBreaching"

  dimensions = {
    LoadBalancer = data.terraform_remote_state.alb.outputs.lb_arn_suffix
    TargetGroup  = data.terraform_remote_state.alb.outputs.target_group_arn_suffix
  }
}

resource "aws_cloudwatch_metric_alarm" "unhealthy_targets" {
  alarm_name          = "haulmer-cards-${var.environment}-unhealthy-targets"
  alarm_description   = "At least one task failed the /health check for 2 minutes"
  namespace           = "AWS/ApplicationELB"
  metric_name         = "UnHealthyHostCount"
  statistic           = "Maximum"
  period              = 60
  evaluation_periods  = 2
  threshold           = 1
  comparison_operator = "GreaterThanOrEqualToThreshold"
  treat_missing_data  = "notBreaching"

  dimensions = {
    LoadBalancer = data.terraform_remote_state.alb.outputs.lb_arn_suffix
    TargetGroup  = data.terraform_remote_state.alb.outputs.target_group_arn_suffix
  }
}

resource "aws_cloudwatch_metric_alarm" "rds_cpu" {
  alarm_name          = "haulmer-cards-${var.environment}-rds-cpu"
  alarm_description   = "Database CPU above 80% for 5 minutes"
  namespace           = "AWS/RDS"
  metric_name         = "CPUUtilization"
  statistic           = "Average"
  period              = 60
  evaluation_periods  = 5
  threshold           = 80
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"

  dimensions = {
    DBInstanceIdentifier = data.terraform_remote_state.rds.outputs.db_identifier
  }
}
