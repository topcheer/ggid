package email

import (
	"strings"
	"testing"
)

// TestPasswordResetHTML_EscapesUserName verifies that the password reset email
// HTML-escapes user-supplied fields to prevent HTML/XSS injection.
func TestPasswordResetHTML_EscapesUserName(t *testing.T) {
	malicious := `<script>alert('xss')</script>`
	html := PasswordResetHTML(PasswordResetData{
		UserName: malicious,
		Link:     "https://ggid.dev/reset?token=abc",
		Expiry:   "30 minutes",
		AppName:  "TestApp",
	})

	// The raw script tag must NOT appear unescaped in the HTML body
	if strings.Contains(html, "<script>") {
		t.Errorf("HTML injection: <script> tag not escaped in password reset email")
	}
	// The escaped version must be present
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("expected escaped &lt;script&gt; in HTML, got: %s", html)
	}
}

// TestPasswordResetHTML_EscapesImgTag verifies that <img> tags in user input
// are escaped (rendered as text, not interpreted as HTML elements).
func TestPasswordResetHTML_EscapesImgTag(t *testing.T) {
	malicious := `<img src=x onerror=alert(1)>`
	html := PasswordResetHTML(PasswordResetData{
		UserName: malicious,
		Link:     "https://ggid.dev/reset?token=abc",
	})

	// The raw <img tag must NOT appear — it must be escaped to &lt;img
	if strings.Contains(html, "<img ") {
		t.Errorf("HTML injection: <img> tag not escaped in password reset email")
	}
	if !strings.Contains(html, "&lt;img") {
		t.Errorf("expected escaped &lt;img in HTML output")
	}
}

func TestEmailVerificationHTML_EscapesUserName(t *testing.T) {
	malicious := `"><script>document.cookie</script>`
	html := EmailVerificationHTML(EmailVerificationData{
		UserName: malicious,
		Link:     "https://ggid.dev/verify?token=abc",
		AppName:  "TestApp",
	})

	if strings.Contains(html, "<script>") {
		t.Errorf("HTML injection: <script> tag not escaped in email verification")
	}
}

func TestWelcomeHTML_EscapesUserName(t *testing.T) {
	malicious := `<b>bold</b><script>alert(1)</script>`
	html := WelcomeHTML(WelcomeData{
		UserName: malicious,
		AppName:  "TestApp",
		Link:     "https://ggid.dev/start",
	})

	if strings.Contains(html, "<script>") {
		t.Errorf("HTML injection: <script> tag not escaped in welcome email")
	}
}

func TestMFACodeHTML_EscapesUserName(t *testing.T) {
	malicious := `<script>alert('xss')</script>`
	html := MFACodeHTML(MFACodeData{
		UserName: malicious,
		Code:     "123456",
	})

	if strings.Contains(html, "<script>") {
		t.Errorf("HTML injection: <script> tag not escaped in MFA code email")
	}
}

// TestPasswordResetHTML_BlocksJavascriptProtocol verifies that the link field
// blocks dangerous protocols like javascript: in the href attribute.
func TestPasswordResetHTML_BlocksJavascriptProtocol(t *testing.T) {
	html := PasswordResetHTML(PasswordResetData{
		UserName: "test",
		Link:     "javascript:alert(1)//",
		Expiry:   "30 minutes",
	})

	// The javascript: protocol must NOT appear in the href attribute
	if strings.Contains(html, "javascript:alert") {
		t.Errorf("Link injection: javascript: protocol not blocked in href")
	}
	// Should be replaced with a safe placeholder
	if !strings.Contains(html, `href="#"`) {
		t.Errorf("expected href='#' for dangerous protocol, got unexpected URL in output")
	}
}

// TestSafeURL_AllowsHTTPS verifies that valid HTTPS links are preserved.
func TestSafeURL_AllowsHTTPS(t *testing.T) {
	cases := []string{
		"https://ggid.dev/reset?token=abc",
		"http://localhost:8080/verify",
		"/relative/path",
	}
	for _, link := range cases {
		result := safeURL(link)
		if result == "#" {
			t.Errorf("safeURL(%q) should not be blocked", link)
		}
	}
}

// TestSafeURL_BlocksDangerousProtocols verifies that dangerous protocols are blocked.
func TestSafeURL_BlocksDangerousProtocols(t *testing.T) {
	cases := []string{
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"vbscript:msgbox(1)",
	}
	for _, link := range cases {
		result := safeURL(link)
		if result != "#" {
			t.Errorf("safeURL(%q) should return '#', got %q", link, result)
		}
	}
}
