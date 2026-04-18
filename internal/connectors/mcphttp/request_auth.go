package mcphttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/infra/logging"
)

// JSONRPCMessageType represents the classified shape of a JSON-RPC message.
type JSONRPCMessageType string

const (
	// JSONRPCMessageRequest is a request with method and id.
	JSONRPCMessageRequest JSONRPCMessageType = "request"
	// JSONRPCMessageNotification is a notification with method but no id.
	JSONRPCMessageNotification JSONRPCMessageType = "notification"
	// JSONRPCMessageResponse is a response with result or error.
	JSONRPCMessageResponse JSONRPCMessageType = "response"
	// JSONRPCMessageBatch is a batch request (array of messages).
	JSONRPCMessageBatch JSONRPCMessageType = "batch"
	// JSONRPCMessageInvalid is malformed or unparseable JSON.
	JSONRPCMessageInvalid JSONRPCMessageType = "invalid"
	// JSONRPCMessageUnknown is valid JSON that doesn't match known shapes.
	JSONRPCMessageUnknown JSONRPCMessageType = "unknown"
)

// JSONRPCMessageInfo contains classified information about a JSON-RPC message.
type JSONRPCMessageInfo struct {
	Type          JSONRPCMessageType
	Method        string
	HasID         bool
	HasResult     bool
	HasError      bool
	BodyParseable bool
}

type jsonrpcRequestPeek struct {
	Method string          `json:"method"`
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

// classifyJSONRPCMessage analyzes the body and returns message classification.
func classifyJSONRPCMessage(body []byte) JSONRPCMessageInfo {
	info := JSONRPCMessageInfo{
		Type:          JSONRPCMessageUnknown,
		BodyParseable: false,
	}

	if len(body) == 0 {
		info.Type = JSONRPCMessageInvalid
		return info
	}

	// Check for batch (array)
	if isArray(body) {
		info.Type = JSONRPCMessageBatch
		info.BodyParseable = true
		return info
	}

	var req jsonrpcRequestPeek
	if err := json.Unmarshal(body, &req); err != nil {
		info.Type = JSONRPCMessageInvalid
		return info
	}

	info.Method = req.Method
	info.HasID = len(req.ID) > 0
	info.HasResult = len(req.Result) > 0
	info.HasError = len(req.Error) > 0
	info.BodyParseable = true

	// Classify based on presence of fields
	if info.HasResult || info.HasError {
		// Response has result or error
		info.Type = JSONRPCMessageResponse
	} else if info.Method != "" {
		// Has method
		if info.HasID {
			info.Type = JSONRPCMessageRequest
		} else {
			info.Type = JSONRPCMessageNotification
		}
	} else {
		// Valid JSON but no recognizable fields
		info.Type = JSONRPCMessageUnknown
	}

	return info
}

// isArray checks if the body starts with '[' (ignoring whitespace).
func isArray(body []byte) bool {
	for _, b := range body {
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			continue
		}
		return b == '['
	}
	return false
}

func peekJSONRPCMethod(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var req jsonrpcRequestPeek
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	return req.Method
}

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

		// Buffer body to allow peeking at method
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			logging.Event(r.Context(), m.logger, slog.LevelError, "mcp.request_auth", "mcp-http", "error", slog.String("reason", "body_read_error"))
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

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

		// Classify JSON-RPC message shape for observability
		msgInfo := classifyJSONRPCMessage(bodyBytes)
		bearerToken := bearerTokenFromHeader(r.Header.Get("Authorization"))
		hasBearerToken := bearerToken != ""

		// Log message classification for observability
		logging.Event(
			r.Context(),
			m.logger,
			slog.LevelDebug,
			"mcp.request_auth",
			"mcp-http",
			"message_classified",
			slog.String("message_type", string(msgInfo.Type)),
			slog.String("method", msgInfo.Method),
			slog.Bool("has_bearer_token", hasBearerToken),
			slog.Bool("body_parseable", msgInfo.BodyParseable),
		)

		identity, err := m.authenticator.Authenticate(r.Context(), bearerToken)

		if err == nil {
			next.ServeHTTP(w, r.WithContext(app.WithAuthenticatedIdentity(r.Context(), identity)))
			return
		}

		// All unauthenticated or unauthorized requests: reject with appropriate error
		status := http.StatusUnauthorized
		if errors.Is(err, app.ErrUnauthorizedIdentity) || errors.Is(err, app.ErrEmailNotVerified) {
			status = http.StatusForbidden
		}
		setMCPAuthChallenge(w.Header(), m.challenge)
		logging.Event(
			r.Context(),
			m.logger,
			slog.LevelWarn,
			"mcp.request_auth",
			"mcp-http",
			"denied",
			slog.String("reason", classifyRequestAuthReason(err)),
			slog.String("method", msgInfo.Method),
			slog.String("message_type", string(msgInfo.Type)),
			slog.Bool("body_parseable", msgInfo.BodyParseable),
		)
		http.Error(w, http.StatusText(status), status)
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
