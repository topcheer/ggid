package i18n

// LoadTranslationsInline is a test helper that loads translations from a Go map.
// This allows tests to avoid filesystem dependencies.
func (t *Translator) LoadTranslationsInline(locale string, msgs map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.translations[locale] = msgs
}
