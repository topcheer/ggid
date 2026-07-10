package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// Azure AD v2.0 common endpoint (works for any Azure AD tenant + personal accounts).
var azureADCommon = oauth2.Endpoint{
	AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
	TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
}

type microsoftConnector struct {
	config *oauth2.Config
}

// NewMicrosoftConnector creates a Microsoft (Azure AD v2.0) OAuth2 social connector.
// Uses the common endpoint which works for any Azure AD tenant or personal Microsoft accounts.
func NewMicrosoftConnector(clientID, clientSecret string) Connector {
	return &microsoftConnector{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes: []string{
				"openid",
				"profile",
				"email",
				"User.Read",
			},
			Endpoint: azureADCommon,
		},
	}
}

func (m *microsoftConnector) ID() string          { return "microsoft" }
func (m *microsoftConnector) DisplayName() string { return "Microsoft" }

func (m *microsoftConnector) GetAuthURL(_ context.Context, state, redirectURI string) (string, error) {
	m.config.RedirectURL = redirectURI
	return m.config.AuthCodeURL(state), nil
}

func (m *microsoftConnector) HandleCallback(ctx context.Context, code, _, redirectURI string) (*UserInfo, error) {
	m.config.RedirectURL = redirectURI
	token, err := m.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("microsoft token exchange: %w", err)
	}

	// Microsoft Graph API user profile endpoint.
	client := m.config.Client(ctx, token)
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/me", nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("microsoft graph API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read microsoft response: %w", err)
	}

	var claims struct {
		ID           string `json:"id"`
		DisplayName  string `json:"displayName"`
		GivenName    string `json:"givenName"`
		Surname      string `json:"surname"`
		Email        string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"`
	}
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, fmt.Errorf("parse microsoft claims: %w", err)
	}

	rawClaims := make(map[string]any)
	_ = json.Unmarshal(body, &rawClaims)

	// Microsoft may put email in "mail" or "userPrincipalName".
	email := claims.Email
	if email == "" {
		email = claims.UserPrincipalName
	}

	name := claims.DisplayName
	if name == "" {
		name = claims.GivenName + " " + claims.Surname
	}

	return &UserInfo{
		Provider:   "microsoft",
		ExternalID: claims.ID,
		Email:      email,
		Name:       name,
		RawClaims:  rawClaims,
	}, nil
}
