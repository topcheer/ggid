package httpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	neturl "net/url"
	"os"
	"sync"

	"github.com/google/uuid"
)

type alertWebhookConfig struct {
	mu       sync.RWMutex
	webhooks []map[string]any
}

var globalAlertWebhooks = &alertWebhookConfig{}

// POST/GET/DELETE /api/v1/audit/alert-webhooks
// DB-backed: uses audit_alert_webhooks table. Falls back to in-memory when pool is nil.
// P1 fix: All operations now require and enforce tenant_id isolation.
func (s *HTTPServer) handleAlertWebhooks(w http.ResponseWriter, r *http.Request) {
	// Require tenant context for all operations
	tid := r.Header.Get("X-Tenant-ID")
	if tid == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "missing tenant context"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		if s.pool != nil {
			rows, err := s.pool.Query(r.Context(), `
				SELECT id::text, url, COALESCE(secret, ''), active, created_at
				FROM audit_alert_webhooks WHERE tenant_id::text = $1 ORDER BY created_at DESC`, tid)
			if err == nil {
				defer rows.Close()
				webhooks := []map[string]any{}
				for rows.Next() {
					var id, url, secret string
					var active bool
					var created interface{}
					_ = rows.Scan(&id, &url, &secret, &active, &created)
					webhooks = append(webhooks, map[string]any{
						"id": id, "url": url, "secret": maskSecret(secret), "active": active, "created_at": created,
					})
				}
				writeJSON(w, http.StatusOK, map[string]any{"webhooks": webhooks})
				return
			}
		}
		// In-memory fallback: filter by tenant
		globalAlertWebhooks.mu.RLock()
		defer globalAlertWebhooks.mu.RUnlock()
		filtered := []map[string]any{}
		for _, h := range globalAlertWebhooks.webhooks {
			if h["tenant_id"] == tid {
				filtered = append(filtered, h)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"webhooks": filtered})

	case http.MethodPost:
		var req struct {
			URL    string `json:"url"`
			Secret string `json:"secret"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if req.URL == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url required"})
			return
		}
		// SECURITY: SSRF prevention — reject localhost, private IPs, link-local
		if err := validateWebhookURL(req.URL); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		hookID := uuid.New().String()
		// SECURITY: hash the secret before storing (never store plaintext)
		secretHash := ""
		if req.Secret != "" {
			h := sha256.Sum256([]byte(req.Secret))
			secretHash = hex.EncodeToString(h[:])
		}
		hook := map[string]any{
			"id":        hookID,
			"url":       req.URL,
			"secret":    "", // never return secret in response
			"active":    true,
			"tenant_id": tid,
		}
		if s.pool != nil {
			_, err := s.pool.Exec(r.Context(), `
				INSERT INTO audit_alert_webhooks (id, tenant_id, url, secret, active)
				VALUES ($1, $2, $3, $4, true)`, hookID, tid, req.URL, secretHash)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save webhook"})
				return
			}
		} else {
			globalAlertWebhooks.mu.Lock()
			globalAlertWebhooks.webhooks = append(globalAlertWebhooks.webhooks, hook)
			globalAlertWebhooks.mu.Unlock()
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":        hookID,
			"url":       req.URL,
			"secret":    maskSecret(req.Secret),
			"active":    true,
			"tenant_id": tid,
		})

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
			return
		}
		if s.pool != nil {
			_, err := s.pool.Exec(r.Context(), `DELETE FROM audit_alert_webhooks WHERE id::text = $1 AND tenant_id::text = $2`, id, tid)
			if err == nil {
				writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
				return
			}
		}
		// In-memory fallback: filter by id AND tenant
		globalAlertWebhooks.mu.Lock()
		defer globalAlertWebhooks.mu.Unlock()
		for i, h := range globalAlertWebhooks.webhooks {
			if h["id"] == id && h["tenant_id"] == tid {
				globalAlertWebhooks.webhooks = append(globalAlertWebhooks.webhooks[:i], globalAlertWebhooks.webhooks[i+1:]...)
				writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
				return
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "webhook not found"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// validateWebhookURL prevents SSRF by rejecting localhost, private IPs,
// link-local, and non-HTTP(S) schemes.
func validateWebhookURL(rawURL string) error {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http/https URLs allowed")
	}
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0" {
		return fmt.Errorf("localhost URLs not allowed")
	}
	// Reject private network ranges
	if ip, err := netip.ParseAddr(host); err == nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return fmt.Errorf("internal network URLs not allowed")
		}
		return nil
	}
	// Hostnames (e.g. kubernetes.default.svc, metadata.google.internal)
	// must not resolve to internal addresses (R11 P1: literal-IP check
	// alone let every internal DNS name through). The DNS check can be
	// disabled explicitly for tests/dev via env.
	if os.Getenv("GGID_WEBHOOK_SKIP_DNS") == "true" {
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("webhook host does not resolve")
	}
	for _, ip := range ips {
		if addr, ok := netip.AddrFromSlice(ip); ok {
			a := addr.Unmap()
			if a.IsLoopback() || a.IsPrivate() || a.IsLinkLocalUnicast() || a.IsLinkLocalMulticast() || a.IsUnspecified() {
				return fmt.Errorf("webhook host resolves to internal address")
			}
		}
	}
	return nil
}
