package server

import (
	"net/http"
	"strings"
	"sync"
)

// scopeDesc holds localized descriptions for a scope.
type scopeDesc struct {
	Name        string            `json:"name"`
	Descriptions map[string]string `json:"descriptions"`
}

var (
	scopeDescMu sync.RWMutex
	scopeDescStore = map[string]*scopeDesc{
		"openid": {Name: "openid", Descriptions: map[string]string{
			"en": "Sign in using OpenID Connect", "zh": "使用 OpenID Connect 登录",
			"ja": "OpenID Connect でサインイン", "de": "Mit OpenID Connect anmelden", "fr": "Se connecter avec OpenID Connect",
		}},
		"profile": {Name: "profile", Descriptions: map[string]string{
			"en": "Access your profile information", "zh": "访问您的个人资料",
			"ja": "プロファイル情報にアクセス", "de": "Auf Profilinformationen zugreifen", "fr": "Accéder aux informations de profil",
		}},
		"email": {Name: "email", Descriptions: map[string]string{
			"en": "Access your email address", "zh": "访问您的邮箱地址",
			"ja": "メールアドレスにアクセス", "de": "Auf E-Mail-Adresse zugreifen", "fr": "Accéder à l'adresse e-mail",
		}},
		"read": {Name: "read", Descriptions: map[string]string{
			"en": "Read access to resources", "zh": "读取资源权限",
			"ja": "リソースの読み取り", "de": "Lesezugriff auf Ressourcen", "fr": "Accès en lecture aux ressources",
		}},
		"write": {Name: "write", Descriptions: map[string]string{
			"en": "Write access to resources", "zh": "写入资源权限",
			"ja": "リソースへの書き込み", "de": "Schreibzugriff auf Ressourcen", "fr": "Accès en écriture aux ressources",
		}},
		"admin": {Name: "admin", Descriptions: map[string]string{
			"en": "Administrative access", "zh": "管理员权限",
			"ja": "管理者アクセス", "de": "Administrativer Zugriff", "fr": "Accès administrateur",
		}},
	}
)

// GET /api/v1/oauth/scopes?lang=zh — list scopes with localized descriptions.
func handleScopesI18n(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "en"
	}
	supported := []string{"en", "zh", "ja", "de", "fr"}
	if !contains(supported, lang) {
		lang = "en"
	}

	// Include custom scopes too
	customScopes.mu.RLock()
	for name := range customScopes.scopes {
		if _, exists := scopeDescStore[name]; !exists {
			scopeDescStore[name] = &scopeDesc{Name: name, Descriptions: map[string]string{
				"en": name, "zh": name, "ja": name, "de": name, "fr": name,
			}}
		}
	}
	customScopes.mu.RUnlock()

	scopeDescMu.RLock()
	result := []map[string]any{}
	for _, sc := range scopeDescStore {
		desc := sc.Descriptions[lang]
		if desc == "" {
			desc = sc.Descriptions["en"]
		}
		result = append(result, map[string]any{
			"name":        sc.Name,
			"description": desc,
			"lang":        lang,
		})
	}
	scopeDescMu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"scopes": result,
		"count":  len(result),
		"lang":   lang,
	})
}

func contains(s []string, v string) bool {
	for _, item := range s {
		if strings.EqualFold(item, v) {
			return true
		}
	}
	return false
}
