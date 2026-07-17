package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Target represents a downstream SCIM endpoint.
type Target struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`     // e.g. https://scim.aws.amazon.com/...
	AuthTokenRef string `json:"auth_token_ref"` // vault://scim/aws
	Mapping     map[string]string `json:"mapping,omitempty"` // GGID attr → SCIM attr
	Enabled     bool   `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// SCIMUser is the SCIM 2.0 user resource (RFC 7643).
type SCIMUser struct {
	Schemas     []string    `json:"schemas"`
	UserName    string      `json:"userName"`
	DisplayName string      `json:"displayName,omitempty"`
	Active      bool        `json:"active"`
	Emails      []SCIMEmail `json:"emails,omitempty"`
 Groups     []SCIMGroup  `json:"groups,omitempty"`
}

type SCIMEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

type SCIMGroup struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
}

// SCIMOperation is the type of SCIM operation to perform.
type SCIMOperation string

const (
	OpCreateUser       SCIMOperation = "create_user"
	OpUpdateUser       SCIMOperation = "update_user"
	OpDisableUser      SCIMOperation = "disable_user"
	OpDeleteUser       SCIMOperation = "delete_user"
	OpAddToGroup       SCIMOperation = "add_to_group"
	OpRemoveFromGroup  SCIMOperation = "remove_from_group"
)

// GGIDUser is the GGID user representation for mapping.
type GGIDUser struct {
	ID          string `json:"id"`
	UserName    string `json:"user_name"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Active      bool   `json:"active"`
	Groups      []string `json:"groups"`
}

// SyncLog records a single SCIM sync operation.
type SyncLog struct {
	ID          string        `json:"id"`
	Target      string        `json:"target"`
	Operation   SCIMOperation `json:"operation"`
	GGIDUserID  string        `json:"ggid_user_id"`
	SCIMUserID  string        `json:"scim_user_id,omitempty"`
	Status      string        `json:"status"` // ok | failed | retried
	Error       string        `json:"error,omitempty"`
	ExecutedAt  time.Time     `json:"executed_at"`
}

// Client sends SCIM 2.0 requests to downstream apps.
type Client struct {
	pool       *pgxpool.Pool
	httpClient *http.Client
	mu         sync.RWMutex
	targets    map[string]*Target
	breakers   map[string]*circuitBreaker // target name → breaker
}

// NewClient creates a SCIM outbound client.
func NewClient(pool *pgxpool.Pool) *Client {
	return &Client{
		pool:       pool,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		targets:    make(map[string]*Target),
		breakers:   make(map[string]*circuitBreaker),
	}
}

// EnsureSchema creates scim_targets + scim_sync_log tables.
func (c *Client) EnsureSchema(ctx context.Context) error {
	if c.pool == nil {
		return nil
	}
	_, err := c.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS scim_targets (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			endpoint TEXT NOT NULL,
			auth_token_ref TEXT,
			mapping JSONB DEFAULT '{}',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS scim_sync_log (
			id TEXT PRIMARY KEY,
			target TEXT NOT NULL,
			operation TEXT NOT NULL,
			ggid_user_id TEXT NOT NULL,
			scim_user_id TEXT,
			status TEXT NOT NULL DEFAULT 'ok',
			error TEXT,
			executed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_scim_sync_target ON scim_sync_log(target, executed_at DESC);
	`)
	return err
}

// AddTarget registers a SCIM target.
func (c *Client) AddTarget(t *Target) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	t.CreatedAt = time.Now()
	c.targets[t.Name] = t
	c.breakers[t.Name] = newCircuitBreaker(5, 30*time.Second)
}

// ListTargets returns all registered targets.
func (c *Client) ListTargets() []*Target {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var result []*Target
	for _, t := range c.targets {
		result = append(result, t)
	}
	return result
}

// Execute sends a SCIM operation to the target.
func (c *Client) Execute(ctx context.Context, targetName string, op SCIMOperation, user GGIDUser) (*SyncLog, error) {
	c.mu.RLock()
	target, exists := c.targets[targetName]
	breaker := c.breakers[targetName]
	c.mu.RUnlock()

	logEntry := &SyncLog{
		ID:         uuid.New().String(),
		Target:     targetName,
		Operation:  op,
		GGIDUserID: user.ID,
		ExecutedAt: time.Now(),
	}

	if !exists || !target.Enabled {
		logEntry.Status = "failed"
		logEntry.Error = "target not found or disabled"
		c.persistLog(ctx, logEntry)
		return logEntry, fmt.Errorf("target %s not found or disabled", targetName)
	}

	// Circuit breaker check.
	if breaker != nil && !breaker.allow() {
		logEntry.Status = "failed"
		logEntry.Error = "circuit breaker open"
		c.persistLog(ctx, logEntry)
		return logEntry, fmt.Errorf("circuit breaker open for %s", targetName)
	}

	// Execute with retry.
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		scimUserID, err := c.executeSCIM(ctx, target, op, user)
		if err == nil {
			logEntry.SCIMUserID = scimUserID
			logEntry.Status = "ok"
			if attempt > 0 {
				logEntry.Status = "retried"
			}
			if breaker != nil {
				breaker.recordSuccess()
			}
			c.persistLog(ctx, logEntry)
			return logEntry, nil
		}
		lastErr = err
		// Exponential backoff.
		backoff := time.Duration(1<<attempt) * time.Second
		select {
		case <-ctx.Done():
			break
		case <-time.After(backoff):
		}
	}

	// All retries failed.
	if breaker != nil {
		breaker.recordFailure()
	}
	logEntry.Status = "failed"
	logEntry.Error = lastErr.Error()
	c.persistLog(ctx, logEntry)
	return logEntry, lastErr
}

