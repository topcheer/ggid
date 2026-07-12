package server

import (
	"encoding/json"
	"net/http"
)

type JWTForgeryPattern struct {
	PatternID    string  `json:"pattern_id"`
	Type         string  `json:"type"`
	Detail       string  `json:"detail"`
	Confidence   float64 `json:"confidence"`
	DetectedCount int    `json:"detected_count"`
}

type GoldenTicketResult struct {
	JWTForgeryPatterns  []JWTForgeryPattern `json:"jwt_forgery_patterns"`
	DetectedCount       int                 `json:"detected_count"`
	FalsePositiveRate   float64             `json:"false_positive_rate"`
	BlockedTokens       int                 `json:"blocked_tokens"`
	RecommendedAction   string              `json:"recommended_action"`
}

func (h *Handler) handleGoldenTicketDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := GoldenTicketResult{
		JWTForgeryPatterns: []JWTForgeryPattern{
			{PatternID: "gt-001", Type: "abnormal_claims", Detail: "Token contains admin claims not present in IdP", Confidence: 0.92, DetectedCount: 3},
			{PatternID: "gt-002", Type: "issuer_mismatch", Detail: "iss claim does not match expected issuer URL", Confidence: 0.88, DetectedCount: 2},
			{PatternID: "gt-003", Type: "signature_anomaly", Detail: "RSA signature uses deprecated key ID (rotated 90d ago)", Confidence: 0.79, DetectedCount: 5},
			{PatternID: "gt-004", Type: "expiry_anomaly", Detail: "Token expiry 24h but max configured is 1h", Confidence: 0.95, DetectedCount: 1},
		},
		DetectedCount:     11,
		FalsePositiveRate: 0.08,
		BlockedTokens:     9,
		RecommendedAction: "Rotate JWT signing keys immediately. Investigate tokens with abnormal_claims pattern. Audit all tokens issued with deprecated key ID.",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
