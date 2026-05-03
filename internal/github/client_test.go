package github

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_NoAuth(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("PATH", "") // ensure `gh` not findable
	c := NewClient()
	assert.Equal(t, AuthNone, c.AuthSource())
}

func TestNewClient_TokenFromEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_fromenv")
	c := NewClient()
	assert.Equal(t, AuthEnv, c.AuthSource())
}

func TestClient_AddsAuthorizationHeaderWhenTokenPresent(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_fromenv")
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	c := NewClient()
	c.BaseURL = srv.URL
	resp, err := c.do("GET", "/anything")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "Bearer ghp_fromenv", got)
}

func TestClient_NoAuthHeader_WhenNoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("PATH", "")
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	c := NewClient()
	c.BaseURL = srv.URL
	resp, err := c.do("GET", "/anything")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Empty(t, got)
}

func TestClient_RateLimited_ReturnsErrRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1777800000")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"rate limit"}`))
	}))
	defer srv.Close()

	c := NewClient()
	c.BaseURL = srv.URL
	_, err := c.do("GET", "/anything")
	assert.ErrorIs(t, err, ErrRateLimited)
}
