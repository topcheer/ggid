package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/oauth2"
)

type linkedinConnector struct {
	config *oauth2.Config
}

// NewLinkedInConnector creates a LinkedIn OAuth2 connector.
func NewLinkedInConnector(clientID, clientSecret string) Connector {
	return &linkedinConnector{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.linkedin.com/oauth/v2/authorization",
				TokenURL: "https://www.linkedin.com/oauth/v2/accessToken",
			},
		},
	}
}

func (c *linkedinConnector) ID() string         { return "linkedin" }
func (c *linkedinConnector) DisplayName() string { return "LinkedIn" }

func (c *linkedinConnector) GetAuthURL(_ context.Context, state, redirectURI string) (string, error) {
	c.config.RedirectURL = redirectURI
	return c.config.AuthCodeURL(state), nil
}

func (c *linkedinConnector) HandleCallback(ctx context.Context, code, state, redirectURI string) (*UserInfo, error) {
	c.config.RedirectURL = redirectURI
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("linkedin token exchange: %w", err)
	}

	client := c.config.Client(ctx, token)
	resp, err := client.Get("https://api.linkedin.com/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("linkedin user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read linkedin response: %w", err)
	}

	var profile struct {
		Sub         string `json:"sub"`
		Name        string `json:"name"`
		GivenName   string `json:"given_name"`
		FamilyName  string `json:"family_name"`
		Email       string `json:"email"`
		Picture     string `json:"picture"`
	}
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("parse linkedin profile: %w", err)
	}

	return &UserInfo{
		Provider:   "linkedin",
		ExternalID: profile.Sub,
		Email:      profile.Email,
		Name:       profile.Name,
		AvatarURL:  profile.Picture,
	}, nil
}
