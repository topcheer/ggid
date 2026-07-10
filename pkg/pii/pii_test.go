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
		{"@example.com", "***"},
		{"user@", "u***@***"},
		{"user@example", "u***@***"},
		{"x.y.z@domain.info", "x***@d***.info"},
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

func TestMaskPhone_ShortAndEdge(t *testing.T) {
	// len <= 4: fully masked with *
	got := MaskPhone("123")
	if got != "***" {
		t.Errorf("MaskPhone(\"123\") = %q, want ***", got)
	}
	got = MaskPhone("abcd")
	if got != "****" {
		t.Errorf("MaskPhone(\"abcd\") = %q, want ****", got)
	}
	got = MaskPhone("")
	if got != "" {
		t.Errorf("MaskPhone(\"\") = %q, want empty", got)
	}
	// len > 4: last 4 preserved
	got = MaskPhone("12345")
	if !strings.HasSuffix(got, "2345") {
		t.Errorf("MaskPhone(\"12345\") = %q, want suffix 2345", got)
	}
}

func TestMaskIP(t *testing.T) {
	got := MaskIP("192.168.1.100")
	if got != "192.168.x.x" {
		t.Errorf("got %s", got)
	}
}

func TestMaskIP_EdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"192.168.1.100", "192.168.x.x"},
		{"10.0.0.1", "10.0.x.x"},
		{"255.255.255.255", "255.255.x.x"},
		{"not_an_ip", "not_an_ip"},
		{"1.2.3", "1.2.3"},
		{"", ""},
	}
	for _, tt := range tests {
		got := MaskIP(tt.input)
		if got != tt.want {
			t.Errorf("MaskIP(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskUUID(t *testing.T) {
	got := MaskUUID("550e8400-e29b-41d4-a716-446655440000")
	if !strings.HasPrefix(got, "550e8400-") {
		t.Errorf("got %s", got)
	}
}

func TestMaskUUID_EdgeCases(t *testing.T) {
	// No dash → returned as-is
	got := MaskUUID("no-dash-here")
	if !strings.HasPrefix(got, "no-") {
		t.Errorf("MaskUUID(\"no-dash-here\") = %q, want prefix 'no-'", got)
	}
	// Empty string → returned as-is
	got = MaskUUID("")
	if got != "" {
		t.Errorf("MaskUUID(\"\") = %q, want empty", got)
	}
	// Standard UUID: first segment kept, rest masked
	got = MaskUUID("550e8400-e29b-41d4-a716-446655440000")
	if !strings.HasPrefix(got, "550e8400-****") {
		t.Errorf("got %q, want 550e8400-**** prefix", got)
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

func TestObfuscate_MultipleEmails(t *testing.T) {
	input := "Send to alice@example.com and bob@test.org"
	output := Obfuscate(input)
	if strings.Contains(output, "alice@") || strings.Contains(output, "bob@") {
		t.Error("emails not masked in multi-email string")
	}
	if !strings.Contains(output, "***") {
		t.Error("expected mask characters")
	}
}

func TestObfuscate_UUID(t *testing.T) {
	input := "User ID: 550e8400-e29b-41d4-a716-446655440000"
	output := Obfuscate(input)
	// UUID: first segment kept, rest masked
	if strings.Contains(output, "e29b-41d4-a716-446655440000") {
		t.Error("UUID middle/end segments not masked")
	}
}

func TestObfuscate_EmptyString(t *testing.T) {
	output := Obfuscate("")
	if output != "" {
		t.Errorf("expected empty string, got %q", output)
	}
}

func TestObfuscate_NoPII(t *testing.T) {
	input := "This is a normal log message without PII."
	output := Obfuscate(input)
	if output != input {
		t.Errorf("expected unchanged output, got %q", output)
	}
}
