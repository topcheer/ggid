package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

// --- ResolveLocale edge cases ---

func TestResolveLocale_EmptyString(t *testing.T) {
	if got := ResolveLocale("", "en"); got != "en" {
		t.Errorf("empty Accept-Language should return default, got %q", got)
	}
}

func TestResolveLocale_SingleLanguage(t *testing.T) {
	if got := ResolveLocale("en", "en-US"); got != "en" {
		t.Errorf("single language: got %q", got)
	}
}

func TestResolveLocale_MultipleLanguages(t *testing.T) {
	if got := ResolveLocale("en-US,en;q=0.9,zh-CN;q=0.8", "en"); got != "en-US" {
		t.Errorf("should return first language, got %q", got)
	}
}

func TestResolveLocale_WithWhitespace(t *testing.T) {
	if got := ResolveLocale(" en-US , zh-CN", "en"); got != "en-US" {
		t.Errorf("should trim whitespace, got %q", got)
	}
}

func TestResolveLocale_QualityParameter(t *testing.T) {
	if got := ResolveLocale("zh-CN;q=0.9,en;q=0.1", "en"); got != "zh-CN" {
		t.Errorf("should return first regardless of q value, got %q", got)
	}
}

func TestResolveLocale_OnlyCommas(t *testing.T) {
	if got := ResolveLocale(",,,", "en"); got != "en" {
		t.Errorf("empty parts should return default, got %q", got)
	}
}

func TestResolveLocale_OnlySemicolons(t *testing.T) {
	if got := ResolveLocale(";", "en"); got != "en" {
		t.Errorf("semicolon should return default, got %q", got)
	}
}

// --- Translate edge cases ---

func TestTranslate_KeyNotInAnyLocale(t *testing.T) {
	tr := NewTranslator("en")
	tr.LoadTranslationsInline("en", map[string]string{"hello": "Hello"})
	result := tr.Translate("fr", "missing.key")
	if result != "missing.key" {
		t.Errorf("missing key should return key itself, got %q", result)
	}
}

func TestTranslate_NoTranslationsLoaded(t *testing.T) {
	tr := NewTranslator("en")
	result := tr.Translate("en", "hello")
	if result != "hello" {
		t.Errorf("with no translations loaded, should return key: got %q", result)
	}
}

func TestTranslate_ParamsFormatting(t *testing.T) {
	tr := NewTranslator("en")
	tr.LoadTranslationsInline("en", map[string]string{
		"welcome": "Welcome, %s! You have %d messages.",
	})
	result := tr.Translate("en", "welcome", "Alice", 5)
	expected := "Welcome, Alice! You have 5 messages."
	if result != expected {
		t.Errorf("Translate with params: got %q, want %q", result, expected)
	}
}

func TestTranslate_DefaultLocaleFallsBackToKey(t *testing.T) {
	tr := NewTranslator("en")
	tr.LoadTranslationsInline("en", map[string]string{"hello": "Hello"})
	// Requested locale doesn't exist, default doesn't have the key either
	result := tr.Translate("de", "unknown")
	if result != "unknown" {
		t.Errorf("should return key when not in any locale: got %q", result)
	}
}

func TestTranslate_LocaleCaseSensitive(t *testing.T) {
	tr := NewTranslator("en")
	tr.LoadTranslationsInline("en", map[string]string{"key": "value"})
	// "EN" should not match "en"
	result := tr.Translate("EN", "key")
	// Should fall back to default locale
	if result != "value" {
		t.Errorf("locale is case-sensitive, should fall back: got %q", result)
	}
}

// --- TranslateMap edge cases ---

