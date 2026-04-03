package mcphttp

import (
	"context"
	"net/http"

	"github.com/Carlos0934/billar/internal/app"
)

type LoginUseCase interface {
	StartLogin(ctx context.Context) (app.LoginIntentDTO, error)
}

func LoginHandler(useCase LoginUseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		result, err := useCase.StartLogin(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, result.LoginURL, http.StatusFound)
	}
}
