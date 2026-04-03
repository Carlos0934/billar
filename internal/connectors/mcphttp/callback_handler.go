package mcphttp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
)

type CallbackUseCase interface {
	HandleOAuthCallback(ctx context.Context, cmd app.HandleOAuthCallbackCommand) (app.SessionDTO, error)
}

func CallbackHandler(useCase CallbackUseCase, stateStore app.StateStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		code := strings.TrimSpace(r.URL.Query().Get("code"))
		state := strings.TrimSpace(r.URL.Query().Get("state"))
		if code == "" || state == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if err := stateStore.Validate(r.Context(), state); err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		_, err := useCase.HandleOAuthCallback(r.Context(), app.HandleOAuthCallbackCommand{Code: code, State: state})
		if err != nil {
			if errors.Is(err, app.ErrUnauthorizedIdentity) || errors.Is(err, app.ErrEmailNotVerified) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/auth/session", http.StatusFound)
	}
}
