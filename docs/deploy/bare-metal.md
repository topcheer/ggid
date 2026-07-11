# Bare Metal Deployment Guide

> Deploy GGID on bare metal or VMs: build from source, systemd services, nginx reverse proxy, PostgreSQL/Redis/NATS setup.

---

## Prerequisites

| Requirement | Version |
|-------------|---------|
| Go | 1.25+ |
| Node.js | 20+ |
| PostgreSQL | 16+ |
| Redis | 7+ |
| NATS | 2.10+ |
| Nginx | 1.24+ (for reverse proxy) |
| systemd | (any modern Linux) |

---

## Step 1: Build from Source

```bash
# Clone
git clone https://github.com/ggid/ggid.git
cd ggid

# Build all services
make build

# Binaries are in bin/:
# bin/gateway, bin/identity, bin/auth, bin/oauth,
# bin/policy, bin/org, bin/audit

# Build console
cd console && pnpm install && pnpm build
cd ..
```

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build all Go services |
| `make test` | Run all tests |
| `make migrate` | Run database migrations |
| `make console-build` | Build the admin console |
| `make docker-build` | Build all Docker images |
| `make clean` | Remove build artifacts |

---

## Step 2: Install Infrastructure

### PostgreSQL

```bash
# Ubuntu/Debian
sudo apt install postgresql-16

# Create database and user
sudo -u postgres psql <<'SQL'
CREATE USER ggid WITH PASSWORD 'secure-password';
CREATE DATABASE ggid OWNER ggid;
SQL

# Run migrations
psql -U ggid -d ggid -f deploy/migrations/01_all_up.sql
psql -U ggid -d ggid -f deploy/migrations/02_add_webauthn_backup_flags.sql
psql -U ggid -d ggid -f deploy/migrations/03_audit_hash_chain.sql
```

### Redis

```bash
sudo apt install redis-server

# Start and enable
sudo systemctl enable redis-server
sudo systemctl start redis-server

# Verify
redis-cli ping
# → PONG
```

### NATS

```bash
# Download
curl -L https://github.com/nats-io/nats-server/releases/latest/download/nats-server-linux-amd64.tar.gz | tar xz
sudo mv nats-server /usr/local/bin/

# Create systemd service
sudo tee /etc/systemd/system/nats.service <<'EOF'
[Unit]
Description=NATS Server
After=network.target

[Service]
ExecStart=/usr/local/bin/nats-server -js -m 8222
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable nats
sudo systemctl start nats
```

---

## Step 3: Install GGID Services

### Create System User

```bash
sudo useradd -r -s /bin/false ggid
sudo mkdir -p /opt/ggid/bin /opt/ggid/config
sudo cp bin/* /opt/ggid/bin/
```

### Environment Configuration

```bash
sudo tee /opt/ggid/config/ggid.env <<'EOF'
DATABASE_URL=postgres://ggid:secure-password@localhost:5432/ggid?sslmode=disable
REDIS_URL=redis://localhost:6379
NATS_URL=nats://localhost:4222
JWT_SECRET=your-production-jwt-secret
LOG_LEVEL=info
EOF

sudo chmod 600 /opt/ggid/config/ggid.env
sudo chown ggid:ggid /opt/ggid/config/ggid.env
```

### systemd Unit Files

#### Gateway

```bash
sudo tee /etc/systemd/system/ggid-gateway.service <<'EOF'
[Unit]
Description=GGID API Gateway
After=network.target postgresql.service redis-server.service nats.service

[Service]
Type=simple
User=ggid
EnvironmentFile=/opt/ggid/config/ggid.env
ExecStart=/opt/ggid/bin/gateway
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF
```

#### Auth Service

```bash
sudo tee /etc/systemd/system/ggid-auth.service <<'EOF'
[Unit]
Description=GGID Auth Service
After=network.target postgresql.service redis-server.service

[Service]
Type=simple
User=ggid
EnvironmentFile=/opt/ggid/config/ggid.env
ExecStart=/opt/ggid/bin/auth
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
```

#### Repeat for each service (identity, oauth, policy, org, audit)

Replace `gateway`/`auth` with each service name in the unit file.

### Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable ggid-gateway ggid-auth ggid-identity ggid-oauth ggid-policy ggid-org ggid-audit
sudo systemctl start ggid-gateway ggid-auth ggid-identity ggid-oauth ggid-policy ggid-org ggid-audit

# Check status
sudo systemctl status ggid-gateway
```

---

## Step 4: Nginx Reverse Proxy

```bash
sudo tee /etc/nginx/sites-available/ggid <<'EOF'
upstream ggid_gateway {
    server 127.0.0.1:8080;
}

server {
    listen 80;
    server_name iam.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name iam.example.com;

    ssl_certificate /etc/letsencrypt/live/iam.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/iam.example.com/privkey.pem;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;

    # Gateway
    location / {
        proxy_pass http://ggid_gateway;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    location /api/ {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://ggid_gateway;
    }
}
EOF

sudo ln -s /etc/nginx/sites-available/ggid /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### TLS with Let's Encrypt

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d iam.example.com
```

---

## Step 5: Console Deployment

```bash
# Build console
cd console
pnpm install
pnpm build

# Serve with nginx
sudo cp -r out/* /var/www/ggid-console/

# Add to nginx config
location /console {
    root /var/www;
    index index.html;
    try_files $uri $uri/ /console/index.html;
}
```

---

## Step 6: Log Management

### journald

```bash
# View logs
sudo journalctl -u ggid-gateway -f
sudo journalctl -u ggid-auth --since "1 hour ago"

# Persistent logging
sudo tee /etc/systemd/journald.conf.d/ggid.conf <<'EOF'
[Journal]
Storage=persistent
SystemMaxUse=2G
EOF
sudo systemctl restart systemd-journald
```

---

## Step 7: Health Monitoring

```bash
# Simple health check cron
sudo tee /etc/cron.d/ggid-health <<'EOF'
*/1 * * * * root curl -sf http://localhost:8080/healthz/deep > /dev/null || systemctl restart ggid-gateway
*/1 * * * * root curl -sf http://localhost:9001/healthz > /dev/null || systemctl restart ggid-auth
EOF
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Service won't start | `journalctl -u ggid-gateway` for errors |
| DB connection refused | Verify PostgreSQL running, DATABASE_URL correct |
| Redis connection refused | Verify Redis running, REDIS_URL correct |
| 502 from Nginx | Backend service down, check systemd status |
| TLS certificate expired | `sudo certbot renew` |

---

*Last updated: 2025-07-11*