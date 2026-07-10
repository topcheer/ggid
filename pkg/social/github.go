package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type githubConnector struct {
	config *oauth2.Config
}

// NewGitHubConnector creates a GitHub OAuth2 social connector.
func NewGitHubConnector(clientID, clientSecret string) Connector {
	return &githubConnector{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes: []string{
				"read:user",
				"user:email",
			},
			Endpoint: github.Endpoint,
		},
	}
}

func (g *githubConnector) ID() string         { return "github" }
func (g *githubConnector) DisplayName() string { return "GitHub" }

func (g *githubConnector) GetAuthURL(_ context.Context, state, redirectURI string) (string, error) {
	g.config.RedirectURL = redirectURI
	return g.config.AuthCodeURL(state), nil
}

func (g *githubConnector) HandleCallback(ctx context.Context, code, _, redirectURI string) (*UserInfo, error) {
	g.config.RedirectURL = redirectURI
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("github token exchange: %w", err)
	}

	client := g.config.Client(ctx, token)

	// Get user profile
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github user API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read github response: %w", err)
	}

	var claims struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, fmt.Errorf("parse github claims: %w", err)
	}

	rawClaims := make(map[string]any)
	_ = json.Unmarshal(body, &rawClaims)

	// If email is empty, fetch from /user/emails
	email := claims.Email
	if email == "" {
		email = g.fetchPrimaryEmail(ctx, client)
	}

	return &UserInfo{
		Provider:   "github",
		ExternalID: fmt.Sprintf("%d", claims.ID),
		Email:      email,
		Name:       claims.Name,
		AvatarURL:  claims.AvatarURL,
		RawClaims:  rawClaims,
	}, nil
}

func (g *githubConnector) fetchPrimaryEmail(ctx context.Context, client *http.Client) string {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return ""
	}
	for _, e := range emails {
		if e.Primary {
			return e.Email
		}
	}
	if len(emails) > 0 {
		return emails[0].Email
	}
	return ""
}
