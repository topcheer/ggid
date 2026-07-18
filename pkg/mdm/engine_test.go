package mdm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEngine_AddAndListConnectors(t *testing.T) {
	engine := NewEngine(nil)
	engine.AddConnector(ConnectorConfig{
		Name: "intune-prod", Type: "intune", Enabled: true,
	})
	engine.AddConnector(ConnectorConfig{
		Name: "jamf-prod", Type: "jamf", Enabled: true,
	})

	connectors := engine.ListConnectors()
	if len(connectors) != 2 {
		t.Fatalf("expected 2 connectors, got %d", len(connectors))
	}
	if engine.GetConnector("intune-prod") == nil {
		t.Fatal("intune-prod should exist")
	}
}

func TestEngine_CreateAdapter(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		connType   string
		expectType string
	}{
		{"intune", "intune"},
		{"jamf", "jamf"},
		{"android_management", "android_management"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		conn := engine.AddConnector(ConnectorConfig{Name: "test-" + tt.connType, Type: tt.connType})
		if tt.expectType == "" {
			if conn.Adapter != nil {
				t.Fatalf("expected nil adapter for type %s", tt.connType)
			}
		} else {
			if conn.Adapter == nil {
				t.Fatalf("expected non-nil adapter for type %s", tt.connType)
			}
			if conn.Adapter.ConnectorType() != tt.expectType {
				t.Fatalf("expected type %s, got %s", tt.expectType, conn.Adapter.ConnectorType())
			}
		}
	}
}

func TestIntuneAdapter_GetDevices_MockServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"value": [
				{"id":"dev-1","deviceName":"iPhone","osVersion":"17.0","operatingSystem":"iOS","complianceState":"compliant","jailBroken":"false","isEncrypted":true},
				{"id":"dev-2","deviceName":"Pixel","osVersion":"14.0","operatingSystem":"Android","complianceState":"noncompliant","jailBroken":"false","isEncrypted":false}
			]
		}`))
	}))
	defer ts.Close()

	adapter := &IntuneAdapter{
		Config: ConnectorConfig{Endpoint: ts.URL, AuthToken: "test-token"},
		client: &http.Client{},
	}

	devices, err := adapter.GetDevices(context.Background())
	if err != nil {
		t.Fatalf("GetDevices failed: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].ComplianceStatus != Compliant {
		t.Fatalf("expected compliant, got %s", devices[0].ComplianceStatus)
	}
	if devices[1].ComplianceStatus != NonCompliant {
		t.Fatalf("expected non_compliant, got %s", devices[1].ComplianceStatus)
	}
	if !devices[0].Encrypted {
		t.Fatal("device 0 should be encrypted")
	}
}

func TestEngine_SyncDevices_UnknownConnector(t *testing.T) {
	engine := NewEngine(nil)
	_, err := engine.SyncDevices(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown connector")
	}
}

func TestEngine_EnsureSchema_NilPool(t *testing.T) {
	engine := NewEngine(nil)
	if err := engine.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
}

func TestEngine_GetAllDevices_NilPool(t *testing.T) {
	engine := NewEngine(nil)
	devices, err := engine.GetAllDevices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if devices != nil {
		t.Fatal("nil pool should return nil devices")
	}
}

func TestJamfAdapter_GetDevices_MockServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"results": [
				{"id":101,"udid":"ABC","osVersion":"14.2","platform":"macOS","managed":true,"fileVault2Enabled":true},
				{"id":102,"udid":"DEF","osVersion":"13.0","platform":"macOS","managed":true,"fileVault2Enabled":false}
			]
		}`))
	}))
	defer ts.Close()

	adapter := &JamfAdapter{
		Config: ConnectorConfig{Endpoint: ts.URL, AuthToken: "test"},
		client: &http.Client{},
	}

	devices, err := adapter.GetDevices(context.Background())
	if err != nil {
		t.Fatalf("GetDevices failed: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].DeviceID != "jamf-101" {
		t.Fatalf("expected jamf-101, got %s", devices[0].DeviceID)
	}
	if !devices[1].Encrypted {
		t.Fatal("device 1 should have fileVault disabled")
	}
}