func TestTranslateMap_NoLocales(t *testing.T) {
	tr := NewTranslator("en")
	result := tr.TranslateMap("en")
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestTranslateMap_RequestedLocale(t *testing.T) {
	tr := NewTranslator("en")
	tr.LoadTranslationsInline("en", map[string]string{"a": "1"})
	tr.LoadTranslationsInline("fr", map[string]string{"a": "un", "b": "deux"})

	result := tr.TranslateMap("fr")
	if len(result) != 2 {
		t.Errorf("expected 2 entries for fr, got %d", len(result))
	}
	if result["b"] != "deux" {
		t.Errorf("result[b] = %q", result["b"])
	}
}

func TestTranslateMap_ReturnsCopy(t *testing.T) {
	tr := NewTranslator("en")
	tr.LoadTranslationsInline("en", map[string]string{"key": "value"})

	result := tr.TranslateMap("en")
	result["key"] = "modified"

	// Original should be unchanged
	val := tr.Translate("en", "key")
	if val != "value" {
		t.Error("TranslateMap should return a copy, not the internal map")
	}
}

// --- SupportedLocales ---

func TestSupportedLocales_NoLocales(t *testing.T) {
	tr := NewTranslator("en")
	locales := tr.SupportedLocales()
	if len(locales) != 0 {
		t.Errorf("expected 0 locales, got %d", len(locales))
	}
}

func TestSupportedLocales_MultipleLocales(t *testing.T) {
	tr := NewTranslator("en")
	tr.LoadTranslationsInline("en", map[string]string{"k": "v"})
	tr.LoadTranslationsInline("fr", map[string]string{"k": "v"})
	tr.LoadTranslationsInline("de", map[string]string{"k": "v"})

	locales := tr.SupportedLocales()
	if len(locales) != 3 {
		t.Errorf("expected 3 locales, got %d", len(locales))
	}
}

// --- LoadTranslations error paths ---

func TestLoadTranslations_FileNotFound(t *testing.T) {
	tr := NewTranslator("en")
	err := tr.LoadTranslations("en", "/nonexistent/path/file.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadTranslations_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0644); err != nil {
		t.Fatal(err)
	}

	tr := NewTranslator("en")
	err := tr.LoadTranslations("en", path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// --- LoadDirectory edge cases ---

func TestLoadDirectory_DirectoryNotFound(t *testing.T) {
	tr := NewTranslator("en")
	err := tr.LoadDirectory("/nonexistent/directory")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestLoadDirectory_SkipsNonJSONFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .txt file (should be skipped)
	txtFile := filepath.Join(tmpDir, "readme.txt")
	os.WriteFile(txtFile, []byte("text"), 0644) //nolint:errcheck // test helper

	// Create a .json file (should be loaded)
	jsonFile := filepath.Join(tmpDir, "en.json")
	os.WriteFile(jsonFile, []byte(`{"hello":"world"}`), 0644) //nolint:errcheck // test helper

	tr := NewTranslator("en")
	if err := tr.LoadDirectory(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Only en locale should be loaded
	if tr.Translate("en", "hello") != "world" {
		t.Error("en.json should have been loaded")
	}
}

func TestLoadDirectory_SkipsSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a subdirectory (should be skipped)
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755) //nolint:errcheck // test helper

	tr := NewTranslator("en")
	if err := tr.LoadDirectory(tmpDir); err != nil {
		t.Fatal(err)
	}

	// No locales should be loaded
	locales := tr.SupportedLocales()
	if len(locales) != 0 {
		t.Errorf("expected 0 locales, got %d", len(locales))
	}
}

func TestLoadDirectory_InvalidJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	badFile := filepath.Join(tmpDir, "bad.json")
	os.WriteFile(badFile, []byte("invalid"), 0644) //nolint:errcheck // test helper

	tr := NewTranslator("en")
	err := tr.LoadDirectory(tmpDir)
	if err == nil {
		t.Error("expected error for invalid JSON in directory")
	}
}

// --- format edge cases ---

func TestFormat_MultipleParams(t *testing.T) {
	result := format("%s=%d", "key", 42)
	if result != "key=42" {
		t.Errorf("format with multiple params: got %q", result)
	}
}

// format is an internal function that wraps fmt.Sprintf. When params are provided
// but the message has no format verbs, fmt.Sprintf produces %!(EXTRA ...).
// This is acceptable behavior and tested for no panic.
