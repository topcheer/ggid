package server

import (
	"encoding/json"
	"net/http"
)

type FingerprintCluster struct {
	ClusterID   string  `json:"cluster_id"`
	Count       int     `json:"count"`
	CommonUA    string  `json:"common_user_agent"`
	Platform    string  `json:"platform"`
	RiskScore   float64 `json:"risk_score"`
}

type SuspiciousFingerprint struct {
	Fingerprint string `json:"fingerprint"`
	Reason      string `json:"reason"`
	Severity    string `json:"severity"`
	SeenCount   int    `json:"seen_count"`
}

type HashDistribution struct {
	HashPrefix string `json:"hash_prefix"`
	Count      int    `json:"count"`
}

type DeviceFingerprintAnalytics struct {
	UniqueFingerprintsCount int                     `json:"unique_fingerprints_count"`
	FingerprintClusters     []FingerprintCluster     `json:"fingerprint_clusters"`
	SuspiciousFingerprints  []SuspiciousFingerprint  `json:"suspicious_fingerprints"`
	CanvasHashDistribution  []HashDistribution       `json:"canvas_hash_distribution"`
	WebGLHashDistribution   []HashDistribution       `json:"webgl_hash_distribution"`
	GeneratedAt             string                   `json:"generated_at"`
}

func (h *Handler) handleDeviceFingerprintAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := DeviceFingerprintAnalytics{
		UniqueFingerprintsCount: 8420,
		FingerprintClusters: []FingerprintCluster{
			{ClusterID: "fc-001", Count: 3200, CommonUA: "Chrome/120 macOS", Platform: "macOS", RiskScore: 0.12},
			{ClusterID: "fc-002", Count: 2100, CommonUA: "Safari/17 iOS", Platform: "iOS", RiskScore: 0.08},
			{ClusterID: "fc-003", Count: 1800, CommonUA: "Chrome/120 Windows", Platform: "Windows", RiskScore: 0.15},
			{ClusterID: "fc-004", Count: 320, CommonUA: "headless-chrome", Platform: "Linux", RiskScore: 0.82},
		},
		SuspiciousFingerprints: []SuspiciousFingerprint{
			{Fingerprint: "fp-headless-001", Reason: "headless_browser", Severity: "high", SeenCount: 450},
			{Fingerprint: "fp-spoofed-042", Reason: "canvas_spoofing_detected", Severity: "high", SeenCount: 120},
			{Fingerprint: "fp-inconsistent-088", Reason: "webgl_canvas_mismatch", Severity: "medium", SeenCount: 85},
			{Fingerprint: "fp-inconsistent-156", Reason: "timezone_locale_mismatch", Severity: "medium", SeenCount: 42},
		},
		CanvasHashDistribution: []HashDistribution{
			{HashPrefix: "a1b2", Count: 4200},
			{HashPrefix: "c3d4", Count: 2800},
			{HashPrefix: "e5f6", Count: 820},
			{HashPrefix: "0000", Count: 600},
		},
		WebGLHashDistribution: []HashDistribution{
			{HashPrefix: "ff00", Count: 3800},
			{HashPrefix: "aacc", Count: 2400},
			{HashPrefix: "1234", Count: 1220},
			{HashPrefix: "ffff", Count: 1000},
		},
		GeneratedAt: "2025-01-15T10:00:00Z",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
