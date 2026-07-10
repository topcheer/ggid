package domain

import (
	"errors"
	"strings"
	"unicode"
)

// PasswordPolicy defines configurable password complexity rules.
// It mirrors the conf.PasswordPolicy but lives in the domain layer for
// reuse by the password service and HTTP handlers without importing conf.
type PasswordPolicy struct {
	MinLength      int      `json:"min_length"`
	RequireUpper   bool     `json:"require_upper"`
	RequireLower   bool     `json:"require_lower"`
	RequireDigit   bool     `json:"require_digit"`
	RequireSpecial bool     `json:"require_special"`
	Blacklist      []string `json:"blacklist,omitempty"`
	HistoryCount   int      `json:"history_count"`
	MaxAttempts    int      `json:"max_attempts"`
}

// Errors returned by password policy validation.
var (
	ErrPolicyTooShort    = errors.New("password is too short")
	ErrPolicyNoUpper     = errors.New("password must contain an uppercase letter")
	ErrPolicyNoLower     = errors.New("password must contain a lowercase letter")
	ErrPolicyNoDigit     = errors.New("password must contain a digit")
	ErrPolicyNoSpecial   = errors.New("password must contain a special character")
	ErrPolicyBlacklisted = errors.New("password is blacklisted")
)

// Validate checks a plaintext password against this policy.
func (p PasswordPolicy) Validate(password string) error {
	if len(password) < p.MinLength {
		return ErrPolicyTooShort
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case 'A' <= ch && ch <= 'Z':
			hasUpper = true
		case 'a' <= ch && ch <= 'z':
			hasLower = true
		case '0' <= ch && ch <= '9':
			hasDigit = true
		case !unicode.IsLetter(ch) && !unicode.IsDigit(ch):
			hasSpecial = true
		}
	}

	if p.RequireUpper && !hasUpper {
		return ErrPolicyNoUpper
	}
	if p.RequireLower && !hasLower {
		return ErrPolicyNoLower
	}
	if p.RequireDigit && !hasDigit {
		return ErrPolicyNoDigit
	}
	if p.RequireSpecial && !hasSpecial {
		return ErrPolicyNoSpecial
	}

	if len(p.Blacklist) > 0 {
		lower := strings.ToLower(password)
		for _, b := range p.Blacklist {
			if lower == strings.ToLower(b) {
				return ErrPolicyBlacklisted
			}
		}
	}

	return nil
}

// StrengthScore returns a 0-4 password strength score based on length and character variety.
func (p PasswordPolicy) StrengthScore(password string) int {
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case 'A' <= ch && ch <= 'Z':
			hasUpper = true
		case 'a' <= ch && ch <= 'z':
			hasLower = true
		case '0' <= ch && ch <= '9':
			hasDigit = true
		case !unicode.IsLetter(ch) && !unicode.IsDigit(ch):
			hasSpecial = true
		}
	}

	score := 0
	if len(password) >= 8 {
		score++
	}
	if len(password) >= 12 {
		score++
	}
	variety := 0
	if hasUpper {
		variety++
	}
	if hasLower {
		variety++
	}
	if hasDigit {
		variety++
	}
	if hasSpecial {
		variety++
	}
	if variety >= 3 {
		score++
	}
	if variety == 4 {
		score++
	}
	if score > 4 {
		score = 4
	}
	return score
}
