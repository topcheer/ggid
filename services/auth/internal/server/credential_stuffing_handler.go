package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// loginAttempt tracks a login attempt for credential stuffing analysis.
type loginAttempt struct {
	IP        string
	UserAgent string
	Success   bool
	Timestamp time.Time
}

var (
	attemptLogMu sync.Mutex
	attemptLog   []loginAttempt
)

// RecordLoginAttempt adds a login attempt to the analysis log.
func RecordLoginAttempt(ip, ua string, success bool) {
	attemptLogMu.Lock()
	defer attemptLogMu.Unlock()
	attemptLog = append(attemptLog, loginAttempt{
		IP: ip, UserAgent: ua, Success: success, Timestamp: time.Now().UTC(),
	})
	// Trim to last 10,000
	if len(attemptLog) > 10000 {
		attemptLog = attemptLog[len(attemptLog)-10000:]
	}
}

// POST /api/v1/auth/detect-credential-stuffing
// Body: {"time_window": "1h", "min_attempts": 10}
func (h *Handler) handleDetectCredentialStuffing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		TimeWindow string `json:"time_window"`
		MinAttempts int   `json:"min_attempts"`
	}
	if r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.TimeWindow == "" {
		req.TimeWindow = "1h"
	}
	if req.MinAttempts <= 0 {
		req.MinAttempts = 10
	}

	dur, err := time.ParseDuration(req.TimeWindow)
	if err != nil || dur <= 0 {
		dur = time.Hour
	}
	cutoff := time.Now().UTC().Add(-dur)

	// Analyze recent attempts
	attemptLogMu.Lock()
	defer attemptLogMu.Unlock()

	ipCounts := make(map[string]int)
	ipSuccess := make(map[string]int)
	uaSet := make(map[string]int)
	total := 0

	for _, a := range attemptLog {
		if a.Timestamp.Before(cutoff) {
			continue
		}
		total++
		ipCounts[a.IP]++
		if a.Success {
			ipSuccess[a.IP]++
		}
		uaSet[a.UserAgent]++
	}

	// Detect credential stuffing indicators
	var blockedIPs []string
	highVolumeIPs := 0
	lowSuccessRate := false

	for ip, count := range ipCounts {
		if count >= req.MinAttempts {
			highVolumeIPs++
			successRate := float64(ipSuccess[ip]) / float64(count)
			if successRate < 0.05 { // <5% success = suspicious
				lowSuccessRate = true
				blockedIPs = append(blockedIPs, ip)
			}
		}
	}

	// User agent diversity (many UAs from same IP = bot)
	uaDiversity := len(uaSet)
	ipSpread := len(ipCounts)

	// Confidence score
	confidence := 0.0
	if highVolumeIPs > 0 {
		confidence += 30
	}
	if lowSuccessRate {
		confidence += 30
	}
	if uaDiversity > 5 {
		confidence += 20
	}
	if ipSpread > 3 {
		confidence += 20
	}
	if confidence > 100 {
		confidence = 100
	}

	detected := confidence >= 60

	if blockedIPs == nil {
		blockedIPs = []string{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"is_detected":       detected,
		"confidence":        int(confidence),
		"total_attempts":    total,
		"unique_ips":        ipSpread,
		"unique_user_agents": uaDiversity,
		"high_volume_ips":   highVolumeIPs,
		"low_success_rate":  lowSuccessRate,
		"blocked_ips":       blockedIPs,
		"time_window":       req.TimeWindow,
		"analyzed_at":       time.Now().UTC().Format(time.RFC3339),
	})
}
