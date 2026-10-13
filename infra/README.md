# Infrastructure

Terraform infrastructure for the Cards API on AWS (ECS Fargate).

## Layout

```text
bootstrap/        State bucket (applied once, local state)
iam/              GitHub OIDC provider and plan, apply and CI roles (applied once by hand)
ecr/              Image repository (permanent, not destroyed after tests)
network/          VPC, subnets, security groups
secrets/          Secrets Manager
rds/              RDS Postgres
alb/              Application Load Balancer (HTTPS)
ecs/              ECS (Fargate) cluster and services
cloudwatch/       Log groups, alarms
  └── <component>/{dev,prod}   One Terraform state per component and environment
```

State key: `<component>/<env>/terraform.tfstate` in the bucket created by `bootstrap/`.

## First-time setup

1. Authenticate locally with an AWS admin profile.
2. Create the state bucket, then the GitHub roles (both by hand, once):
   ```bash
   cd infra/bootstrap
   terraform init && terraform apply
   cd ../iam
   terraform init && terraform apply
   ```
   If `haulmer-cards-tfstate` is taken, change `state_bucket_name` and update
   `bucket` in `infra/iam/backend.tf` and every `infra/*/*/backend.tf`.
3. In GitHub, set:
   - repository variable `AWS_PLAN_ROLE_ARN` = `plan_role_arn` output of `iam`
   - repository variable `AWS_CI_ROLE_ARN` = `ci_role_arn` output of `iam`
   - environment `prod` variable `AWS_APPLY_ROLE_ARN` = `apply_role_arn` output of `iam`
   - required reviewers on the `prod` environment
4. Check connectivity:
   ```bash
   cd infra/network/prod
   terraform init && terraform plan   # prints the connected account_id
   ```

## Pipeline

`.github/workflows/terraform.yml` (paths under `infra/`)

- Pull request: `fmt`, `validate` and `plan` for changed stacks (read-only role).
- Merge to `main`: `apply` for changed stacks, in dependency order, through the `prod` environment.
- Only environments listed in `ENABLED_ENVS` run (currently `prod`). `dev/` folders exist but are not applied.

## Cost notes

NAT Gateway, VPC endpoints and the ALB have fixed monthly costs. Create an AWS Budget alert
before deploying and run `terraform destroy` (reverse order) after testing.
