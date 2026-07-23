package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type googleConnector struct {
	config *oauth2.Config
}

// NewGoogleConnector creates a Google OAuth2 social connector.
func NewGoogleConnector(clientID, clientSecret string) Connector {
	return &googleConnector{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
	}
}

func (g *googleConnector) ID() string          { return "google" }
func (g *googleConnector) DisplayName() string { return "Google" }

func (g *googleConnector) GetAuthURL(_ context.Context, state, redirectURI string) (string, error) {
	g.config.RedirectURL = redirectURI
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOnline), nil
}

func (g *googleConnector) HandleCallback(ctx context.Context, code, _, redirectURI string) (*UserInfo, error) {
	g.config.RedirectURL = redirectURI
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google token exchange: %w", err)
	}

	client := g.config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("google userinfo: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read google response: %w", err)
	}

	var claims struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Picture  string `json:"picture"`
		Verified bool   `json:"verified_email"`
	}
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, fmt.Errorf("parse google claims: %w", err)
	}

	rawClaims := make(map[string]any)
	_ = json.Unmarshal(body, &rawClaims)

	return &UserInfo{
		Provider:      "google",
		ExternalID:     claims.ID,
		Email:         claims.Email,
		Name:          claims.Name,
		AvatarURL:      claims.Picture,
		EmailVerified:  claims.Verified,
		RawClaims:      rawClaims,
	}, nil
}

var _ = http.StatusOK
