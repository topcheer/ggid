# GGID SAML Service Provider Setup Guide

Complete guide for registering SAML 2.0 Service Providers (SP) with GGID as
the Identity Provider (IdP). Covers setup for Grafana, Jenkins, Tableau, and
generic SAML applications.

---

## Table of Contents

- [Overview](#overview)
- [GGID IdP Metadata](#ggid-idp-metadata)
- [Registering a Service Provider](#registering-a-service-provider)
- [Grafana SSO Setup](#grafana-sso-setup)
- [Jenkins SSO Setup](#jenkins-sso-setup)
- [Tableau SSO Setup](#tableau-sso-setup)
- [Generic SP Setup](#generic-sp-setup)
- [Attribute Mapping](#attribute-mapping)
- [Troubleshooting](#troubleshooting)

---

## Overview

GGID acts as a SAML 2.0 Identity Provider (IdP). Applications act as Service
Providers (SP) and delegate authentication to GGID via SAML assertions.

```
User → Grafana (SP) → Redirect to GGID (IdP)
                          ├── User authenticates
                          ├── GGID issues SAML assertion
                          └── Redirect back to Grafana with assertion
Grafana verifies assertion → User logged in
```

---

## GGID IdP Metadata

GGID exposes a standard SAML metadata document at:

```
https://iam.example.com/saml/metadata
```

### Download Metadata

```bash
# Download IdP metadata XML
curl -o ggid-idp-metadata.xml https://iam.example.com/saml/metadata

# Or reference the URL directly in SP configuration
```

### Key Information from Metadata

| Field | Value |
|-------|-------|
| Entity ID | `https://iam.example.com/saml` |
| SSO URL (Redirect) | `https://iam.example.com/saml/sso` |
| SSO URL (POST) | `https://iam.example.com/saml/sso/post` |
| SLO URL | `https://iam.example.com/saml/slo` |
| Signing Certificate | Available in metadata XML |
| NameID Format | Configurable per SP |

---

## Registering a Service Provider

Before an application can use GGID for SSO, register it as a SAML SP:

```bash
curl -X POST $API/api/v1/saml/sp \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{
        "name": "Grafana",
        "entity_id": "https://grafana.example.com",
        "assertion_consumer_service_url": "https://grafana.example.com/saml/acs",
        "name_id_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
        "attributes": [
            { "name": "email", "value": "{{user.email}}" },
            { "name": "name", "value": "{{user.name}}" },
            { "name": "role", "value": "{{user.roles}}" }
        ]
    }'
```

### SP Registration Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name |
| `entity_id` | Yes | Unique SP identifier (usually SP URL) |
| `assertion_consumer_service_url` | Yes | URL where GGID posts SAML assertions |
| `name_id_format` | No | Default: `emailAddress` |
| `attributes` | No | Custom attributes in assertion |
| `sign_assertion` | No | Sign the SAML assertion (default: true) |
| `sign_response` | No | Sign the SAML response wrapper (default: true) |
| `encryption_cert` | No | SP's public cert for encrypted assertions |

### List Registered SPs

```bash
curl $API/api/v1/saml/sp \
    -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Delete an SP

```bash
curl -X DELETE $API/api/v1/saml/sp/$SP_ID \
    -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## Grafana SSO Setup

### Step 1: Register Grafana in GGID

```bash
curl -X POST $API/api/v1/saml/sp \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{
        "name": "Grafana",
        "entity_id": "https://grafana.example.com",
        "assertion_consumer_service_url": "https://grafana.example.com/saml/acs",
        "name_id_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
        "attributes": [
            { "name": "Email", "value": "{{user.email}}" },
            { "name": "Name", "value": "{{user.name}}" },
            { "name": "Role", "value": "{{user.roles}}" }
        ]
    }'
```

### Step 2: Configure Grafana

Edit `grafana.ini` or use environment variables:

```ini
[auth.saml]
enabled = true
single_sign_on_url = https://iam.example.com/saml/sso/post
idp_metadata_url = https://iam.example.com/saml/metadata
sp_entity_id = https://grafana.example.com
assertion_attribute_name = Name
assertion_attribute_login = Email
assertion_attribute_email = Email
assertion_attribute_role = Role
allow_sign_up = true
role_values_editor = editor
role_values_admin = admin
```

Or via environment variables (Docker):

```bash
GF_AUTH_SAML_ENABLED=true
GF_AUTH_SAML_SINGLE_SIGN_ON_URL=https://iam.example.com/saml/sso/post
GF_AUTH_SAML_IDP_METADATA_URL=https://iam.example.com/saml/metadata
GF_AUTH_SAML_SP_ENTITY_ID=https://grafana.example.com
GF_AUTH_SAML_ASSERTION_ATTRIBUTE_EMAIL=Email
GF_AUTH_SAML_ASSERTION_ATTRIBUTE_NAME=Name
```

### Step 3: Map Roles

Grafana maps SAML `Role` attribute to Grafana roles:

| SAML Role | Grafana Role |
|-----------|-------------|
| `admin` | Admin |
| `editor` | Editor |
| `viewer` | Viewer |

---

## Jenkins SSO Setup

### Step 1: Register Jenkins in GGID

```bash
curl -X POST $API/api/v1/saml/sp \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{
        "name": "Jenkins",
        "entity_id": "https://jenkins.example.com",
        "assertion_consumer_service_url": "https://jenkins.example.com/securityRealm/finishLogin",
        "name_id_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
    }'
```

### Step 2: Install SAML Plugin

1. Go to **Manage Jenkins** > **Manage Plugins**
2. Install the **SAML Plugin** (plugin ID: `saml`)

### Step 3: Configure Jenkins

```groovy
// Jenkins Configuration as Code (JCasC)
jenkins:
  securityRealm:
    saml:
      idpMetadataConfiguration:
        url: "https://iam.example.com/saml/metadata"
        period: 60
      binding: "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      usernameAttributeName: "Email"
      emailAttributeName: "Email"
      fullNameAttributeName: "Name"
      groupsAttributeName: "Role"
      managerInternalGroups: false
```

### Step 4: Configure Authorization

```groovy
jenkins:
  authorizationStrategy:
    roleBased:
      roles:
        global:
          - name: "admin"
            description: "GGID Admin role"
            permissions:
              - "Overall/Administer"
            assignments:
              - "admin"  # SAML Role attribute
          - name: "developer"
            description: "GGID Editor role"
            permissions:
              - "Overall/Read"
              - "Job/Build"
              - "Job/Cancel"
            assignments:
              - "editor"
```

---

## Tableau SSO Setup

### Step 1: Register Tableau in GGID

```bash
curl -X POST $API/api/v1/saml/sp \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{
        "name": "Tableau",
        "entity_id": "tableau",
        "assertion_consumer_service_url": "https://tableau.example.com/saml/acs",
        "name_id_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
    }'
```

### Step 2: Download GGID Certificate

```bash
# Download IdP signing certificate
curl -o ggid-saml-cert.pem https://iam.example.com/saml/cert
```

### Step 3: Configure Tableau Server

```bash
# Enable SAML on Tableau Server
tsm authentication saml enable
tsm settings import -f saml-config.json
tsm pending-changes apply

# saml-config.json:
{
  "configEntities": {
    "samlSettings": {
      "_type": "samlSettingsType",
      "enabled": true,
      "idpEntityId": "https://iam.example.com/saml",
      "idpMetadataUrl": "https://iam.example.com/saml/metadata",
      "idpCertificate": "GGID signing certificate content",
      "spEntityId": "tableau",
      "returnUrl": "https://tableau.example.com/saml/acs"
    }
  }
}
```

---

## Generic SP Setup

For applications not listed above, use these generic steps:

### Step 1: Gather SP Information

Collect from the application's SAML configuration page:
- **SP Entity ID** — unique identifier
- **ACS URL** — Assertion Consumer Service URL
- **NameID format** — usually `emailAddress`
- **Required attributes** — which user fields are needed

### Step 2: Register in GGID

```bash
curl -X POST $API/api/v1/saml/sp \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{
        "name": "My Application",
        "entity_id": "<SP Entity ID>",
        "assertion_consumer_service_url": "<ACS URL>",
        "name_id_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
        "attributes": [
            { "name": "email", "value": "{{user.email}}" },
            { "name": "name", "value": "{{user.name}}" },
            { "name": "groups", "value": "{{user.groups}}" }
        ]
    }'
```

### Step 3: Configure the Application

Point the application to GGID's IdP metadata:

| Setting | Value |
|---------|-------|
| IdP Metadata URL | `https://iam.example.com/saml/metadata` |
| IdP SSO URL | `https://iam.example.com/saml/sso` |
| IdP Entity ID | `https://iam.example.com/saml` |
| IdP Certificate | From metadata XML |

### Step 4: Test Login

```
1. Navigate to the application
2. Application redirects to GGID login page
3. User enters credentials
4. GGID posts SAML assertion to ACS URL
5. Application validates assertion → user logged in
```

---

## Attribute Mapping

GGID supports template variables for SAML attributes:

| Template | Resolved To |
|----------|-------------|
| `{{user.email}}` | User's email address |
| `{{user.name}}` | User's full name |
| `{{user.username}}` | Username |
| `{{user.id}}` | Internal UUID |
| `{{user.roles}}` | Comma-separated role list |
| `{{user.groups}}` | Comma-separated group list |
| `{{user.tenant_id}}` | Tenant UUID |
| `{{user.department}}` | User's department (if set) |

### Custom Claims

For custom attributes, use the claims API:

```bash
curl -X PUT $API/api/v1/saml/sp/$SP_ID/attributes \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "attributes": [
            { "name": "department", "value": "{{user.department}}" },
            { "name": "manager", "value": "{{user.manager_email}}" },
            { "name": "cost_center", "value": "{{user.cost_center}}" }
        ]
    }'
```

---

## Troubleshooting

### Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| `No SP registered for entity_id` | Entity ID mismatch | Verify SP entity_id matches registration |
| `Invalid signature` | Certificate mismatch | Download latest IdP metadata |
| `ACS URL mismatch` | Wrong ACS URL | Update SP registration with correct URL |
| `Clock skew` | Server time difference | Sync NTP, increase clock skew tolerance |
| `NameID format not supported` | Wrong format | Use standard formats (emailAddress, unspecified) |
| `Attribute missing` | Template unresolved | Check user has the attribute set |

### Debug SAML Response

Use the SAML Tracer browser extension or SAML test endpoint:

```bash
# Enable SAML debug logging
SAML_DEBUG=true

# Test SAML flow
curl -v $API/saml/sso?SAMLRequest=...
```

---

## References

- [SAML Package](../pkg/saml/) — GGID SAML implementation
- [API Reference](./api-reference.md) — REST API
- [Integration Examples](./integration-examples.md) — SAML code samples
