package social

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

// Apple uses a slightly non-standard OAuth2 flow: the client_secret is a JWT
// signed with the Apple Sign-In private key, not a static string.
// This connector accepts a pre-generated client_secret JWT (valid for up to 6 months).
// For production use, the secret should be refreshed periodically.

type appleConnector struct {
	config       *oauth2.Config
	clientSecretJWT string // JWT-signed client secret
}

// NewAppleConnector creates an Apple Sign-In OAuth2 social connector.
// clientSecretJWT is a JWT signed with the team's private key (Apple requires this).
// Use GenerateAppleClientSecret() to create it from a .p8 key file.
func NewAppleConnector(clientID, clientSecretJWT string) Connector {
	return &appleConnector{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecretJWT,
			Scopes:       []string{"name", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://appleid.apple.com/auth/authorize",
				TokenURL: "https://appleid.apple.com/auth/token",
			},
		},
		clientSecretJWT: clientSecretJWT,
	}
}

func (a *appleConnector) ID() string          { return "apple" }
func (a *appleConnector) DisplayName() string { return "Apple" }

func (a *appleConnector) GetAuthURL(_ context.Context, state, redirectURI string) (string, error) {
	a.config.RedirectURL = redirectURI
	// Apple requires response_mode=form_post for name/email scopes.
	return a.config.AuthCodeURL(state, oauth2.SetAuthURLParam("response_mode", "form_post")), nil
}

func (a *appleConnector) HandleCallback(ctx context.Context, code, _, redirectURI string) (*UserInfo, error) {
	a.config.RedirectURL = redirectURI
	token, err := a.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("apple token exchange: %w", err)
	}

	// Apple returns user info in the id_token JWT.
	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in Apple response")
	}

	claims, err := parseJWTClaims(idToken)
	if err != nil {
		return nil, fmt.Errorf("parse apple id_token: %w", err)
	}

	externalID, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	emailVerified, _ := claims["email_verified"].(string)

	name := ""
	// Apple only returns name on first authorization — it comes in the form_post user payload,
	// not in the id_token. If available in RawClaims, extract it.
	if firstName, ok := claims["firstName"].(string); ok {
		name = firstName
	}
	if lastName, ok := claims["lastName"].(string); ok {
		name = strings.TrimSpace(name + " " + lastName)
	}

	_ = emailVerified // Apple sends "true"/"false" as string

	return &UserInfo{
		Provider:   "apple",
		ExternalID: externalID,
		Email:      email,
		Name:       name,
		RawClaims:  claims,
	}, nil
}

// AppleFormCallback handles Apple's form_post response mode.
// Apple POSTs the code and optional user JSON to the redirect URI.
// This extracts the authorization code from the form body.
func AppleFormCallback(r *http.Request) (code, state string, userJSON string) {
	_ = r.ParseForm()
	code = r.FormValue("code")
	state = r.FormValue("state")
	userJSON = r.FormValue("user")
	return
}

// ParseAppleUserJSON parses the "user" object from Apple's first-authorization form_post.
type appleUser struct {
	Name struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	} `json:"name"`
	Email string `json:"email"`
}

// ParseAppleUser extracts name and email from Apple's user form payload.
func ParseAppleUser(userJSON string) (name, email string) {
	if userJSON == "" {
		return "", ""
	}
	var u appleUser
	if err := json.Unmarshal([]byte(userJSON), &u); err != nil {
		return "", ""
	}
	name = strings.TrimSpace(u.Name.FirstName + " " + u.Name.LastName)
	email = u.Email
	return name, email
}

var _ = ParseAppleUser // keep ParseAppleUser exported
