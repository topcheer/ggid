# Deployment Troubleshooting

> Common deployment issues and their solutions across Docker, Kubernetes, and bare metal.

---

## Quick Diagnosis

```bash
# Docker
docker compose ps
docker compose logs --tail=50 <service>

# Kubernetes
kubectl get pods -n ggid
kubectl describe pod <pod> -n ggid
kubectl logs <pod> -n ggid

# Bare metal
sudo systemctl status ggid-gateway
sudo journalctl -u ggid-gateway --since "10 min ago"
```

---

## Common Issues

### Container / Pod Won't Start

| Symptom | Cause | Solution |
|---------|-------|----------|
| `OOMKilled` | Memory limit too low | Increase `resources.limits.memory` in Helm values or docker-compose |
| `CrashLoopBackOff` | App error on startup | Check logs: `kubectl logs <pod>` or `docker logs <container>` |
| `ErrImagePull` | Image not in registry | Verify image exists: `docker pull <image>`; check `imageRegistry` config |
| `ImagePullBackOff` | Auth to private registry | Create imagePullSecret: `kubectl create secret docker-registry ...` |
| `Pending` (K8s) | No available storage | Check StorageClass; K3s uses `local-path`, EKS uses `gp3` |
| Port conflict | Another process on same port | `lsof -i :8080` (macOS) or `ss -tlnp | grep 8080` (Linux) |

### Database Issues

| Symptom | Cause | Solution |
|---------|-------|----------|
| `connection refused` to DB | PostgreSQL not running | `docker compose start postgres` or `sudo systemctl start postgresql` |
| `role ggid does not exist` | DB user not created | Run migrations: `psql -f deploy/migrations/01_all_up.sql` |
| `relation does not exist` | Migrations not run | Run all SQL files in `deploy/migrations/` in order |
| `too many connections` | Pool exhaustion | Increase `max_connections` in PostgreSQL config |
| Policy/Org/Audit DB error | Wrong env var format | These services use `DB_HOST`/`DB_PORT`/etc, NOT `DATABASE_URL` |

### Authentication Issues

| Symptom | Cause | Solution |
|---------|-------|----------|
| 401 on all requests | JWT_SECRET mismatch | Ensure same JWT_SECRET across gateway, auth, oauth |
| 401 on all requests | Token expired | Refresh: `POST /api/v1/auth/refresh` with refresh token |
| 429 Too Many Requests | Rate limit exceeded | Wait 60s, or flush Redis: `redis-cli FLUSHDB` (dev only) |
| 403 Forbidden | Missing scope | Assign role with required permission via Policy API |
| 403 `mfa_required` | Step-up auth required | Complete MFA: `POST /api/v1/auth/mfa/verify` |
| Register returns 409 | Username exists | Use a different `username` value |
| Register returns 500 | Empty `username` field | `username` is the credential identifier (not `email`) |
| Login returns 423 | Account locked | Brute-force protection: wait 60s or restart auth container (dev) |

### Networking Issues

| Symptom | Cause | Solution |
|---------|-------|----------|
| 502 Bad Gateway | Backend service down | Check service health: `curl localhost:8080/healthz/deep` |
| 502 Bad Gateway | Service URL mismatch | Verify backend service name in gateway route table |
| 503 Service Unavailable | Circuit breaker open | Wait for reset timeout (30s default) or check failing backend |
| Connection timeout | Firewall blocking | Check security groups / iptables rules |
| DNS resolution failure | CoreDNS issue (K8s) | `kubectl rollout restart deployment coredns -n kube-system` |

### TLS / Certificate Issues

| Symptom | Cause | Solution |
|---------|-------|----------|
| `ERR_CERT_AUTHORITY_INVALID` | Self-signed cert | Install CA cert or use Let's Encrypt |
| Certificate expired | Auto-renewal failed | `certbot renew` or check cert-manager: `kubectl get certificates -A` |
| `ERR_SSL_PROTOCOL_ERROR` | TLS version mismatch | Ensure TLS 1.2+; check nginx/ingress config |
| HSTS redirect loop | HTTPS not configured | Ensure backend serves HTTPS or terminate TLS at ingress |

### NATS / Audit Issues

| Symptom | Cause | Solution |
|---------|-------|----------|
| NATS unhealthy | Missing `-m 8222` flag | Add monitoring flag: `command: ["-js", "-m", "8222"]` |
| Audit events missing | NATS JetStream not enabled | Ensure `-js` flag in NATS command |
| Audit 404 | Route mismatch | Use `/api/v1/audit` (alias) not just `/api/v1/audit/events` |
| Webhook not delivered | SSRF protection blocking URL | Webhook URL must NOT resolve to private IP range |

### Console Issues

| Symptom | Cause | Solution |
|---------|-------|----------|
| Blank page after login | `NEXT_PUBLIC_GGID_URL` not set | Set env var to gateway URL (e.g., `http://localhost:8080`) |
| CORS errors in browser | Gateway CORS not configured | Set `CORS_ALLOWED_ORIGINS` to console origin |
| API calls return 401 | Console not sending JWT | Check `Authorization: Bearer` header in browser DevTools |

---

## Debug Checklist

1. Check container/pod status — is it Running?
2. Check logs — any errors on startup?
3. Check env vars — are required vars set? (`JWT_SECRET`, `DATABASE_URL`, `REDIS_URL`)
4. Check network — can services reach each other?
5. Check health endpoint — `curl /healthz/deep`
6. Check dependencies — PostgreSQL, Redis, NATS all healthy?
7. Check JWT_SECRET — same across all services?

---

## Getting Help

- [Architecture Overview](../architecture.md)
- [Operations Runbook](../operations-runbook.md)
- [Troubleshooting Guide](../troubleshooting.md)
- [GitHub Issues](https://github.com/ggid/ggid/issues)

---

*Last updated: 2025-07-11*