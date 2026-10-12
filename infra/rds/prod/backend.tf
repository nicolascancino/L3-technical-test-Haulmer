terraform {
  backend "s3" {
    bucket       = "haulmer-cards-tfstate"
    key          = "rds/prod/terraform.tfstate"
    region       = "us-east-1"
    encrypt      = true
    use_lockfile = true
  }
}
