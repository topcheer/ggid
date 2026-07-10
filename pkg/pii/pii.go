package pii

import (
	"regexp"
	"strings"
)

// PII obfuscation for log streams and audit trails.
// Masks sensitive data while preserving enough structure for debugging.

var (
	emailRegex    = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phoneRegex    = regexp.MustCompile(`\+?[0-9]{1,3}?[-.\s]?\(?[0-9]{3}\)?[-.\s]?[0-9]{3,4}[-.\s]?[0-9]{4}`)
	ipRegex       = regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)
	uuidRegex     = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	ssnRegex      = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	creditCardRegex = regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`)
)

// MaskEmail masks an email address: "user@example.com" → "u***@e***.com"
func MaskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || len(parts[0]) == 0 {
		return "***"
	}
	local := string(parts[0][0]) + "***"
	domainParts := strings.SplitN(parts[1], ".", 2)
	domain := "***"
	if len(domainParts) >= 2 {
		domain = string(domainParts[0][0]) + "***." + domainParts[1]
	}
	return local + "@" + domain
}

// MaskPhone masks a phone number: "+1-234-567-8901" → "+1-***-***-8901"
func MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return strings.Repeat("*", len(phone))
	}
	return strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
}

// MaskIP masks an IPv4 address: "192.168.1.100" → "192.168.x.x"
func MaskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	return parts[0] + "." + parts[1] + ".x.x"
}

// MaskUUID keeps only the first segment: "550e8400-e29b-..." → "550e8400-****-..."
func MaskUUID(id string) string {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) < 2 {
		return id
	}
	return parts[0] + "-****-****-****-************"
}

// Obfuscate applies all masking rules to a string (for log output).
func Obfuscate(s string) string {
	s = emailRegex.ReplaceAllStringFunc(s, MaskEmail)
	s = phoneRegex.ReplaceAllStringFunc(s, MaskPhone)
	s = ipRegex.ReplaceAllStringFunc(s, MaskIP)
	s = uuidRegex.ReplaceAllStringFunc(s, MaskUUID)
	s = ssnRegex.ReplaceAllStringFunc(s, func(string) string { return "***-**-****" })
	s = creditCardRegex.ReplaceAllStringFunc(s, func(string) string { return "****-****-****-****" })
	return s
}
