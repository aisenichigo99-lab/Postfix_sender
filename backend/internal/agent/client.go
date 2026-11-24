package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"postfix-manager/internal/postfix"
)

type Client struct {
	transport http.RoundTripper
}

func NewClient(socketPath string) *Client {
	return &Client{
		transport: &http.Transport{
			DialContext: func(
				ctx context.Context,
				network string,
				address string,
			) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(
					ctx,
					"unix",
					socketPath,
				)
			},
		},
	}
}

func (c *Client) request(
	ctx context.Context,
	method string,
	path string,
	body []byte,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		method,
		"http://unix"+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{
		Transport: c.transport,
	}

	return client.Do(req)
}

func (c *Client) Health(ctx context.Context) error {
	resp, err := c.request(
		ctx,
		http.MethodGet,
		"/v1/health",
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent returned HTTP %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Install(
	ctx context.Context,
	req postfix.InstallRequest,
	report func(int, string),
) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("cannot encode installation request: %w", err)
	}

	resp, err := c.request(
		ctx,
		http.MethodPost,
		"/v1/postfix/install",
		body,
	)
	if err != nil {
		return fmt.Errorf("agent connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent returned HTTP %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4096), 1024*1024)

	completed := false

	for scanner.Scan() {
		var event Event

		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return fmt.Errorf("invalid agent event: %w", err)
		}

		switch event.Type {
		case "log":
			if report != nil {
				if event.Progress <= 0 {
					// Ajouter un log sans modifier la progression.
					report(-1, event.Message)
				} else {
					report(event.Progress, event.Message)
				}
			}

		case "result":
			if !event.Success {
				message := event.Error
				if message == "" {
					message = "installation failed"
				}
				return fmt.Errorf("%s", message)
			}

			completed = true

			if report != nil {
				report(100, event.Message)
			}

		default:
			return fmt.Errorf("unknown agent event: %s", event.Type)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("agent stream interrupted: %w", err)
	}

	if !completed {
		return fmt.Errorf("agent stream ended without a result")
	}

	return nil
}

func (c *Client) Diagnostic(
	ctx context.Context,
	report func(int, string),
) error {
	resp, err := c.request(
		ctx,
		http.MethodGet,
		"/v1/diagnostic",
		nil,
	)
	if err != nil {
		return fmt.Errorf("agent connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent returned HTTP %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4096), 1024*1024)

	completed := false

	for scanner.Scan() {
		var event Event

		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return fmt.Errorf("invalid agent event: %w", err)
		}

		switch event.Type {
		case "log":
			if report != nil {
				report(event.Progress, event.Message)
			}

		case "result":
			if !event.Success {
				return fmt.Errorf("diagnostic failed: %s", event.Error)
			}
			completed = true

		default:
			return fmt.Errorf("unknown event type: %s", event.Type)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if !completed {
		return fmt.Errorf("agent stream ended unexpectedly")
	}

	return nil
}
