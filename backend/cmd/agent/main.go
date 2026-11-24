package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"postfix-manager/internal/agent"
	"postfix-manager/internal/system"
	"strconv"
	"strings"
	"syscall"
	"time"

	"postfix-manager/internal/postfix"
)

const socketPath = "/run/postfix-manager/agent.sock"

type peerUIDListener struct {
	net.Listener
	allowedUID uint32
}

func (l *peerUIDListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}

		unixConn, ok := conn.(*net.UnixConn)
		if !ok {
			conn.Close()
			continue
		}

		raw, err := unixConn.SyscallConn()
		if err != nil {
			conn.Close()
			continue
		}

		var credentials *syscall.Ucred
		var socketErr error

		err = raw.Control(func(fd uintptr) {
			credentials, socketErr = syscall.GetsockoptUcred(
				int(fd),
				syscall.SOL_SOCKET,
				syscall.SO_PEERCRED,
			)
		})

		if err != nil || socketErr != nil || credentials == nil {
			log.Println("Unable to verify socket peer")
			conn.Close()
			continue
		}

		if credentials.Uid != l.allowedUID {
			log.Printf("Rejected socket peer UID: %d", credentials.Uid)
			conn.Close()
			continue
		}

		return conn, nil
	}
}

func installHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limiter la taille du JSON : les identifiants SMTP sont transmis
	// dans le corps de la requête, jamais dans l'URL.
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req postfix.InstallRequest

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid installation request", http.StatusBadRequest)
		return
	}

	// Refuser les données supplémentaires après le premier objet JSON.
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "unexpected data after request", http.StatusBadRequest)
		return
	}

	// L'installation peut nécessiter plusieurs minutes.
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	controller := http.NewResponseController(w)

	sendEvent := func(event agent.Event) {
		if err := encoder.Encode(event); err != nil {
			return
		}

		_ = controller.Flush()
	}

	sendEvent(agent.Event{
		Type:     "log",
		Message:  "Préparation de l'installation de Postfix...",
		Progress: 1,
	})

	result, err := postfix.Install(ctx, req, func(message string) {
		sendEvent(agent.Event{
			Type:     "log",
			Message:  message,
			Progress: 0,
		})
	})

	if err != nil {
		sendEvent(agent.Event{
			Type:    "result",
			Message: "Échec de l'installation de Postfix",
			Success: false,
			Error:   fmt.Sprintf("%v", err),
		})
		return
	}

	if !result.Success {
		sendEvent(agent.Event{
			Type:    "result",
			Message: "L'installation de Postfix n'a pas abouti",
			Success: false,
			Error:   "installation non confirmée",
		})
		return
	}

	sendEvent(agent.Event{
		Type:     "result",
		Message:  "Postfix et le relais SMTP sont configurés.",
		Progress: 100,
		Success:  true,
	})
}

func diagnosticHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	controller := http.NewResponseController(w)

	send := func(event agent.Event) error {
		if err := json.NewEncoder(w).Encode(event); err != nil {
			return err
		}
		return controller.Flush()
	}

	// Début du diagnostic réel.
	if err := send(agent.Event{
		Type:     "log",
		Progress: 5,
		Message:  "Starting real system diagnostic",
	}); err != nil {
		return
	}

	info := system.DetectContext(r.Context())

	total := len(info.Checks)
	blocking := 0
	warnings := 0

	for i, check := range info.Checks {
		if r.Context().Err() != nil {
			return
		}

		switch check.Status {
		case "blocking":
			blocking++
		case "warning":
			warnings++
		}

		progress := 90
		if total > 0 {
			progress = 10 + ((i + 1) * 80 / total)
		}

		message := strings.ToUpper(check.Status) +
			" | " + check.Label +
			" | " + check.Message

		if err := send(agent.Event{
			Type:     "log",
			Progress: progress,
			Message:  message,
		}); err != nil {
			return
		}
	}

	readiness := "not ready"
	if info.Ready {
		readiness = "ready"
	}

	summary := "Diagnostic completed. " +
		strconv.Itoa(blocking) + " blocking issue(s), " +
		strconv.Itoa(warnings) + " warning(s). " +
		"Installation readiness: " + readiness + "."

	_ = send(agent.Event{
		Type:     "result",
		Progress: 100,
		Success:  true,
		Message:  summary,
	})
}

func main() {
	if os.Geteuid() != 0 {
		log.Fatal("The Postfix Manager agent must run as root")
	}

	uidValue := os.Getenv("POSTFIX_MANAGER_API_UID")
	if uidValue == "" {
		log.Fatal("POSTFIX_MANAGER_API_UID is required")
	}

	uid, err := strconv.ParseUint(uidValue, 10, 32)
	if err != nil {
		log.Fatal("Invalid POSTFIX_MANAGER_API_UID")
	}

	if err := os.MkdirAll(filepath.Dir(socketPath), 0750); err != nil {
		log.Fatal(err)
	}

	// Vérifier le socket existant avant toute suppression.
	if info, err := os.Lstat(socketPath); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			log.Fatal("Socket path exists but is not a Unix socket")
		}

		conn, dialErr := net.DialTimeout("unix", socketPath, 300*time.Millisecond)
		if dialErr == nil {
			conn.Close()
			log.Fatal("Another agent is already listening")
		}

		if err := os.Remove(socketPath); err != nil {
			log.Fatal(err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Fatal(err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	if err := os.Chmod(socketPath, 0660); err != nil {
		log.Fatal(err)
	}

	secureListener := &peerUIDListener{
		Listener:   listener,
		allowedUID: uint32(uid),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/v1/postfix/install", installHandler)

	mux.HandleFunc("GET /v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","agent":"postfix-manager"}`))
	})

	mux.HandleFunc("GET /v1/diagnostic", diagnosticHandler)

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("Postfix Manager System Agent listening securely on Unix socket")

	if err := server.Serve(secureListener); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
