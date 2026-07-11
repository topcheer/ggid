# GGID Terraform Module

Deploys GGID IAM Platform on Kubernetes via Helm.

## Quick Start

```bash
# Create terraform.tfvars
cat > terraform.tfvars <<'EOF'
domain       = "iam.example.com"
jwt_secret   = "$(openssl rand -base64 32)"
db_password  = "super-secure-password"
EOF

# Deploy
terraform init
terraform plan
terraform apply
```

## Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `domain` | **Yes** | — | Domain name for GGID |
| `jwt_secret` | **Yes** | — | JWT signing secret |
| `db_password` | **Yes** | — | PostgreSQL password |
| `namespace` | No | `ggid` | K8s namespace |
| `image_registry` | No | `""` | Container registry |
| `storage_class` | No | `""` | PVC storage class |
| `ingress_class` | No | `nginx` | Ingress controller |
| `tls_issuer` | No | `letsencrypt-prod` | cert-manager issuer |
| `db_storage_size` | No | `20Gi` | DB PVC size |
| `gateway_replicas` | No | `2` | Gateway replicas |

## Outputs

| Output | Description |
|--------|-------------|
| `gateway_url` | `https://iam.example.com` |
| `namespace` | `ggid` |
| `gateway_service_name` | `ggid-ggid-gateway` |
| `helm_release_status` | `deployed` |

## Using a Private Registry

```hcl
# terraform.tfvars
image_registry = "registry.iot2.win"
domain         = "ggid.iot2.win"
```

## Destroy

```bash
terraform destroy
```

---

*Part of [GGID IAM Platform](https://github.com/ggid/ggid)*