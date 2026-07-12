package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDelegationValidator_ValidateDelegation_Valid(t *testing.T) {
	dv := NewDelegationValidator(3)
	result, err := dv.ValidateDelegation(uuid.New(), uuid.New(), []string{"read"}, 3)
	if err != nil {
		t.Fatalf("ValidateDelegation: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got: %s", result.Reason)
	}
}

func TestDelegationValidator_SelfDelegation(t *testing.T) {
	dv := NewDelegationValidator(3)
	id := uuid.New()
	result, _ := dv.ValidateDelegation(id, id, []string{"read"}, 3)
	if result.Valid {
		t.Error("self-delegation should be invalid")
	}
}

func TestDelegationValidator_NilIDs(t *testing.T) {
	dv := NewDelegationValidator(3)
	result, _ := dv.ValidateDelegation(uuid.Nil, uuid.New(), []string{"read"}, 3)
	if result.Valid {
		t.Error("nil delegator should be invalid")
	}
}

func TestDelegationValidator_EmptyScopes(t *testing.T) {
	dv := NewDelegationValidator(3)
	result, _ := dv.ValidateDelegation(uuid.New(), uuid.New(), nil, 3)
	if result.Valid {
		t.Error("empty scopes should be invalid")
	}
}

func TestDelegationValidator_CheckDepth_OK(t *testing.T) {
	dv := NewDelegationValidator(3)
	chain := []DelegationLink{{DelegatorID: uuid.New(), DelegateeID: uuid.New()}}
	result, _ := dv.CheckDelegationDepth(chain)
	if !result.Valid {
		t.Error("depth 1 should be valid")
	}
}

func TestDelegationValidator_CheckDepth_Exceeds(t *testing.T) {
	dv := NewDelegationValidator(2)
	chain := []DelegationLink{{}, {}, {}}
	result, _ := dv.CheckDelegationDepth(chain)
	if result.Valid {
		t.Error("depth 3 should exceed max 2")
	}
}

func TestDelegationValidator_ScopeNarrowing_OK(t *testing.T) {
	dv := NewDelegationValidator(3)
	ok, _ := dv.CheckScopeNarrowing([]string{"read", "write", "delete"}, []string{"read"})
	if !ok {
		t.Error("subset should be allowed")
	}
}

func TestDelegationValidator_ScopeNarrowing_Denied(t *testing.T) {
	dv := NewDelegationValidator(3)
	ok, _ := dv.CheckScopeNarrowing([]string{"read"}, []string{"delete"})
	if ok {
		t.Error("non-subset should be denied")
	}
}

func TestDelegationValidator_ScopeNarrowing_Wildcard(t *testing.T) {
	dv := NewDelegationValidator(3)
	ok, _ := dv.CheckScopeNarrowing([]string{"*"}, []string{"anything"})
	if !ok {
		t.Error("wildcard should allow any scope")
	}
}

func TestDelegationValidator_Expiry_Expired(t *testing.T) {
	dv := NewDelegationValidator(3)
	expired := time.Now().Add(-1 * time.Hour)
	chain := []DelegationLink{{ExpiresAt: &expired}}
	ok, _ := dv.CheckDelegationExpiry(chain)
	if ok {
		t.Error("expired chain should fail")
	}
}

func TestDelegationValidator_Expiry_Valid(t *testing.T) {
	dv := NewDelegationValidator(3)
	future := time.Now().Add(1 * time.Hour)
	chain := []DelegationLink{{ExpiresAt: &future}}
	ok, _ := dv.CheckDelegationExpiry(chain)
	if !ok {
		t.Error("non-expired chain should pass")
	}
}

func TestDelegationValidator_CircularDetection(t *testing.T) {
	dv := NewDelegationValidator(5)
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	chain := []DelegationLink{
		{DelegatorID: a, DelegateeID: b},
		{DelegatorID: b, DelegateeID: c},
		{DelegatorID: c, DelegateeID: a}, // circular: a -> b -> c -> a
	}
	isCircular, _ := dv.CheckCircularDelegation(chain)
	if !isCircular {
		t.Error("should detect circular delegation")
	}
}

func TestDelegationValidator_NoCircular(t *testing.T) {
	dv := NewDelegationValidator(5)
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	chain := []DelegationLink{
		{DelegatorID: a, DelegateeID: b},
		{DelegatorID: b, DelegateeID: c},
	}
	isCircular, _ := dv.CheckCircularDelegation(chain)
	if isCircular {
		t.Error("should not detect circular in linear chain")
	}
}

func TestDelegationValidator_ValidateChain_Valid(t *testing.T) {
	dv := NewDelegationValidator(5)
	a := uuid.New()
	b := uuid.New()
	future := time.Now().Add(1 * time.Hour)
	chain := []DelegationLink{
		{DelegatorID: a, DelegateeID: b, Scopes: []string{"read"}, ExpiresAt: &future},
	}
	result, _ := dv.ValidateChain(chain)
	if !result.Valid {
		t.Errorf("expected valid chain, got: %s", result.Reason)
	}
}

func TestDelegationValidator_ValidateChain_SelfDelegation(t *testing.T) {
	dv := NewDelegationValidator(5)
	id := uuid.New()
	chain := []DelegationLink{{DelegatorID: id, DelegateeID: id}}
	result, _ := dv.ValidateChain(chain)
	if result.Valid {
		t.Error("self-delegation chain should be invalid")
	}
}
