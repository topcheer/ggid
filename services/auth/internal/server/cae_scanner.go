package server

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StartCAEScanner launches a background goroutine that periodically scans
// all active sessions and evaluates them against Conditional Access Policies.
// If a policy denies access, the session is revoked via the SessionRevocationManager.
//
// The scanner runs every 15 minutes by default. It can be cancelled via the
// provided context.
func (h *Handler) StartCAEScanner(ctx context.Context, pool *pgxpool.Pool, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Minute
	}

	go func() {
		log.Printf("CAE: background scanner started (interval=%s)", interval)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("CAE: background scanner stopped")
				return
			case <-ticker.C:
				h.runCAESweep(ctx, pool)
			}
		}
	}()
}

// runCAESweep queries all active sessions and evaluates each against CAP policies.
func (h *Handler) runCAESweep(ctx context.Context, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}

	// Query active sessions (not expired, within the last hour of activity).
	rows, err := pool.Query(ctx, `
		SELECT id, tenant_id, user_id,
		       COALESCE(ip_address::text, ''),
		       0
		FROM sessions
		WHERE expires_at > now()
		  AND revoked_at IS NULL
		  AND created_at > now() - interval '24 hours'
		LIMIT 500
	`)
	if err != nil {
		log.Printf("CAE: failed to query sessions: %v", err)
		return
	}
	defer rows.Close()

	evaluated := 0
	denied := 0

	for rows.Next() {
		var (
			sessionID uuid.UUID
			tenantID  uuid.UUID
			userID    string
			ipAddress string
			riskScore int
		)
		if err := rows.Scan(&sessionID, &tenantID, &userID, &ipAddress, &riskScore); err != nil {
			continue
		}

		action := h.EvaluateSessionForCAE(tenantID, sessionID.String(), userID, ipAddress, riskScore)
		evaluated++

		if action == "deny" || action == "revoke" {
			denied++
			log.Printf("CAE: session %s denied for user %s (policy=%s), revoking", sessionID, userID, action)

			// Revoke the session: delete it from the DB and invalidate tokens.
			if _, err := pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID); err != nil {
				log.Printf("CAE: failed to delete session %s: %v", sessionID, err)
			}

			// Publish session revocation event for other services.
			if h.auditPublisher != nil {
				h.publishAuditEventWithMeta(nil, "session.cae_revoke", "success",
					"session", sessionID.String(), sessionID,
					map[string]any{
						"user_id":    userID,
						"action":     action,
						"risk_score": riskScore,
						"ip_address": ipAddress,
					})
			}
		}
	}

	if evaluated > 0 {
		log.Printf("CAE: sweep completed — %d sessions evaluated, %d denied", evaluated, denied)
	}
}
