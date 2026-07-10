package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// GitLab self-hosted and gitlab.com are supported via the BaseURL field.
type gitlabConnector struct {
	config  *oauth2.Config
	baseURL string
}

// NewGitLabConnector creates a GitLab OAuth2 social connector.
// baseURL should be "https://gitlab.com" for SaaS or the instance URL for self-hosted.
func NewGitLabConnector(clientID, clientSecret, baseURL string) Connector {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	return &gitlabConnector{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes: []string{
				"read_user",
				"openid",
				"profile",
				"email",
			},
			Endpoint: oauth2.Endpoint{
				AuthURL:  baseURL + "/oauth/authorize",
				TokenURL: baseURL + "/oauth/token",
			},
		},
		baseURL: baseURL,
	}
}

func (g *gitlabConnector) ID() string          { return "gitlab" }
func (g *gitlabConnector) DisplayName() string { return "GitLab" }

func (g *gitlabConnector) GetAuthURL(_ context.Context, state, redirectURI string) (string, error) {
	g.config.RedirectURL = redirectURI
	return g.config.AuthCodeURL(state), nil
}

func (g *gitlabConnector) HandleCallback(ctx context.Context, code, _, redirectURI string) (*UserInfo, error) {
	g.config.RedirectURL = redirectURI
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("gitlab token exchange: %w", err)
	}

	client := g.config.Client(ctx, token)

	// GitLab user API.
	apiURL := g.baseURL + "/api/v4/user"
	req, _ := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab user API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read gitlab response: %w", err)
	}

	var claims struct {
		ID        int    `json:"id"`
		Username  string `json:"username"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, fmt.Errorf("parse gitlab claims: %w", err)
	}

	rawClaims := make(map[string]any)
	_ = json.Unmarshal(body, &rawClaims)

	return &UserInfo{
		Provider:   "gitlab",
		ExternalID: fmt.Sprintf("%d", claims.ID),
		Email:      claims.Email,
		Name:       claims.Name,
		AvatarURL:  claims.AvatarURL,
		RawClaims:  rawClaims,
	}, nil
}
