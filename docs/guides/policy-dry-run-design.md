# Policy Dry-Run Design

This guide covers dry-run mode, simulation API, decision tracing, impact analysis, before/after comparison, and GGID's policy dry-run implementation.

## Overview

Policy dry-run mode allows administrators to test policy changes before deploying them to production. It evaluates requests against both the current policy and the proposed policy, showing what would change without actually enforcing the new rules.

## Dry-Run Mode

### How It Works

```
1. Admin defines a proposed policy change
2. Dry-run evaluates incoming requests against BOTH:
   a. Current policy → actual decision (enforced)
   b. Proposed policy → simulated decision (logged only)
3. Results are compared and stored for analysis
4. Admin reviews impact before deploying
```

### Modes

| Mode | Description | Enforcement |
|---|---|---|
| observe | Log both decisions, enforce current | Current policy enforced |
| shadow | Log both decisions, enforce current | Same as observe + detailed trace |
| compare | Log differences only, enforce current | Only log when decisions differ |
| test | Evaluate against test data, enforce nothing | No enforcement at all |

## Simulation API

### Evaluate a Single Request

```bash
POST /api/v1/policy/dry-run
Authorization: Bearer <admin_token>
Content-Type: application/json

{
  "proposed_policy": {
    "version": "2",
    "rules": [
      {
        "effect": "deny",
        "actions": ["delete"],
        "resources": ["users:*"],
        "conditions": {
          "time_after": "18:00",
          "time_before": "09:00"
        }
      }
    ]
  },
  "test_cases": [
    {
      "subject": "user-uuid",
      "action": "delete",
      "resource": "users:123",
      "context": {
        "time": "2026-07-12T20:00:00Z",
        "ip": "192.168.1.50"
      }
    }
  ]
}
```

### Response

```json
{
  "results": [
    {
      "test_case": 0,
      "current_decision": "allow",
      "proposed_decision": "deny",
      "changed": true,
      "current_rule": "default-allow",
      "proposed_rule": "deny-delete-after-hours",
      "trace": {
        "rules_evaluated": 3,
        "matched_rule": "deny-delete-after-hours",
        "condition_results": {
          "time_after_18:00": true,
          "time_before_09:00": false,
          "combined": true
        }
      }
    }
  ],
  "summary": {
    "total_cases": 1,
    "changed": 1,
    "would_allow": 0,
    "would_deny": 1,
    "no_change": 0
  }
}
```

## Decision Trace

### Trace Structure

Each policy evaluation produces a trace showing exactly which rules were evaluated and why:

```go
type DecisionTrace struct {
    Request      PolicyRequest  `json:"request"`
    RulesEvaluated []RuleTrace  `json:"rules_evaluated"`
    FinalDecision string        `json:"final_decision"`
    MatchedRule   string        `json:"matched_rule"`
    Duration      time.Duration `json:"duration"`
}

type RuleTrace struct {
    RuleID       string                 `json:"rule_id"`
    RuleName     string                 `json:"rule_name"`
    Effect       string                 `json:"effect"`  // allow, deny
    Matched      bool                   `json:"matched"`
    Conditions   map[string]bool        `json:"conditions"`
    Skipped      string                 `json:"skipped_reason,omitempty"`
}
```

### Example Trace

```json
{
  "request": {
    "subject": "user-123",
    "action": "delete",
    "resource": "users:456"
  },
  "rules_evaluated": [
    {
      "rule_id": "rule-1",
      "rule_name": "deny-delete-after-hours",
      "effect": "deny",
      "matched": true,
      "conditions": {
        "time_after_18:00": true,
        "time_before_09:00": false,
        "action_is_delete": true,
        "resource_is_users": true
      }
    },
    {
      "rule_id": "rule-2",
      "rule_name": "allow-admin-delete",
      "effect": "allow",
      "matched": false,
      "skipped": "subject not admin"
    },
    {
      "rule_id": "default",
      "rule_name": "default-deny",
      "effect": "deny",
      "matched": false,
      "skipped": "preceding rule matched"
    }
  ],
  "final_decision": "deny",
  "matched_rule": "deny-delete-after-hours",
  "duration": "0.3ms"
}
```

## Impact Analysis

### Summary Statistics

```bash
GET /api/v1/policy/dry-run/impact?policy_id=proposed-v2&period=24h
Authorization: Bearer <admin_token>

Response:
{
  "period": "24h",
  "total_requests_evaluated": 15420,
  "impact": {
    "would_change": 342,
    "would_allow_more": 50,
    "would_deny_more": 292,
    "no_change": 15078
  },
  "affected_users": 45,
  "affected_resources": ["users:123", "users:456"],
  "by_action": {
    "delete": { "changed": 292, "deny_to_allow": 0, "allow_to_deny": 292 },
    "read": { "changed": 0, "deny_to_allow": 0, "allow_to_deny": 0 },
    "write": { "changed": 50, "deny_to_allow": 50, "allow_to_deny": 0 }
  },
  "risk_assessment": {
    "level": "medium",
    "concerns": [
      "292 delete operations would be denied that are currently allowed",
      "50 write operations would be newly allowed"
    ]
  }
}
```

### Affected Users

