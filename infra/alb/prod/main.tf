data "terraform_remote_state" "network" {
  backend = "s3"

  config = {
    bucket = "haulmer-cards-tfstate"
    key    = "network/${var.environment}/terraform.tfstate"
    region = var.aws_region
  }
}

# Self-signed certificate: free, enough for the test. A real domain with an ACM-issued certificate is the production path.
resource "tls_private_key" "main" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "tls_self_signed_cert" "main" {
  private_key_pem = tls_private_key.main.private_key_pem

  subject {
    common_name = "haulmer-cards.local"
  }

  validity_period_hours = 720
  allowed_uses          = ["key_encipherment", "digital_signature", "server_auth"]
}

resource "aws_acm_certificate" "main" {
  private_key      = tls_private_key.main.private_key_pem
  certificate_body = tls_self_signed_cert.main.cert_pem
}

resource "aws_lb" "main" {
  name               = "haulmer-cards-${var.environment}"
  load_balancer_type = "application"
  subnets            = data.terraform_remote_state.network.outputs.public_subnet_ids
  security_groups    = [data.terraform_remote_state.network.outputs.alb_security_group_id]
}

resource "aws_lb_target_group" "api" {
  name        = "haulmer-cards-${var.environment}"
  port        = 8080
  protocol    = "HTTP"
  target_type = "ip"
  vpc_id      = data.terraform_remote_state.network.outputs.vpc_id

  health_check {
    path    = "/health"
    matcher = "200"
  }
}

resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.main.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = aws_acm_certificate.main.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.api.arn
  }
}

output "dns_name" {
  value = aws_lb.main.dns_name
}

output "target_group_arn" {
  value = aws_lb_target_group.api.arn
}

output "lb_arn_suffix" {
  value = aws_lb.main.arn_suffix
}

output "target_group_arn_suffix" {
  value = aws_lb_target_group.api.arn_suffix
}
