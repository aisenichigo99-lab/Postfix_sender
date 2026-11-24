package main

import (
	"log"
	"net/http"

	"postfix-manager/internal/agent"
	"postfix-manager/internal/api"
	"postfix-manager/internal/jobs"
)

const agentSocket = "/run/postfix-manager/agent.sock"

func main() {
	mux := http.NewServeMux()

	jobManager := jobs.NewManager()
	agentClient := agent.NewClient(agentSocket)

	mux.HandleFunc("/api/system", api.SystemInfoHandler)

	mux.Handle("/api/jobs/", &api.JobsAPI{
		Manager: jobManager,
	})

	installationAPI := &api.InstallationAPI{
		Manager: jobManager,
		Agent:   agentClient,
	}

	mux.HandleFunc(
		"/api/installation",
		installationAPI.Create,
	)
	testAPI := &api.TestJobAPI{
		Manager: jobManager,
		Agent:   agentClient,
	}

	mux.HandleFunc("/api/test-job", testAPI.Create)
	agentAPI := &api.AgentAPI{
		Client: agentClient,
	}

	mux.HandleFunc("/api/agent/health", agentAPI.Health)

	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}

	log.Println("Postfix Manager API started on 127.0.0.1:8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
