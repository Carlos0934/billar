package mcphttp

import (
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"net/http"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/infra/logging"
)

// localIdentity is the fixed identity injected after successful API key auth.
var localIdentity = app.AuthenticatedIdentity{
	Email:         "mcp@local",
	EmailVerified: true,
	Subject:       "mcp-api-key",
	Issuer:        "billar://local",
}

// APIKeyAuthMiddleware validates Bearer API keys using SHA-256 constant-time comparison.
type APIKeyAuthMiddleware struct {
	hashedKeys [][]byte
	logger     *slog.Logger
}

// NewAPIKeyAuthMiddleware returns a middleware that accepts any of the provided valid keys.
// Keys are hashed at construction time so raw values are not retained in memory beyond setup.
func NewAPIKeyAuthMiddleware(validKeys []string, logger *slog.Logger) APIKeyAuthMiddleware {
	hashed := make([][]byte, 0, len(validKeys))
	for _, k := range validKeys {
		h := sha256.Sum256([]byte(k))
		hashed = append(hashed, h[:])
	}
	return APIKeyAuthMiddleware{hashedKeys: hashed, logger: logger}
}

// Wrap wraps an http.Handler with API key authentication.
func (m APIKeyAuthMiddleware) Wrap(next http.Handler) http.Handler {
	if next == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerTokenFromHeader(r.Header.Get("Authorization"))
		if token == "" {
			logging.Event(r.Context(), m.logger, slog.LevelWarn, "mcp.request_auth", "mcp-http", "denied", slog.String("reason", "missing_bearer_token"))
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		incoming := sha256.Sum256([]byte(token))
		for _, key := range m.hashedKeys {
			if subtle.ConstantTimeCompare(incoming[:], key) == 1 {
				ctx := app.WithAuthenticatedIdentity(r.Context(), localIdentity)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		logging.Event(r.Context(), m.logger, slog.LevelWarn, "mcp.request_auth", "mcp-http", "denied", slog.String("reason", "invalid_bearer_token"))
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	})
}
