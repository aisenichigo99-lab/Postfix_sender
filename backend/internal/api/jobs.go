package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"postfix-manager/internal/jobs"
)

type JobsAPI struct {
	Manager *jobs.Manager
}

func (a *JobsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/jobs/"), "/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	id := parts[0]

	if len(parts) == 2 && parts[1] == "events" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		a.streamEvents(w, r, id)
		return
	}

	if len(parts) != 1 || r.Method != http.MethodGet {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	job, ok := a.Manager.Get(id)
	if !ok {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (a *JobsAPI) streamEvents(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	sub, ok := a.Manager.Subscribe(id)
	if !ok {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}
	defer sub.Cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	controller := http.NewResponseController(w)

	send := func(event jobs.Event) error {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintf(w, "event: progress\ndata: %s\n\n", data); err != nil {
			return err
		}

		return controller.Flush()
	}

	// Envoyer d'abord l'historique existant.
	for _, event := range sub.Snapshot {
		if err := send(event); err != nil {
			return
		}
		if event.Status.Terminal() {
			return
		}
	}

	// Puis suivre les nouveaux événements.
	for {
		select {
		case <-r.Context().Done():
			return

		case event := <-sub.Events:
			if err := send(event); err != nil {
				return
			}
			if event.Status.Terminal() {
				return
			}
		}
	}
}
