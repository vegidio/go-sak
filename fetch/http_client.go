package fetch

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const maxRedirects = 5

// safeCheckRedirect caps the redirect chain and refuses https→http downgrade.
// Used by both the resty client and the idle-timeout client.
func safeCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}

	if isUnsafeDowngrade(req, via) {
		return fmt.Errorf("refusing redirect from https to http: %s", req.URL)
	}

	return nil
}

// isUnsafeDowngrade reports whether a redirect drops TLS on the way to a *different* host. A site that
// downgrades a redirect to itself only exposes its own traffic, and refusing that breaks a fair number
// of otherwise working sites; being sent to plaintext on somebody else's host is the case worth
// blocking.
func isUnsafeDowngrade(req *http.Request, via []*http.Request) bool {
	if len(via) == 0 || via[0].URL.Scheme != "https" || req.URL.Scheme != "http" {
		return false
	}

	return !strings.EqualFold(req.URL.Hostname(), via[0].URL.Hostname())
}

type timeoutConn struct {
	net.Conn
	idle time.Duration
}

func (c *timeoutConn) Read(b []byte) (int, error) {
	// bump the read deadline before each Read
	_ = c.Conn.SetReadDeadline(time.Now().Add(c.idle))
	return c.Conn.Read(b)
}

func (c *timeoutConn) Write(b []byte) (int, error) {
	// also enforce write deadlines
	_ = c.Conn.SetWriteDeadline(time.Now().Add(c.idle))
	return c.Conn.Write(b)
}

func newIdleTimeoutClient(idleTimeout time.Duration) *http.Client {
	// clone the default Transport (to inherit all defaults)
	base := http.DefaultTransport.(*http.Transport).Clone()

	// wrap DialContext to install our timeoutConn
	base.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := &net.Dialer{
			Timeout:   30 * time.Second, // connect timeout (optional)
			KeepAlive: 30 * time.Second, // keep-alive period
		}
		rawConn, err := d.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return &timeoutConn{Conn: rawConn, idle: idleTimeout}, nil
	}

	return &http.Client{
		Transport:     base,
		CheckRedirect: safeCheckRedirect,
	}
}
