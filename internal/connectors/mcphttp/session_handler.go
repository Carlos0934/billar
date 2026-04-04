package mcphttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/infra/logging"
)

type StatusUseCase interface {
	Status(ctx context.Context) (app.SessionStatusDTO, error)
}

type LogoutUseCase interface {
	Logout(ctx context.Context) (app.LogoutDTO, error)
}

func SessionStatusHandler(useCase StatusUseCase, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		result, err := useCase.Status(r.Context())
		if err != nil {
			logging.Event(r.Context(), logger, slog.LevelError, "session.status", "mcp-http", "error", slog.String("reason", "internal_error"))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		logging.Event(r.Context(), logger, slog.LevelInfo, "session.status", "mcp-http", "success", slog.String("status", result.Status))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func LogoutHandler(useCase LogoutUseCase, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		result, err := useCase.Logout(r.Context())
		if err != nil {
			logging.Event(r.Context(), logger, slog.LevelError, "session.logout", "mcp-http", "error", slog.String("reason", "internal_error"))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		logging.Event(r.Context(), logger, slog.LevelInfo, "session.logout", "mcp-http", "success")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}
