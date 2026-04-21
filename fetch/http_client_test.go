package fetch

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIdleTimeoutClient(t *testing.T) {
	t.Run("returns a usable http client", func(t *testing.T) {
		c := newIdleTimeoutClient(5 * time.Second)
		require.NotNil(t, c)
		require.NotNil(t, c.Transport)
		assert.NotNil(t, c.CheckRedirect, "CheckRedirect must be configured")
	})

	t.Run("successful GET via custom client", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "hello")
		}))
		defer server.Close()

		c := newIdleTimeoutClient(2 * time.Second)

		resp, err := c.Get(server.URL)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(body))
	})
}

func TestSafeCheckRedirect(t *testing.T) {
	mustURL := func(s string) *url.URL {
		u, err := url.Parse(s)
		require.NoError(t, err)
		return u
	}

	t.Run("allows under the cap", func(t *testing.T) {
		req := &http.Request{URL: mustURL("https://example.com/b")}
		via := []*http.Request{{URL: mustURL("https://example.com/a")}}
		assert.NoError(t, safeCheckRedirect(req, via))
	})

	t.Run("rejects at the cap", func(t *testing.T) {
		req := &http.Request{URL: mustURL("https://example.com/final")}
		via := make([]*http.Request, maxRedirects)
		for i := range via {
			via[i] = &http.Request{URL: mustURL("https://example.com/x")}
		}
		err := safeCheckRedirect(req, via)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stopped after")
	})

	t.Run("rejects https to http downgrade", func(t *testing.T) {
		req := &http.Request{URL: mustURL("http://example.com/b")}
		via := []*http.Request{{URL: mustURL("https://example.com/a")}}
		err := safeCheckRedirect(req, via)
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "https to http"))
	})

	t.Run("allows http to https upgrade", func(t *testing.T) {
		req := &http.Request{URL: mustURL("https://example.com/b")}
		via := []*http.Request{{URL: mustURL("http://example.com/a")}}
		assert.NoError(t, safeCheckRedirect(req, via))
	})
}
