package api

import (
	"encoding/json"
	"net/http"

	"postfix-manager/internal/system"
)

func SystemInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := system.Detect()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(info)
}