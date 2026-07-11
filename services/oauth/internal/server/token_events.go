package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// TokenEvent represents a token lifecycle event for the SSE stream.
type TokenEvent struct {
	Type      string    `json:"type"` // issued, refreshed, revoked, expired
	TokenID   string    `json:"token_id"`
	ClientID  string    `json:"client_id"`
	UserID    string    `json:"user_id"`
	Scope     string    `json:"scope"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	tokenEventMu sync.Mutex
	tokenEvents  []TokenEvent
	tokenSubs    []chan TokenEvent
)

// EmitTokenEvent broadcasts a token lifecycle event to all SSE subscribers.
func EmitTokenEvent(evt TokenEvent) {
	tokenEventMu.Lock()
	defer tokenEventMu.Unlock()
	tokenEvents = append(tokenEvents, evt)
	if len(tokenEvents) > 10000 {
		tokenEvents = tokenEvents[len(tokenEvents)-5000:]
	}
	for _, ch := range tokenSubs {
		select {
		case ch <- evt:
		default: // drop if subscriber is slow
		}
	}
}

// GET /api/v1/oauth/token-events/stream (SSE)
func handleTokenEventStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Subscribe
	ch := make(chan TokenEvent, 100)
	tokenEventMu.Lock()
	tokenSubs = append(tokenSubs, ch)
	// Send recent history
	recent := tokenEvents
	if len(recent) > 50 {
		recent = recent[len(recent)-50:]
	}
	tokenEventMu.Unlock()
	defer func() {
		tokenEventMu.Lock()
		for i, s := range tokenSubs {
			if s == ch {
				tokenSubs = append(tokenSubs[:i], tokenSubs[i+1:]...)
				break
			}
		}
		tokenEventMu.Unlock()
		close(ch)
	}()

	// Send recent events first
	for _, evt := range recent {
		fmt.Fprintf(w, "data: {\"type\":\"%s\",\"token_id\":\"%s\",\"client_id\":\"%s\",\"timestamp\":\"%s\"}\n\n",
			evt.Type, evt.TokenID, evt.ClientID, evt.Timestamp.Format(time.RFC3339))
	}
	flusher.Flush()

	// Stream new events
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-ch:
			fmt.Fprintf(w, "data: {\"type\":\"%s\",\"token_id\":\"%s\",\"client_id\":\"%s\",\"user_id\":\"%s\",\"scope\":\"%s\",\"timestamp\":\"%s\"}\n\n",
				evt.Type, evt.TokenID, evt.ClientID, evt.UserID, evt.Scope, evt.Timestamp.Format(time.RFC3339))
			flusher.Flush()
		case <-time.After(30 * time.Second):
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}
