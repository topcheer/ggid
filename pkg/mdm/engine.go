package mdm

import (
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

// ComplianceStatus represents an MDM device's compliance state.
type ComplianceStatus string

const (
	Compliant    ComplianceStatus = "compliant"
	NonCompliant ComplianceStatus = "non_compliant"
	Unknown      ComplianceStatus = "unknown"
)

// Device represents an MDM-enrolled device.
type Device struct {
	ID              string           `json:"id"`
	DeviceID        string           `json:"device_id"`         // GGID device ID
	ConnectorID     string           `json:"connector_id"`
	OSVersion       string           `json:"os_version"`
	OS              string           `json:"os"`                 // iOS/macOS/Android/Windows
	ComplianceStatus ComplianceStatus `json:"compliance_status"`
	Managed         bool             `json:"managed"`
	Jailbroken      bool             `json:"jailbroken"`
	Encrypted       bool             `json:"encrypted"`
	PostureData     map[string]any   `json:"posture_data,omitempty"`
	LastSeen        time.Time        `json:"last_seen"`
}

// ConnectorConfig holds MDM connector configuration.
type ConnectorConfig struct {
	Name       string `json:"name"`
	Type       string `json:"type"`        // intune/jamf/android_management
	Endpoint   string `json:"endpoint"`    // API base URL
	AuthToken  string `json:"auth_token"`  // bearer token
	TenantID   string `json:"tenant_id"`   // Intune tenant or Jamf instance
	Enabled    bool   `json:"enabled"`
}

// MDMConnector is the interface for MDM platform adapters.
type MDMConnector interface {
	GetDevices(ctx context.Context) ([]Device, error)
	GetCompliance(ctx context.Context, deviceID string) (*Device, error)
	GetPosture(ctx context.Context, deviceID string) (map[string]any, error)
	ConnectorType() string
}

// Connector is a registered MDM connector with its config.
type Connector struct {
	ID        string          `json:"id"`
	Config    ConnectorConfig `json:"config"`
	Adapter   MDMConnector    `json:"-"`
	LastSync  *time.Time      `json:"last_sync,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// Engine manages MDM connectors and device sync.
type Engine struct {
	pool       *pgxpool.Pool
	httpClient *http.Client
	mu         sync.RWMutex
	connectors map[string]*Connector // name → connector
}

// NewEngine creates an MDM engine.
func NewEngine(pool *pgxpool.Pool) *Engine {
	return &Engine{
		pool:       pool,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		connectors: make(map[string]*Connector),
	}
}

// EnsureSchema creates mdm_connectors + mdm_devices tables.
func (e *Engine) EnsureSchema(ctx context.Context) error {
	if e.pool == nil {
		return nil
	}
	_, err := e.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS mdm_connectors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL,
			config JSONB NOT NULL DEFAULT '{}',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			last_sync TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS mdm_devices (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			connector_id TEXT NOT NULL,
			os_version TEXT,
			os TEXT,
			compliance_status TEXT NOT NULL DEFAULT 'unknown',
			managed BOOLEAN DEFAULT FALSE,
			jailbroken BOOLEAN DEFAULT FALSE,
			encrypted BOOLEAN DEFAULT FALSE,
			posture_data JSONB DEFAULT '{}',
			last_seen TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_mdm_devices_device ON mdm_devices(device_id);
		CREATE INDEX IF NOT EXISTS idx_mdm_devices_connector ON mdm_devices(connector_id);
	`)
	return err
}

// AddConnector registers a new MDM connector.
func (e *Engine) AddConnector(cfg ConnectorConfig) *Connector {
	e.mu.Lock()
	defer e.mu.Unlock()

	conn := &Connector{
		ID:        uuid.New().String(),
		Config:    cfg,
		CreatedAt: time.Now(),
	}

	// Create the appropriate adapter.
	conn.Adapter = e.createAdapter(cfg)

	e.connectors[cfg.Name] = conn
	return conn
}

// ListConnectors returns all registered connectors.
func (e *Engine) ListConnectors() []*Connector {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var result []*Connector
	for _, c := range e.connectors {
		result = append(result, c)
	}
	return result
}

// GetConnector returns a connector by name.
func (e *Engine) GetConnector(name string) *Connector {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.connectors[name]
}

