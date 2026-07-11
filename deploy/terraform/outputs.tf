# GGID Terraform Outputs

output "gateway_url" {
  description = "GGID Gateway URL"
  value       = "https://${var.domain}"
}

output "gateway_service_name" {
  description = "Gateway Kubernetes service name"
  value       = "${helm_release.ggid.name}-ggid-gateway"
}

output "auth_service_name" {
  description = "Auth Kubernetes service name"
  value       = "${helm_release.ggid.name}-ggid-auth"
}

output "identity_service_name" {
  description = "Identity Kubernetes service name"
  value       = "${helm_release.ggid.name}-ggid-identity"
}

output "oauth_service_name" {
  description = "OAuth Kubernetes service name"
  value       = "${helm_release.ggid.name}-ggid-oauth"
}

output "policy_service_name" {
  description = "Policy Kubernetes service name"
  value       = "${helm_release.ggid.name}-ggid-policy"
}

output "org_service_name" {
  description = "Org Kubernetes service name"
  value       = "${helm_release.ggid.name}-ggid-org"
}

output "audit_service_name" {
  description = "Audit Kubernetes service name"
  value       = "${helm_release.ggid.name}-ggid-audit"
}

output "namespace" {
  description = "Kubernetes namespace where GGID is deployed"
  value       = kubernetes_namespace.ggid.metadata[0].name
}

output "helm_release_status" {
  description = "Helm release status"
  value       = helm_release.ggid.status
}
