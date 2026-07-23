package social

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
)

// appleConnector implements Apple Sign-In using the OAuth2 flow.
type appleConnector struct {
	config       *oauth2.Config
	clientSecret string // JWT-based, generated from Apple Developer account
}

// NewAppleConnector creates an Apple Sign-In connector.
// The clientSecret is a JWT signed with the Apple Developer private key.
func NewAppleConnector(clientID, clientSecret string) Connector {
	return &appleConnector{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       []string{"name", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://appleid.apple.com/auth/authorize",
				TokenURL: "https://appleid.apple.com/auth/token",
			},
		},
		clientSecret: clientSecret,
	}
}

// ParseAppleUser parses the user JSON returned by Apple during the first login.
// Apple sends: {"name":{"firstName":"John","lastName":"Doe"},"email":"john@example.com"}
// This is only sent on the FIRST authorization; subsequent logins only have the ID token.
func ParseAppleUser(userJSON string) (name, email string) {
	if userJSON == "" {
		return "", ""
	}
	var data struct {
		Name struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal([]byte(userJSON), &data); err != nil {
		return "", ""
	}
	name = strings.TrimSpace(data.Name.FirstName + " " + data.Name.LastName)
	return name, data.Email
}

func (c *appleConnector) ID() string          { return "apple" }
func (c *appleConnector) DisplayName() string { return "Apple" }

func (c *appleConnector) GetAuthURL(_ context.Context, state, redirectURI string) (string, error) {
	c.config.RedirectURL = redirectURI
	// Apple requires response_mode=form_post for name/email scopes.
	return c.config.AuthCodeURL(state, oauth2.SetAuthURLParam("response_mode", "form_post")), nil
}

func (c *appleConnector) HandleCallback(ctx context.Context, code, state, redirectURI string) (*UserInfo, error) {
	c.config.RedirectURL = redirectURI
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("apple token exchange: %w", err)
	}

	// Apple returns user info in the ID token.
	idToken, ok := token.Extra("id_token").(string)
	if !ok || idToken == "" {
		return nil, fmt.Errorf("apple: no id_token in response")
	}

	profile, err := decodeAppleIDToken(idToken)
	if err != nil {
		return nil, fmt.Errorf("apple: decode id_token: %w", err)
	}

	return &UserInfo{
		Provider:      "apple",
		ExternalID:     profile.Sub,
		Email:         profile.Email,
		Name:          profile.Name,
		EmailVerified:  profile.EmailVerified == "true",
	}, nil
}

// appleProfile represents claims in an Apple ID token.
type appleProfile struct {
	Sub            string `json:"sub"`
	Email          string `json:"email"`
	EmailVerified  string `json:"email_verified"`
	IsPrivateEmail string `json:"is_private_email"`
	Name           string `json:"name"`
}

// decodeAppleIDToken extracts user info from an Apple ID token JWT.
func decodeAppleIDToken(idToken string) (*appleProfile, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var profile appleProfile
	if err := json.Unmarshal(payload, &profile); err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}

	return &profile, nil
}
