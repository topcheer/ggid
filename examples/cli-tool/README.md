# GGID CLI Tool Demo

A command-line tool demonstrating **OAuth 2.0 Device Authorization Grant (RFC 8628)** authentication with GGID.

## Quick Start

```bash
# Login (opens browser for device code flow)
go run main.go login

# Show current user info
go run main.go whoami

# Print access token
go run main.go token

# Logout
go run main.go logout
```

## How It Works

1. **login** — Calls GGID `/api/v1/oauth/device_authorization` to get a device code and user code. Displays the verification URL and user code. Polls `/api/v1/oauth/token` until the user completes authentication in their browser.

2. **whoami** — Uses the stored access token to call GGID `/api/v1/oauth/userinfo` and displays user identity.

3. **token** — Prints the raw JWT access token (useful for piping to other tools).

4. **logout** — Deletes the stored token file.

## Use Cases

- CLI tools that need user authentication without embedding a browser
- CI/CD pipelines that need human approval before proceeding
- IoT devices with limited input capabilities
- SSH sessions where opening a browser is impractical

## Token Storage

Tokens are stored in `~/.ggid-cli-token.json` with the access token, refresh token, and expiry time.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GGID_URL` | `https://ggid.iot2.win` | GGID gateway URL |
| `GGID_TENANT` | `00000000-...` | Tenant ID |
