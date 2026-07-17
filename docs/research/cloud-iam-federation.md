# Cloud IAM Federation: AWS, Azure AD, and GCP Integration for GGID

> **Focus**: Making GGID act as a centralized Identity Provider (IdP) for cloud platforms — enabling users to authenticate once in GGID and access AWS, Azure, and GCP resources without separate cloud credentials.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Problem Statement](#2-problem-statement)
3. [Federation Protocols Overview](#3-federation-protocols-overview)
4. [AWS IAM Federation](#4-aws-iam-federation)
5. [Azure AD / Microsoft Entra ID Federation](#5-azure-ad--microsoft-entra-id-federation)
6. [Google Cloud Platform Federation](#6-google-cloud-platform-federation)
7. [GGID Current State Analysis](#7-ggid-current-state-analysis)
8. [Gap Analysis](#8-gap-analysis)
9. [Proposed Architecture](#9-proposed-architecture)
10. [Database Schema](#10-database-schema)
11. [API Design](#11-api-design)
12. [Claim Mapping Engine](#12-claim-mapping-engine)
13. [SCIM Provisioning to Cloud Providers](#13-scim-provisioning-to-cloud-providers)
14. [Security Considerations](#14-security-considerations)
15. [Console UI Design](#15-console-ui-design)
16. [Competitive Differentiation](#16-competitive-differentiation)
17. [Implementation Backlog](#17-implementation-backlog)

---

## 1. Executive Summary

Enterprise customers using GGID need to federate identity into **AWS IAM Identity Center**, **Azure AD / Microsoft Entra ID**, and **Google Cloud Platform** — so that a single GGID login grants access to cloud consoles, APIs, and resources across all three major cloud providers.

GGID already implements SAML 2.0 IdP endpoints (`/saml/idp/metadata`, `/saml/idp/sso`) and OIDC discovery (`/.well-known/openid-configuration`). However, **cloud-provider-specific federation configuration is missing**: there is no way to map GGID roles to AWS IAM roles, Azure AD app roles, or GCP workforce pool permissions. There is no wizard to generate the metadata/Terraform snippets needed to configure trust on the cloud side.

**Recommendation**: Build a **Cloud Federation Manager** — a set of cloud-provider-specific configuration modules that:
1. Generate SAML/OIDC metadata pre-configured for each cloud provider's requirements
2. Map GGID users/groups/roles to cloud-specific permission models (AWS IAM roles, Azure app roles, GCP workforce pool attributes)
3. Emit the correct SAML attributes / OIDC claims that cloud providers expect
4. Auto-provision users via SCIM 2.0 to AWS IAM Identity Center
5. Provide step-by-step setup wizards in the Console UI with copy-paste Terraform snippets

**Estimated effort**: 4 sprints for MVP (AWS + Azure SAML federation, claim mapping, Console wizard) + 2 sprints for GCP + SCIM.

---

## 2. Problem Statement

### The Multi-Cloud Identity Challenge

Organizations operating across AWS, Azure, and GCP face a fundamental identity problem:

```
┌──────────────────────────────────────────────────────────┐
│                  WITHOUT FEDERATION                       │
│                                                          │
│   User Alice has:                                        │
│   ├── AWS: iam-user:alice@prod (access key + secret)     │
│   ├── Azure: alice@corp.onmicrosoft.com (password)       │
│   ├── GCP: alice@corp-gcp.iam.gserviceaccount.com        │
│   └── GGID: alice@corp.com (password + MFA)              │
│                                                          │
│   Problems:                                              │
│   - 4 separate credentials to manage                     │
│   - No centralized password policy                       │
│   - No single MFA enforcement                            │
│   - Onboarding/offboarding touches 4 systems             │
│   - No centralized audit trail                           │
│   - Credential sprawl = security risk                    │
└──────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────┐
│                   WITH FEDERATION                         │
│                                                          │
│   User Alice has:                                        │
│   └── GGID: alice@corp.com (password + MFA)              │
│       │                                                  │
│       ├── SAML assertion → AWS IAM Identity Center       │
│       │   └── Maps to AWS role: DeveloperAccess          │
│       │                                                  │
│       ├── OIDC token → Azure AD App Registration         │
│       │   └── Maps to Azure role: Contributor            │
│       │                                                  │
│       └── SAML assertion → GCP Workforce Identity Pool   │
│           └── Maps to GCP role: roles/viewer             │
│                                                          │
│   Benefits:                                              │
│   - Single credential, single MFA                        │
│   - Centralized password policy                          │
│   - Instant deprovisioning from all clouds               │
│   - Centralized audit trail                              │
│   - No long-lived cloud credentials                      │
└──────────────────────────────────────────────────────────┘
```

---

## 3. Federation Protocols Overview

### SAML 2.0 (Enterprise Federation)

SAML 2.0 is the dominant protocol for enterprise cloud federation. AWS IAM Identity Center, Azure AD, and GCP Workforce Identity Federation all support SAML 2.0.

| Aspect | Detail |
|--------|--------|
| **Token format** | XML assertion, signed with X.509 certificate |
| **Binding** | HTTP-Redirect, HTTP-POST, HTTP-Artifact |
| **Trust model** | IdP metadata exchange (entity ID + signing cert) |
| **Claim format** | `<saml:Attribute>` elements in assertion |
| **Session** | Browser cookie set by the cloud provider after assertion |
| **Expiry** | Assertion validity period (typically 5 min) |
| **GGID status** | **Implemented** — `/saml/idp/metadata`, `/saml/idp/sso` endpoints exist |

### OIDC (Modern Federation)

OpenID Connect is used for workload identity federation (service-to-service) and some workforce scenarios.

| Aspect | Detail |
|--------|--------|
| **Token format** | JWT (JSON Web Token) |
| **Trust model** | Issuer URL + JWKS endpoint discovery |
| **Claim format** | JSON claims in JWT payload |
| **Session** | Short-lived access token (typically 1 hour) |
| **Expiry** | Token `exp` claim |
| **GGID status** | **Implemented** — `/.well-known/openid-configuration`, `/oauth/jwks` endpoints exist |

### When to Use Each

| Scenario | Protocol | Why |
|----------|----------|-----|
| AWS IAM Identity Center (workforce SSO) | SAML 2.0 | AWS requires SAML for Identity Center |
| AWS IAM role assumption (web identity) | OIDC | AWS supports OIDC for IRSA / workload federation |
| Azure AD app integration | SAML 2.0 or OIDC | Azure supports both; OIDC preferred for new apps |
| Azure managed identity federation | OIDC | Workload identity federation uses OIDC tokens |
| GCP Workforce Identity Pool | SAML 2.0 | GCP recommends SAML for workforce federation |
| GCP Workload Identity Pool | OIDC | GCP uses OIDC for workload (service-to-service) federation |
| GitHub Actions → cloud | OIDC | Cloud providers trust GitHub's OIDC tokens |

---

## 4. AWS IAM Federation

### Architecture

```
    User           GGID (IdP)          AWS IAM Identity Center      AWS Console
      │                │                        │                      │
      │ 1. Login       │                        │                      │
      ├───────────────►│                        │                      │
      │                │ AuthN + MFA            │                      │
      │ 2. Click "AWS" │                        │                      │
      ├───────────────►│                        │                      │
      │                │ 3. SAML Response       │                      │
      │                │   (with role attr)     │                      │
      │◄───────────────┤                        │                      │
      │                                         │                      │
      │ 4. POST SAML Response                   │                      │
      ├────────────────────────────────────────►│                      │
      │                                         │ 5. Parse assertion   │
      │                                         │   Map role attribute │
      │                                         │   to AWS permission  │
      │                                         ├─────────────────────►│
      │                                         │                      │
      │ 6. Redirect to AWS Console              │                      │
      │◄────────────────────────────────────────┤                      │
      │                                                                │
      │ 7. AWS Console (authenticated as role)                         │
      │◄───────────────────────────────────────────────────────────────┤
```

### AWS-Specific SAML Attributes

AWS IAM Identity Center expects specific SAML attributes:

```xml
<!-- Required: Role attribute containing comma-separated role ARN -->
<saml:Attribute Name="https://aws.amazon.com/SAML/Attributes/Role">
  <saml:AttributeValue>
    arn:aws:iam::123456789012:role/GGID-Developer,arn:aws:iam::123456789012:saml-provider/GGID
  </saml:AttributeValue>
</saml:Attribute>

<!-- Required: RoleSessionName (identifies the user in CloudTrail) -->
<saml:Attribute Name="https://aws.amazon.com/SAML/Attributes/RoleSessionName">
  <saml:AttributeValue>alice@corp.com</saml:AttributeValue>
</saml:Attribute>

<!-- Optional: SessionDuration (900-43200 seconds) -->
<saml:Attribute Name="https://aws.amazon.com/SAML/Attributes/SessionDuration">
  <saml:AttributeValue>3600</saml:AttributeValue>
</saml:Attribute>

<!-- Optional: SourceIdentity (for ABAC / attribute-based access control) -->
<saml:Attribute Name="https://aws.amazon.com/SAML/Attributes/SourceIdentity">
  <saml:AttributeValue>alice@corp.com</saml:AttributeValue>
</saml:Attribute>

<!-- Optional: PrincipalTag:* (for ABAC tag-based policies) -->
<saml:Attribute Name="https://aws.amazon.com/SAML/Attributes/PrincipalTag:Department">
  <saml:AttributeValue>Engineering</saml:AttributeValue>
</saml:Attribute>
```

### GGID → AWS Role Mapping

GGID roles map to AWS IAM roles via a configurable mapping table:

| GGID Role | AWS IAM Role ARN | Session Duration | Notes |
|-----------|------------------|-----------------|-------|
| `admin` | `arn:aws:iam::ACCOUNT:role/GGID-Admin` | 3600s | Full admin |
| `developer` | `arn:aws:iam::ACCOUNT:role/GGID-Developer` | 3600s | Read + deploy |
| `viewer` | `arn:aws:iam::ACCOUNT:role/GGID-Viewer` | 7200s | Read-only |
| `finance` | `arn:aws:iam::ACCOUNT:role/GGID-Finance` | 1800s | Shorter session |

### AWS IAM Identity Provider Setup (Terraform)

```hcl
# Create SAML identity provider in AWS
resource "aws_iam_saml_provider" "ggid" {
  name                   = "GGID"
  saml_metadata_document = data.http.ggid_saml_metadata.response_body
}

data "http" "ggid_saml_metadata" {
  url = "https://ggid.corp.com/saml/idp/metadata"
}

# Create IAM role that trusts GGID SAML provider
resource "aws_iam_role" "developer" {
  name = "GGID-Developer"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = aws_iam_saml_provider.ggid.arn
      }
      Action = "sts:AssumeRoleWithSAML"
      Condition = {
        StringEquals = {
          "SAML:aud" = "https://signin.aws.amazon.com/saml"
        }
      }
    }]
  })
}

# Attach permissions policy
resource "aws_iam_role_policy_attachment" "developer_readonly" {
  role       = aws_iam_role.developer.name
  policy_arn = "arn:aws:iam::aws:policy/ReadOnlyAccess"
}
```

### AWS IAM Identity Center (SCIM Provisioning)

For multi-account AWS environments, **IAM Identity Center** is the recommended approach. GGID acts as both SAML IdP and SCIM 2.0 provider:

```
GGID User Created → SCIM POST /Users → AWS IAM Identity Center → Provisioned across N accounts
GGID User Deleted → SCIM DELETE /Users/{id} → AWS IAM Identity Center → Deprovisioned
```

---

## 5. Azure AD / Microsoft Entra ID Federation

### Architecture

```
    User           GGID (IdP)          Azure AD App Registration      Azure Portal
      │                │                        │                        │
      │ 1. Login       │                        │                        │
      ├───────────────►│                        │                        │
      │ 2. Click "Azure"│                       │                        │
      ├───────────────►│                        │                        │
      │                │ 3. SAML Response       │                        │
      │                │   (with roles claim)   │                        │
      │◄───────────────┤                        │                        │
      │                                         │                        │
      │ 4. POST SAML to Azure ACS URL           │                        │
      ├────────────────────────────────────────►│                        │
      │                                         │ 5. Map app roles       │
      │                                         │   to Azure RBAC roles  │
      │                                         ├───────────────────────►│
      │ 6. Redirect to Azure Portal             │                        │
      │◄────────────────────────────────────────┤                        │
```

### Azure-Specific SAML Claims

Azure AD enterprise applications expect specific claim types:

```xml
<!-- Required: User identifier (usually email) -->
<saml:Attribute Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress">
  <saml:AttributeValue>alice@corp.com</saml:AttributeValue>
</saml:Attribute>

<!-- Required: Display name -->
<saml:Attribute Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name">
  <saml:AttributeValue>Alice Chen</saml:AttributeValue>
</saml:Attribute>

<!-- Required: Application roles -->
<saml:Attribute Name="http://schemas.microsoft.com/ws/2008/06/identity/claims/role">
  <saml:AttributeValue>Developer</saml:AttributeValue>
  <saml:AttributeValue>Contributor</saml:AttributeValue>
</saml:Attribute>

<!-- Optional: Department (for dynamic groups) -->
<saml:Attribute Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/department">
  <saml:AttributeValue>Engineering</saml:AttributeValue>
</saml:Attribute>

<!-- Optional: Group memberships (object IDs of Azure AD groups) -->
<saml:Attribute Name="http://schemas.microsoft.com/ws/2008/06/identity/claims/groups">
  <saml:AttributeValue>a1b2c3d4-e5f6-...</saml:AttributeValue>
</saml:Attribute>
```

### GGID → Azure App Role Mapping

| GGID Role | Azure App Role | Azure RBAC Role |
|-----------|---------------|-----------------|
| `admin` | `Admin` | `Owner` |
| `developer` | `Developer` | `Contributor` |
| `viewer` | `Reader` | `Reader` |

### OIDC Federation for Azure Workload Identity

For service-to-service (workload) federation, Azure uses OIDC token exchange:

```
GGID issues OIDC token → Azure AD validates token via GGID JWKS →
Azure issues access token for Azure resource → Workload accesses Azure resource
```

Azure configuration (Terraform):

```hcl
# Create federated identity credential on Azure AD app
resource "azuread_application_federated_identity_credential" "ggid" {
  application_object_id = azuread_application.app.object_id
  display_name          = "GGID-Federation"
  description           = "Trust GGID OIDC tokens"
  audiences             = ["api://${azuread_application.app.client_id}"]
  issuer                = "https://ggid.corp.com/oauth"
  subject               = "ggid:workload:ci-pipeline"
}
```

---

## 6. Google Cloud Platform Federation

### Workforce Identity Federation (Human Users)

GCP recommends **Workforce Identity Pool** with SAML for workforce federation:

```
    User           GGID (IdP)          GCP Workforce Pool           GCP Console
      │                │                     │                          │
      │ 1. Login       │                     │                          │
      ├───────────────►│                     │                          │
      │ 2. Click "GCP" │                     │                          │
      ├───────────────►│                     │                          │
      │                │ 3. SAML Response    │                          │
      │                │   (with attributes) │                          │
      │◄───────────────┤                     │                          │
      │                                      │                          │
      │ 4. POST SAML to GCP workforce ACS    │                          │
      ├─────────────────────────────────────►│                          │
      │                                      │ 5. Map pool attributes   │
      │                                      │   to GCP IAM permissions │
      │                                      ├─────────────────────────►│
      │ 6. Redirect to GCP Console           │                          │
      │◄─────────────────────────────────────┤                          │
```

### GCP-Specific Attribute Mapping

GCP Workforce Identity Pool uses CEL-based attribute mapping:

```yaml
# Attribute mapping in GCP workforce pool provider
attribute_mapping:
  google.subject:        "assertion.subject"
  google.display_name:   "assertion.attributes.displayName[0]"
  google.groups:         "assertion.attributes.groups"

  # Custom attributes for ABAC
  attribute.department:  "assertion.attributes.department[0]"
  attribute.role:        "assertion.attributes.role[0]"

# Attribute conditions (CEL expression)
attribute_condition: >
  assertion.attributes.department[0] in ['Engineering', 'DevOps', 'Security']
```

### GGID → GCP Role Mapping

| GGID Role | GCP IAM Role | Scope |
|-----------|-------------|-------|
| `admin` | `roles/owner` | Organization |
| `developer` | `roles/editor` | Project |
| `viewer` | `roles/viewer` | Project |

### GCP Workforce Pool Setup (Terraform)

```hcl
# Create workforce identity pool
resource "google_iam_workforce_pool" "ggid" {
  workforce_pool_id = "ggid-federation"
  parent            = "organizations/123456789"
  location          = "global"
  display_name      = "GGID Federation"
  session_duration  = "28800s"
}

# Create SAML provider in the pool
resource "google_iam_workforce_pool_provider" "saml" {
  workforce_pool_id         = google_iam_workforce_pool.ggid.workforce_pool_id
  location                  = "global"
  provider_id               = "ggid-saml"
  display_name              = "GGID SAML IdP"

  attribute_mapping = {
    "google.subject"      = "assertion.subject"
    "google.display_name" = "assertion.attributes.displayName[0]"
    "google.groups"       = "assertion.attributes.groups"
  }

  attribute_condition = "assertion.attributes.tenant[0] == 'production'"

  saml {
    idp_metadata_xml = data.http.ggid_saml_metadata.response_body
  }
}

data "http" "ggid_saml_metadata" {
  url = "https://ggid.corp.com/saml/idp/metadata"
}

# Bind GGID groups to GCP roles
resource "google_project_iam_member" "dev_access" {
  project = "my-project"
  role    = "roles/editor"
  member  = "principalSet://iam.googleapis.com/${google_iam_workforce_pool.ggid.name}/attribute.role/developer"
}
```

---

## 7. GGID Current State Analysis

### Existing Federation Infrastructure

| Component | File | Status |
|-----------|------|--------|
| SAML IdP metadata endpoint | `services/oauth/internal/server/server.go:967` | **Implemented** — `/saml/idp/metadata` |
| SAML IdP SSO endpoint | `services/oauth/internal/server/server.go:979` | **Implemented** — `/saml/idp/sso` |
| SAML IdP SLO endpoint | `services/oauth/internal/server/server.go:1074` | **Implemented** — `/saml/idp/slo` |
| SAML SP metadata | `services/oauth/internal/server/server.go:846` | **Implemented** — `/saml/metadata` |
| SAML SP ACS | `services/oauth/internal/server/server.go:854` | **Implemented** — `/saml/acs` |
| SAML SP-initiated SSO | `services/oauth/internal/server/server.go:914` | **Implemented** — `/saml/sso` |
| OIDC Discovery | `services/oauth/internal/server/server.go:213` | **Implemented** — `/.well-known/openid-configuration` |
| JWKS endpoint | `services/oauth/internal/server/server.go:225` | **Implemented** — `/oauth/jwks` |
| Per-tenant IdP config | `services/identity/internal/idpconfig/idpconfig.go` | **Implemented** — CRUD for IdP configs |
| Token Exchange (RFC 8693) | `services/oauth/internal/service/token_exchange.go` | **Implemented** — delegation tokens |
| SAML assertion building | `pkg/saml/` | **Implemented** — signing, parsing, attribute extraction |
| IssueSAMLToken | `services/oauth/internal/service/oauth_service.go:752` | **Implemented** — JWT from SAML NameID |

### What GGID Already Does Well

1. **SAML IdP endpoints exist** — can sign assertions and serve metadata
2. **OIDC discovery + JWKS** — cloud providers can discover GGID's signing keys
3. **Per-tenant IdP configs** — each tenant can configure its own federation
4. **Token exchange** — RFC 8693 delegation chain support
5. **MFA** — users authenticate with TOTP/WebAuthn before getting SAML assertions

### What's Missing

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No cloud-specific claim mapping** | SAML assertions don't include AWS role ARNs, Azure app roles, or GCP workforce attributes |
| 2 | **No cloud provider registry** | No data model for "AWS account federation," "Azure AD app registration," or "GCP workforce pool" |
| 3 | **No Terraform/metadata export** | No way to generate the Terraform snippets cloud admins need |
| 4 | **No SCIM provisioning to AWS** | Users aren't auto-provisioned to AWS IAM Identity Center |
| 5 | **No Console federation wizard** | Admins must manually configure SAML/OIDC trust on both sides |
| 6 | **No attribute-based access control (ABAC)** | SAML assertions don't include principal tags for AWS ABAC |
| 7 | **No federation health monitoring** | No way to test if federation trust is still valid |
| 8 | **No multi-account AWS support** | Single SAML provider, no Identity Center integration |

---

## 8. Gap Analysis

### Detailed Use Cases That Fail Today

| # | Use Case | Current Behavior | Expected Behavior |
|---|----------|-----------------|-------------------|
| 1 | "Alice clicks AWS Console button in GGID" | No AWS Console button exists | SAML assertion with `https://aws.amazon.com/SAML/Attributes/Role` → AWS Console login |
| 2 | "GGID Developer role maps to AWS DeveloperAccess role" | No role mapping | Configurable mapping table: GGID role → AWS IAM role ARN |
| 3 | "User created in GGID is auto-provisioned in AWS" | Manual AWS user creation | SCIM 2.0 push to AWS IAM Identity Center |
| 4 | "Admin gets Terraform snippet for AWS setup" | Must write manually | Console generates copy-paste Terraform |
| 5 | "Azure AD app roles from GGID roles" | SAML assertion has no app roles | `claims/role` attribute mapped from GGID roles |
| 6 | "GCP workforce pool attributes from GGID" | No GCP-specific attributes | SAML attributes mapped to GCP CEL expressions |
| 7 | "Federation trust breaks (cert rotation)" | No detection | Health check validates IdP metadata + signing cert |
| 8 | "Per-tenant AWS account mapping" | Global SAML, no tenant-specific roles | Each tenant maps to different AWS accounts |

---

## 9. Proposed Architecture

### Cloud Federation Manager

```
                    ┌───────────────────────────────────────┐
                    │          GGID OAuth Service           │
                    │                                       │
                    │   ┌───────────────────────────────┐   │
                    │   │   Cloud Federation Manager    │   │
                    │   │                               │   │
                    │   │  ┌─────────┐ ┌──────────┐   │   │
                    │   │  │  AWS    │ │  Azure   │   │   │
                    │   │  │ Module  │ │ Module   │   │   │
                    │   │  └────┬────┘ └────┬─────┘   │   │
                    │   │       │           │          │   │
                    │   │  ┌────┴───────────┴────┐    │   │
                    │   │  │  Claim Mapping      │    │   │
                    │   │  │  Engine             │    │   │
                    │   │  │                     │    │   │
                    │   │  │  GGID Role → Cloud  │    │   │
                    │   │  │  Role Mapping Table │    │   │
                    │   │  └─────────────────────┘    │   │
                    │   │                               │   │
                    │   │  ┌─────────┐ ┌──────────┐   │   │
                    │   │  │  GCP    │ │ SCIM     │   │   │
                    │   │  │ Module  │ │ Module   │   │   │
                    │   │  └─────────┘ └──────────┘   │   │
                    │   └───────────────────────────────┘   │
                    │                    │                  │
                    │   ┌────────────────▼──────────────┐   │
                    │   │   SAML Assertion Builder       │   │
                    │   │   (cloud-specific attributes)  │   │
                    │   └────────────────┬──────────────┘   │
                    │                    │                  │
                    └────────────────────┼──────────────────┘
                                         │
                          ┌──────────────┼──────────────┐
                          │              │              │
                          ▼              ▼              ▼
                   ┌────────────┐ ┌────────────┐ ┌────────────┐
                   │ AWS IAM    │ │ Azure AD   │ │ GCP        │
                   │ Identity   │ │ App        │ │ Workforce  │
                   │ Center     │ │            │ │ Pool       │
                   └────────────┘ └────────────┘ └────────────┘
```

### Component Design

#### Cloud Federation Config (Per-Tenant)

Each tenant can configure multiple cloud federation targets:

```go
// CloudFederationConfig represents a cloud provider federation setup.
type CloudFederationConfig struct {
    ID              uuid.UUID       `json:"id"`
    TenantID        uuid.UUID       `json:"tenant_id"`
    Provider        CloudProvider   `json:"provider"`     // aws, azure, gcp
    Name            string          `json:"name"`         // "Production AWS"
    Protocol        string          `json:"protocol"`     // "saml" or "oidc"

    // Provider-specific configuration
    AWSConfig       *AWSFedConfig   `json:"aws_config,omitempty"`
    AzureConfig     *AzureFedConfig `json:"azure_config,omitempty"`
    GCPConfig       *GCPFedConfig   `json:"gcp_config,omitempty"`

    // Role mapping (GGID role → cloud role)
    RoleMappings    []RoleMapping   `json:"role_mappings"`

    // Claim/attribute mapping
    AttributeMapping map[string]string `json:"attribute_mapping"`

    // SAML settings
    SAMLEntityID    string          `json:"saml_entity_id"`
    ACSURL          string          `json:"acs_url"`         // Cloud provider's ACS URL

    // SCIM provisioning
    SCIMEnabled     bool            `json:"scim_enabled"`
    SCIMEndpoint    string          `json:"scim_endpoint"`
    SCIMToken       string          `json:"-"`               // Encrypted, never returned

    // Status
    Enabled         bool            `json:"enabled"`
    CreatedAt       time.Time       `json:"created_at"`
    UpdatedAt       time.Time       `json:"updated_at"`
}

type CloudProvider string
const (
    CloudProviderAWS   CloudProvider = "aws"
    CloudProviderAzure CloudProvider = "azure"
    CloudProviderGCP   CloudProvider = "gcp"
)

// AWSFedConfig holds AWS-specific federation parameters.
type AWSFedConfig struct {
    AccountID           string `json:"account_id"`            // 12-digit AWS account
    IAMIdentityCenter   bool   `json:"iam_identity_center"`  // Using Identity Center
    Region              string `json:"region"`                // e.g., us-east-1
    SessionDuration     int    `json:"session_duration"`     // seconds (900-43200)
    SAMLProviderName    string `json:"saml_provider_name"`   // e.g., "GGID"
}

// RoleMapping maps a GGID role to a cloud-specific role identifier.
type RoleMapping struct {
    GGIDRoleKey    string `json:"ggid_role_key"`    // e.g., "developer"
    CloudRoleARN   string `json:"cloud_role_arn"`   // AWS: arn:aws:iam::...
    CloudRoleName  string `json:"cloud_role_name"`  // Azure: app role name; GCP: role name
    Priority       int    `json:"priority"`         // Lower = higher priority
}
```

---

## 10. Database Schema

```sql
-- Cloud federation configurations
CREATE TABLE cloud_federation_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    provider        VARCHAR(32) NOT NULL,         -- 'aws', 'azure', 'gcp'
    name            VARCHAR(128) NOT NULL,         -- Display name
    protocol        VARCHAR(16) NOT NULL,         -- 'saml', 'oidc'
    config_json     JSONB NOT NULL,                -- Provider-specific config
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, provider, name)
);

-- Role mappings: GGID role → cloud role
CREATE TABLE cloud_role_mappings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    federation_id       UUID NOT NULL REFERENCES cloud_federation_configs(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL,
    ggid_role_key       VARCHAR(128) NOT NULL,
    cloud_role_arn      TEXT,                    -- AWS: full ARN
    cloud_role_name     VARCHAR(256),             -- Azure/GCP: role name
    priority            INT DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Attribute mappings: SAML/OIDC claim → cloud attribute
CREATE TABLE cloud_attribute_mappings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    federation_id       UUID NOT NULL REFERENCES cloud_federation_configs(id) ON DELETE CASCADE,
    ggid_attribute      VARCHAR(256) NOT NULL,    -- e.g., "email", "department", "roles"
    cloud_attribute     VARCHAR(256) NOT NULL,    -- e.g., "https://aws.amazon.com/SAML/Attributes/RoleSessionName"
    transform           VARCHAR(128),             -- Optional: "lowercase", "prefix:dev-", "template:{value}@corp.com"
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- SCIM provisioning state
CREATE TABLE cloud_scim_state (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    federation_id       UUID NOT NULL REFERENCES cloud_federation_configs(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL,             -- GGID user ID
    cloud_user_id       VARCHAR(256),              -- Cloud provider's user ID
    status              VARCHAR(32) NOT NULL,      -- 'pending', 'provisioned', 'failed', 'deprovisioned'
    last_sync_at        TIMESTAMPTZ,
    error               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(federation_id, user_id)
);

-- Federation health checks
CREATE TABLE cloud_federation_health (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    federation_id       UUID NOT NULL REFERENCES cloud_federation_configs(id) ON DELETE CASCADE,
    check_type          VARCHAR(64) NOT NULL,      -- 'saml_metadata', 'cert_expiry', 'scim_connectivity'
    status              VARCHAR(32) NOT NULL,      -- 'healthy', 'warning', 'critical'
    message             TEXT,
    checked_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_federation_tenant ON cloud_federation_configs (tenant_id);
CREATE INDEX idx_federation_provider ON cloud_federation_configs (tenant_id, provider);
CREATE INDEX idx_role_mapping_federation ON cloud_role_mappings (federation_id);
CREATE INDEX idx_scim_state_user ON cloud_scim_state (federation_id, user_id);
CREATE INDEX idx_health_federation ON cloud_federation_health (federation_id, checked_at DESC);
```

---

## 11. API Design

### Cloud Federation Management

```
# Create cloud federation target
POST /api/v1/oauth/cloud-federation
Content-Type: application/json

{
    "provider": "aws",
    "name": "Production AWS",
    "protocol": "saml",
    "config": {
        "account_id": "123456789012",
        "iam_identity_center": false,
        "region": "us-east-1",
        "session_duration": 3600,
        "saml_provider_name": "GGID"
    },
    "acs_url": "https://signin.aws.amazon.com/saml",
    "role_mappings": [
        {
            "ggid_role_key": "admin",
            "cloud_role_arn": "arn:aws:iam::123456789012:role/GGID-Admin"
        },
        {
            "ggid_role_key": "developer",
            "cloud_role_arn": "arn:aws:iam::123456789012:role/GGID-Developer"
        }
    ]
}

Response:
{
    "id": "uuid",
    "provider": "aws",
    "name": "Production AWS",
    "saml_metadata_url": "https://ggid.corp.com/saml/idp/metadata",
    "terraform_snippet": "# Copy-paste Terraform to configure AWS trust\n...",
    "status": "active"
}

# List federation targets
GET /api/v1/oauth/cloud-federation?tenant_id={tenant}

# Get federation target with role mappings
GET /api/v1/oauth/cloud-federation/{id}

# Update role mappings
PUT /api/v1/oauth/cloud-federation/{id}/role-mappings
[
    {"ggid_role_key": "viewer", "cloud_role_arn": "arn:aws:iam::123456789012:role/GGID-Viewer"}
]

# Generate Terraform snippet
GET /api/v1/oauth/cloud-federation/{id}/terraform

# Response: Full Terraform module to configure cloud-side trust
```

### Federation Login (User initiates cloud console access)

```
# Initiate federation login
POST /api/v1/oauth/cloud-federation/{id}/login
{
    "user_id": "uuid",
    "session_token": "current JWT"
}

# Response: SAML response or redirect URL
{
    "method": "POST",
    "action": "https://signin.aws.amazon.com/saml",
    "saml_response": "base64-encoded-saml-response",
    "relay_state": "https://console.aws.amazon.com/"
}
```

### Health Check

```
# Test federation health
POST /api/v1/oauth/cloud-federation/{id}/health-check

# Response
{
    "overall": "healthy",
    "checks": [
        {
            "type": "saml_metadata",
            "status": "healthy",
            "message": "Metadata endpoint accessible, signing cert valid until 2027-01-01"
        },
        {
            "type": "cert_expiry",
            "status": "warning",
            "message": "Signing certificate expires in 45 days"
        },
        {
            "type": "scim_connectivity",
            "status": "healthy",
            "message": "SCIM endpoint reachable, last sync 2 min ago"
        }
    ]
}
```

### SCIM Provisioning

```
# Trigger SCIM sync for a user
POST /api/v1/oauth/cloud-federation/{id}/scim/sync
{
    "user_id": "uuid"
}

# Get SCIM provisioning status
GET /api/v1/oauth/cloud-federation/{id}/scim/status
```

---

## 12. Claim Mapping Engine

### How It Works

When GGID builds a SAML assertion for cloud federation, the **Claim Mapping Engine** transforms GGID user attributes into cloud-provider-specific SAML attributes:

```go
// ClaimMappingEngine transforms GGID attributes into cloud-specific claims.
type ClaimMappingEngine struct {
    mappings []AttributeMapping
}

// AttributeMapping defines how a GGID attribute maps to a cloud claim.
type AttributeMapping struct {
    GGIDSource      string  // Source attribute: "email", "roles", "department", "display_name"
    CloudAttribute  string  // Target SAML attribute name / OIDC claim
    Transform       string  // Optional transform: "lower", "prefix:arn:...", "template:..."
    Required        bool    // If true, assertion fails if source is empty
}

// BuildCloudSAMLAttributes generates cloud-specific SAML attributes.
func (e *ClaimMappingEngine) BuildCloudSAMLAttributes(
    user *User,
    roles []string,
    fedConfig *CloudFederationConfig,
) (map[string][]string, error) {
    attrs := make(map[string][]string)

    for _, mapping := range e.mappings {
        var value string
        switch mapping.GGIDSource {
        case "email":
            value = user.Email
        case "display_name":
            value = user.FullName
        case "department":
            value = user.Department
        case "roles":
            // Special handling for roles — map through RoleMappings
            cloudRoles := e.mapRolesToCloud(roles, fedConfig.RoleMappings)
            attrs[mapping.CloudAttribute] = cloudRoles
            continue
        }

        // Apply transform
        value = applyTransform(value, mapping.Transform)

        if value == "" && mapping.Required {
            return nil, fmt.Errorf("required attribute %s is empty", mapping.GGIDSource)
        }

        attrs[mapping.CloudAttribute] = []string{value}
    }

    return attrs, nil
}
```

### AWS Default Mapping

```go
var AWSDefaultMappings = []AttributeMapping{
    {
        GGIDSource:     "roles",
        CloudAttribute: "https://aws.amazon.com/SAML/Attributes/Role",
        Transform:      "aws_role_arn",  // Maps GGID roles to AWS role ARNs
        Required:       true,
    },
    {
        GGIDSource:     "email",
        CloudAttribute: "https://aws.amazon.com/SAML/Attributes/RoleSessionName",
        Required:       true,
    },
    {
        GGIDSource:     "department",
        CloudAttribute: "https://aws.amazon.com/SAML/Attributes/PrincipalTag:Department",
    },
}
```

### Azure Default Mapping

```go
var AzureDefaultMappings = []AttributeMapping{
    {
        GGIDSource:     "email",
        CloudAttribute: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
        Required:       true,
    },
    {
        GGIDSource:     "display_name",
        CloudAttribute: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
        Required:       true,
    },
    {
        GGIDSource:     "roles",
        CloudAttribute: "http://schemas.microsoft.com/ws/2008/06/identity/claims/role",
        Transform:      "azure_app_role",  // Maps GGID roles to Azure app roles
        Required:       true,
    },
}
```

### GCP Default Mapping

```go
var GCPDefaultMappings = []AttributeMapping{
    {
        GGIDSource:     "email",
        CloudAttribute: "subject",  // Maps to google.subject
        Required:       true,
    },
    {
        GGIDSource:     "display_name",
        CloudAttribute: "displayName",
    },
    {
        GGIDSource:     "department",
        CloudAttribute: "department",
    },
    {
        GGIDSource:     "roles",
        CloudAttribute: "role",
        Transform:      "gcp_workforce_role",
    },
}
```

---

## 13. SCIM Provisioning to Cloud Providers

### SCIM 2.0 Server for AWS IAM Identity Center

GGID already implements SCIM 2.0 endpoints (Identity service). For cloud federation, GGID needs to act as a **SCIM client** — pushing user changes to cloud providers:

```
GGID User Created    → SCIM POST /Users → Cloud Provider → Provisioned
GGID User Updated    → SCIM PUT /Users/{id} → Cloud Provider → Updated
GGID User Disabled   → SCIM PATCH /Users/{id} (active=false) → Cloud Provider → Disabled
GGID User Deleted    → SCIM DELETE /Users/{id} → Cloud Provider → Deprovisioned
GGID User Role Added → SCIM PATCH /Users/{id} (add roles) → Cloud Provider → Permissions updated
```

### SCIM Client Implementation

```go
// SCIMClient pushes user changes to cloud providers via SCIM 2.0.
type SCIMClient struct {
    endpoint string
    token    string
    http     *http.Client
}

// ProvisionUser creates a user in the cloud provider.
func (c *SCIMClient) ProvisionUser(ctx context.Context, user *User, roles []string) error {
    scimUser := SCIMUser{
        Schemas:    []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
        UserName:   user.Email,
        Name:       SCIMName{GivenName: user.FirstName, FamilyName: user.LastName},
        DisplayName: user.FullName,
        Emails:     []SCIMEmail{{Value: user.Email, Type: "work", Primary: true}},
        Active:     true,
        Roles:      roles,
    }

    body, _ := json.Marshal(scimUser)
    req, _ := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/Users", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+c.token)
    req.Header.Set("Content-Type", "application/scim+json")

    resp, err := c.http.Do(req)
    if err != nil {
        return fmt.Errorf("SCIM provision failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        return fmt.Errorf("SCIM provision returned %d", resp.StatusCode)
    }

    // Store cloud user ID for future updates
    var result SCIMUser
    json.NewDecoder(resp.Body).Decode(&result)
    return c.storeCloudUserID(user.ID, result.ID)
}
```

---

## 14. Security Considerations

### Trust Chain Security

| Risk | Mitigation |
|------|-----------|
| **Signing key compromise** | Rotate SAML signing cert regularly (90 days), notify all federation targets |
| **Confused deputy attack** | Per-tenant OIDC issuer URLs (AWS recommendation); `aud` claim validation |
| **Token replay** | Short assertion validity (5 min), `NotBefore`/`NotOnOrAfter` enforcement |
| **Man-in-the-middle** | Require TLS for all SAML endpoints, HSTS headers |
| **SCIM token leak** | Encrypted storage (AES-256-GCM in `pkg/crypto`), never logged |
| **Role escalation** | Mapping table is admin-only; role changes audited |
| **Stale federation** | Health checks validate cert expiry, metadata accessibility |

### Certificate Lifecycle

```
1. GGID generates self-signed RSA key (4096-bit) for SAML signing
2. Certificate included in IdP metadata
3. Cloud provider imports metadata → trusts certificate
4. On rotation:
   a. Generate new key pair
   b. Serve new certificate in metadata (dual-key period)
   c. Notify cloud admin to re-import metadata
   d. After grace period (7 days), remove old key
5. Health check monitors cert expiry → alerts at 60/30/7 days
```

---

## 15. Console UI Design

### Cloud Federation Dashboard

```
┌──────────────────────────────────────────────────────────────────┐
│  Cloud Federation                                                │
│                                                                  │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐     │
│  │  AWS           │  │  Azure         │  │  GCP           │     │
│  │  Production    │  │  Corp App      │  │  Workforce Pool│     │
│  │  ● Healthy     │  │  ● Healthy     │  │  ◐ Cert exp    │     │
│  │  247 users     │  │  180 users     │  │  92 users      │     │
│  │  [Configure]   │  │  [Configure]   │  │  [Renew Cert]  │     │
│  └────────────────┘  └────────────────┘  └────────────────┘     │
│                                                                  │
│  + Add Federation Target                                         │
│                                                                  │
│  Recent Activity                                                 │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ 2026-07-17 10:32  alice@corp.com → AWS Console login      │  │
│  │ 2026-07-17 10:28  bob@corp.com → Azure Portal login       │  │
│  │ 2026-07-17 09:15  SCIM sync: 3 users provisioned to AWS   │  │
│  │ 2026-07-17 08:00  Health check: All healthy                │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### Setup Wizard (AWS Example)

```
Step 1: Choose Provider
  ◉ AWS        ○ Azure        ○ GCP

Step 2: Provider Details
  ┌──────────────────────────────────────────────────┐
  │ AWS Account ID:    [123456789012         ]       │
  │ Region:            [us-east-1            ]       │
  │ Session Duration:  [3600] seconds (900-43200)    │
  │ IAM Identity Center: ☐ Use Identity Center       │
  └──────────────────────────────────────────────────┘

Step 3: Role Mapping
  ┌──────────────────────────────────────────────────┐
  │ GGID Role        →  AWS IAM Role ARN             │
  │ admin            →  arn:aws:iam::...:role/Admin  │
  │ developer        →  arn:aws:iam::...:role/Dev    │
  │ + Add Mapping                                      │
  └──────────────────────────────────────────────────┘

Step 4: Configure AWS Trust
  ┌──────────────────────────────────────────────────┐
  │ 1. Download SAML metadata:                       │
  │    [Download Metadata XML]                       │
  │                                                  │
  │ 2. Create SAML provider in AWS IAM:              │
  │    [Copy AWS CLI Command]                        │
  │                                                  │
  │ 3. Or use Terraform:                             │
  │    [Copy Terraform Snippet]                      │
  │                                                  │
  │ 4. Create IAM roles with trust policy:           │
  │    [Copy Trust Policy JSON]                      │
  └──────────────────────────────────────────────────┘

Step 5: Test Federation
  [Launch Test Login] → Opens AWS Console in new tab
```

---

## 16. Competitive Differentiation

| Feature | GGID (proposed) | Auth0 | Keycloak | Okta | AWS IAM IdC |
|---------|-----------------|-------|----------|------|-------------|
| SAML IdP for AWS | **Yes** | Via Actions | Yes | Yes | N/A (consumer) |
| SAML IdP for Azure | **Yes** | Via Actions | Yes | Yes | No |
| SAML IdP for GCP | **Yes** | Via Actions | Yes | Yes | No |
| Role mapping UI | **Yes (wizard)** | Code | XML config | Visual | Console |
| Terraform export | **Yes** | No | No | No | No |
| SCIM to AWS IdC | **Yes** | Yes | No | Yes | N/A |
| Health monitoring | **Yes** | No | No | Partial | Yes |
| Per-tenant configs | **Yes** | Stores | Realms | Yes | No |
| Open source | **Yes (Apache 2.0)** | No | Yes | No | No |

**Key differentiator**: GGID would be the **only open-source IAM** with built-in cloud federation wizard, Terraform snippet generation, and multi-cloud SCIM provisioning.

---

## 17. Implementation Backlog

### P0 — Core Federation (3 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 1 | Cloud federation data model | PostgreSQL tables for configs, role mappings, attribute mappings | 2 days |
| 2 | Cloud federation service | CRUD service for federation configs with per-tenant isolation | 3 days |
| 3 | Claim mapping engine | Transform GGID attributes to AWS/Azure/GCP SAML attributes | 4 days |
| 4 | AWS SAML federation module | Role ARN generation, `https://aws.amazon.com/SAML/Attributes/*` attributes | 3 days |
| 5 | Azure SAML federation module | Azure claim URIs, app role mapping | 3 days |
| 6 | SAML assertion builder integration | Wire claim mapping into existing SAML response builder | 2 days |
| 7 | Federation login endpoint | `POST /cloud-federation/{id}/login` → SAML response | 2 days |
| 8 | Terraform snippet generator | Generate provider-specific Terraform for AWS/Azure/GCP | 3 days |
| 9 | Unit tests | 90%+ coverage for mapping engine, federation service | 3 days |

### P1 — Enhanced Features (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 10 | GCP workforce federation | SAML attributes for GCP Workforce Identity Pool, CEL attribute mapping | 3 days |
| 11 | SCIM client | Push user changes to AWS IAM Identity Center via SCIM 2.0 | 4 days |
| 12 | Health monitoring | Periodic health checks: metadata access, cert expiry, SCIM connectivity | 3 days |
| 13 | OIDC workload federation | Issue OIDC tokens for workload identity (IRSA-style) | 3 days |
| 14 | Attribute-based access control | Principal tags in SAML for AWS ABAC policies | 2 days |
| 15 | Integration tests | End-to-end federation login tests | 3 days |

### P2 — Console UI (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 16 | Federation dashboard | Card grid showing all cloud targets with health status | 2 days |
| 17 | Setup wizard (AWS) | Multi-step wizard with metadata download, Terraform copy, test login | 4 days |
| 18 | Setup wizard (Azure) | Azure-specific wizard steps | 2 days |
| 19 | Setup wizard (GCP) | GCP-specific wizard with workforce pool config | 2 days |
| 20 | Role mapping editor | Drag-and-drop GGID role → cloud role mapping table | 2 days |
| 21 | Activity log | Federation login events, SCIM sync events | 2 days |
| 22 | Certificate management | View SAML signing cert, rotation, expiry alerts | 2 days |

### P3 — Advanced Features (Future)

| # | Task | Description |
|---|------|-------------|
| 23 | Multi-account AWS | Support multiple AWS accounts per tenant |
| 24 | Conditional access | Per-federation risk policies (e.g., block AWS login from new IP) |
| 25 | Federation analytics | Usage metrics: login frequency, role distribution, session duration |
| 26 | Self-service portal | Users see available cloud targets and request access |
| 27 | Break-glass access | Emergency cloud access via federation with enhanced logging |
| 28 | Cross-cloud ABAC | Consistent attribute-based policies across AWS, Azure, and GCP |

---

## References

- [AWS IAM Identity Providers](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers.html) — SAML/OIDC federation into AWS
- [AWS IAM Identity Center](https://aws.amazon.com/iam/identity-center/) — Multi-account workforce SSO
- [Microsoft Entra ID Enterprise Apps](https://learn.microsoft.com/en-us/entra/identity/enterprise-apps/) — SAML/OIDC app federation
- [Azure Workload Identity Federation](https://learn.microsoft.com/en-us/entra/workload-id/workload-identity-federation) — OIDC token trust
- [GCP Workforce Identity Federation](https://cloud.google.com/iam/docs/workforce-identity-federation) — External IdP integration
- [GCP Workload Identity Pool](https://cloud.google.com/iam/docs/workload-identity-federation) — Service-to-service federation
- [Federating into Azure, GCP and AWS with OIDC](https://awsteele.com/blog/2025/07/27/federating-into-azure-gcp-and-aws-with-oidc.html) — Multi-cloud OIDC guide
- [SAML and OIDC-Based Federation for Multi-Cloud](https://oneuptime.com/blog/post/2026-02-17-how-to-implement-saml-and-oidc-based-federation-for-multi-cloud-identity/view) — Implementation guide
- [SCIM 2.0 Protocol](https://datatracker.ietf.org/doc/html/rfc7644) — SCIM provisioning protocol
- [AWS SAML Clams Reference](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_saml_assertions.html) — Required SAML attributes for AWS
- [Confused Deputy Prevention](https://docs.aws.amazon.com/IAM/latest/UserGuide/confused-deputy.html) — Per-tenant OIDC issuer URLs
