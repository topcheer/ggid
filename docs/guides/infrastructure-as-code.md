# Infrastructure as Code

Terraform module structure, state management, drift detection, per-environment config, secrets in IaC, and CI/CD integration.

## Module Structure

```
infra/
├── modules/
│   ├── vpc/              # Network
│   ├── eks/              # Kubernetes cluster
│   ├── rds/              # PostgreSQL
│   ├── elasticache/      # Redis
│   ├── nats/             # NATS JetStream
│   ├── cert-manager/     # TLS certificates
│   └── ggid-services/    # GGID microservices
├── environments/
│   ├── dev/              # Development
│   ├── staging/          # Pre-production
│   └── prod/             # Production
└── shared/               # Shared resources (DNS, IAM)
```

## State Management

### Remote Backend (S3 + DynamoDB)

```hcl
terraform {
  backend "s3" {
    bucket         = "ggid-tfstate"
    key            = "prod/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "ggid-tflock"  # State locking
  }
}
```

### State Rules

| Rule | Enforcement |
|------|-------------|
| Never commit .tfstate | .gitignore |
| Always use remote backend | CI validates |
| Lock during apply | DynamoDB |
| One state per environment | Separate keys |
| State contains no secrets | Use variables from Vault |

## Drift Detection

```bash
# Nightly drift check
terraform plan -detailed-exitcode
# Exit 0: no changes
# Exit 2: drift detected → alert
```

```yaml
drift_check:
  schedule: "0 3 * * *"
  steps:
    - run: terraform plan -detailed-exitcode
    - if: exit_code == 2
      run: |
        terraform plan -no-color > drift.txt
        slack-alert "Infrastructure drift detected in prod" --file drift.txt
```

## Per-Environment Config

```hcl
# environments/prod/main.tf
module "ggid" {
  source = "../../modules/ggid-services"

  environment      = "prod"
  cluster_endpoint = module.eks.endpoint
  db_host          = module.rds.endpoint
  redis_host       = module.elasticache.endpoint

  replicas = {
    gateway   = 3
    auth      = 3
    identity  = 2
  }

  db_pool_size = 25
  redis_size   = 20

  feature_flags = {
    new_auth_flow = false
  }
}
```

## Secrets in IaC

```hcl
# NEVER hardcode secrets
# BAD
db_password = "super-secret"

# GOOD: from Vault
data "vault_generic_secret" "db" {
  path = "secret/ggid/prod/db"
}

module "rds" {
  source = "../../modules/rds"
  password = data.vault_generic_secret.db.data["password"]
}
```

## CI/CD Integration

```yaml
terraform_deploy:
  steps:
    - name: fmt-check
      run: terraform fmt -check -recursive

    - name: validate
      run: terraform validate

    - name: plan
      run: terraform plan -out=tfplan

    - name: apply (staging auto, prod manual)
      if: github.ref == 'refs/heads/main'
      run: terraform apply -auto-approve tfplan
      environment: production  # Requires manual approval
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Drift detected | Any → review |
| Apply failures | Any → investigate |
| State lock stuck | >5 min → force unlock |
| Plan diff size | >50 resources → review |

## See Also

- [Blue-Green Deployment](blue-green-deployment.md)
- [Zero-Downtime Deployment](zero-downtime-deployment.md)
- [Secret Sprawl Prevention](secret-sprawl-prevention.md)
- [Cost Monitoring](cost-monitoring.md)
