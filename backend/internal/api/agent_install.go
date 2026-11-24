package api

import (
	"encoding/json"
	"net/http"

	"postfix-manager/internal/postfix"
)

func AgentInstallHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Invalid installation request", http.StatusBadRequest)
		return
	}

	result, err := postfix.Install(
		r.Context(),
		req,
		nil,
	)

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   err.Error(),
			"logs":    result.Logs,
		})
		return
	}

	json.NewEncoder(w).Encode(result)
}
