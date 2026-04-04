package mcphttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/infra/logging"
)

type CallbackUseCase interface {
	HandleOAuthCallback(ctx context.Context, cmd app.HandleOAuthCallbackCommand) (app.SessionDTO, error)
}

func CallbackHandler(useCase CallbackUseCase, stateStore app.StateStore, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		code := strings.TrimSpace(r.URL.Query().Get("code"))
		state := strings.TrimSpace(r.URL.Query().Get("state"))
		if code == "" || state == "" {
			logging.Event(r.Context(), logger, slog.LevelWarn, "auth.callback", "mcp-http", "denied", slog.String("reason", "missing_code_or_state"))
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if err := stateStore.Validate(r.Context(), state); err != nil {
			logging.Event(r.Context(), logger, slog.LevelWarn, "auth.callback", "mcp-http", "denied", slog.String("reason", "invalid_state"))
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		_, err := useCase.HandleOAuthCallback(r.Context(), app.HandleOAuthCallbackCommand{Code: code, State: state})
		if err != nil {
			if errors.Is(err, app.ErrUnauthorizedIdentity) || errors.Is(err, app.ErrEmailNotVerified) {
				logging.Event(r.Context(), logger, slog.LevelWarn, "auth.callback", "mcp-http", "denied", slog.String("reason", classifyHTTPAuthReason(err)))
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			logging.Event(r.Context(), logger, slog.LevelError, "auth.callback", "mcp-http", "error", slog.String("reason", classifyHTTPAuthReason(err)))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		logging.Event(r.Context(), logger, slog.LevelInfo, "auth.callback", "mcp-http", "success")

		http.Redirect(w, r, "/auth/session", http.StatusFound)
	}
}

func classifyHTTPAuthReason(err error) string {
	if errors.Is(err, app.ErrUnauthorizedIdentity) {
		return "unauthorized_identity"
	}
	if errors.Is(err, app.ErrEmailNotVerified) {
		return "email_not_verified"
	}
	return "internal_error"
}
