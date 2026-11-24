package postfix

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
)

const (
	maxCommandOutput = 1024 * 1024 // 1 MiB
	maxLogLine       = 64 * 1024   // 64 KiB
)

type streamWriter struct {
	mu      sync.Mutex
	pending bytes.Buffer
	output  bytes.Buffer
	report  func(string)
}

func (s *streamWriter) Write(p []byte) (int, error) {
	s.mu.Lock()

	n := len(p)

	// Conserver une sortie limitée en mémoire pour le diagnostic.
	remaining := maxCommandOutput - s.output.Len()
	if remaining > 0 {
		if remaining > len(p) {
			remaining = len(p)
		}
		_, _ = s.output.Write(p[:remaining])
	}

	_, _ = s.pending.Write(p)

	var lines []string

	for s.pending.Len() > 0 {
		data := s.pending.Bytes()
		newline := bytes.IndexByte(data, '\n')

		if newline >= 0 {
			line := string(data[:newline])
			s.pending.Next(newline + 1)
			lines = append(lines, strings.TrimSpace(line))
			continue
		}

		// Éviter qu'une ligne anormalement longue ne consomme
		// une quantité illimitée de mémoire.
		if s.pending.Len() >= maxLogLine {
			line := string(s.pending.Next(maxLogLine))
			lines = append(lines, strings.TrimSpace(line))
			continue
		}

		break
	}

	report := s.report
	s.mu.Unlock()

	// Ne jamais appeler le callback en conservant le mutex.
	if report != nil {
		for _, line := range lines {
			if line != "" {
				report(line)
			}
		}
	}

	return n, nil
}

func (s *streamWriter) Flush() {
	s.mu.Lock()

	remaining := strings.TrimSpace(s.pending.String())
	s.pending.Reset()

	report := s.report
	s.mu.Unlock()

	if report != nil && remaining != "" {
		report(remaining)
	}
}

func (s *streamWriter) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.output.String()
}

// mergeEnvironment remplace proprement les variables d'environnement
// fournies, sans conserver plusieurs valeurs pour une même variable.
func mergeEnvironment(extra []string) []string {
	values := make(map[string]string)

	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}

	for _, entry := range extra {
		key, value, ok := strings.Cut(entry, "=")
		if ok && key != "" {
			values[key] = value
		}
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	env := make([]string, 0, len(keys))
	for _, key := range keys {
		env = append(env, key+"="+values[key])
	}

	return env
}

// runCommand exécute une commande sans shell.
// Les arguments ne doivent jamais contenir de mot de passe ou de secret.
func runCommand(
	ctx context.Context,
	name string,
	args []string,
	env []string,
	stdin string,
	report func(string),
) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = mergeEnvironment(env)

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	stream := &streamWriter{
		report: report,
	}

	cmd.Stdout = stream
	cmd.Stderr = stream

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("impossible de démarrer %s: %w", name, err)
	}

	err := cmd.Wait()
	stream.Flush()

	output := stream.String()

	if err != nil {
		if ctx.Err() != nil {
			return output, fmt.Errorf(
				"commande %s interrompue: %w",
				name,
				ctx.Err(),
			)
		}

		return output, fmt.Errorf(
			"commande %s échouée: %w",
			name,
			err,
		)
	}

	return output, nil
}
