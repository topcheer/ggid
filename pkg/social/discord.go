package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/oauth2"
)

type discordConnector struct {
	config *oauth2.Config
}

// NewDiscordConnector creates a Discord OAuth2 connector.
func NewDiscordConnector(clientID, clientSecret string) Connector {
	return &discordConnector{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		Scopes:       []string{"identify", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discord.com/api/oauth2/authorize",
				TokenURL: "https://discord.com/api/oauth2/token",
			},
		},
	}
}

func (c *discordConnector) ID() string         { return "discord" }
func (c *discordConnector) DisplayName() string { return "Discord" }

func (c *discordConnector) GetAuthURL(_ context.Context, state, redirectURI string) (string, error) {
	c.config.RedirectURL = redirectURI
	return c.config.AuthCodeURL(state), nil
}

func (c *discordConnector) HandleCallback(ctx context.Context, code, state, redirectURI string) (*UserInfo, error) {
	c.config.RedirectURL = redirectURI
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("discord token exchange: %w", err)
	}

	client := c.config.Client(ctx, token)
	resp, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		return nil, fmt.Errorf("discord user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read discord response: %w", err)
	}

	var profile struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Avatar   string `json:"avatar"`
	}
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("parse discord profile: %w", err)
	}

	avatarURL := ""
	if profile.Avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", profile.ID, profile.Avatar)
	}

	return &UserInfo{
		Provider:   "discord",
		ExternalID: profile.ID,
		Email:      profile.Email,
		Name:       profile.Username,
		AvatarURL:  avatarURL,
	}, nil
}
