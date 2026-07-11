package service

import (
	"github.com/ggid/ggid/pkg/pii"
)

// obfuscateForLog masks PII fields before logging.
// This ensures that email addresses, phone numbers, IP addresses, etc.
// are never written to logs in plaintext.
func obfuscateForLog(s string) string {
	return pii.Obfuscate(s)
}

// obfuscateEmail is a convenience wrapper for masking a single email.
func obfuscateEmail(email string) string {
	return pii.MaskEmail(email)
}
