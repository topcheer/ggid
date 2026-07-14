package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTranslator(t *testing.T) {
	tr := NewTranslator("en")
	if tr == nil {
		t.Fatal("expected non-nil translator")
	}
	if tr.defaultLocale != "en" {
		t.Errorf("expected default locale 'en', got '%s'", tr.defaultLocale)
	}
}

func TestTranslate_MissingLocale(t *testing.T) {
	tr := NewTranslator("en")
	got := tr.Translate("zh", "nonexistent")
	if got != "nonexistent" {
		t.Errorf("expected key as fallback, got %q", got)
	}
}

func TestTranslate_WithParams(t *testing.T) {
	tr := NewTranslator("en")
	tr.translations["en"] = map[string]string{
		"welcome": "Hello %s, you have %d messages",
	}
	got := tr.Translate("en", "welcome", "Alice", 5)
	if got != "Hello Alice, you have 5 messages" {
		t.Errorf("unexpected translation: %q", got)
	}
}

func TestTranslate_FallbackToDefault(t *testing.T) {
	tr := NewTranslator("en")
	tr.translations["en"] = map[string]string{
		"login": "Login",
	}
	// Request fr, should fall back to en
	got := tr.Translate("fr", "login")
	if got != "Login" {
		t.Errorf("expected fallback to 'Login', got %q", got)
	}
}

func TestTranslate_ExactLocale(t *testing.T) {
	tr := NewTranslator("en")
	tr.translations["en"] = map[string]string{"greeting": "Hello"}
	tr.translations["zh-CN"] = map[string]string{"greeting": "你好"}
	got := tr.Translate("zh-CN", "greeting")
	if got != "你好" {
		t.Errorf("expected '你好', got %q", got)
	}
}

func TestTranslateMap(t *testing.T) {
	tr := NewTranslator("en")
	tr.translations["en"] = map[string]string{
		"login": "Login",
		"logout": "Logout",
	}
	m := tr.TranslateMap("en")
	if len(m) != 2 {
		t.Errorf("expected 2 translations, got %d", len(m))
	}
	if m["login"] != "Login" {
		t.Error("expected login key")
	}
}

func TestTranslateMap_Fallback(t *testing.T) {
	tr := NewTranslator("en")
	tr.translations["en"] = map[string]string{"key": "value"}
	m := tr.TranslateMap("fr")
	if m["key"] != "value" {
		t.Error("expected fallback to default locale")
	}
}

func TestSupportedLocales(t *testing.T) {
	tr := NewTranslator("en")
	tr.translations["en"] = map[string]string{}
	tr.translations["zh-CN"] = map[string]string{}
	tr.translations["ja"] = map[string]string{}
	locales := tr.SupportedLocales()
	if len(locales) != 3 {
		t.Errorf("expected 3 locales, got %d", len(locales))
	}
}

func TestLoadTranslations(t *testing.T) {
	dir := t.TempDir()
	json := `{"login": "Sign In", "register": "Create Account"}`
	path := filepath.Join(dir, "en.json")
	if err := os.WriteFile(path, []byte(json), 0644); err != nil {
		t.Fatal(err)
	}

	tr := NewTranslator("en")
	if err := tr.LoadTranslations("en", path); err != nil {
		t.Fatal(err)
	}
	got := tr.Translate("en", "login")
	if got != "Sign In" {
		t.Errorf("expected 'Sign In', got %q", got)
	}
}

func TestLoadDirectory(t *testing.T) {
	dir := t.TempDir()
	enJSON := `{"login": "Login"}`
	zhJSON := `{"login": "登录"}`
	if err := os.WriteFile(filepath.Join(dir, "en.json"), []byte(enJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "zh-CN.json"), []byte(zhJSON), 0644); err != nil {
		t.Fatal(err)
	}
	// Non-JSON file should be skipped
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("skip"), 0644) //nolint:errcheck // test helper

	tr := NewTranslator("en")
	if err := tr.LoadDirectory(dir); err != nil {
		t.Fatal(err)
	}

	if tr.Translate("en", "login") != "Login" {
		t.Error("expected English login")
	}
	if tr.Translate("zh-CN", "login") != "登录" {
		t.Error("expected Chinese login")
	}
}

func TestResolveLocale(t *testing.T) {
	tests := []struct {
		header   string
		fallback string
		want     string
	}{
		{"en-US,en;q=0.9", "zh-CN", "en-US"},
		{"zh-CN,zh;q=0.9,en;q=0.8", "en", "zh-CN"},
		{"", "en", "en"},
		{"fr-FR", "en", "fr-FR"},
		{"ja", "en", "ja"},
	}
	for _, tt := range tests {
		got := ResolveLocale(tt.header, tt.fallback)
		if got != tt.want {
			t.Errorf("ResolveLocale(%q, %q) = %q, want %q", tt.header, tt.fallback, got, tt.want)
		}
	}
}

func TestFormat_NoParams(t *testing.T) {
	got := format("Hello World")
	if got != "Hello World" {
		t.Errorf("unexpected: %q", got)
	}
}
