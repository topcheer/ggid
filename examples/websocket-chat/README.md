# GGID WebSocket Chat Demo

A real-time WebSocket chat room demonstrating **GGID JWT authentication** — clients must present a valid GGID JWT to connect.

## Architecture

```
Client (Browser)                          Server (Node.js)
     │                                        │
     │  1. POST /api/v1/auth/login            │
     │  ─────────────────────────────────────►│  (GGID Gateway)
     │  ◄─────────────── access_token (JWT)   │
     │                                        │
     │  2. ws://host/ws?token=<JWT>           │
     │  ─────────────────────────────────────►│  3. Verify JWT via JWKS
     │                                        │     jwks-rsa → GGID /oauth/jwks
     │  ◄─── 4001 close (if invalid)         │     RS256 signature check
     │  OR                                    │     expiry check (60s skew)
     │  ◄─── connection accepted ────────────│
     │                                        │
     │  4. { type: "message", text: "hi" }   │
     │  ─────────────────────────────────────►│  Broadcast to all clients
     │  ◄─── broadcast with user_id + roles  │
```

## Quick Start

### 1. Install

```bash
cd examples/websocket-chat
npm install
```

### 2. Configure

```bash
export GGID_URL=https://ggid.iot2.win
export GGID_TENANT_ID=00000000-0000-0000-0000-000000000001
export PORT=5050
```

### 3. Run

```bash
npm start
```

Open `http://localhost:5050` in your browser.

### 4. Login

Enter your GGID username and password. The client will:
1. Login via GGID API to get a JWT
2. Connect to WebSocket with the JWT as query param
3. Server verifies the JWT against GGID JWKS
4. On success, you join the chat room

### Alternative: Direct Token

Get a token via CLI:
```bash
node get-token.js admin password123
```

Then paste the JWT into the "connect with a JWT token directly" field.

## How JWT Verification Works

1. **Client connects**: `ws://localhost:5050/ws?token=eyJhbG...`
2. **Server extracts** the `token` query parameter
3. **Server verifies** the JWT:
   - Fetches public keys from `${GGID_URL}/oauth/jwks` (cached 5 min)
   - Matches `kid` in JWT header to the correct signing key
   - Verifies RS256 signature
   - Checks expiry (60s clock tolerance)
4. **On failure**: WebSocket closed with code `4001` and reason
5. **On success**: User identity (`sub`, `roles`, `name`, `email`) is attached to the connection

## Close Codes

| Code | Meaning |
|------|---------|
| 4001 | Missing or invalid JWT token |
| 1000 | Normal closure |

## Message Types

### Client → Server

```json
{ "type": "message", "text": "Hello world" }
{ "type": "ping" }
```

### Server → Client

```json
{ "type": "system", "message": "Welcome!", "user": { "userId": "...", "name": "Alice", "roles": ["admin"] }, "online": 3 }
{ "type": "user_joined", "user": { "userId": "...", "name": "Bob", "roles": ["user"] }, "online": 4 }
{ "type": "user_left", "user": { "userId": "...", "name": "Bob", "roles": ["user"] }, "online": 3 }
{ "type": "message", "user": { "userId": "...", "name": "Alice", "roles": ["admin"] }, "text": "Hello world", "timestamp": 1700000000000 }
{ "type": "pong", "timestamp": 1700000000000 }
```

## Files

```
websocket-chat/
├── package.json          # express, ws, jsonwebtoken, jwks-rsa
├── server.js             # WebSocket server + JWT verification
├── get-token.js          # Helper: login and print JWT
├── public/index.html     # Chat client (login + chat UI)
└── README.md
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GGID_URL` | `https://ggid.iot2.win` | GGID gateway URL |
| `GGID_TENANT_ID` | `00000000-0000-0000-0000-000000000001` | Tenant UUID |
| `PORT` | `5050` | HTTP/WS server port |

## Security Notes

- JWT is passed as a query parameter (suitable for WebSocket — headers are not supported in browser WS API)
- Use `wss://` (TLS) in production to protect the token in transit
- JWKS keys are cached for 5 minutes with rate limiting (10 requests/min)
- Messages are truncated to 1000 characters
- The token is never stored server-side; it's verified on connection only

## License

Apache-2.0
