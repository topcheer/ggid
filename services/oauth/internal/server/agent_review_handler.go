package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type AgentReview struct {
	ID              string    `json:"id"`
	AgentID         string    `json:"agent_id"`
	Reviewer        string    `json:"reviewer"`
	ScopesReviewed  []string  `json:"scopes_reviewed"`
	Decision        string    `json:"decision"`
	Comment         string    `json:"comment"`
	Timestamp       time.Time `json:"timestamp"`
}

var (
	reviewStore = make(map[string]*AgentReview)
	reviewMu    sync.RWMutex
	reviewSeq   int
)

func handleAgentReviewCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var rv AgentReview
	if err := json.NewDecoder(r.Body).Decode(&rv); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if rv.AgentID == "" || rv.Reviewer == "" {
		http.Error(w, `{"error":"agent_id and reviewer required"}`, http.StatusBadRequest)
		return
	}
	reviewMu.Lock()
	reviewSeq++
	rv.ID = fmtReviewID(reviewSeq)
	rv.Timestamp = time.Now()
	reviewStore[rv.ID] = &rv
	reviewMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rv)
}

func handleAgentReviewList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	reviewMu.RLock()
	defer reviewMu.RUnlock()
	var list []*AgentReview
	for _, rv := range reviewStore {
		list = append(list, rv)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func handleAgentReviewGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, `{"error":"agent id required"}`, http.StatusBadRequest)
		return
	}
	agentID := parts[4]
	reviewMu.RLock()
	defer reviewMu.RUnlock()
	for _, rv := range reviewStore {
		if rv.AgentID == agentID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(rv)
			return
		}
	}
	http.Error(w, `{"error":"review not found"}`, http.StatusNotFound)
}

func handleAgentReviewUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, `{"error":"review id required"}`, http.StatusBadRequest)
		return
	}
	reviewID := parts[len(parts)-1]
	var rv AgentReview
	if err := json.NewDecoder(r.Body).Decode(&rv); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	reviewMu.Lock()
	defer reviewMu.Unlock()
	existing, ok := reviewStore[reviewID]
	if !ok {
		http.Error(w, `{"error":"review not found"}`, http.StatusNotFound)
		return
	}
	existing.Decision = rv.Decision
	existing.Comment = rv.Comment
	existing.Timestamp = time.Now()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func fmtReviewID(n int) string {
	const hex = "0123456789abcdef"
	if n == 0 {
		return "rev_0"
	}
	var buf []byte
	for n > 0 {
		buf = append([]byte{hex[n%16]}, buf...)
		n /= 16
	}
	return "rev_" + string(buf)
}