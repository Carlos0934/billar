package mcphttp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/infra/logging"
)

type RequestAuthenticator interface {
	Authenticate(ctx context.Context, bearerToken string) (app.AuthenticatedIdentity, error)
}

type MCPHTTPAuthMiddleware struct {
	authenticator RequestAuthenticator
	challenge     app.OAuthChallengeDTO
	logger        *slog.Logger
}

func NewMCPHTTPAuthMiddleware(authenticator RequestAuthenticator, challenge app.OAuthChallengeDTO, logger *slog.Logger) MCPHTTPAuthMiddleware {
	return MCPHTTPAuthMiddleware{
		authenticator: authenticator,
		challenge:     challenge,
		logger:        logger,
	}
}

func (m MCPHTTPAuthMiddleware) Wrap(next http.Handler) http.Handler {
	if next == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.authenticator == nil {
			logging.Event(r.Context(), m.logger, slog.LevelError, "mcp.request_auth", "mcp-http", "error", slog.String("reason", "missing_request_authenticator"))
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}

		hasAuthorization := strings.TrimSpace(r.Header.Get("Authorization")) != ""
		looksBearer := strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Authorization"))), "bearer ")
		logging.Event(
			r.Context(),
			m.logger,
			slog.LevelDebug,
			"mcp.request_auth",
			"mcp-http",
			"received",
			slog.String("path", r.URL.Path),
			slog.Bool("has_authorization", hasAuthorization),
			slog.Bool("looks_bearer", looksBearer),
		)

		identity, err := m.authenticator.Authenticate(r.Context(), bearerTokenFromHeader(r.Header.Get("Authorization")))
		if err != nil {
			status := http.StatusUnauthorized
			if errors.Is(err, app.ErrUnauthorizedIdentity) || errors.Is(err, app.ErrEmailNotVerified) {
				status = http.StatusForbidden
			}
			setMCPAuthChallenge(w.Header(), m.challenge)
			logging.Event(r.Context(), m.logger, slog.LevelWarn, "mcp.request_auth", "mcp-http", "denied", slog.String("reason", classifyRequestAuthReason(err)))
			http.Error(w, http.StatusText(status), status)
			return
		}

		next.ServeHTTP(w, r.WithContext(app.WithAuthenticatedIdentity(r.Context(), identity)))
	})
}

func bearerTokenFromHeader(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	tokenType, token, found := strings.Cut(value, " ")
	if !found || !strings.EqualFold(strings.TrimSpace(tokenType), "Bearer") {
		return ""
	}

	return strings.TrimSpace(token)
}

func setMCPAuthChallenge(headers http.Header, challenge app.OAuthChallengeDTO) {
	if headers == nil {
		return
	}

	params := []string{`Bearer realm="billar-mcp"`}
	if resource := strings.TrimSpace(challenge.ResourceURI); resource != "" {
		params = append(params, fmt.Sprintf(`resource_metadata=%q`, resource+"/.well-known/oauth-protected-resource"))
	}
	headers.Set("WWW-Authenticate", strings.Join(params, ", "))
	if resource := strings.TrimSpace(challenge.ResourceURI); resource != "" {
		headers.Set("MCP-Resource", resource)
	}
}

func classifyRequestAuthReason(err error) string {
	switch {
	case err == nil:
		return ""
	case strings.Contains(strings.ToLower(err.Error()), "tokeninfo_issuer_mismatch"):
		return "tokeninfo_issuer_mismatch"
	case strings.Contains(strings.ToLower(err.Error()), "tokeninfo_audience_mismatch"):
		return "tokeninfo_audience_mismatch"
	case strings.Contains(strings.ToLower(err.Error()), "tokeninfo_expired"):
		return "tokeninfo_expired"
	case strings.Contains(strings.ToLower(err.Error()), "tokeninfo_rejected"):
		return "tokeninfo_rejected"
	case strings.Contains(strings.ToLower(err.Error()), "userinfo_unauthorized"):
		return "userinfo_unauthorized"
	case strings.Contains(strings.ToLower(err.Error()), "userinfo_missing_subject"):
		return "userinfo_missing_subject"
	case errors.Is(err, app.ErrMissingBearerToken):
		return "missing_bearer_token"
	case errors.Is(err, app.ErrInvalidBearerToken):
		return "invalid_bearer_token"
	case errors.Is(err, app.ErrUnauthorizedIdentity):
		return "unauthorized_identity"
	case errors.Is(err, app.ErrEmailNotVerified):
		return "email_not_verified"
	default:
		return "internal_error"
	}
}
