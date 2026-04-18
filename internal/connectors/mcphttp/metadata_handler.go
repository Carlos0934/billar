package mcphttp

import (
	"encoding/json"
	"net/http"

	"github.com/Carlos0934/billar/internal/app"
)

type metadataResponse struct {
	ResourceURI            string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
}

func MetadataHandler(challenge app.OAuthChallengeDTO) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(metadataResponse{
			ResourceURI:            challenge.ResourceURI,
			AuthorizationServers:   challenge.AuthorizationServers,
			BearerMethodsSupported: []string{"bearer"},
		})
	}
}
