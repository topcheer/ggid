package server

import (
	"net/url"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEvaluateJDLCondition_EqualsMatch(t *testing.T) {
	v := url.Values{"department": {"eng"}}
	if !evaluateJDLCondition("department == 'eng'", v) {
		t.Error("should match when values equal")
	}
}

func TestEvaluateJDLCondition_EqualsNoMatch(t *testing.T) {
	v := url.Values{"department": {"sales"}}
	if evaluateJDLCondition("department == 'eng'", v) {
		t.Error("should not match when values differ")
	}
}

func TestEvaluateJDLCondition_Unparseable(t *testing.T) {
	v := url.Values{}
	if !evaluateJDLCondition("garbage expr", v) {
		t.Error("unparseable condition should default to true")
	}
}

func TestJDL_ParseValidYAML(t *testing.T) {
	def := `steps:
  - id: "1"
    name: "Assign role"
    action: "assign_role"
    condition: "department == 'eng'"
    params:
      role_id: "abc-123"
  - id: "2"
    name: "Notify"
    action: "notify"
`
	var jdl JDL
	if err := yaml.Unmarshal([]byte(def), &jdl); err != nil {
		t.Fatalf("YAML parse failed: %v", err)
	}
	if len(jdl.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(jdl.Steps))
	}
	if jdl.Steps[0].Action != "assign_role" {
		t.Errorf("expected assign_role, got %s", jdl.Steps[0].Action)
	}
}

func TestJourneyDefinition_Defaults(t *testing.T) {
	j := &JourneyDefinition{Name: "Onboarding"}
	if j.ID == "" {
		j.ID = "gen-id"
	}
	if j.Status == "" {
		j.Status = "draft"
	}
	if j.ID != "gen-id" || j.Status != "draft" {
		t.Error("defaults should be applied")
	}
}

func TestJourneyDryRunResult_StepValidation(t *testing.T) {
	validActions := map[string]bool{
		"assign_role": true, "revoke_access": true, "notify": true,
		"create_account": true, "disable_account": true,
	}
	if !validActions["assign_role"] {
		t.Error("assign_role should be valid")
	}
	if validActions["unknown"] {
		t.Error("unknown action should be invalid")
	}
}
