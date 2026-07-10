package pii

import (
	"strings"
	"testing"
)

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user@example.com", "u***@e***.com"},
		{"a@b.co", "a***@b***.co"},
		{"", "***"},
	}
	for _, tt := range tests {
		got := MaskEmail(tt.input)
		if got != tt.want {
			t.Errorf("MaskEmail(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskPhone(t *testing.T) {
	got := MaskPhone("+1-234-567-8901")
	if !strings.HasSuffix(got, "8901") {
		t.Errorf("expected last 4 digits, got %s", got)
	}
}

func TestMaskIP(t *testing.T) {
	got := MaskIP("192.168.1.100")
	if got != "192.168.x.x" {
		t.Errorf("got %s", got)
	}
}

func TestMaskUUID(t *testing.T) {
	got := MaskUUID("550e8400-e29b-41d4-a716-446655440000")
	if !strings.HasPrefix(got, "550e8400-") {
		t.Errorf("got %s", got)
	}
}

func TestObfuscate(t *testing.T) {
	input := "User john@example.com from 192.168.1.1 called from +1-234-567-8901"
	output := Obfuscate(input)
	if strings.Contains(output, "john@example.com") {
		t.Error("email not masked")
	}
	if strings.Contains(output, "192.168.1.1") {
		t.Error("IP not masked")
	}
	if strings.Contains(output, "+1-234-567-8901") {
		t.Error("phone not masked")
	}
}

func TestObfuscate_SSN(t *testing.T) {
	input := "SSN: 123-45-6789"
	output := Obfuscate(input)
	if strings.Contains(output, "123-45-6789") {
		t.Error("SSN not masked")
	}
}

func TestObfuscate_CreditCard(t *testing.T) {
	input := "Card: 4111-1111-1111-1111"
	output := Obfuscate(input)
	if strings.Contains(output, "4111") {
		t.Error("credit card not masked")
	}
}
