# GGID Terraform Module
# Deploys GGID IAM platform via Helm on Kubernetes
# Usage: terraform init && terraform apply

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = ">= 2.12.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.26.0"
    }
  }
}

# ---- Providers ----

provider "helm" {
  kubernetes {
    config_path = var.kubeconfig_path
  }
}

provider "kubernetes" {
  config_path = var.kubeconfig_path
}

# ---- Namespace ----

resource "kubernetes_namespace" "ggid" {
  metadata {
    name = var.namespace
    labels = {
      "app.kubernetes.io/part-of" = "ggid"
    }
  }
}

# ---- GGID Helm Release ----

resource "helm_release" "ggid" {
  name       = "ggid"
  repository = "https://charts.ggid.dev"
  chart      = "ggid"
  version    = var.chart_version
  namespace  = kubernetes_namespace.ggid.metadata[0].name

  create_namespace = false
  wait             = true
  wait_for_jobs    = true
  timeout          = 300

  values = [
    yamlencode({
      global = {
        imageRegistry = var.image_registry
        storageClass  = var.storage_class
      }

      # External databases — when set, disable bundled services
      postgresql = {
        enabled = var.external_database_host == ""
        auth = {
          password = var.db_password
        }
        primary = {
          persistence = {
            size = var.db_storage_size
          }
        }
      }

      externalDatabase = {
        host     = var.external_database_host
        port     = var.external_database_port
        username = "ggid"
        password = var.db_password
        database = "ggid"
      }

      redis = {
        enabled = var.external_redis_host == ""
        auth = {
          password = var.redis_password
        }
      }

      externalRedis = {
        host     = var.external_redis_host
        port     = var.external_redis_port
        password = var.redis_password
      }

      nats = {
        enabled = var.external_nats_host == ""
      }

      externalNats = {
        host = var.external_nats_host
        port = var.external_nats_port
      }

      gateway = {
        replicaCount = var.gateway_replicas
        env = {
          JWT_SECRET = var.jwt_secret
        }
      }

      auth = {
        env = {
          JWT_SECRET = var.jwt_secret
        }
      }

      oauth = {
        env = {
          JWT_SECRET   = var.jwt_secret
          OAUTH_ISSUER = "https://${var.domain}"
        }
      }

      ingress = {
        enabled  = true
        className = var.ingress_class
        annotations = {
          "cert-manager.io/cluster-issuer" = var.tls_issuer
        }
        hosts = [{
          host = var.domain
          paths = [{
            path     = "/"
            pathType = "Prefix"
          }]
        }]
        tls = [{
          secretName = "ggid-tls"
          hosts      = [var.domain]
        }]
      }
    })
  ]
}
