package api

import (
	"context"
	"encoding/json"
	"net/http"

	"postfix-manager/internal/agent"
	"postfix-manager/internal/jobs"
)

type TestJobAPI struct {
	Manager *jobs.Manager
	Agent   *agent.Client
}

func (a *TestJobAPI) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	job, err := a.Manager.Submit(
		"agent.integration.test",
		func(
			ctx context.Context,
			report func(int, string),
		) error {
			report(5, "Connecting to System Agent")

			return a.Agent.Diagnostic(
				ctx,
				func(progress int, message string) {
					report(progress, message)
				},
			)
		},
	)

	if err != nil {
		http.Error(w, "Unable to create job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(job)
}
