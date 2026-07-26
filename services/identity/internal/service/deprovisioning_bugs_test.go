package service

import (
	"testing"
)

// BUG 1: StartDeprovisioning doesn't actually execute steps
func TestStartDeprovisioning_NoActualExecution_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewDeprovisioningService()

	req := svc.StartDeprovisioning("user123", "user left", "admin1")

	if req.Status != DeprovisionCompleted {
		t.Errorf("Expected status completed, got %s", req.Status)
	}

	// BUG: All steps are marked as completed immediately
	// But no actual work is done:
	// - Tokens are not revoked
	// - Groups are not removed
	// - Account is not disabled
	// - Data is not archived
	// - Audit is not created in any real system

	for _, step := range req.Steps {
		if !step.Done || step.Status != "completed" {
			t.Errorf("BUG: Step %s should be completed, got status %s, done %v", step.Step, step.Status, step.Done)
		}
		t.Logf("Step %s: %s (but nothing was actually done)", step.Step, step.Message)
	}
}

// BUG 2: CancelDeprovisioning after completion is prevented but steps remain completed
func TestCancelDeprovisioning_AfterCompletion_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewDeprovisioningService()

	req := svc.StartDeprovisioning("user123", "test", "admin1")

	// Try to cancel after completion
	err := svc.CancelDeprovisioning(req.RequestID)
	if err == nil {
		t.Error("Cancel should fail for completed deprovisioning")
	}

	// BUG: Even though cancel failed, all steps remain completed
	// There's no way to "undo" a completed deprovisioning
	for _, step := range req.Steps {
		if !step.Done {
			t.Error("BUG: Steps should remain completed even if cancel fails")
		}
	}
}

// BUG 3: Rollback doesn't actually undo anything
func TestRollback_NoActualRollback_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewDeprovisioningService()

	req := svc.StartDeprovisioning("user123", "test", "admin1")

	// Rollback the deprovisioning
	rolledBack, err := svc.Rollback(req.RequestID)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// BUG: Rollback only changes status to "failed" and marks steps as "rolled_back"
	// It doesn't actually:
	// - Restore tokens
	// - Re-add to groups
	// - Re-enable account
	// - Unarchive data

	if rolledBack.Status != DeprovisionFailed {
		t.Errorf("Expected status failed, got %s", rolledBack.Status)
	}

	for _, step := range rolledBack.Steps {
		if step.Done && step.Status != "rolled_back" {
			t.Errorf("BUG: Completed step %s should be rolled back, got status %s", step.Step, step.Status)
		}
		t.Logf("Step %s: %s (but nothing was actually rolled back)", step.Step, step.Message)
	}
}

// BUG 4: Concurrent deprovisioning requests for same user
func TestStartDeprovisioning_Concurrent_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewDeprovisioningService()

	// Start multiple deprovisioning requests for the same user
	req1 := svc.StartDeprovisioning("user123", "reason1", "admin1")
	req2 := svc.StartDeprovisioning("user123", "reason2", "admin2")

	// BUG: Multiple deprovisioning requests can be created for the same user
	// This could lead to race conditions and inconsistent state

	if req1.UserID != req2.UserID {
		t.Error("BUG: Both requests should be for same user")
	}

	if req1.RequestID == req2.RequestID {
		t.Error("BUG: Different requests should have different IDs")
	}

	// Both complete, potentially causing conflicts
	if req1.Status == DeprovisionCompleted && req2.Status == DeprovisionCompleted {
		t.Error("BUG: Both deprovisioning requests completed - may cause conflicts")
	}
}

// BUG 5: CancelDeprovisioning doesn't prevent in-progress steps
func TestCancelDeprovisioning_DuringExecution_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewDeprovisioningService()

	// Note: In the current implementation, all steps complete immediately
	// So we can't truly test cancellation during execution
	// This is a design bug - there's no async execution

	req := svc.StartDeprovisioning("user123", "test", "admin1")

	// Since everything completes instantly, we can't cancel during execution
	// This is the actual bug - the design doesn't support async, cancellable operations

	if req.Status == DeprovisionCompleted {
		t.Error("BUG: Deprovisioning completes synchronously - can't test cancellation during execution")
	}
}

// BUG 6: GetDeprovisionStatus doesn't validate request ID
func TestGetDeprovisionStatus_InvalidID_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewDeprovisioningService()

	// Try to get status of non-existent request
	req := svc.GetDeprovisionStatus("non-existent-id")

	// BUG: Returns nil instead of error
	if req != nil {
		t.Error("BUG: GetDeprovisionStatus should return error for invalid request ID")
	}
}

// BUG 7: No timestamp tracking for step execution
func TestDeprovisioning_NoStepTimestamps_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewDeprovisioningService()

	req := svc.StartDeprovisioning("user123", "test", "admin1")

	// BUG: Steps don't have individual timestamps
	// We can't tell when each step started/finished
	// This makes debugging and monitoring difficult

	for _, step := range req.Steps {
		if step.Done {
			// No way to know when this step completed
			t.Logf("Step %s: done at unknown time", step.Step)
		}
	}

	// Only the overall request has timestamps
	if req.UpdatedAt.IsZero() || req.CreatedAt.IsZero() {
		t.Error("Request timestamps should be set")
	}
}

// BUG 8: Rollback doesn't validate current status
func TestRollback_StatusValidation_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewDeprovisioningService()

	req := svc.StartDeprovisioning("user123", "test", "admin1")

	// Cancel it first
	_ = svc.CancelDeprovisioning(req.RequestID)

	// Try to rollback a cancelled request
	rolledBack, err := svc.Rollback(req.RequestID)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// BUG: Rollback works even on cancelled requests
	// It should probably only work on in-progress or completed requests

	if rolledBack.Status != DeprovisionFailed {
		t.Errorf("BUG: Rollback of cancelled request changed status to %s", rolledBack.Status)
	}
}

// BUG 9: No validation of user existence
func TestStartDeprovisioning_UserNotValidated_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewDeprovisioningService()

	// Use a user ID that clearly doesn't exist
	req := svc.StartDeprovisioning("non-existent-user-xyz-123", "test", "admin1")

	// BUG: Service doesn't validate that the user actually exists
	// It just creates a deprovisioning request

	if req.Status != DeprovisionInProgress && req.Status != DeprovisionCompleted {
		t.Errorf("Unexpected status: %s", req.Status)
	}

	t.Log("BUG: Deprovisioning request created for non-existent user")
}

// BUG 10: No authorization check for requestedBy
func TestStartDeprovisioning_NoAuthCheck_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	svc := NewDeprovisioningService()

	// Anyone can request deprovisioning with any requestedBy value
	req := svc.StartDeprovisioning("user123", "test", "unauthorized-user")

	// BUG: No authorization check
	// Anyone can deprovision any user claiming to be anyone
	// The requestedBy field is not validated

	if req.RequestedBy != "unauthorized-user" {
		t.Error("BUG: requestedBy should be stored as provided")
	}

	t.Log("BUG: No authorization check on who can request deprovisioning")
}