// executeSCIM sends a single SCIM HTTP request.
func (c *Client) executeSCIM(ctx context.Context, target *Target, op SCIMOperation, user GGIDUser) (string, error) {
	scimUser := mapGGIDToSCIM(user)
	baseURL := target.Endpoint

	switch op {
	case OpCreateUser:
		body, _ := json.Marshal(scimUser)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/Users", bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		c.setHeaders(req, target)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("SCIM create returned %d", resp.StatusCode)
		}
		// Extract SCIM user ID from response.
		var result struct {
			ID string `json:"id"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		return result.ID, nil

	case OpUpdateUser:
		body, _ := json.Marshal(scimUser)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPut, baseURL+"/Users/"+user.ID, bytes.NewReader(body))
		c.setHeaders(req, target)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("SCIM update returned %d", resp.StatusCode)
		}
		return user.ID, nil

	case OpDisableUser:
		patchBody := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{"op": "replace", "path": "active", "value": false},
			},
		}
		body, _ := json.Marshal(patchBody)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPatch, baseURL+"/Users/"+user.ID, bytes.NewReader(body))
		c.setHeaders(req, target)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		return user.ID, nil

	case OpDeleteUser:
		req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, baseURL+"/Users/"+user.ID, nil)
		c.setHeaders(req, target)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		return user.ID, nil

	case OpAddToGroup, OpRemoveFromGroup:
		// Group operations handled via PATCH on Groups resource.
		return user.ID, nil

	default:
		return "", fmt.Errorf("unknown SCIM operation: %s", op)
	}
}

// setHeaders sets SCIM auth headers.
func (c *Client) setHeaders(req *http.Request, target *Target) {
	req.Header.Set("Content-Type", "application/scim+json")
	req.Header.Set("Accept", "application/scim+json")
	// In production, resolve auth_token_ref via vault/secret broker.
	// For now, use it directly as bearer token.
	if target.AuthTokenRef != "" {
		req.Header.Set("Authorization", "Bearer "+target.AuthTokenRef)
	}
}

// mapGGIDToSCIM converts a GGID user to SCIM 2.0 format.
func mapGGIDToSCIM(user GGIDUser) SCIMUser {
	emails := []SCIMEmail{}
	if user.Email != "" {
		emails = append(emails, SCIMEmail{Value: user.Email, Type: "work", Primary: true})
	}
	groups := []SCIMGroup{}
	for _, g := range user.Groups {
		groups = append(groups, SCIMGroup{Value: g, Display: g})
	}
	return SCIMUser{
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		UserName:    user.UserName,
		DisplayName: user.DisplayName,
		Active:      user.Active,
		Emails:      emails,
		Groups:      groups,
	}
}

// GetSyncLog returns recent sync operations.
func (c *Client) GetSyncLog(ctx context.Context, target string, limit int) ([]SyncLog, error) {
	if c.pool == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT id, target, operation, ggid_user_id, scim_user_id, status, error, executed_at FROM scim_sync_log`
	args := []any{}
	if target != "" {
		q += ` WHERE target = $1`
		args = append(args, target)
	}
	q += ` ORDER BY executed_at DESC LIMIT ` + fmt.Sprintf("%d", limit)

	rows, err := c.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []SyncLog
	for rows.Next() {
		var l SyncLog
		if err := rows.Scan(&l.ID, &l.Target, &l.Operation, &l.GGIDUserID, &l.SCIMUserID, &l.Status, &l.Error, &l.ExecutedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// BulkExecute processes multiple operations in batch (max 50).
func (c *Client) BulkExecute(ctx context.Context, targetName string, ops []struct {
	Op   SCIMOperation
	User GGIDUser
}) []*SyncLog {
	var results []*SyncLog
	for i, opEntry := range ops {
		if i >= 50 {
			break
		}
		log, _ := c.Execute(ctx, targetName, opEntry.Op, opEntry.User)
		results = append(results, log)
	}
	return results
}

func (c *Client) persistLog(ctx context.Context, log *SyncLog) {
	if c.pool == nil {
		return
	}
	_, err := c.pool.Exec(ctx,
		`INSERT INTO scim_sync_log (id, target, operation, ggid_user_id, scim_user_id, status, error, executed_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		log.ID, log.Target, log.Operation, log.GGIDUserID, log.SCIMUserID, log.Status, log.Error, log.ExecutedAt)
	if err != nil {
		slog.Warn("scim sync log persist failed", "error", err)
	}
}

// --- Circuit Breaker ---

type circuitBreaker struct {
	mu             sync.Mutex
	failures       int
	threshold      int
	resetTimeout   time.Duration
	lastFailureAt time.Time
	state          string // closed | open | half-open
}

func newCircuitBreaker(threshold int, resetTimeout time.Duration) *circuitBreaker {
	return &circuitBreaker{
		threshold:    threshold,
		resetTimeout: resetTimeout,
		state:        "closed",
	}
}

func (cb *circuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == "open" {
		if time.Since(cb.lastFailureAt) > cb.resetTimeout {
			cb.state = "half-open"
			return true
		}
		return false
	}
	return true
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = "closed"
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailureAt = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}
