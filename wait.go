package daemon

import (
	"context"
	"fmt"
	"net"
	"time"
)

// DefaultWaitPollInterval is how often WaitForSocket and ProbeUntilAlive
// re-check between attempts. The context's deadline bounds the total wait,
// so the interval only sets how soon after the daemon is up the wait
// returns.
const DefaultWaitPollInterval = 50 * time.Millisecond

// WaitOption configures WaitForSocket and ProbeUntilAlive.
type WaitOption func(*waitConfig)

// waitConfig is the resolved option set of the wait helpers.
type waitConfig struct {
	pollInterval time.Duration
}

// WithPollInterval sets how often WaitForSocket and ProbeUntilAlive re-check
// (default DefaultWaitPollInterval). A non-positive interval keeps the default.
func WithPollInterval(d time.Duration) WaitOption {
	return func(cfg *waitConfig) {
		if d > 0 {
			cfg.pollInterval = d
		}
	}
}

// WaitForSocket blocks until a listener accepts connections on the unix
// socket at socketPath, or until ctx is done, in which case the returned
// error wraps ctx.Err(). The first check runs before any pause, so a socket
// that is already up returns immediately.
//
// Only a completed dial counts. A socket file nobody accepts on — what a
// crashed daemon leaves behind — does not, which is what tells this apart
// from polling os.Stat. Use ProbeUntilAlive when the daemon must also
// answer HTTP.
func WaitForSocket(ctx context.Context, socketPath string, opts ...WaitOption) error {
	return pollUntil(ctx, resolveWaitConfig(opts).pollInterval, "wait for socket "+socketPath, func() bool {
		return dialSocket(ctx, socketPath)
	})
}

// ProbeUntilAlive blocks until a daemon answers GET healthPath on socketPath,
// or until ctx is done, in which case the returned error wraps ctx.Err().
// Any status code counts: the probe measures whether the socket is served,
// not application health. Each probe is bounded by DefaultProbeTimeout.
func ProbeUntilAlive(ctx context.Context, socketPath, healthPath string, opts ...WaitOption) error {
	return pollUntil(
		ctx,
		resolveWaitConfig(opts).pollInterval,
		"probe "+healthPath+" on socket "+socketPath,
		func() bool {
			return isSocketAlive(ctx, socketPath, healthPath, DefaultProbeTimeout)
		},
	)
}

// resolveWaitConfig applies opts to the wait defaults.
func resolveWaitConfig(opts []WaitOption) waitConfig {
	cfg := waitConfig{pollInterval: DefaultWaitPollInterval}

	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return cfg
}

// pollUntil runs probe immediately and then every interval until it reports
// true. A done ctx ends the wait with an error wrapping ctx.Err(), named by
// what so the caller sees which wait timed out.
func pollUntil(ctx context.Context, interval time.Duration, what string, probe func() bool) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%s: %w", what, err)
		}

		if probe() {
			return nil
		}

		select {
		case <-ctx.Done():
		case <-ticker.C:
		}
	}
}

// dialSocket reports whether a listener accepts connections on socketPath.
func dialSocket(ctx context.Context, socketPath string) bool {
	var dialer net.Dialer

	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return false
	}

	_ = conn.Close()

	return true
}
