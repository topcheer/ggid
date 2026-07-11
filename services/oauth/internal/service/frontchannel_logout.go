package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FrontChannelSession tracks an OIDC session for front-channel logout.
type FrontChannelSession struct {
	ID                uuid.UUID
	ClientID          string
	UserID            string
	FrontChannelURI   string // client's registered front_channel_logout_uri
	CreatedAt         time.Time
	LoggedOut         bool
}

var (
	fcMu       sync.RWMutex
	fcSessions = make(map[string]*FrontChannelSession) // sessionID → session
)

// RegisterFrontChannelSession creates or updates a front-channel logout session.
func RegisterFrontChannelSession(sessionID, clientID, userID, frontChannelURI string) *FrontChannelSession {
	fcMu.Lock()
	defer fcMu.Unlock()
	s := &FrontChannelSession{
		ID:              uuid.New(),
		ClientID:        clientID,
		UserID:          userID,
		FrontChannelURI: frontChannelURI,
		CreatedAt:       time.Now().UTC(),
	}
	fcSessions[sessionID] = s
	return s
}

// FrontChannelLogout processes logout and returns all front_channel_logout_uris
// for active sessions belonging to the same user (across multiple clients).
func FrontChannelLogout(sessionID string) ([]string, error) {
	fcMu.Lock()
	defer fcMu.Unlock()

	session, ok := fcSessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	if session.LoggedOut {
		return nil, fmt.Errorf("already logged out")
	}

	// Collect all active sessions for the same user (different clients)
	var uris []string
	userID := session.UserID
	session.LoggedOut = true

	for _, s := range fcSessions {
		if s.UserID == userID && !s.LoggedOut {
			if s.FrontChannelURI != "" {
				uris = append(uris, s.FrontChannelURI)
			}
			s.LoggedOut = true
		}
	}

	return uris, nil
}

// GetFrontChannelSession returns a session by ID.
func GetFrontChannelSession(sessionID string) (*FrontChannelSession, error) {
	fcMu.RLock()
	defer fcMu.RUnlock()
	s, ok := fcSessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return s, nil
}

// ResetFrontChannelSessions clears all sessions (for testing).
func ResetFrontChannelSessions() {
	fcMu.Lock()
	defer fcMu.Unlock()
	fcSessions = make(map[string]*FrontChannelSession)
}
