package service

import (
	"github.com/ggid/ggid/pkg/pii"
)

// obfuscateForLog masks PII fields before logging.
// Ensures email addresses, phone numbers, IPs, etc. are never in plaintext logs.
func obfuscateForLog(s string) string {
	return pii.Obfuscate(s)
}

// obfuscateEmail masks a single email address for log output.
func obfuscateEmail(email string) string {
	return pii.MaskEmail(email)
}
