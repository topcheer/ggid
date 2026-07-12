# Migration from Keycloak to GGID

This guide covers migrating from Keycloak to GGID — realm export, user import, SAML/OIDC config migration, role mapping, and session cut-over.

> **Related**: [Keycloak Migration Guide](../keycloak-migration-guide.md), [Migration from Auth0/Okta](migration-from-auth0-okta.md)

## Concept Mapping

| Keycloak | GGID | Notes |
|----------|------|-------|
| Realm | Tenant | Top-level isolation |
| Client | OAuth Client | Register redirect URIs |
| Role | Role | RBAC with key field |
| Group | Organization/Group | Org tree with LTREE |
| Identity Provider | Auth Provider | SAML, OIDC, social |
| User Federation (LDAP) | LDAP Provider | authprovider chain |
| Realm Events | Audit Events | NATS + hash chain |
| Required Actions | MFA/Password Reset | Auth service |
| Authentication Flows | Login Flows | Configurable steps |

## Step 1: Export Realm from Keycloak

```bash
# Export realm with users
/opt/keycloak/bin/kc.sh export \
  --realm my-realm \
  --dir /tmp/keycloak-export \
  --users realm_file

# Output: /tmp/keycloak-export/my-realm-realm.json
```

### Extract Users

```bash
cat my-realm-realm.json | jq '[.users[] | {
  username: .username,
  email: .email,
  first_name: .firstName,
  last_name: .lastName,
  enabled: .enabled,
  created: .createdTimestamp,
  roles: [.realmMappings[].name],
  groups: [.groups[]]
}]' > keycloak-users.json
```

### Extract Roles

```bash
cat my-realm-realm.json | jq '[.roles.realm[] | {
  key: .name,
  name: .name,
  description: .description // ""
}]' > keycloak-roles.json
```

### Extract Clients

```bash
cat my-realm-realm.json | jq '[.clients[] | select(.clientId != "account" and .clientId != "admin-cli" and .clientId != "realm-management") | {
  client_id: .clientId,
  name: .name,
  redirect_uris: .redirectUris,
  web_origins: .webOrigins,
  public_client: .publicClient,
  bearer_only: .bearerOnly,
  standard_flow: .standardFlowEnabled,
  service_account: .serviceAccountsEnabled
}]' > keycloak-clients.json
```

## Step 2: Import Roles to GGID

```bash
cat keycloak-roles.json | jq -c '.[]' | while read role; do
  key=$(echo "$role" | jq -r '.key')
  name=$(echo "$role" | jq -r '.name')
  curl -X POST https://api.ggid.example.com/api/v1/roles \
    -H "Authorization: Bearer $GGID_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d "{\"key\": \"$key\", \"name\": \"$name\"}"
done
```

## Step 3: Import Users via SCIM

```bash
cat keycloak-users.json | jq -c '.[]' | while read user; do
  username=$(echo "$user" | jq -r '.username')
  email=$(echo "$user" | jq -r '.email')
  curl -X POST https://api.ggid.example.com/scim/v2/Users \
    -H "Authorization: Bearer $GGID_TOKEN" \
    -H "Content-Type: application/scim+json" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d "{\"schemas\": [\"urn:ietf:params:scim:schemas:core:2.0:User\"], \"userName\": \"$username\", \"emails\": [{\"value\": \"$email\", \"primary\": true}]}"
done
```

> **Note**: Keycloak uses PBKDF2 for passwords. GGID uses Argon2id. Passwords cannot be migrated — users must reset.

## Step 4: Migrate Clients (OAuth Apps)

```bash
cat keycloak-clients.json | jq -c '.[]' | while read client; do
  curl -X POST https://api.ggid.example.com/api/v1/oauth/clients \
    -H "Authorization: Bearer $GGID_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d "$client"
done
```

## Step 5: Migrate Identity Providers

| Keycloak IdP | GGID Config |
|-------------|------------|
| SAML (AddProvider) | Upload metadata in Settings → SSO |
| OIDC (AddProvider) | Configure in Settings → SSO |
| Google | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` |
| GitHub | `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET` |
| Microsoft | `MICROSOFT_CLIENT_ID`, `MICROSOFT_CLIENT_SECRET` |
| LDAP User Federation | `LDAP_URL`, `LDAP_BIND_DN`, `LDAP_BASE_DN` env vars |

## Step 6: Update Application Code

### Java/Spring (Keycloak Adapter → GGID)

```java
// BEFORE: Keycloak Spring Boot adapter
// application.properties: keycloak.realm=my-realm

// AFTER: GGID JWT verification
@Bean
public GGIDAuthFilter authFilter() {
    JwtVerifier verifier = new JwtVerifier("https://api.ggid.example.com/.well-known/jwks.json");
    return new GGIDAuthFilter(verifier);
}
```

### JavaScript (keycloak-js → GGID)

```javascript
// BEFORE
import Keycloak from 'keycloak-js';
const kc = new Keycloak({ url, realm, clientId });

// AFTER
import { GGIDClient } from '@ggid/node';
const client = new GGIDClient({ gatewayUrl, tenantId });
```

## Cut-Over Strategy

1. Deploy GGID alongside Keycloak
2. Migrate roles and users
3. Update one application to use GGID
4. Test login flow
5. Gradually migrate remaining apps
6. Monitor for 30 days
7. Decommission Keycloak

## Post-Migration Checklist

- [ ] All users imported
- [ ] Password reset emails sent
- [ ] Roles mapped correctly
- [ ] OAuth clients reconfigured
- [ ] SAML/OIDC IdPs reconfigured
- [ ] LDAP federation tested
- [ ] Application code updated
- [ ] Keycloak kept running for 30-day monitoring

## See Also

- [Migration from Auth0/Okta](migration-from-auth0-okta.md)
- [SDK Migration Guide](sdk-migration-guide.md)
- [SCIM Provisioning](scim-provisioning.md)
