# Production Readiness Checklist

> Pre-deployment checklist with verification commands for each item.

---

## TLS & Network

- [ ] TLS certificate valid
  ```bash
  openssl s_client -connect iam.example.com:443 -servername iam.example.com </dev/null 2>/dev/null | openssl x509 -noout -dates
  ```
- [ ] HSTS header set
  ```bash
  curl -sI https://iam.example.com | grep -i strict-transport
  ```
- [ ] gRPC TLS between services
  ```bash
  grep GRPC_TLS /etc/ggid/gateway.env
  ```

## Database

- [ ] RLS enabled on all tenant tables
  ```bash
  psql $DATABASE_URL -c "SELECT relname, relrowsecurity FROM pg_class WHERE relrowsecurity = true;"
  ```
- [ ] SSL mode required
  ```bash
  psql $DATABASE_URL -c "SHOW ssl;"
  # Should show 'on'
  ```
- [ ] Daily backups configured
  ```bash
  crontab -l | grep pg_dump
  ```

## Redis

- [ ] AUTH password set
  ```bash
  redis-cli -a $REDIS_PASSWORD ping  # Should return PONG
  ```
- [ ] Persistence enabled (AOF)
  ```bash
  redis-cli CONFIG GET appendonly  # Should be 'yes'
  ```

## NATS JetStream

- [ ] Stream persisted
  ```bash
  curl http://localhost:8222/jsz?streams=true | jq .
  ```
- [ ] Audit consumer active
  ```bash
  curl http://localhost:8222/jsz?consumers=true | jq .
  ```

## JWT & Secrets

- [ ] RSA key generated (not default)
  ```bash
  ls -la /etc/ggid/keys/  # Should have private+public key
  ```
- [ ] JWT_SECRET set (non-empty)
  ```bash
  test -n "$JWT_SECRET" && echo "OK" || echo "MISSING"
  ```

## Monitoring

- [ ] Prometheus scraping /metrics
  ```bash
  curl -s http://localhost:8080/metrics | head -5
  ```
- [ ] Grafana dashboard imported
- [ ] Alerts for: 5xx rate, DB connections, NATS lag

## SIEM

- [ ] Audit events forwarding to SIEM
  ```bash
  docker logs ggid-siem-connector --tail 5
  ```

## Audit Retention

- [ ] Retention policy set
  ```bash
  grep AUDIT_RETENTION /etc/ggid/audit.env
  ```
- [ ] Hash chain verification passing
  ```bash
  curl -s http://localhost:8080/api/v1/audit/verify | jq .verified
  # Should be true
  ```

## Rate Limiting

- [ ] Gateway rate limiter enabled
  ```bash
  curl -s -o /dev/null -w '%{http_code}' -H 'X-Tenant-ID: test' http://localhost:8080/api/v1/users
  # Hit 100 times rapidly, should get 429
  ```

---

*See: [Production Checklist](../deploy/production-checklist.md) | [Security Hardening](security-hardening.md) | [Operations Runbook](../operations-runbook.md)*

*Last updated: 2025-07-11*
