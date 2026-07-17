package posture

import "testing"

func TestEvaluate_FullyCompliant(t *testing.T) {
	engine := NewEngine(nil)
	engine.SetPolicy("t1", DefaultPosturePolicy("t1"))

	result := engine.Evaluate("t1", PostureInput{
		DeviceID:      "dev-1",
		OSVersion:     "14.2",
		DiskEncrypted: true,
		Jailbroken:    false,
		ScreenLock:    true,
		MDMEnrolled:   true,
		CertValid:     true,
	})

	if !result.Compliant {
		t.Fatalf("fully compliant device should pass, score=%d", result.Score)
	}
	if result.Action != "allow" {
		t.Fatalf("expected allow, got %s", result.Action)
	}
	if result.Score != 100 {
		t.Fatalf("expected score 100, got %d", result.Score)
	}
}

func TestEvaluate_Jailbroken_Block(t *testing.T) {
	engine := NewEngine(nil)
	engine.SetPolicy("t1", DefaultPosturePolicy("t1"))

	result := engine.Evaluate("t1", PostureInput{
		DeviceID:      "dev-jb",
		OSVersion:     "14.2",
		DiskEncrypted: true,
		Jailbroken:    true, // jailbroken!
		ScreenLock:    true,
		MDMEnrolled:   true,
		CertValid:     true,
	})

	if result.Compliant {
		t.Fatal("jailbroken device should not be compliant")
	}
	if result.Action != "block" {
		t.Fatalf("expected block for jailbroken, got %s", result.Action)
	}
}

func TestEvaluate_LowOSVersion(t *testing.T) {
	engine := NewEngine(nil)
	engine.SetPolicy("t1", DefaultPosturePolicy("t1"))

	result := engine.Evaluate("t1", PostureInput{
		DeviceID:      "dev-old",
		OSVersion:     "12.0", // below min 13.0
		DiskEncrypted: true,
		Jailbroken:    false,
		CertValid:     true,
	})

	// OS check is optional (weight 15) — device may still be compliant if score >= 70.
	if result.Score == 100 {
		t.Fatal("low OS version should reduce score")
	}
}

func TestEvaluate_DiskNotEncrypted(t *testing.T) {
	engine := NewEngine(nil)
	engine.SetPolicy("t1", DefaultPosturePolicy("t1"))

	result := engine.Evaluate("t1", PostureInput{
		DeviceID:      "dev-noenc",
		OSVersion:     "14.2",
		DiskEncrypted: false, // required check fails
		Jailbroken:    false,
		CertValid:     true,
	})

	if result.Compliant {
		t.Fatal("disk not encrypted should be non-compliant (required check)")
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("14.2", "13.0") <= 0 {
		t.Error("14.2 >= 13.0")
	}
	if compareVersions("12.0", "13.0") >= 0 {
		t.Error("12.0 < 13.0")
	}
	if compareVersions("14.2", "14.2") != 0 {
		t.Error("14.2 == 14.2")
	}
	if compareVersions("14.10", "14.9") <= 0 {
		t.Error("14.10 >= 14.9")
	}
}

func TestDefaultPosturePolicy(t *testing.T) {
	policy := DefaultPosturePolicy("t1")
	if len(policy.Checks) != 6 {
		t.Fatalf("expected 6 checks, got %d", len(policy.Checks))
	}
	if policy.MinScore != 70 {
		t.Fatalf("expected min_score 70, got %d", policy.MinScore)
	}
	// Verify required checks.
	requiredCount := 0
	for _, c := range policy.Checks {
		if c.Required {
			requiredCount++
		}
	}
	if requiredCount != 3 {
		t.Fatalf("expected 3 required checks (disk/jailbreak/cert), got %d", requiredCount)
	}
}

func TestEvaluate_UnknownTenant_UsesDefault(t *testing.T) {
	engine := NewEngine(nil)
	// No policy set for this tenant.
	result := engine.Evaluate("unknown-tenant", PostureInput{
		DeviceID:      "dev-1",
		OSVersion:     "14.2",
		DiskEncrypted: true,
		Jailbroken:    false,
		CertValid:     true,
	})

	if !result.Compliant {
		t.Fatal("should use default policy and be compliant")
	}
}

func TestEvaluate_StepUpAction(t *testing.T) {
	engine := NewEngine(nil)
	engine.SetPolicy("t1", PosturePolicy{
		TenantID: "t1",
		MinScore: 90,
		Action:   "step_up",
		Checks: []PostureCheck{
			{Name: "disk_encrypted", Required: false, Weight: 50},
			{Name: "screen_lock", Required: false, Weight: 50},
		},
	})

	// Only one check passes → score 50 < 90.
	result := engine.Evaluate("t1", PostureInput{
		DeviceID:      "dev-1",
		DiskEncrypted: true,
		ScreenLock:    false,
	})

	if result.Action != "step_up" {
		t.Fatalf("expected step_up, got %s", result.Action)
	}
}
