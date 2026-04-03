package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var ErrMissingIDToken = errors.New("oauth2 token response missing id_token")

type GoogleOIDC struct {
	oauthConfig *oauth2.Config
	verifier    *oidc.IDTokenVerifier
}

func NewGoogleOIDC(ctx context.Context, issuerURL, clientID, clientSecret, redirectURL string) (*GoogleOIDC, error) {
	issuerURL = strings.TrimSpace(issuerURL)
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	redirectURL = strings.TrimSpace(redirectURL)
	if issuerURL == "" {
		return nil, errors.New("issuer url is required")
	}
	if clientID == "" {
		return nil, errors.New("client id is required")
	}
	if clientSecret == "" {
		return nil, errors.New("client secret is required")
	}
	if redirectURL == "" {
		return nil, errors.New("redirect url is required")
	}

	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}

	return &GoogleOIDC{
		oauthConfig: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  redirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}, nil
}

func (g *GoogleOIDC) AuthorizationURL(ctx context.Context, state string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if g == nil || g.oauthConfig == nil {
		return "", errors.New("google oidc is not configured")
	}
	return g.oauthConfig.AuthCodeURL(strings.TrimSpace(state)), nil
}

func (g *GoogleOIDC) ExchangeCodeForIDToken(ctx context.Context, code string) (string, error) {
	if g == nil || g.oauthConfig == nil {
		return "", errors.New("google oidc is not configured")
	}

	token, err := g.oauthConfig.Exchange(ctx, strings.TrimSpace(code))
	if err != nil {
		return "", fmt.Errorf("exchange authorization code: %w", err)
	}

	rawToken, ok := token.Extra("id_token").(string)
	if !ok || strings.TrimSpace(rawToken) == "" {
		return "", ErrMissingIDToken
	}

	return rawToken, nil
}

func (g *GoogleOIDC) VerifyIDToken(ctx context.Context, rawToken string) (app.AuthenticatedIdentity, error) {
	if g == nil || g.verifier == nil {
		return app.AuthenticatedIdentity{}, errors.New("google oidc is not configured")
	}

	verified, err := g.verifier.Verify(ctx, strings.TrimSpace(rawToken))
	if err != nil {
		return app.AuthenticatedIdentity{}, fmt.Errorf("verify oidc token: %w", err)
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := verified.Claims(&claims); err != nil {
		return app.AuthenticatedIdentity{}, fmt.Errorf("decode oidc claims: %w", err)
	}

	return app.AuthenticatedIdentity{
		Email:         strings.TrimSpace(claims.Email),
		EmailVerified: claims.EmailVerified,
		Subject:       verified.Subject,
		Issuer:        verified.Issuer,
	}, nil
}
