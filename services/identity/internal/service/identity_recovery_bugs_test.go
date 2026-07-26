package service

import (
	"testing"
	"time"
)

// BUG 1: VerifyRecoveryToken searches linearly through all requests
func TestVerifyRecoveryToken_Performance_Bug(t *testing.T) {
	t.Skip("BUG 1: VerifyRecoveryToken uses linear O(n) search — should use a map for token lookup")

	svc := NewIdentityRecoveryService()
	userID := "user123"

	for i := 0; i < 1000; i++ {
		_, _ = svc.InitiateRecovery(userID, RecoveryEmail)
	}

	start := time.Now()
	_, err := svc.VerifyRecoveryToken(userID, "nonexistent-token")
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Should have returned error for nonexistent token")
	}

	if elapsed > 10*time.Millisecond {
		t.Error("BUG: VerifyRecoveryToken is slow due to linear search - should use a map")
	}
}

// BUG 2: Multiple recovery requests for same user can be created
func TestInitiateRecovery_MultipleRequests_Bug(t *testing.T) {
	t.Skip("BUG 2: Multiple active recovery requests allowed for same user — security risk")

	svc := NewIdentityRecoveryService()
	userID := "user123"

	req1, _ := svc.InitiateRecovery(userID, RecoveryEmail)
	time.Sleep(10 * time.Millisecond)
	req2, _ := svc.InitiateRecovery(userID, RecoveryEmail)
	time.Sleep(10 * time.Millisecond)
	req3, _ := svc.InitiateRecovery(userID, RecoveryPhone)

	if req1.RequestID == req2.RequestID {
		t.Error("Different requests should have different IDs")
	}

	if req1.Status != RecoveryInitiated {
		t.Errorf("Expected status initiated, got %s", req1.Status)
	}
	if req2.Status != RecoveryInitiated {
		t.Errorf("Expected status initiated, got %s", req2.Status)
	}
	if req3.Status != RecoveryInitiated {
		t.Errorf("Expected status initiated, got %s", req3.Status)
	}

	t.Error("BUG: Multiple active recovery requests should not be allowed for security")
}

// BUG 3: CompleteRecovery doesn't verify the token again
func TestCompleteRecovery_NoTokenVerification_Bug(t *testing.T) {
	t.Skip("BUG 3: CompleteRecovery doesn't verify the token — anyone with request ID can complete")

	svc := NewIdentityRecoveryService()
	userID := "user123"

	req, _ := svc.InitiateRecovery(userID, RecoveryEmail)

	// Bypass the 24h wait period to test the actual bug (token not verified)
	svc.mu.Lock()
	req.WaitUntil = time.Now().Add(-1 * time.Second)
	svc.mu.Unlock()

	completed, err := svc.CompleteRecovery(req.RequestID, "new-password")
	if err != nil {
		t.Fatalf("CompleteRecovery failed: %v", err)
	}

	if completed.Status != RecoveryCompleted {
		t.Errorf("Expected status completed, got %s", completed.Status)
	}

	t.Error("BUG: CompleteRecovery should verify the token before allowing completion")
}

// BUG 4: Recovery token doesn't have rate limiting
func TestInitiateRecovery_NoRateLimit_Bug(t *testing.T) {
	t.Skip("BUG 4: No rate limiting on recovery initiation — spam attack risk")

	svc := NewIdentityRecoveryService()
	userID := "user123"

	requestCount := 100
	for i := 0; i < requestCount; i++ {
		_, err := svc.InitiateRecovery(userID, RecoveryEmail)
		if err != nil {
			t.Fatalf("InitiateRecovery failed: %v", err)
		}
	}

	t.Logf("BUG WARNING: Created %d recovery requests without rate limiting", requestCount)
}

// BUG 5: Time-delayed recovery can be bypassed
func TestCompleteRecovery_BypassTimeDelay_Bug(t *testing.T) {
	// This test passes — time delay is enforced correctly
	svc := NewIdentityRecoveryService()
	userID := "user123"

	req, _ := svc.InitiateRecovery(userID, RecoveryEmail)

	_, err := svc.CompleteRecovery(req.RequestID, "new-password")
	if err == nil {
		t.Error("BUG: Should enforce time-delayed recovery wait period")
	} else {
		t.Log("Time delay is enforced (good)")
	}
}

// BUG 5b: Time-delayed recovery doesn't require token after wait
func TestCompleteRecovery_NoTokenAfterWait_Bug(t *testing.T) {
	t.Skip("BUG 5b: Recovery completes without token verification after wait period")

	svc := NewIdentityRecoveryService()
	userID := "user123"

	req, _ := svc.InitiateRecovery(userID, RecoveryEmail)

	// Bypass the 24h wait period (avoid sleeping 24h in tests)
	svc.mu.Lock()
	req.WaitUntil = time.Now().Add(-1 * time.Second)
	svc.mu.Unlock()

	completed, err := svc.CompleteRecovery(req.RequestID, "new-password")
	if err != nil {
		t.Fatalf("CompleteRecovery failed after wait: %v", err)
	}

	if completed.Status == RecoveryCompleted {
		t.Error("BUG: Recovery completed without token verification, just request ID")
	}
}

