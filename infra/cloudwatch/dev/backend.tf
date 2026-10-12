terraform {
  backend "s3" {
    bucket       = "haulmer-cards-tfstate"
    key          = "cloudwatch/dev/terraform.tfstate"
    region       = "us-east-1"
    encrypt      = true
    use_lockfile = true
  }
}
