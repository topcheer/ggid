package server

import (
	"testing"

	"github.com/google/uuid"
)

func TestEvaluatePosture_AllCompliant(t *testing.T) {
	dp := &DevicePosture{
		Checks: map[string]any{
			"disk_encrypted":     true,
			"screen_lock_enabled": true,
			"antivirus_active":    true,
			"firewall_enabled":    true,
			"os_up_to_date":       true,
			"jailbroken":          false,
			"managed":             true,
		},
	}
	evaluatePosture(dp)
	if dp.ComplianceScore != 100 {
		t.Errorf("expected score 100, got %d", dp.ComplianceScore)
	}
	if !dp.Compliant {
		t.Error("should be compliant")
	}
	if dp.TrustLevel != "trusted" {
		t.Errorf("expected trusted, got %s", dp.TrustLevel)
	}
}

func TestEvaluatePosture_Jailbroken(t *testing.T) {
	dp := &DevicePosture{
		Checks: map[string]any{
			"disk_encrypted":  true,
			"antivirus_active": true,
			"os_up_to_date":   true,
			"jailbroken":      true, // critical fail
			"managed":         true,
		},
	}
	evaluatePosture(dp)
	if dp.Compliant {
		t.Error("jailbroken device should not be compliant")
	}
	if dp.TrustLevel != "untrusted" {
		t.Errorf("expected untrusted, got %s", dp.TrustLevel)
	}
}

func TestEvaluatePosture_NoEncryption(t *testing.T) {
	dp := &DevicePosture{
		Checks: map[string]any{
			"disk_encrypted":  false, // critical fail
			"antivirus_active": true,
			"os_up_to_date":   true,
			"jailbroken":      false,
		},
	}
	evaluatePosture(dp)
	if dp.Compliant {
		t.Error("unencrypted disk should not be compliant")
	}
}

func TestEvaluatePosture_EmptyChecks(t *testing.T) {
	dp := &DevicePosture{Checks: map[string]any{}}
	evaluatePosture(dp)
	if dp.ComplianceScore != 0 {
		t.Errorf("empty checks should have score 0, got %d", dp.ComplianceScore)
	}
	if dp.Compliant {
		t.Error("empty checks should not be compliant")
	}
}

func TestEvaluatePosture_PartialCompliance(t *testing.T) {
	dp := &DevicePosture{
		Checks: map[string]any{
			"disk_encrypted":     true,
			"screen_lock_enabled": true,
			"jailbroken":          false,
			// missing: antivirus, os_update, firewall, managed
		},
	}
	evaluatePosture(dp)
	if dp.ComplianceScore >= 85 {
		t.Errorf("partial compliance should be < 85, got %d", dp.ComplianceScore)
	}
	if dp.ComplianceScore < 30 {
		t.Errorf("3 checks should give >= 30, got %d", dp.ComplianceScore)
	}
}

func TestDevicePostureRepo_NilPool(t *testing.T) {
	repo := newDevicePostureRepo(nil, nil)
	dp, err := repo.GetByDevice(nil, uuid.New(), "dev-1")
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if dp != nil {
		t.Error("nil pool should return nil posture")
	}
}
