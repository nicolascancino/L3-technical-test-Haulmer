terraform {
  backend "s3" {
    bucket       = "haulmer-cards-tfstate"
    key          = "ecr/prod/terraform.tfstate"
    region       = "us-east-1"
    encrypt      = true
    use_lockfile = true
  }
}
