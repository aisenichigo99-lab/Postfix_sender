package api

import (
	"context"
	"encoding/json"
	"net/http"

	"io"
	"postfix-manager/internal/agent"
	"postfix-manager/internal/jobs"
	"postfix-manager/internal/postfix"
)

type InstallationAPI struct {
	Manager *jobs.Manager
	Agent   *agent.Client
}

func (a *InstallationAPI) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	defer r.Body.Close()

	var req postfix.InstallRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "Unexpected request data", http.StatusBadRequest)
		return
	}

	job, err := a.Manager.Submit(
		"postfix.install",
		func(
			ctx context.Context,
			report func(int, string),
		) error {
			report(5, "Connecting to System Agent")

			err := a.Agent.Install(ctx, req, func(progress int, message string) {
				if progress <= 0 {
					report(-1, message)
					return
				}

				report(progress, message)
			})

			if err != nil {
				return err
			}

			return nil
		},
	)

	if err != nil {
		http.Error(w, "Unable to create installation job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(job)
}
