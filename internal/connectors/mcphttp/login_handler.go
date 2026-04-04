package mcphttp

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/infra/logging"
)

type LoginUseCase interface {
	StartLogin(ctx context.Context) (app.LoginIntentDTO, error)
}

func LoginHandler(useCase LoginUseCase, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		result, err := useCase.StartLogin(r.Context())
		if err != nil {
			logging.Event(r.Context(), logger, slog.LevelError, "session.start_login", "mcp-http", "error", slog.String("reason", "internal_error"))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		logging.Event(r.Context(), logger, slog.LevelInfo, "session.start_login", "mcp-http", "success")

		http.Redirect(w, r, result.LoginURL, http.StatusFound)
	}
}
