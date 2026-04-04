package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Carlos0934/billar/internal/app"
)

const (
	googleTokenInfoURL = "https://oauth2.googleapis.com/tokeninfo"
	googleUserInfoURL  = "https://openidconnect.googleapis.com/v1/userinfo"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type GoogleAccessTokenAuthenticator struct {
	httpClient       HTTPClient
	tokenInfoURL     string
	userInfoURL      string
	expectedAudience string
	allowedIssuers   map[string]struct{}
}

func NewGoogleAccessTokenAuthenticator(issuerURL, clientID string) (*GoogleAccessTokenAuthenticator, error) {
	issuerURL = strings.TrimSpace(issuerURL)
	clientID = strings.TrimSpace(clientID)
	if issuerURL == "" {
		return nil, errors.New("issuer url is required")
	}
	if clientID == "" {
		return nil, errors.New("client id is required")
	}

	return &GoogleAccessTokenAuthenticator{
		httpClient:       &http.Client{Timeout: 10 * time.Second},
		tokenInfoURL:     googleTokenInfoURL,
		userInfoURL:      googleUserInfoURL,
		expectedAudience: clientID,
		allowedIssuers:   allowedIssuerSet(issuerURL),
	}, nil
}

func (g *GoogleAccessTokenAuthenticator) AuthenticateAccessToken(ctx context.Context, rawToken string) (app.AuthenticatedIdentity, error) {
	if g == nil || g.httpClient == nil {
		return app.AuthenticatedIdentity{}, errors.New("google access token authenticator is not configured")
	}

	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return app.AuthenticatedIdentity{}, app.ErrAccessTokenRejected
	}

	metadata, err := g.fetchTokenInfo(ctx, rawToken)
	if err != nil {
		return app.AuthenticatedIdentity{}, err
	}
	issuer := normalizeIssuer(metadata.Issuer)
	if issuer != "" {
		if _, ok := g.allowedIssuers[issuer]; !ok {
			return app.AuthenticatedIdentity{}, fmt.Errorf("tokeninfo_issuer_mismatch: %w", app.ErrAccessTokenRejected)
		}
	}
	if !tokenAudienceMatches(metadata.Audience, g.expectedAudience) {
		return app.AuthenticatedIdentity{}, fmt.Errorf("tokeninfo_audience_mismatch: %w", app.ErrAccessTokenRejected)
	}
	if metadata.ExpiresIn <= 0 {
		return app.AuthenticatedIdentity{}, fmt.Errorf("tokeninfo_expired: %w", app.ErrAccessTokenRejected)
	}

	user, err := g.fetchUserInfo(ctx, rawToken)
	if err != nil {
		return app.AuthenticatedIdentity{}, err
	}
	if user.Subject == "" {
		return app.AuthenticatedIdentity{}, fmt.Errorf("userinfo_missing_subject: %w", app.ErrAccessTokenRejected)
	}

	return app.AuthenticatedIdentity{
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Subject:       user.Subject,
		Issuer:        issuer,
	}, nil
}

type googleTokenInfoResponse struct {
	Audience  string `json:"aud"`
	ExpiresIn int64  `json:"expires_in,string"`
	Issuer    string `json:"iss"`
	Scope     string `json:"scope"`
	IssuedTo  string `json:"issued_to"`
	Error     string `json:"error_description"`
}

type googleUserInfoResponse struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func (g *GoogleAccessTokenAuthenticator) fetchTokenInfo(ctx context.Context, rawToken string) (googleTokenInfoResponse, error) {
	endpoint, err := url.Parse(g.tokenInfoURL)
	if err != nil {
		return googleTokenInfoResponse{}, fmt.Errorf("parse token info url: %w", err)
	}
	query := endpoint.Query()
	query.Set("access_token", rawToken)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return googleTokenInfoResponse{}, fmt.Errorf("build token info request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return googleTokenInfoResponse{}, fmt.Errorf("fetch token info: %w", err)
	}
	defer resp.Body.Close()

	var payload googleTokenInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return googleTokenInfoResponse{}, fmt.Errorf("decode token info response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if payload.Error != "" {
			return googleTokenInfoResponse{}, fmt.Errorf("fetch token info: tokeninfo_rejected: %w", app.ErrAccessTokenRejected)
		}
		return googleTokenInfoResponse{}, fmt.Errorf("fetch token info: unexpected status %d", resp.StatusCode)
	}

	payload.Issuer = strings.TrimSpace(payload.Issuer)
	payload.Audience = strings.TrimSpace(firstNonEmpty(payload.Audience, payload.IssuedTo))
	return payload, nil
}

func (g *GoogleAccessTokenAuthenticator) fetchUserInfo(ctx context.Context, rawToken string) (googleUserInfoResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.userInfoURL, nil)
	if err != nil {
		return googleUserInfoResponse{}, fmt.Errorf("build user info request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+rawToken)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return googleUserInfoResponse{}, fmt.Errorf("fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return googleUserInfoResponse{}, fmt.Errorf("fetch user info: userinfo_unauthorized: %w", app.ErrAccessTokenRejected)
	}
	if resp.StatusCode != http.StatusOK {
		return googleUserInfoResponse{}, fmt.Errorf("fetch user info: unexpected status %d", resp.StatusCode)
	}

	var payload googleUserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return googleUserInfoResponse{}, fmt.Errorf("decode user info response: %w", err)
	}
	payload.Subject = strings.TrimSpace(payload.Subject)
	payload.Email = strings.TrimSpace(payload.Email)
	return payload, nil
}

func tokenAudienceMatches(got, expected string) bool {
	got = strings.TrimSpace(got)
	expected = strings.TrimSpace(expected)
	if got == "" || expected == "" {
		return false
	}
	for _, candidate := range strings.FieldsFunc(got, func(r rune) bool {
		return r == ' ' || r == ','
	}) {
		if strings.TrimSpace(candidate) == expected {
			return true
		}
	}
	return false
}

func normalizeIssuer(issuer string) string {
	issuer = strings.TrimSpace(issuer)
	if issuer == "accounts.google.com" {
		return "https://accounts.google.com"
	}
	return strings.TrimRight(issuer, "/")
}

func allowedIssuerSet(issuer string) map[string]struct{} {
	issuer = normalizeIssuer(issuer)
	allowed := map[string]struct{}{}
	if issuer == "" {
		return allowed
	}
	allowed[issuer] = struct{}{}
	if strings.HasPrefix(issuer, "https://") {
		allowed[strings.TrimPrefix(issuer, "https://")] = struct{}{}
	}
	return allowed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
