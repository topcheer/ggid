package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// DevicePosture represents a device's security posture signals and evaluation.
type DevicePosture struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID      `json:"tenant_id"`
	DeviceID        string         `json:"device_id"`
	UserID          *uuid.UUID     `json:"user_id,omitempty"`
	TrustLevel      string         `json:"trust_level"`
	ComplianceScore int            `json:"compliance_score"`
	Compliant       bool           `json:"compliant"`
	Checks          map[string]any `json:"checks"`
	LastCheckAt     *time.Time     `json:"last_check_at,omitempty"`
	LastSeen        *time.Time     `json:"last_seen,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// PostureCheckInput is the payload for updating device posture.
type PostureCheckInput struct {
	DeviceID string         `json:"device_id"`
	UserID   string         `json:"user_id,omitempty"`
	Checks   map[string]any `json:"checks"`
}

// PostureResult includes the evaluation outcome.
type PostureResult struct {
	DeviceID        string         `json:"device_id"`
	Compliant       bool           `json:"compliant"`
	PostureScore    int            `json:"posture_score"`
	TrustLevel      string         `json:"trust_level"`
	Checks          map[string]any `json:"checks"`
	PolicyResults   []PolicyResult `json:"policy_results,omitempty"`
	EvaluatedAt     time.Time      `json:"evaluated_at"`
	ExpiresAt       time.Time      `json:"expires_at"`
}

// PolicyResult is the outcome of evaluating one posture policy.
type PolicyResult struct {
	Policy   string `json:"policy"`
	Result   string `json:"result"` // compliant, non_compliant
	Message  string `json:"message,omitempty"`
}

// devicePostureRepo manages device posture persistence + Redis cache.
type devicePostureRepo struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
}

func newDevicePostureRepo(pool *pgxpool.Pool, rdb *redis.Client) *devicePostureRepo {
	return &devicePostureRepo{pool: pool, rdb: rdb}
}

func (r *devicePostureRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS device_posture (
			id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id       UUID NOT NULL,
			device_id       TEXT NOT NULL,
			user_id         UUID,
			trust_level     TEXT NOT NULL DEFAULT 'unknown',
			compliance_score INT NOT NULL DEFAULT 0,
			compliant       BOOLEAN NOT NULL DEFAULT FALSE,
			checks          JSONB NOT NULL DEFAULT '{}',
			last_check_at   TIMESTAMPTZ,
			last_seen       TIMESTAMPTZ,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE(tenant_id, device_id)
		);
		CREATE INDEX IF NOT EXISTS idx_device_posture_user ON device_posture(tenant_id, user_id);
		CREATE INDEX IF NOT EXISTS idx_device_posture_device ON device_posture(tenant_id, device_id);
	`)
	return err
}

// Upsert inserts or updates device posture signals.
func (r *devicePostureRepo) Upsert(ctx context.Context, dp *DevicePosture) error {
	if r.pool == nil {
		return nil
	}
	checksJSON, _ := json.Marshal(dp.Checks)
	now := time.Now()
	dp.LastCheckAt = &now
	dp.UpdatedAt = now

	// Evaluate compliance before storing.
	evaluatePosture(dp)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO device_posture (tenant_id, device_id, user_id, trust_level, compliance_score, compliant, checks, last_check_at, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		ON CONFLICT (tenant_id, device_id) DO UPDATE SET
			trust_level = EXCLUDED.trust_level, compliance_score = EXCLUDED.compliance_score,
			compliant = EXCLUDED.compliant, checks = EXCLUDED.checks,
			last_check_at = EXCLUDED.last_check_at, last_seen = EXCLUDED.last_seen, updated_at = now()`,
		dp.TenantID, dp.DeviceID, dp.UserID, dp.TrustLevel, dp.ComplianceScore, dp.Compliant,
		checksJSON, now)
	if err != nil {
		return err
	}

	// Invalidate Redis cache.
	r.invalidateCache(ctx, dp.TenantID, dp.DeviceID)
	return nil
}

// GetByDevice retrieves posture for a device (with Redis cache).
func (r *devicePostureRepo) GetByDevice(ctx context.Context, tenantID uuid.UUID, deviceID string) (*DevicePosture, error) {
	if r.rdb != nil {
		key := fmt.Sprintf("ggid:posture:%s:%s", tenantID, deviceID)
		if val, err := r.rdb.Get(ctx, key).Result(); err == nil {
			var cached DevicePosture
			if json.Unmarshal([]byte(val), &cached) == nil {
				return &cached, nil
			}
		}
	}

	if r.pool == nil {
		return nil, nil
	}

	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, device_id, user_id, trust_level, compliance_score, compliant, checks, last_check_at, last_seen, created_at, updated_at
		FROM device_posture WHERE tenant_id = $1 AND device_id = $2`, tenantID, deviceID)

	var dp DevicePosture
	var checksJSON []byte
	err := row.Scan(&dp.ID, &dp.TenantID, &dp.DeviceID, &dp.UserID, &dp.TrustLevel, &dp.ComplianceScore, &dp.Compliant, &checksJSON, &dp.LastCheckAt, &dp.LastSeen, &dp.CreatedAt, &dp.UpdatedAt)
	if err != nil {
		return nil, nil // not found
	}
	if len(checksJSON) > 0 {
		json.Unmarshal(checksJSON, &dp.Checks)
	}

	// Cache for 5 minutes.
	if r.rdb != nil {
		key := fmt.Sprintf("ggid:posture:%s:%s", tenantID, deviceID)
		if data, err := json.Marshal(dp); err == nil {
			r.rdb.Set(ctx, key, data, 5*time.Minute)
		}
	}

	return &dp, nil
}

