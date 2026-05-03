// Package github is a minimal client for the GitHub REST API endpoints
// Vellum needs: latest release tag, contents of a directory, and asset download.
//
// Auth is best-effort: GITHUB_TOKEN env first, then `gh auth token` if `gh`
// is in PATH and authenticated, otherwise unauthenticated (60 req/h).
package github

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// AuthSource indicates how the token (if any) was obtained.
type AuthSource int

const (
	AuthNone AuthSource = iota
	AuthEnv
	AuthGH
)

func (a AuthSource) String() string {
	switch a {
	case AuthEnv:
		return "GITHUB_TOKEN env"
	case AuthGH:
		return "gh auth token"
	default:
		return "unauthenticated"
	}
}

// ErrRateLimited is returned when GitHub responds with 403 + X-RateLimit-Remaining: 0.
var ErrRateLimited = errors.New("github rate limit exceeded")

const defaultBaseURL = "https://api.github.com"

// Client is a small wrapper around http.Client with token + base URL.
// BaseURL is exported so tests can point it at httptest.Server.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client

	token  string
	source AuthSource
}

// NewClient resolves auth and returns a ready-to-use client.
func NewClient() *Client {
	token, source := resolveToken()
	return &Client{
		BaseURL: defaultBaseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token:  token,
		source: source,
	}
}

// AuthSource reports which token source was selected.
func (c *Client) AuthSource() AuthSource { return c.source }

// resolveToken looks up a GitHub token, in priority order:
//  1. GITHUB_TOKEN env var
//  2. `gh auth token` if `gh` is in PATH and exits 0 with non-empty output
//  3. none
func resolveToken() (string, AuthSource) {
	if t := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); t != "" {
		return t, AuthEnv
	}
	if _, err := exec.LookPath("gh"); err == nil {
		out, err := exec.Command("gh", "auth", "token").Output()
		if err == nil {
			t := strings.TrimSpace(string(out))
			if t != "" {
				return t, AuthGH
			}
		}
	}
	return "", AuthNone
}

func (c *Client) do(method, path string) (*http.Response, error) {
	url := c.BaseURL + path
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0" {
		_ = drainAndClose(resp.Body)
		return nil, fmt.Errorf("%w (resets at unix %s)", ErrRateLimited, resp.Header.Get("X-RateLimit-Reset"))
	}

	return resp, nil
}

func drainAndClose(b io.ReadCloser) error {
	_, _ = io.Copy(io.Discard, b)
	return b.Close()
}