// SyncDevices fetches all devices from a connector and persists them.
func (e *Engine) SyncDevices(ctx context.Context, connectorName string) ([]Device, error) {
	e.mu.RLock()
	conn, exists := e.connectors[connectorName]
	e.mu.RUnlock()

	if !exists || conn.Adapter == nil {
		return nil, fmt.Errorf("connector %s not found", connectorName)
	}

	devices, err := conn.Adapter.GetDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("sync %s: %w", connectorName, err)
	}

	// Persist devices.
	for i := range devices {
		devices[i].ConnectorID = conn.ID
		e.persistDevice(ctx, &devices[i])

		// Check for compliance change → trigger webhook.
		if devices[i].ComplianceStatus == NonCompliant {
			slog.Warn("MDM device non-compliant", "device", devices[i].DeviceID, "connector", connectorName)
		}
	}

	// Update last sync time.
	now := time.Now()
	conn.LastSync = &now

	return devices, nil
}

// GetAllDevices returns all MDM-enrolled devices from PG.
func (e *Engine) GetAllDevices(ctx context.Context) ([]Device, error) {
	if e.pool == nil {
		return nil, nil
	}
	rows, err := e.pool.Query(ctx,
		`SELECT id, device_id, connector_id, os_version, os, compliance_status, managed, jailbroken, encrypted, posture_data, last_seen
		FROM mdm_devices ORDER BY last_seen DESC LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDevices(rows)
}

// GetDeviceCompliance returns compliance for a specific device.
func (e *Engine) GetDeviceCompliance(ctx context.Context, deviceID string) (*Device, error) {
	if e.pool == nil {
		return nil, nil
	}
	var d Device
	var postureJSON []byte
	err := e.pool.QueryRow(ctx,
		`SELECT id, device_id, connector_id, os_version, os, compliance_status, managed, jailbroken, encrypted, posture_data, last_seen
		FROM mdm_devices WHERE device_id = $1 ORDER BY last_seen DESC LIMIT 1`, deviceID).
		Scan(&d.ID, &d.DeviceID, &d.ConnectorID, &d.OSVersion, &d.OS, &d.ComplianceStatus, &d.Managed, &d.Jailbroken, &d.Encrypted, &postureJSON, &d.LastSeen)
	if err != nil {
		return nil, nil
	}
	if postureJSON != nil {
		_ = json.Unmarshal(postureJSON, &d.PostureData)
	}
	return &d, nil
}

// createAdapter returns the appropriate MDM adapter for the config type.
func (e *Engine) createAdapter(cfg ConnectorConfig) MDMConnector {
	switch cfg.Type {
	case "intune":
		return &IntuneAdapter{Config: cfg, client: e.httpClient}
	case "jamf":
		return &JamfAdapter{Config: cfg, client: e.httpClient}
	case "android_management":
		return &AndroidAdapter{Config: cfg, client: e.httpClient}
	default:
		return nil
	}
}

func (e *Engine) persistDevice(ctx context.Context, d *Device) {
	if e.pool == nil {
		return
	}
	postureJSON, _ := json.Marshal(d.PostureData)
	_, err := e.pool.Exec(ctx,
		`INSERT INTO mdm_devices (id, device_id, connector_id, os_version, os, compliance_status, managed, jailbroken, encrypted, posture_data, last_seen)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (device_id, connector_id) DO UPDATE SET compliance_status=EXCLUDED.compliance_status, managed=EXCLUDED.managed, posture_data=EXCLUDED.posture_data, last_seen=EXCLUDED.last_seen`,
		uuid.New().String(), d.DeviceID, d.ConnectorID, d.OSVersion, d.OS, d.ComplianceStatus, d.Managed, d.Jailbroken, d.Encrypted, postureJSON, d.LastSeen)
	if err != nil {
		slog.Warn("mdm device persist failed", "error", err)
	}
}

// scanDevices converts pgx rows to Device slice.
func scanDevices(rows interface{ Next() bool; Scan(...any) error }) ([]Device, error) {
	var devices []Device
	for rows.Next() {
		var d Device
		var postureJSON []byte
		if err := rows.Scan(&d.ID, &d.DeviceID, &d.ConnectorID, &d.OSVersion, &d.OS, &d.ComplianceStatus, &d.Managed, &d.Jailbroken, &d.Encrypted, &postureJSON, &d.LastSeen); err != nil {
			continue
		}
		if postureJSON != nil {
			_ = json.Unmarshal(postureJSON, &d.PostureData)
		}
		devices = append(devices, d)
	}
	return devices, nil
}
