package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"postfix-manager/internal/agent"
)

type AgentAPI struct {
	Client *agent.Client
}

func (a *AgentAPI) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := a.Client.Health(ctx); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unavailable",
			"error":  "System agent is not reachable",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "connected",
		"agent":  "postfix-manager",
	})
}
