package social

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// oidcConnector implements the Connector interface for any OIDC-compliant IdP.
type oidcConnector struct {
	id          string
	name        string
	config      *oauth2.Config
	userInfoURL string // optional: if non-empty, call userinfo endpoint after token exchange
}

// NewGenericOIDCConnector creates a connector for any OIDC provider.
func NewGenericOIDCConnector(id, name, clientID, clientSecret, authURL, tokenURL, userInfoURL string, scopes []string) Connector {
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}
	return &oidcConnector{
		id:   id,
		name: name,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
		},
		userInfoURL: userInfoURL,
	}
}

func (o *oidcConnector) ID() string          { return o.id }
func (o *oidcConnector) DisplayName() string { return o.name }

func (o *oidcConnector) GetAuthURL(_ context.Context, state, redirectURI string) (string, error) {
	o.config.RedirectURL = redirectURI
	return o.config.AuthCodeURL(state), nil
}

func (o *oidcConnector) HandleCallback(ctx context.Context, code, _, redirectURI string) (*UserInfo, error) {
	o.config.RedirectURL = redirectURI
	token, err := o.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oidc token exchange: %w", err)
	}

	// Extract user info from the id_token
	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in OIDC response")
	}

	claims, err := parseJWTClaims(idToken)
	if err != nil {
		return nil, fmt.Errorf("parse oidc id_token: %w", err)
	}

	externalID, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)

	return &UserInfo{
		Provider:   o.id,
		ExternalID: externalID,
		Email:      email,
		Name:       name,
		AvatarURL:  picture,
		RawClaims:  claims,
	}, nil
}
