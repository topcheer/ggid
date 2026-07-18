package server

import (
	"encoding/json"
	"net/http"
)

type SyntheticFlaggedAccount struct {
	Email             string  `json:"email"`
	RegistrationSource string `json:"registration_source"`
	DisposableDomain  bool    `json:"disposable_domain"`
	AccountAgeDays    int     `json:"account_age_days"`
	RiskScore         float64 `json:"risk_score"`
}

type SyntheticIdentityResult struct {
	FlaggedAccounts         []SyntheticFlaggedAccount `json:"flagged_accounts"`
	DisposableDomainsBlocklist []string      `json:"disposable_domains_blocklist"`
	AutoBlock               bool             `json:"auto_block"`
	TotalScanned            int              `json:"total_scanned"`
	FlaggedCount            int              `json:"flagged_count"`
}

func (h *Handler) handleSyntheticIdentityDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := SyntheticIdentityResult{
		FlaggedAccounts: []SyntheticFlaggedAccount{
			{Email: "user1@temp-mail.dev", RegistrationSource: "self_signup", DisposableDomain: true, AccountAgeDays: 1, RiskScore: 0.92},
			{Email: "user2@guerrillamail.com", RegistrationSource: "self_signup", DisposableDomain: true, AccountAgeDays: 0, RiskScore: 0.88},
			{Email: "user3@10minutemail.com", RegistrationSource: "api", DisposableDomain: true, AccountAgeDays: 2, RiskScore: 0.85},
			{Email: "testuser@protonmail.com", RegistrationSource: "self_signup", DisposableDomain: false, AccountAgeDays: 3, RiskScore: 0.45},
		},
		DisposableDomainsBlocklist: []string{"temp-mail.dev", "guerrillamail.com", "10minutemail.com", "mailinator.com", "throwaway.email"},
		AutoBlock:   true,
		TotalScanned: 8420,
		FlaggedCount: 4,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
