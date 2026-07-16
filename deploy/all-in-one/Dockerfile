# syntax=docker/dockerfile:1
# All-in-one GGID image: PostgreSQL + Redis + NATS + all 7 services + console in one container.
# Usage:
#   docker build -f deploy/all-in-one/Dockerfile -t ggid/ggid-all-in-one:latest .
#   docker run -p 8080:8080 -p 3000:3000 ggid/ggid-all-in-one:latest

# ---------------- Stage 1: Go service builder ----------------
FROM golang:1.25-alpine AS go-builder
WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build all service binaries
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/identity-server ./services/identity/cmd/ && \
    CGO_ENABLED=0 go build -o /bin/auth-server ./services/auth/cmd/ && \
    CGO_ENABLED=0 go build -o /bin/oauth-server ./services/oauth/cmd/ && \
    CGO_ENABLED=0 go build -o /bin/policy-server ./services/policy/cmd/ && \
    CGO_ENABLED=0 go build -o /bin/org-server ./services/org/cmd/ && \
    CGO_ENABLED=0 go build -o /bin/audit-server ./services/audit/cmd/ && \
    CGO_ENABLED=0 go build -o /bin/gateway-server ./services/gateway/cmd/ && \
    CGO_ENABLED=0 go build -o /bin/ggid-migrate ./services/ggid-migrate/ && \
    CGO_ENABLED=0 go build -o /bin/mcp-server ./services/mcp/cmd/

# ---------------- Stage 2: Console SDK builder ----------------
FROM node:22-alpine AS console-sdk-builder
WORKDIR /sdk

# Build React SDK
COPY sdk/react/package.json sdk/react/tsconfig.json ./
COPY sdk/react/src ./src
RUN npm install --production=false && npm run build

# ---------------- Stage 3: Console app builder ----------------
FROM node:22-alpine AS console-builder
WORKDIR /app
COPY console/package.json ./
RUN npm install --no-package-lock

COPY --from=console-sdk-builder /sdk/package.json /tmp/sdk-pkg.json
COPY --from=console-sdk-builder /sdk/dist /tmp/sdk-dist
RUN mkdir -p node_modules/@ggid/sdk-react && \
    cp /tmp/sdk-pkg.json node_modules/@ggid/sdk-react/package.json && \
    cp -r /tmp/sdk-dist node_modules/@ggid/sdk-react/dist

COPY console/ ./
ENV NEXT_TELEMETRY_DISABLED=1
ENV NEXT_PUBLIC_API_URL=
ENV NEXT_PUBLIC_TENANT_ID=00000000-0000-0000-0000-000000000001
ENV PGPASSWORD=ggid
ENV GATEWAY_URL=http://127.0.0.1:8080
RUN npm run build

# ---------------- Stage 4: Final runtime image ----------------
FROM node:22-alpine AS runtime
WORKDIR /app

# Install system dependencies: supervisord, PostgreSQL, Redis, NATS, openssl, curl
RUN apk add --no-cache \
    supervisor \
    postgresql16 postgresql16-contrib \
    redis \
    nats-server \
    openssl \
    curl \
    ca-certificates \
    tzdata

# Create postgres user and data directory
RUN mkdir -p /var/lib/postgresql/data /run/postgresql /var/log/postgresql && \
    chown -R postgres:postgres /var/lib/postgresql /run/postgresql /var/log/postgresql

# Copy all Go binaries
COPY --from=go-builder /bin/identity-server /usr/local/bin/
COPY --from=go-builder /bin/auth-server /usr/local/bin/
COPY --from=go-builder /bin/oauth-server /usr/local/bin/
COPY --from=go-builder /bin/policy-server /usr/local/bin/
COPY --from=go-builder /bin/org-server /usr/local/bin/
COPY --from=go-builder /bin/audit-server /usr/local/bin/
COPY --from=go-builder /bin/gateway-server /usr/local/bin/
COPY --from=go-builder /bin/mcp-server /usr/local/bin/
COPY --from=go-builder /bin/ggid-migrate /usr/local/bin/

# Copy migration SQL files
COPY --from=go-builder /build/deploy/migrations /app/migrations

# Copy console standalone build
COPY --from=console-builder /app/.next/standalone /app/console
COPY --from=console-builder /app/.next/static /app/console/.next/static
COPY --from=console-builder /app/public /app/console/public

# Copy supervisord config and entrypoint
COPY deploy/all-in-one/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY deploy/all-in-one/entrypoint.sh /app/entrypoint.sh
COPY deploy/all-in-one/postgres-start.sh /app/postgres-start.sh
COPY deploy/all-in-one/wait-for-db.sh /app/wait-for-db.sh
COPY deploy/seed.sh /app/seed.sh
RUN chmod +x /app/entrypoint.sh /app/postgres-start.sh /app/wait-for-db.sh /app/seed.sh

# Generate RSA key pair for JWT signing
RUN mkdir -p /app/configs && \
    openssl genrsa -out /app/configs/rsa_private.pem 2048 && \
    openssl rsa -in /app/configs/rsa_private.pem -pubout -out /app/configs/rsa_public.pem

# Environment variables for all-in-one mode
ENV DB_HOST=localhost
ENV DB_PORT=5432
ENV DB_USER=ggid
ENV DB_PASSWORD=ggid
ENV DB_DATABASE=ggid
ENV DB_SSL_MODE=disable
ENV DATABASE_URL=postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable
ENV REDIS_URL=redis://localhost:6379
ENV NATS_URL=nats://localhost:4222
ENV GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true
ENV GATEWAY_ADDR=:8080
ENV JWT_PUBLIC_KEY_PATH=/app/configs/rsa_public.pem
ENV OAUTH_PRIVATE_KEY_PATH=/app/configs/rsa_private.pem
ENV OAUTH_PUBLIC_KEY_PATH=/app/configs/rsa_public.pem
ENV IDENTITY_HTTP_ADDR=:8081
ENV IDENTITY_GRPC_ADDR=:50051
ENV AUTH_HTTP_ADDR=:9001
ENV AUTH_GRPC_ADDR=:50052
ENV OAUTH_HTTP_ADDR=:9005
ENV OAUTH_GRPC_ADDR=:50055
ENV POLICY_HTTP_ADDR=:8070
ENV POLICY_GRPC_ADDR=:9070
ENV ORG_HTTP_ADDR=:8071
ENV ORG_GRPC_ADDR=:9071
ENV AUDIT_HTTP_ADDR=:8072
ENV AUDIT_GRPC_ADDR=:9072
ENV IDENTITY_SERVICE_URL=http://127.0.0.1:8081
ENV AUTH_SERVICE_URL=http://127.0.0.1:9001
ENV USERS_SERVICE_URL=http://127.0.0.1:8081
ENV OAUTH_SERVICE_URL=http://127.0.0.1:9005
ENV POLICY_SERVICE_URL=http://127.0.0.1:8070
ENV ORG_SERVICE_URL=http://127.0.0.1:8071
ENV AUDIT_SERVICE_URL=http://127.0.0.1:8072
ENV OPERATOR_SERVICE_URL=http://127.0.0.1:9090
ENV GATEWAY_URL=http://127.0.0.1:8080
ENV HOSTNAME=
ENV NEXT_PUBLIC_API_URL=
ENV NEXT_PUBLIC_TENANT_ID=00000000-0000-0000-0000-000000000001
ENV PGPASSWORD=ggid

# Ports: gateway + console + internal services (optional)
# Only expose gateway + console to host. Backend services communicate internally.
# Do NOT expose backend ports (8081/9001/9005/8070-8072) — they bypass gateway auth.
EXPOSE 8080 3000

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
