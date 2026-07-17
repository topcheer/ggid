package server

import (
	"encoding/json"
	"net/http"
	"strings"
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

var reviewSeq int

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
	reviewSeq++
	rv.ID = fmtReviewID(reviewSeq)
	rv.Timestamp = time.Now()
	if mapRepoVar != nil {
		b, _ := json.Marshal(rv)
		var dataMap map[string]any
		json.Unmarshal(b, &dataMap)
		mapRepoVar.Store(r.Context(), "oauth_agent_reviews", rv.ID, dataMap)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rv)
}

func handleAgentReviewList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var list []map[string]any
	if mapRepoVar != nil {
		list, _ = mapRepoVar.List(r.Context(), "oauth_agent_reviews")
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
	if mapRepoVar != nil {
		rows, _ := mapRepoVar.List(r.Context(), "oauth_agent_reviews")
		for _, row := range rows {
			if aid, ok := row["agent_id"].(string); ok && aid == agentID {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(row)
				return
			}
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
	if mapRepoVar != nil {
		existing, err := mapRepoVar.Get(r.Context(), "oauth_agent_reviews", reviewID)
		if err != nil {
			http.Error(w, `{"error":"review not found"}`, http.StatusNotFound)
			return
		}
		existing["decision"] = rv.Decision
		existing["comment"] = rv.Comment
		existing["timestamp"] = time.Now()
		mapRepoVar.Store(r.Context(), "oauth_agent_reviews", reviewID, existing)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(existing)
		return
	}
	http.Error(w, `{"error":"review not found"}`, http.StatusNotFound)
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
