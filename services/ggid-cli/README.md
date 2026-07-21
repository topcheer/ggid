# ggid-cli — GGID Remote Management CLI

A comprehensive command-line tool for managing GGID instances remotely. Authentication uses Dynamic Client Registration (RFC 7591) to register in the console (default) tenant, then exchanges client credentials for access tokens.

## Build

```bash
make cli
# Binary at: services/ggid-cli/bin/ggid
```

Or directly:

```bash
go build -o bin/ggid ./services/ggid-cli/cmd/
```

## Quick Start

```bash
# 1. Login (registers via DCR + exchanges client_credentials for token)
./ggid login --server http://localhost:8080

# 2. Verify identity
./ggid whoami

# 3. Manage your instance
./ggid users list
./ggid dashboard
./ggid system health
```

## Authentication Flow

```
ggid login
   │
   ├─► POST /api/v1/oauth/register   (DCR: register CLI as OAuth client)
   │      client_name = "ggid-cli"
   │      grant_types = [client_credentials, refresh_token]
   │      tenant = console (00000000-0000-0000-0000-000000000001)
   │   ◄──  client_id + client_secret
   │
   ├─► POST /api/v1/oauth/token       (client_credentials grant)
   │      grant_type = client_credentials
   │      client_id + client_secret
   │   ◄──  access_token
   │
   └─► Save to ~/.ggid/config.json
```

Credentials are stored in `~/.ggid/config.json` with file permissions 0600.

## Commands

### Auth

| Command | Description |
|---------|-------------|
| `login` | Authenticate via DCR + token exchange |
| `logout` | Clear stored credentials |
| `whoami` | Show current session and token claims |
| `version` | Show CLI version |

### Users

```bash
ggid users list [--page N] [--size N] [--search STR] [--status S]
ggid users get <id>
ggid users create --username X --email Y --password Z [--display-name N] [--phone P] [--roles a,b]
ggid users update <id> [--email Y] [--status active|inactive|locked]
ggid users delete <id>
ggid users lock <id>
ggid users unlock <id>
```

### Roles

```bash
ggid roles list
ggid roles get <id>
ggid roles create --name X [--description Y] [--permissions a,b,c]
ggid roles delete <id>
ggid roles assign --user <id> --role <id>
ggid roles revoke --user <id> --role <id>
```

### Organizations

```bash
ggid orgs list
ggid orgs get <id>
ggid orgs create --name X [--parent ID] [--type department]
ggid orgs delete <id>
ggid orgs tree
ggid orgs members <id>
```

### Audit

```bash
ggid audit events [--page N] [--size N] [--type T] [--status S] [--from DATE] [--to DATE]
ggid audit dashboard
```

### Policies

```bash
ggid policies list
ggid policies get <id>
ggid policies create --name X [--description Y] [--effect allow|deny]
ggid policies delete <id>
ggid policies check --user <id> --action <action> --resource <resource>
```

### OAuth Clients

```bash
ggid oauth clients list
ggid oauth clients get <id>
ggid oauth clients create --name X [--type confidential|public] [--grant-types a,b] [--scopes "openid profile"]
ggid oauth clients delete <id>
ggid oauth clients rotate-secret <id>
```

### Tenants

```bash
ggid tenants list
ggid tenants get <id>
ggid tenants create --name X --slug Y [--description Z]
ggid tenants delete <id>
ggid tenants resolve --slug Y
ggid tenants suspend <id>
ggid tenants activate <id>
```

### Sessions

```bash
ggid sessions list [--page N] [--size N]
ggid sessions revoke <id>
```

### System

```bash
ggid system health
ggid system status
ggid system bootstrap --admin-user X --admin-password Y [--admin-email Z]
ggid system initialized
ggid system routes
```

### Webhooks

```bash
ggid webhooks list
ggid webhooks create --url X --events "a,b,c"
ggid webhooks delete <id>
ggid webhooks test <id>
ggid webhooks catalog
```

### Dashboard

```bash
ggid dashboard
```

## Output Formats

| Format | Flag | Description |
|--------|------|-------------|
| Table (default) | `--table` | Human-readable tables |
| JSON | `--json` | Machine-parseable JSON |

Or set via environment variable:

```bash
export GGID_OUTPUT=json
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GGID_SERVER_URL` | Gateway base URL |
| `GGID_OUTPUT` | Output format: `json` or `table` |

## Architecture

```
services/ggid-cli/
├── cmd/
│   └── main.go                    # Entry point + command router
├── internal/
│   ├── config/
│   │   └── config.go              # ~/.ggid/config.json persistence
│   ├── client/
│   │   ├── client.go              # HTTP client (GET/POST/PUT/PATCH/DELETE)
│   │   └── auth.go                # DCR registration + token exchange
│   ├── output/
│   │   └── output.go              # JSON/table formatting utilities
│   └── commands/
│       ├── context.go             # Shared command context
│       ├── auth.go                # login/logout/whoami
│       ├── users.go               # user CRUD + lock/unlock
│       ├── roles.go               # role CRUD + assign/revoke
│       ├── organizations.go       # org CRUD + tree/members
│       ├── policies.go            # policy CRUD + check
│       ├── audit.go               # audit events + dashboard
│       ├── oauth_clients.go       # OAuth client CRUD
│       ├── tenants.go             # tenant CRUD + suspend/activate
│       ├── sessions.go            # session list + revoke
│       ├── system.go              # health/status/bootstrap/routes
│       ├── webhooks.go            # webhook CRUD + test/catalog
│       └── dashboard.go           # aggregate stats
├── Dockerfile
└── README.md
```

## Dependencies

Zero external Go dependencies — uses only the standard library.

## Security

- Config file stored at `~/.ggid/config.json` with `0600` permissions
- Config directory `~/.ggid/` created with `0700` permissions
- Client secret stored locally; only access token is sent in requests
- All API communication over HTTP (use HTTPS in production via Gateway)

## Docker

```bash
docker build -f services/ggid-cli/Dockerfile -t ggid-cli .
docker run --rm ggid-cli help
```
