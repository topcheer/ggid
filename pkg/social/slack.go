package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/oauth2"
)

type slackConnector struct {
	config *oauth2.Config
}

// NewSlackConnector creates a Slack OAuth2 connector.
func NewSlackConnector(clientID, clientSecret string) Connector {
	return &slackConnector{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       []string{"identity.basic", "identity.email", "identity.avatar"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://slack.com/oauth/v2/authorize",
				TokenURL: "https://slack.com/api/oauth.v2.access",
			},
		},
	}
}

func (c *slackConnector) ID() string         { return "slack" }
func (c *slackConnector) DisplayName() string { return "Slack" }

func (c *slackConnector) GetAuthURL(_ context.Context, state, redirectURI string) (string, error) {
	c.config.RedirectURL = redirectURI
	return c.config.AuthCodeURL(state), nil
}

func (c *slackConnector) HandleCallback(ctx context.Context, code, state, redirectURI string) (*UserInfo, error) {
	c.config.RedirectURL = redirectURI
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("slack token exchange: %w", err)
	}

	client := c.config.Client(ctx, token)
	resp, err := client.Get("https://slack.com/api/users.identity")
	if err != nil {
		return nil, fmt.Errorf("slack user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read slack response: %w", err)
	}

	var profile struct {
		OK   bool   `json:"ok"`
		User struct {
			Name  string `json:"name"`
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
		UserStruct struct {
			Image512 string `json:"image_512"`
		} `json:"user"`
		Team struct {
			Name string `json:"name"`
		} `json:"team"`
	}
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("parse slack profile: %w", err)
	}
	if !profile.OK {
		return nil, fmt.Errorf("slack API returned not ok")
	}

	return &UserInfo{
		Provider:   "slack",
		ExternalID: profile.User.ID,
		Email:      profile.User.Email,
		Name:       profile.User.Name,
		AvatarURL:  "",
	}, nil
}