func (r *devicePostureRepo) invalidateCache(ctx context.Context, tenantID uuid.UUID, deviceID string) {
	if r.rdb == nil {
		return
	}
	key := fmt.Sprintf("ggid:posture:%s:%s", tenantID, deviceID)
	r.rdb.Del(ctx, key)
}

// evaluatePosture calculates compliance_score and compliant flag from checks.
// Each check contributes to the score. Critical failures set compliant=false.
func evaluatePosture(dp *DevicePosture) {
	score := 0
	maxScore := 0
	compliant := true

	// Define weighted checks.
	checks := []struct {
		key      string
		weight   int
		critical bool
	}{
		{"disk_encrypted", 15, true},
		{"screen_lock_enabled", 10, false},
		{"antivirus_active", 15, true},
		{"firewall_enabled", 10, false},
		{"os_up_to_date", 15, true},
		{"jailbroken", 20, true}, // inverted: false = good
		{"managed", 15, false},
	}

	for _, c := range checks {
		maxScore += c.weight
		val, exists := dp.Checks[c.key]
		if !exists {
			continue
		}

		if c.key == "jailbroken" {
			isJailbroken, _ := val.(bool)
			if !isJailbroken {
				score += c.weight
			} else if c.critical {
				compliant = false
			}
		} else {
			isOK, _ := val.(bool)
			if isOK {
				score += c.weight
			} else if c.critical {
				compliant = false
			}
		}
	}

	if maxScore > 0 {
		dp.ComplianceScore = score * 100 / maxScore
	} else {
		dp.ComplianceScore = 0
	}
	dp.Compliant = compliant && dp.ComplianceScore >= 60

	// Set trust level based on score and compliance.
	if !compliant {
		dp.TrustLevel = "untrusted" // any critical failure → untrusted
	} else if dp.ComplianceScore >= 85 {
		dp.TrustLevel = "trusted"
	} else if dp.ComplianceScore >= 60 {
		dp.TrustLevel = "conditional"
	} else {
		dp.TrustLevel = "untrusted"
	}
}

// handleDevicePosture routes device posture endpoints.
// GET  /api/v1/identity/devices/{id}/posture
// PUT  /api/v1/identity/devices/{id}/posture
func (h *HTTPHandler) handleDevicePosture(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	// Extract device_id from path: /api/v1/identity/devices/{id}/posture
	parts := splitPath(r.URL.Path)
	if len(parts) < 5 {
		writeJSONError(w, http.StatusBadRequest, "device id required")
		return
	}
	deviceID := parts[3]
	if deviceID == "" {
		writeJSONError(w, http.StatusBadRequest, "device id required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		dp, err := h.devicePostureRepo.GetByDevice(r.Context(), tc.TenantID, deviceID)
		if err != nil || dp == nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"device_id":    deviceID,
				"compliant":    false,
				"posture_score": 0,
				"trust_level":  "unknown",
				"message":      "no posture data for this device",
			})
			return
		}

		now := time.Now()
		result := PostureResult{
			DeviceID:     dp.DeviceID,
			Compliant:    dp.Compliant,
			PostureScore: dp.ComplianceScore,
			TrustLevel:   dp.TrustLevel,
			Checks:       dp.Checks,
			EvaluatedAt:  now,
			ExpiresAt:    now.Add(5 * time.Minute),
		}

		writeJSON(w, http.StatusOK, result)

	case http.MethodPut:
		var input PostureCheckInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid body")
			return
		}
		input.DeviceID = deviceID

		dp := &DevicePosture{
			TenantID: tc.TenantID,
			DeviceID: deviceID,
			Checks:   input.Checks,
		}
		if input.UserID != "" {
			uid, _ := uuid.Parse(input.UserID)
			dp.UserID = &uid
		}

		if err := h.devicePostureRepo.Upsert(r.Context(), dp); err != nil {
			slog.Error("device posture upsert error", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to update posture")
			return
		}

		// Emit CAEP-style event for compliance change via NATS.
		if h.auditPublisher != nil {
			_ = h.auditPublisher // NATS publish would go here in full integration
		}
		slog.Info("CAEP device-compliance-change",
			"device", deviceID, "score", dp.ComplianceScore, "compliant", dp.Compliant, "trust", dp.TrustLevel)

		writeJSON(w, http.StatusOK, dp)

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func splitPath(path string) []string {
	return splitString(path, '/')
}

func splitString(s string, sep byte) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}
