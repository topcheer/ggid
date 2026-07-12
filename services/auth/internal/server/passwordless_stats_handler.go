package server

import (
	"encoding/json"
	"net/http"
)

type MethodDist struct {
	Method   string  `json:"method"`
	Count    int     `json:"count"`
	Pct      float64 `json:"pct"`
}

type DeviceStat struct {
	DeviceType    string  `json:"device_type"`
	Attempts      int     `json:"attempts"`
	SuccessRate   float64 `json:"success_rate"`
	AvgTimeMs     float64 `json:"avg_completion_time_ms"`
}

type PasswordlessStats struct {
	TotalAttempts       int          `json:"total_attempts"`
	MethodDistribution  []MethodDist `json:"method_distribution"`
	OverallSuccessRate  float64      `json:"success_rate"`
	AvgCompletionTimeMs float64      `json:"avg_completion_time_ms"`
	AbandonmentRate     float64      `json:"abandonment_rate"`
	ByDeviceType        []DeviceStat `json:"by_device_type"`
	GeneratedAt         string       `json:"generated_at"`
}

func (h *Handler) handlePasswordlessStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := PasswordlessStats{
		TotalAttempts: 8920,
		MethodDistribution: []MethodDist{
			{Method: "magic_link", Count: 4200, Pct: 47.1},
			{Method: "passkey", Count: 3100, Pct: 34.8},
			{Method: "biometric", Count: 1620, Pct: 18.1},
		},
		OverallSuccessRate:  0.872,
		AvgCompletionTimeMs: 4250,
		AbandonmentRate:     0.128,
		ByDeviceType: []DeviceStat{
			{DeviceType: "mobile", Attempts: 5100, SuccessRate: 0.89, AvgTimeMs: 3800},
			{DeviceType: "desktop", Attempts: 3200, SuccessRate: 0.85, AvgTimeMs: 5100},
			{DeviceType: "tablet", Attempts: 620, SuccessRate: 0.91, AvgTimeMs: 3900},
		},
		GeneratedAt: "2025-01-15T10:00:00Z",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
