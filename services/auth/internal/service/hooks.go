package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// HookEvent represents when a hook fires in the auth flow.
type HookEvent string

const (
	HookPreLogin     HookEvent = "pre_login"
	HookPostLogin    HookEvent = "post_login"
	HookPreRegister  HookEvent = "pre_register"
	HookPostRegister HookEvent = "post_register"
)

// AuthHook defines a tenant-customizable hook that calls an external webhook.
type AuthHook struct {
	ID       string
	TenantID string
	Event    HookEvent
	URL      string
	Headers  map[string]string
	Enabled  bool
}

// HookPayload is sent to the webhook endpoint.
type HookPayload struct {
	Event     HookEvent `json:"event"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id,omitempty"`
	Username  string    `json:"username,omitempty"`
	IP        string    `json:"ip,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Result    string    `json:"result,omitempty"` // "success" or "failure"
}

// HookResponse is the expected response from the webhook.
type HookResponse struct {
	Allow   bool   `json:"allow"`
	Message string `json:"message,omitempty"`
}

// HookManager manages auth hooks. It is concurrency-safe.
type HookManager struct {
	mu    sync.RWMutex
	hooks map[string]*AuthHook // keyed by hook ID
}

// NewHookManager creates a new HookManager.
func NewHookManager() *HookManager {
	return &HookManager{hooks: make(map[string]*AuthHook)}
}

// RegisterHook adds or updates a hook.
func (m *HookManager) RegisterHook(hook *AuthHook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks[hook.ID] = hook
}

// RemoveHook deletes a hook by ID.
func (m *HookManager) RemoveHook(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.hooks, id)
}

// ExecuteHooks calls all registered webhooks for the given event.
// Pre-hooks can block the operation (returning Allow=false stops the flow).
// Post-hooks are fire-and-forget (errors logged but don't affect flow).
func (m *HookManager) ExecuteHooks(ctx context.Context, event HookEvent, payload *HookPayload) error {
	m.mu.RLock()
	hooks := make([]*AuthHook, 0)
	for _, h := range m.hooks {
		if h.Enabled && h.Event == event {
			hooks = append(hooks, h)
		}
	}
	m.mu.RUnlock()

	for _, hook := range hooks {
		if err := m.callWebhook(ctx, hook, payload); err != nil {
			// Pre-hooks: return error to block.
			if event == HookPreLogin || event == HookPreRegister {
				return fmt.Errorf("hook %s denied: %w", hook.ID, err)
			}
			// Post-hooks: ignore errors.
		}
	}
	return nil
}

func (m *HookManager) callWebhook(ctx context.Context, hook *AuthHook, payload *HookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hook.Headers {
		req.Header.Set(k, v)
	}

	// SSRF protection: block private/loopback/link-local addresses and pin resolved IP.
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: ssrfSafeDialContext,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}

	// For pre-hooks, parse the response to check Allow.
	if hook.Event == HookPreLogin || hook.Event == HookPreRegister {
		var hookResp HookResponse
		if err := json.NewDecoder(resp.Body).Decode(&hookResp); err == nil {
			if !hookResp.Allow {
				return fmt.Errorf("operation denied: %s", hookResp.Message)
			}
		}
	}

	return nil
}

// ssrfSafeDialContext blocks connections to private/loopback/link-local IPs
// to prevent SSRF via auth hook URLs. Pins resolved IP to prevent DNS rebinding.
func ssrfSafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, _ := net.SplitHostPort(addr)
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	testMode := os.Getenv("GGID_ENV") == "test" || os.Getenv("GGID_DEV_MODE") == "true"
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			if testMode && ip.IsLoopback() {
				continue
			}
			return nil, fmt.Errorf("SSRF blocked: %s resolves to private IP %s", host, ip)
		}
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
}
