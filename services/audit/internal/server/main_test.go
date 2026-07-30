package httpserver

import (
	"os"
	"testing"
)

// TestMain disables the webhook DNS-resolution SSRF check for the test
// suite — tests use non-resolvable fixture hostnames (example.com etc.).
// The check itself is covered by TestValidateWebhookURL cases using
// literal IPs and localhost.
func TestMain(m *testing.M) {
	_ = os.Setenv("GGID_WEBHOOK_SKIP_DNS", "true")
	os.Exit(m.Run())
}
