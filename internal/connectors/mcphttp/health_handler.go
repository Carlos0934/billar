package mcphttp

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Carlos0934/billar/internal/app"
)

type HealthUseCase interface {
	Status(ctx context.Context) (app.HealthDTO, error)
}

func HealthHandler(useCase HealthUseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		result, err := useCase.Status(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}
