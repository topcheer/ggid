package server

import (
	"testing"
)

func TestMaskValue_FullMask(t *testing.T) {
	result := MaskValue("secret123", "full_mask")
	if result != "*********" {
		t.Errorf("expected *********, got %s", result)
	}
}

func TestMaskValue_PartialMask(t *testing.T) {
	result := MaskValue("alice@example.com", "partial_mask")
	if len(result) != len("alice@example.com") {
		t.Errorf("partial mask should preserve length, got %d", len(result))
	}
	if result[:2] != "al" {
		t.Errorf("partial mask should preserve first 2 chars, got %s", result[:2])
	}
}

func TestMaskValue_ShortValue(t *testing.T) {
	result := MaskValue("ab", "partial_mask")
	if result != "****" {
		t.Errorf("short value partial mask should be ****, got %s", result)
	}
}

func TestMaskValue_None(t *testing.T) {
	result := MaskValue("hello", "none")
	if result != "hello" {
		t.Errorf("none rule should return original, got %s", result)
	}
}

func TestMaskValue_Empty(t *testing.T) {
	result := MaskValue("", "full_mask")
	if result != "" {
		t.Errorf("empty string full_mask should return empty, got %s", result)
	}
}
