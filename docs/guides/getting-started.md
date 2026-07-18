# Getting Started with GGID

This guide walks you through setting up GGID from scratch — from installation to your first authenticated API call.

## Prerequisites

- [Docker](https://docker.com) 24+ and Docker Compose
- `curl` and `jq` (for API testing)
- Ports 8080 (API Gateway) and 3000 (Console) available

## Option A: All-in-One Docker (Fastest)

The all-in-one image bundles PostgreSQL, Redis, NATS, all microservices, and the console in a single container.

```bash
# Build the all-in-one image
docker build -f deploy/all-in-one/Dockerfile -t ggid/ggid-all-in-one:latest .

# Run it
docker run -d --name ggid \
  -p 8080:8080 \
  -p 3000:3000 \
  ggid/ggid-all-in-one:latest

# Wait for services to start (~30 seconds)
sleep 30

# Verify
curl http://localhost:8080/healthz
# Expected: {"status":"ok"}
```

Open the console at `http://localhost:3000`.

## Option B: Docker Compose (Development)

For development with individual service containers:

```bash
git clone https://github.com/topcheer/ggid.git
cd ggid

# Start infrastructure (PostgreSQL, Redis, NATS, LDAP)
make docker-run

# Apply database migrations
make migrate-up

# Seed initial data (admin user, roles, permissions)
bash deploy/seed.sh

# Build and run all services
make build
```

### Default Service Ports

| Service | Port | Description |
|---------|------|-------------|
| Gateway | 8080 | API Gateway (entry point) |
| Console | 3000 | Next.js Admin Console |
| Identity | 8081 | User/Group management |
| Auth | 9001 | Authentication service |
| OAuth | 9005 | OAuth 2.1 / OIDC |
| Policy | 8070 | RBAC/ABAC/ReBAC |
| Audit | 8072 | Audit & ITDR |

## Option C: Build from Source

```bash
git clone https://github.com/topcheer/ggid.git
cd ggid

# Start infrastructure
make docker-run
make migrate-up

# Build all services
make build

# Run services (in separate terminals or via supervisord)
./bin/identity-server &
./bin/auth-server &
./bin/oauth-server &
./bin/policy-server &
./bin/org-server &
./bin/audit-server &
./bin/gateway-server &

# Start console
cd console && npm install && npm run dev
```

## First Login

### Default Credentials

| Username | Password |
|----------|----------|
| `admin` | `Admin@123456` |

### Login via API

```bash
# Login and extract token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"Admin@123456"}' | jq -r .access_token)

echo "Token: ${TOKEN:0:50}..."
```

### Login via Console

1. Open `http://localhost:3000` in your browser
2. Enter username: `admin`
3. Enter password: `Admin@123456`
4. Click Sign In

## Core API Operations

All API calls go through the Gateway at port **8080**. The Gateway routes to backend services internally.

### 1. Create a User

```bash
curl -s -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "email": "john@corp.com",
    "password": "SecurePass@123",
    "name": "John Doe",
    "username": "john"
  }' | jq .
```

### 2. List Users

```bash
curl -s http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq '.users | length'
```

### 3. Create a Role

```bash
curl -s -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "key": "developer",
    "name": "Developer",
    "description": "Software developer role"
  }' | jq .
```

### 4. Register an OAuth Client

```bash
curl -s -X POST http://localhost:8080/api/v1/oauth/clients \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "client_name": "My App",
    "redirect_uris": ["http://localhost:3000/callback"],
    "grant_types": ["authorization_code"],
    "response_types": ["code"],
    "token_endpoint_auth_method": "client_secret_post",
    "scope": "openid profile email"
  }' | jq .
```

### 5. Query Audit Events

```bash
curl -s "http://localhost:8080/api/v1/audit/events?limit=10" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq .
```

## OAuth 2.1 Authorization Flow

Once you have a client, test the OAuth flow:

```bash
# 1. Authorization request (browser)
# Open in browser:
# http://localhost:8080/api/v1/oauth/authorize?client_id=YOUR_CLIENT_ID&redirect_uri=http://localhost:3000/callback&response_type=code&scope=openid&state=random123

# 2. Exchange code for token
curl -s -X POST http://localhost:8080/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=AUTHORIZATION_CODE" \
  -d "client_id=YOUR_CLIENT_ID" \
  -d "client_secret=YOUR_CLIENT_SECRET" \
  -d "redirect_uri=http://localhost:3000/callback" | jq .
```

## Configuration

### Environment Variables

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable` | PostgreSQL connection |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection |
| `NATS_URL` | `nats://localhost:4222` | NATS connection |
| `GATEWAY_ADDR` | `:8080` | Gateway listen address |
| `CONSOLE_PORT` | `3000` | Console port |
| `JWT_SIGNING_KEY_PATH` | `/app/configs/rsa_private.pem` | RSA private key for JWT |
| `INTERNAL_AUTH_SECRET` | `change-me-in-production` | Inter-service auth secret |
| `NEXT_PUBLIC_TENANT_ID` | `00000000-0000-0000-0000-000000000001` | Default tenant |

### RSA Key Generation

GGID requires an RSA key pair for JWT signing:

```bash
mkdir -p configs
openssl genrsa -out configs/rsa_private.pem 2048
openssl rsa -in configs/rsa_private.pem -pubout -out configs/rsa_public.pem
```

## Console Tour

After login, the console provides:

1. **Dashboard** — System overview, active sessions, risk score
2. **Users** — User lifecycle management
3. **Roles & Permissions** — RBAC configuration
4. **OAuth Clients** — Client application management
5. **Audit Trail** — Hash-chained event log
6. **Security Center** — ITDR alerts, risk analytics
7. **Settings** — System configuration, branding, i18n

## Troubleshooting

### Services won't start

```bash
# Check logs
docker logs ggid 2>&1 | tail -50

# For docker-compose:
docker compose -f deploy/docker-compose.yaml logs --tail=50
```

### Database connection failed

```bash
# Verify PostgreSQL is running
docker exec ggid-postgres pg_isready -U ggid

# Check migration status
make migrate-up
```

### Login fails with "invalid credentials"

- Verify you're using `username` field (not `email`) in the API
- Default credentials: username=`admin`, password=`Admin@123456`
- Ensure `X-Tenant-ID: 00000000-0000-0000-0000-000000000001` header is set

### 404 on API calls

- All API calls go through the Gateway at port **8080**
- Do NOT call backend services directly (8081, 9001, etc.)
- Gateway routes: `/api/v1/*` (no service prefix needed)

## Next Steps

- Read the [Architecture Guide](docs/research/zero-trust-maturity-assessment.md)
- Explore [API Reference](docs/guides/api-reference.md)
- Set up [OAuth 2.1 clients](docs/guides/oauth-2-1-compliance-checklist.md)
- Configure [MFA](docs/guides/mfa-architecture.md)
- Explore [SDK integration](docs/guides/sdk-integration-guide.md)
