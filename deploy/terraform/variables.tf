# GGID Terraform Variables

variable "kubeconfig_path" {
  description = "Path to kubeconfig file"
  type        = string
  default     = "~/.kube/config"
}

variable "namespace" {
  description = "Kubernetes namespace for GGID"
  type        = string
  default     = "ggid"
}

variable "chart_version" {
  description = "GGID Helm chart version"
  type        = string
  default     = "1.0.0"
}

variable "image_registry" {
  description = "Container image registry (e.g., ghcr.io/ggid, registry.iot2.win)"
  type        = string
  default     = ""
}

variable "storage_class" {
  description = "Kubernetes StorageClass for PVCs"
  type        = string
  default     = ""
}

variable "domain" {
  description = "Domain name for GGID (e.g., iam.example.com)"
  type        = string
}

variable "ingress_class" {
  description = "Ingress controller class"
  type        = string
  default     = "nginx"
}

variable "tls_issuer" {
  description = "cert-manager ClusterIssuer name"
  type        = string
  default     = "letsencrypt-prod"
}

# ---- Secrets ----

variable "jwt_secret" {
  description = "JWT signing secret (shared by gateway, auth, oauth). Generate: openssl rand -base64 32"
  type        = string
  sensitive   = true
}

variable "db_password" {
  description = "PostgreSQL password"
  type        = string
  sensitive   = true
}

variable "redis_password" {
  description = "Redis password"
  type        = string
  sensitive   = true
  default     = ""
}

# ---- Sizing ----

variable "db_storage_size" {
  description = "PostgreSQL PVC size"
  type        = string
  default     = "20Gi"
}

variable "gateway_replicas" {
  description = "Number of gateway replicas"
  type        = number
  default     = 2
}
