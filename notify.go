package daemon

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	sd "github.com/coreos/go-systemd/v22/daemon"
)

// sd_notify activity counters. NOTIFY_SOCKET and WATCHDOG_USEC are
// process-global state (set once by systemd before exec), so the counters
// are process-global too: ReadNotifyCounters is safe for concurrent use and
// needs no per-instance wiring. Expose them from a /metrics endpoint to make
// readiness and watchdog keepalive observable without parsing logs.
//
//nolint:gochecknoglobals // process-global sd_notify state by design; docs/decisions/2026-10-03_v040-notify-api-shape.md
var (
	notifyReadySent     atomic.Int64
	notifyStoppingSent  atomic.Int64
	notifyWatchdogPings atomic.Int64
	notifyFailures      atomic.Int64
)

// NotifyCounters is a point-in-time snapshot of the process-global sd_notify
// activity counters.
type NotifyCounters struct {
	// ReadySent counts successful READY=1 notifications.
	ReadySent int64

	// StoppingSent counts successful STOPPING=1 notifications.
	StoppingSent int64

	// WatchdogPings counts successful WATCHDOG=1 keepalive pings.
	WatchdogPings int64

	// NotifyFailures counts failed sd_notify sends (NOTIFY_SOCKET set but
	// unreachable, or the write failed). Failures are logged as warnings,
	// never fatal: a daemon that cannot reach systemd must still run.
	NotifyFailures int64
}

// ReadNotifyCounters returns a snapshot of the process-global sd_notify
// activity counters.
func ReadNotifyCounters() NotifyCounters {
	return NotifyCounters{
		ReadySent:      notifyReadySent.Load(),
		StoppingSent:   notifyStoppingSent.Load(),
		WatchdogPings:  notifyWatchdogPings.Load(),
		NotifyFailures: notifyFailures.Load(),
	}
}

// NotifyReady sends READY=1 to systemd, marking the service fully started
// (a type=notify service reaches "active" only after this). A no-op when
// NOTIFY_SOCKET is unset (foreground or non-systemd runs); failures are
// logged as warnings, never fatal. A nil logger defaults to slog.Default().
func NotifyReady(logger *slog.Logger) {
	sendNotify(logger, "READY", sd.SdNotifyReady, &notifyReadySent)
}

// NotifyStopping sends STOPPING=1 to systemd, marking the service as going
// down. A no-op when NOTIFY_SOCKET is unset; failures are logged as
// warnings, never fatal. A nil logger defaults to slog.Default().
func NotifyStopping(logger *slog.Logger) {
	sendNotify(logger, "STOPPING", sd.SdNotifyStopping, &notifyStoppingSent)
}

// Watchdog sends WATCHDOG=1 keepalive pings every interval until ctx is
// canceled. Call it with the interval from WatchdogInterval, which already
// applies the sd_watchdog_enabled(3) recommendation (WatchdogSec/2).
// Non-positive intervals disable the loop (return immediately) instead of
// panicking. A nil logger defaults to slog.Default().
func Watchdog(ctx context.Context, logger *slog.Logger, interval time.Duration) {
	if interval <= 0 {
		return
	}

	if logger == nil {
		logger = slog.Default()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendNotify(logger, "WATCHDOG", sd.SdNotifyWatchdog, &notifyWatchdogPings)
		}
	}
}

// WatchdogInterval reports whether the service manager expects watchdog
// keepalive pings from this process and, if so, the recommended ping
// interval: half of the WatchdogSec timeout, per sd_watchdog_enabled(3)
// ("It is recommended that a daemon sends a keep-alive notification message
// to the service manager every half of the time returned here").
// Returns (0, false) when no watchdog is configured, the variables name a
// different process, or the check fails (logged as a warning).
func WatchdogInterval(logger *slog.Logger) (time.Duration, bool) {
	if logger == nil {
		logger = slog.Default()
	}

	timeout, err := sd.SdWatchdogEnabled(false)
	if err != nil {
		logger.Warn("sd_watchdog check failed", "error", err)

		return 0, false
	}

	if timeout <= 0 {
		return 0, false
	}

	return timeout / 2, true
}

// sendNotify sends one sd_notify state, counts the outcome, and logs it.
// Advisory by contract: a failure to reach systemd never aborts the caller.
func sendNotify(logger *slog.Logger, label, state string, counter *atomic.Int64) {
	if logger == nil {
		logger = slog.Default()
	}

	wasSent, err := sd.SdNotify(false, state)
	if err != nil {
		notifyFailures.Add(1)
		logger.Warn("sd_notify "+label+" failed", "error", err)
	} else if wasSent {
		counter.Add(1)
		logger.Debug("sd_notify " + state + " sent")
	}
}
