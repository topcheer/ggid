# ERP Java Demo — GGID IAM Integration

Java ERP demo with fine-grained permission control using GGID Java SDK.

## Quick Start

```bash
# Build SDK jar (first time only)
cd ../../sdk/java && mvn package -DskipTests
cp target/ggid-sdk-1.0.0.jar ../../examples/erp-java/lib/

# Build demo
cd ../../examples/erp-java && mvn package

# Run
GGID_URL=https://ggid.iot2.win \
TENANT_ID=00000000-0000-0000-0000-000000000001 \
PORT=8080 \
java -jar target/erp-java-demo-1.0.0.jar
```

## 7 Modules

| Module | Endpoint | Required Permission |
|--------|----------|-------------------|
| Auth | POST /auth/login | (public) |
| Users | GET/POST/DELETE /users | users:read / users:write |
| Roles | GET/POST /roles | roles:read / roles:write |
| Orgs | GET/POST /orgs | (auth) / settings:write |
| Inventory | GET/POST/DELETE /inventory | inventory:read / inventory:write |
| Orders | GET/POST/PUT /orders | orders:read / orders:write / orders:approve |
| Audit | GET /audit | audit:read |

## Permission Matrix

| Permission | Viewer | Sales | Manager | Admin |
|---|---|---|---|---|
| inventory:read | ✅ | ✅ | ✅ | ✅ |
| inventory:write | ❌ | ❌ | ✅ | ✅ |
| orders:read | ✅ | ✅ | ✅ | ✅ |
| orders:read:all | ❌ | ❌ | ✅ | ✅ |
| orders:write | ❌ | ✅ | ✅ | ✅ |
| orders:approve | ❌ | ❌ | ✅ | ✅ |
| users:read | ❌ | ❌ | ❌ | ✅ |
| users:write | ❌ | ❌ | ❌ | ✅ |
| audit:read | ❌ | ❌ | ✅ | ✅ |

## Row-Level Filtering

Orders module implements ABAC row-level filtering:
- Users with `orders:read:all` see ALL orders
- Users without it only see orders matching their `org_id` JWT claim

## Architecture

```
Client → Bearer JWT → BaseHandler.requireAuth() → GGIDClient.verifyToken()
                     → GGIDUser.hasPermission("inventory:read")
                     → Module handler executes CRUD
```

All operations are logged to the audit module.