```bash
GET /api/v1/policy/dry-run/affected-users?policy_id=proposed-v2
Authorization: Bearer <admin_token>

Response:
{
  "users": [
    {
      "user_id": "user-123",
      "user_name": "alice",
      "affected_requests": 15,
      "changes": [
        { "action": "delete", "resource": "users:456", "from": "allow", "to": "deny" }
      ]
    }
  ]
}
```

## Before/After Comparison

### Policy Diff

```bash
GET /api/v1/policy/dry-run/diff?current=v1&proposed=v2
Authorization: Bearer <admin_token>

Response:
{
  "added_rules": [
    {
      "id": "deny-delete-after-hours",
      "effect": "deny",
      "actions": ["delete"],
      "resources": ["users:*"],
      "conditions": { "time_after": "18:00" }
    }
  ],
  "modified_rules": [
    {
      "id": "allow-admin-all",
      "changes": {
        "resources": { "from": ["*"], "to": ["users:*", "roles:*"] }
      }
    }
  ],
  "removed_rules": [
    {
      "id": "allow-all-internal"
    }
  ]
}
```

### Decision Matrix

| Scenario | Current | Proposed | Change |
|---|---|---|---|
| Admin delete user (business hours) | Allow | Allow | No change |
| Admin delete user (after hours) | Allow | Deny | Breaking |
| User read own profile | Allow | Allow | No change |
| User delete another user | Deny | Deny | No change |
| Developer write config | Deny | Allow | New access |

## Audit Dry-Run

All dry-run evaluations are logged for audit purposes:

```go
func auditDryRun(eval *DryRunEvaluation) {
    audit.Log(AuditEvent{
        Type:       "policy_dry_run",
        AdminID:    eval.AdminID,
        PolicyID:   eval.ProposedPolicyID,
        TestCase:   eval.TestCase,
        CurrentDecision: eval.CurrentDecision,
        ProposedDecision: eval.ProposedDecision,
        Changed:    eval.Changed,
        Timestamp:  time.Now(),
    })
}
```

### Audit Query

```bash
GET /api/v1/audit/events?type=policy_dry_run&policy_id=proposed-v2
Authorization: Bearer <admin_token>
```

## GGID Implementation

### Dry-Run Service

```go
type DryRunService struct {
    policyEngine *PolicyEngine
    auditStore   AuditStore
    config       DryRunConfig
}

func (s *DryRunService) Evaluate(
    ctx context.Context,
    proposedPolicy *Policy,
    testCases []PolicyRequest,
) (*DryRunResult, error) {

    results := make([]TestCaseResult, len(testCases))
    changed := 0
    wouldAllow := 0
    wouldDeny := 0

    for i, tc := range testCases {
        // Evaluate against current policy
        currentDecision, currentTrace := s.policyEngine.Evaluate(tc)

        // Evaluate against proposed policy
        proposedDecision, proposedTrace := s.policyEngine.EvaluateWith(tc, proposedPolicy)

        isChanged := currentDecision != proposedDecision

        results[i] = TestCaseResult{
            TestCase:          i,
            CurrentDecision:   currentDecision,
            ProposedDecision:  proposedDecision,
            Changed:           isChanged,
            CurrentRule:       currentTrace.MatchedRule,
            ProposedRule:      proposedTrace.MatchedRule,
            Trace:             proposedTrace,
        }

        if isChanged {
            changed++
            if proposedDecision == "allow" {
                wouldAllow++
            } else {
                wouldDeny++
            }
        }

        // Audit
        s.auditDryRun(tc, currentDecision, proposedDecision, isChanged)
    }

    return &DryRunResult{
        Results: results,
        Summary: DryRunSummary{
            Total:     len(testCases),
            Changed:   changed,
            WouldAllow: wouldAllow,
            WouldDeny: wouldDeny,
            NoChange:  len(testCases) - changed,
        },
    }, nil
}
```

### Configuration

```yaml
policy:
  dry_run:
    enabled: true
    mode: "observe"  # observe, shadow, compare, test
    max_test_cases: 1000
    retention: 30d  # How long to keep dry-run results
    auto_cleanup: true
    require_admin: true  # Only admins can run dry-runs
    audit_all: true
```

### API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/policy/dry-run` | POST | Run dry-run with test cases |
| `/api/v1/policy/dry-run/impact` | GET | Get impact analysis for a proposed policy |
| `/api/v1/policy/dry-run/diff` | GET | Get diff between current and proposed |
| `/api/v1/policy/dry-run/affected-users` | GET | List users affected by proposed change |
| `/api/v1/policy/dry-run/results` | GET | Get stored dry-run results |
| `/api/v1/policy/dry-run/{id}` | DELETE | Delete a dry-run result |

## Best Practices

1. **Always dry-run before deploying** — Never push policy changes without testing
2. **Use real test data** — Synthetic data may miss edge cases
3. **Review breaking changes** — Pay special attention to allow→deny changes
4. **Check affected users** — Identify who will be impacted
5. **Run for 24h minimum** — Capture all time-based rule variations
6. **Involve stakeholders** — Share results with affected teams
7. **Audit all dry-runs** — Track who tested what and when
8. **Version policies** — Keep history of all policy versions
9. **Test edge cases** — Boundary conditions, empty inputs, wildcards
10. **Document changes** — Record rationale for each policy change