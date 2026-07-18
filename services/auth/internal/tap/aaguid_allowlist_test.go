package tap

import (
	"testing"
)

func TestAllowlist_AddAndCheck(t *testing.T) {
	al := NewAllowlist()
	al.Add(AAGUIDEntry{AAGUID: "test-uuid-1", Name: "YubiKey 5"})

	if !al.IsApproved("test-uuid-1") {
		t.Fatal("approved AAGUID should pass")
	}
	if al.IsApproved("unknown-uuid") {
		t.Fatal("unknown AAGUID should not pass")
	}
}

func TestAllowlist_CheckAttestation_Approved(t *testing.T) {
	al := NewAllowlist()
	al.Add(AAGUIDEntry{AAGUID: "abc-123", Name: "Titan"})

	err := al.CheckAttestation("abc-123")
	if err != nil {
		t.Fatalf("approved authenticator should pass: %v", err)
	}
}

func TestAllowlist_CheckAttestation_Rejected(t *testing.T) {
	al := NewAllowlist()

	err := al.CheckAttestation("unknown-uuid")
	if err == nil {
		t.Fatal("unknown AAGUID should be rejected")
	}
}

func TestAllowlist_CheckAttestation_EmptyAAGUID(t *testing.T) {
	al := NewAllowlist()
	err := al.CheckAttestation("")
	if err == nil {
		t.Fatal("empty AAGUID should be rejected")
	}
}

func TestAllowlist_DeniedStatus(t *testing.T) {
	al := NewAllowlist()
	al.Add(AAGUIDEntry{AAGUID: "denied-1", Name: "Bad Key", Status: AAGUIDDenied})

	if al.IsApproved("denied-1") {
		t.Fatal("denied AAGUID should not be approved")
	}
}

func TestAllowlist_Remove(t *testing.T) {
	al := NewAllowlist()
	al.Add(AAGUIDEntry{AAGUID: "rm-1", Name: "Temp"})
	al.Remove("rm-1")

	if al.IsApproved("rm-1") {
		t.Fatal("removed AAGUID should not be approved")
	}
}

func TestAllowlist_PopulateFromMDS(t *testing.T) {
	al := NewAllowlist()
	mdsData := map[string]struct {
		Name        string
		Description string
	}{
		"uuid-1": {"YubiKey 5", "Yubico"},
		"uuid-2": {"Titan", "Google"},
	}

	count := al.PopulateFromMDS(mdsData, "admin")
	if count != 2 {
		t.Fatalf("expected 2 MDS entries, got %d", count)
	}
	if !al.IsApproved("uuid-1") {
		t.Fatal("MDS-populated AAGUID should be approved")
	}
}

func TestDefaultKnownAuthenticators(t *testing.T) {
	mds := DefaultKnownAuthenticators()
	if len(mds) < 5 {
		t.Fatalf("expected at least 5 default authenticators, got %d", len(mds))
	}
	// Check YubiKey exists.
	found := false
	for aaguid := range mds {
		if aaguid == "cb69481e-8ff7-4039-93ec-0a2729a154a8" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected YubiKey 5 NFC in defaults")
	}
}

func TestHashAAGUID(t *testing.T) {
	h := HashAAGUID("test-uuid")
	if len(h) != 16 {
		t.Fatalf("expected 16-char hash, got %d", len(h))
	}
	if h == "test-uuid" {
		t.Fatal("hash should not equal input")
	}
}

func TestAllowlist_List(t *testing.T) {
	al := NewAllowlist()
	al.Add(AAGUIDEntry{AAGUID: "l1", Name: "Key1"})
	al.Add(AAGUIDEntry{AAGUID: "l2", Name: "Key2"})

	entries := al.List()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}
