// Package i18n provides internationalization support for GGID hosted pages and emails.
// Supports loading translations from JSON files and rendering in the user's locale.
package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Translator manages translations for multiple locales.
type Translator struct {
	mu         sync.RWMutex
	translations map[string]map[string]string // locale -> key -> translation
	defaultLocale string
}

// NewTranslator creates a new translator with the given default locale.
func NewTranslator(defaultLocale string) *Translator {
	return &Translator{
		translations:  make(map[string]map[string]string),
		defaultLocale: defaultLocale,
	}
}

// LoadTranslations loads translations from a JSON file for a specific locale.
// The JSON should be a flat map of key -> translation string.
func (t *Translator) LoadTranslations(locale, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("i18n: failed to read %s: %w", path, err)
	}

	var msgs map[string]string
	if err := json.Unmarshal(data, &msgs); err != nil {
		return fmt.Errorf("i18n: failed to parse %s: %w", path, err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.translations[locale] = msgs
	return nil
}

// LoadDirectory loads all locale JSON files from a directory.
// Files must be named like "en.json", "zh-CN.json", etc.
func (t *Translator) LoadDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("i18n: failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		locale := strings.TrimSuffix(entry.Name(), ".json")
		if err := t.LoadTranslations(locale, filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

// Translate returns the translation for the given key in the specified locale.
// Falls back to default locale, then to the key itself.
func (t *Translator) Translate(locale, key string, params ...interface{}) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Try requested locale
	if msgs, ok := t.translations[locale]; ok {
		if msg, ok := msgs[key]; ok {
			return format(msg, params...)
		}
	}

	// Fall back to default locale
	if locale != t.defaultLocale {
		if msgs, ok := t.translations[t.defaultLocale]; ok {
			if msg, ok := msgs[key]; ok {
				return format(msg, params...)
			}
		}
	}

	// Return the key itself as last resort
	return key
}

// TranslateMap returns all translations for a given locale.
func (t *Translator) TranslateMap(locale string) map[string]string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if msgs, ok := t.translations[locale]; ok {
		result := make(map[string]string, len(msgs))
		for k, v := range msgs {
			result[k] = v
		}
		return result
	}

	// Fallback to default
	if msgs, ok := t.translations[t.defaultLocale]; ok {
		result := make(map[string]string, len(msgs))
		for k, v := range msgs {
			result[k] = v
		}
		return result
	}

	return map[string]string{}
}

// SupportedLocales returns a list of loaded locale codes.
func (t *Translator) SupportedLocales() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	locales := make([]string, 0, len(t.translations))
	for locale := range t.translations {
		locales = append(locales, locale)
	}
	return locales
}

// ResolveLocale extracts the best matching locale from an Accept-Language header.
func ResolveLocale(acceptLanguage, defaultLocale string) string {
	if acceptLanguage == "" {
		return defaultLocale
	}

	// Parse "en-US,en;q=0.9,zh-CN;q=0.8" format
	parts := strings.Split(acceptLanguage, ",")
	for _, part := range parts {
		lang := strings.TrimSpace(strings.Split(part, ";")[0])
		// Normalize: "en-US" → "en-US", "en" → "en"
		if lang != "" {
			return lang
		}
	}
	return defaultLocale
}

// format applies sprintf-style formatting if params are provided.
func format(msg string, params ...interface{}) string {
	if len(params) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, params...)
}