// BUG 6: CancelRecovery doesn't prevent completion
func TestCancelRecovery_Prevention_Bug(t *testing.T) {
	t.Skip("BUG 6: Cancelled recovery can still be completed — needs status check in CompleteRecovery")

	svc := NewIdentityRecoveryService()
	userID := "user123"

	req, _ := svc.InitiateRecovery(userID, RecoveryEmail)

	err := svc.CancelRecovery(req.RequestID)
	if err != nil {
		t.Fatalf("CancelRecovery failed: %v", err)
	}

	// Bypass the 24h wait period
	svc.mu.Lock()
	req.WaitUntil = time.Now().Add(-1 * time.Second)
	svc.mu.Unlock()

	completed, err := svc.CompleteRecovery(req.RequestID, "new-password")
	if err == nil && completed.Status == RecoveryCompleted {
		t.Error("BUG: Cancelled recovery should not be completable")
	}
}

// BUG 7: Recovery method not validated
func TestInitiateRecovery_InvalidMethod_Bug(t *testing.T) {
	t.Skip("BUG 7: Recovery method not validated — any string accepted")

	svc := NewIdentityRecoveryService()
	userID := "user123"

	invalidMethod := RecoveryMethod("invalid_method")
	req, err := svc.InitiateRecovery(userID, invalidMethod)

	if err != nil {
		t.Fatalf("InitiateRecovery failed: %v", err)
	}

	if req.Method != invalidMethod {
		t.Errorf("Expected method %s, got %s", invalidMethod, req.Method)
	}

	t.Error("BUG: Recovery methods should be validated")
}

// BUG 8: VerifyRecoveryToken doesn't check if user is locked
func TestVerifyRecoveryToken_LockedUser_Bug(t *testing.T) {
	t.Skip("BUG 8: Recovery doesn't check if user is locked/suspended")

	svc := NewIdentityRecoveryService()
	userID := "locked_user_123"

	req, _ := svc.InitiateRecovery(userID, RecoveryEmail)

	verified, err := svc.VerifyRecoveryToken(userID, req.Token)
	if err != nil {
		t.Fatalf("VerifyRecoveryToken failed: %v", err)
	}

	if verified != nil {
		t.Error("BUG: Recovery should be blocked for locked users, but it's not checked")
	}
}

// BUG 9: Recovery token is predictable
func TestInitiateRecovery_TokenPredictability_Bug(t *testing.T) {
	t.Skip("BUG 9: Token format is predictable (rtok_{seq}_{timestamp}) — should use crypto random")

	svc := NewIdentityRecoveryService()

	tokens := make(map[string]bool)
	for i := 0; i < 10; i++ {
		req, _ := svc.InitiateRecovery("user", RecoveryEmail)
		if tokens[req.Token] {
			t.Error("BUG: Duplicate token generated")
		}
		tokens[req.Token] = true
		t.Logf("Token %d: %s", i+1, req.Token)
	}

	for token := range tokens {
		if len(token) < 32 {
			t.Logf("BUG WARNING: Token '%s' is short and predictable", token)
		}
	}
}

// BUG 10: GetRecoveryAuditTrail doesn't paginate
func TestGetRecoveryAuditTrail_Pagination_Bug(t *testing.T) {
	t.Skip("BUG 10: Audit trail returns all entries without pagination")

	svc := NewIdentityRecoveryService()

	for i := 0; i < 10000; i++ {
		_, _ = svc.InitiateRecovery("user", RecoveryEmail)
	}

	audit := svc.GetRecoveryAuditTrail()

	if len(audit) >= 10000 {
		t.Logf("BUG WARNING: Returning %d audit entries without pagination", len(audit))
		t.Error("BUG: Audit trail should be paginated")
	}
}

// BUG 11: CleanupExpired doesn't remove expired requests from memory
func TestCleanupExpired_MemoryLeak_Bug(t *testing.T) {
	t.Skip("BUG 11: CleanupExpired marks expired but doesn't remove from memory — memory leak")

	svc := NewIdentityRecoveryService()

	for i := 0; i < 100; i++ {
		req, _ := svc.InitiateRecovery("user", RecoveryEmail)
		req.ExpiresAt = time.Now().Add(-1 * time.Hour)
	}

	count := svc.CleanupExpired()
	t.Logf("Cleaned up %d expired requests", count)

	t.Error("BUG: CleanupExpired should remove expired requests from memory, not just mark them")
}

// BUG 12: Recovery request ID is predictable
func TestInitiateRecovery_PredictableIDs_Bug(t *testing.T) {
	t.Skip("BUG 12: Request IDs follow predictable pattern (rec_{seq}) — enumeration risk")

	svc := NewIdentityRecoveryService()

	ids := make([]string, 5)
	for i := 0; i < 5; i++ {
		req, _ := svc.InitiateRecovery("user", RecoveryEmail)
		ids[i] = req.RequestID
		t.Logf("Request ID %d: %s", i+1, req.RequestID)
	}

	for i, id := range ids {
		_ = "rec_" + string(rune(i+1))
		if len(id) < 10 {
			t.Logf("BUG WARNING: Request ID '%s' is short and predictable", id)
		}
	}
}