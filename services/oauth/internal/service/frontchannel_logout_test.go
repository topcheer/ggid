package service

import (
	"testing"
)

func TestFrontChannel_RegisterAndGet(t *testing.T) {
	ResetFrontChannelSessions()
	RegisterFrontChannelSession("sess1", "client-a", "user1", "https://app-a.com/logout")
	s, err := GetFrontChannelSession("sess1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if s.ClientID != "client-a" || s.UserID != "user1" {
		t.Error("session data mismatch")
	}
	if s.FrontChannelURI != "https://app-a.com/logout" {
		t.Error("front channel URI mismatch")
	}
}

func TestFrontChannel_LogoutReturnsURIs(t *testing.T) {
	ResetFrontChannelSessions()
	RegisterFrontChannelSession("sess1", "client-a", "user1", "https://a.com/logout")
	RegisterFrontChannelSession("sess2", "client-b", "user1", "https://b.com/logout")

	uris, err := FrontChannelLogout("sess1")
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	if len(uris) != 1 {
		t.Errorf("expected 1 URI (client-b still active), got %d", len(uris))
	}
}

func TestFrontChannel_LogoutAlreadyDone(t *testing.T) {
	ResetFrontChannelSessions()
	RegisterFrontChannelSession("s1", "c", "u", "https://x.com/logout")
	FrontChannelLogout("s1")

	_, err := FrontChannelLogout("s1")
	if err == nil {
		t.Error("should error on double logout")
	}
}

func TestFrontChannel_LogoutNotFound(t *testing.T) {
	ResetFrontChannelSessions()
	_, err := FrontChannelLogout("nonexistent")
	if err == nil {
		t.Error("should error for nonexistent session")
	}
}

func TestFrontChannel_MultiClientLogout(t *testing.T) {
	ResetFrontChannelSessions()
	RegisterFrontChannelSession("s1", "c1", "user-x", "https://c1.com/logout")
	RegisterFrontChannelSession("s2", "c2", "user-x", "https://c2.com/logout")
	RegisterFrontChannelSession("s3", "c3", "user-x", "https://c3.com/logout")
	RegisterFrontChannelSession("s4", "c4", "user-y", "https://c4.com/logout")

	uris, _ := FrontChannelLogout("s1")
	if len(uris) != 2 {
		t.Errorf("expected 2 remaining URIs for user-x, got %d", len(uris))
	}
}

func TestFrontChannel_EmptyURI(t *testing.T) {
	ResetFrontChannelSessions()
	RegisterFrontChannelSession("s1", "c1", "u1", "") // no front_channel_logout_uri

	uris, err := FrontChannelLogout("s1")
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	if len(uris) != 0 {
		t.Error("should return no URIs when none configured")
	}
}
