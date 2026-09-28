package daemon

import (
	"context"
	"net"
	"net/http"
	"time"
)

// DefaultDialTimeout bounds connection establishment to a daemon socket.
const DefaultDialTimeout = 5 * time.Second

// idleConnTimeout caps how long idle connections live in the transport's
// pool. Matches http.DefaultTransport's default (90s).
const idleConnTimeout = 90 * time.Second

// maxIdleConnsPerHost overrides Go's default of 2 idle connections per host.
// All daemon traffic targets a single unix socket, so the default of 2
// bottlenecked concurrent requests from long-lived clients. Unix socket
// connections are cheap (kernel memory only), so a generous limit avoids
// needless dial churn under burst load.
const maxIdleConnsPerHost = 32

// NewHTTPClient creates an *http.Client that dials the given unix socket path.
func NewHTTPClient(socketPath string, timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{Timeout: DefaultDialTimeout}).DialContext(ctx, "unix", socketPath)
			},
			MaxIdleConnsPerHost: maxIdleConnsPerHost,
			IdleConnTimeout:     idleConnTimeout,
		},
		Timeout: timeout,
	}
}

// NewHTTPClientNoTimeout creates an *http.Client with no total timeout,
// suitable for long-lived connections like SSE watch streams. Relies on context
// cancellation for teardown.
func NewHTTPClientNoTimeout(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{Timeout: DefaultDialTimeout}).DialContext(ctx, "unix", socketPath)
			},
			MaxIdleConnsPerHost: maxIdleConnsPerHost,
			IdleConnTimeout:     idleConnTimeout,
		},
	}
}
