# M2M Service-to-Service Authentication Demo

This demo shows how two microservices can authenticate to each other using
GGID's `client_credentials` OAuth 2.0 grant — no human user involved.

```
┌────────────┐     client_credentials     ┌──────────┐
│ service-a  │ ──────────────────────────► │   GGID   │
│ (caller)   │ ◄────── access_token ────── │ Gateway  │
└────────────┘                             └──────────┘
      │
      │  GET /api/data  (Authorization: Bearer <jwt>)
      ▼
┌────────────┐
│ service-b  │  verify JWT against GGID JWKS
│ (callee)   │  → allow or reject
└────────────┘
```

## Architecture

- **service-a** (port 5001) — Obtains an M2M access token from GGID using
  `client_credentials` grant, then calls service-b with the token as a Bearer
  header. Caches the token until expiry.

- **service-b** (port 5002) — Receives requests, extracts the Bearer JWT,
  fetches GGID's JWKS public keys, and verifies the RSA-SHA256 signature
  in-process. No external JWT library needed.

## Quick Start

### 1. Prerequisites

- Go 1.25+
- A GGID deployment (default: `https://ggid.iot2.win`)
- An OAuth client registered in GGID with `client_credentials` grant

### 2. Register an OAuth Client

If you don't have a client_id/client_secret yet, register one via DCR:

```bash
curl -k -X POST https://ggid.iot2.win/api/v1/oauth/register \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "m2m-service-a",
    "redirect_uris": [],
    "grant_types": ["client_credentials"],
    "token_endpoint_auth_method": "client_secret_post"
  }'
```

Save the returned `client_id` and `client_secret`.

### 3. Start service-b (the protected service)

```bash
cd examples/m2m-service

GGID_URL=https://ggid.iot2.win \
GGID_TENANT_ID=00000000-0000-0000-0000-000000000001 \
PORT=5002 \
go run . -mode b
```

### 4. Start service-a (the caller)

In another terminal:

```bash
cd examples/m2m-service

GGID_URL=https://ggid.iot2.win \
GGID_TENANT_ID=00000000-0000-0000-0000-000000000001 \
CLIENT_ID=<your_client_id> \
CLIENT_SECRET=<your_client_secret> \
PORT=5001 \
SERVICE_B_URL=http://localhost:5002 \
go run . -mode a
```

### 5. Trigger the M2M call

```bash
# service-a obtains a token from GGID, then calls service-b
curl http://localhost:5001/call-service-b
```

Expected output:
```json
{
  "service": "service-b",
  "message": "data retrieved successfully",
  "timestamp": "2025-01-15T10:30:00Z",
  "caller": "<client_id>",
  "tenant": "00000000-0000-0000-0000-000000000001",
  "scopes": ""
}
```

### 6. Test POST (create data via M2M)

```bash
curl -X POST http://localhost:5001/call-service-b-post \
  -H "Content-Type: application/json" \
  -d '{"item": "widget", "quantity": 42}'
```

### 7. Verify service-b rejects invalid tokens

```bash
# No token → 401
curl http://localhost:5002/api/data

# Garbage token → 401
curl -H "Authorization: Bearer invalid.token.here" http://localhost:5002/api/data
```

## Environment Variables

| Variable | Default | Service | Description |
|----------|---------|---------|-------------|
| `GGID_URL` | `https://ggid.iot2.win` | Both | GGID gateway URL |
| `GGID_TENANT_ID` | `00000000-...001` | Both | Tenant UUID |
| `CLIENT_ID` | — | A | OAuth client ID for client_credentials |
| `CLIENT_SECRET` | — | A | OAuth client secret |
| `PORT` | `5001` / `5002` | Both | Listen port |
| `SERVICE_B_URL` | `http://localhost:5002` | A | service-b base URL |

## API Endpoints

### service-a (port 5001)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/call-service-b` | Obtains M2M token, calls service-b GET /api/data |
| POST | `/call-service-b-post` | Obtains M2M token, calls service-b POST /api/data |

### service-b (port 5002)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/data` | Protected — requires valid GGID JWT |
| POST | `/api/data` | Protected — requires valid GGID JWT |

## How It Works

1. **Token acquisition**: service-a POSTs to `/api/v1/oauth/token` with
   `grant_type=client_credentials`, `client_id`, and `client_secret`.

2. **Token caching**: service-a caches the access token in memory and only
   refreshes when it's within 30 seconds of expiry.

3. **Token verification**: service-b fetches GGID's JWKS
   (`/.well-known/jwks.json`), builds an RSA public key from the JWK's
   `n` and `e` parameters, and verifies the JWT signature using
   `crypto/rsa.VerifyPKCS1v15`.

4. **No external dependencies**: The demo uses only Go standard library —
   no JWT library needed. RSA key construction and signature verification
   are implemented from scratch.

## License

Apache-2.0
