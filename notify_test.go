package daemon

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// notifyTests is the shared table for the NotifyReady/NotifyStopping tables:
// same three environments (unset, live unixgram stand-in for NOTIFY_SOCKET,
// unreachable path), different expected wire message and counter.
type notifyExpectation struct {
	name        string
	notifyEnv   string
	listen      bool
	wantMessage string
}

func runNotifyTest(
	t *testing.T,
	tt notifyExpectation,
	notify func(),
	counterName string,
	counterBefore int64,
	counterAfter func() int64,
) {
	t.Helper()

	var pc net.PacketConn

	if tt.listen {
		var err error

		pc, err = (&net.ListenConfig{}).ListenPacket(
			t.Context(), "unixgram", filepath.Join(t.TempDir(), "notify.sock"),
		)
		if err != nil {
			t.Fatalf("create unixgram listener: %v", err)
		}

		defer func() { _ = pc.Close() }()

		t.Setenv("NOTIFY_SOCKET", pc.LocalAddr().String())
	} else {
		t.Setenv("NOTIFY_SOCKET", tt.notifyEnv)
	}

	notify()

	if tt.wantMessage == "" {
		return
	}

	buf := make([]byte, 64)

	if err := pc.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	n, _, err := pc.ReadFrom(buf)
	if err != nil {
		t.Fatalf("did not receive %s notification: %v", counterName, err)
	}

	if got := string(buf[:n]); got != tt.wantMessage {
		t.Errorf("notification = %q, want %q", got, tt.wantMessage)
	}

	if got := counterAfter(); got != counterBefore+1 {
		t.Errorf("%s = %d, want %d", counterName, got, counterBefore+1)
	}
}

func TestNotifyReady(t *testing.T) {
	// Cannot use t.Parallel: t.Setenv is not allowed in parallel tests.
	tests := []struct {
		name        string
		notifyEnv   string
		listen      bool
		wantMessage string
	}{
		{
			name:        "unset NOTIFY_SOCKET is a no-op",
			notifyEnv:   "",
			wantMessage: "",
		},
		{
			name:        "unixgram NOTIFY_SOCKET receives READY=1",
			listen:      true,
			wantMessage: "READY=1",
		},
		{
			name:        "unreachable NOTIFY_SOCKET logs and continues",
			notifyEnv:   filepath.Join(t.TempDir(), "missing.sock"),
			wantMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := ReadNotifyCounters()

			runNotifyTest(t, tt, func() { NotifyReady(nil) },
				"ReadySent", before.ReadySent,
				func() int64 { return ReadNotifyCounters().ReadySent })
		})
	}
}

func TestNotifyStopping(t *testing.T) {
	// Cannot use t.Parallel: t.Setenv is not allowed in parallel tests.
	tests := []struct {
		name        string
		notifyEnv   string
		listen      bool
		wantMessage string
	}{
		{
			name:        "unset NOTIFY_SOCKET is a no-op",
			notifyEnv:   "",
			wantMessage: "",
		},
		{
			name:        "unixgram NOTIFY_SOCKET receives STOPPING=1",
			listen:      true,
			wantMessage: "STOPPING=1",
		},
		{
			name:        "unreachable NOTIFY_SOCKET logs and continues",
			notifyEnv:   filepath.Join(t.TempDir(), "missing.sock"),
			wantMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := ReadNotifyCounters()

			runNotifyTest(t, tt, func() { NotifyStopping(nil) },
				"StoppingSent", before.StoppingSent,
				func() int64 { return ReadNotifyCounters().StoppingSent })
		})
	}
}

func TestNotifyFailureCounter(t *testing.T) {
	// Cannot use t.Parallel: t.Setenv is not allowed in parallel tests.
	t.Setenv("NOTIFY_SOCKET", filepath.Join(t.TempDir(), "missing.sock"))

	before := ReadNotifyCounters().NotifyFailures

	NotifyReady(nil)
	NotifyStopping(nil)

	if got := ReadNotifyCounters().NotifyFailures; got != before+2 {
		t.Errorf("NotifyFailures = %d, want %d", got, before+2)
	}
}

func TestWatchdog_ExitsOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan struct{})

	go func() {
		Watchdog(ctx, nil, 10*time.Millisecond)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watchdog did not exit after context cancellation")
	}
}

func TestWatchdog_NonPositiveIntervalReturnsImmediately(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})

	go func() {
		Watchdog(t.Context(), nil, 0)
		Watchdog(t.Context(), nil, -time.Second)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watchdog did not return for a non-positive interval")
	}
}

func TestWatchdog_SendsWatchdogNotifications(t *testing.T) {
	// Cannot run in parallel with other tests because t.Setenv is not allowed
	// under t.Parallel in Go 1.26.

	// Create a temporary unixgram socket to stand in for systemd's NOTIFY_SOCKET.
	tmpDir := t.TempDir()

	socketPath := filepath.Join(tmpDir, "notify.sock")

	pc, err := (&net.ListenConfig{}).ListenPacket(t.Context(), "unixgram", socketPath)
	if err != nil {
		t.Fatalf("create unixgram listener: %v", err)
	}

	defer func() { _ = pc.Close() }()

	t.Setenv("NOTIFY_SOCKET", socketPath)

	pingsBefore := ReadNotifyCounters().WatchdogPings

	ctx, cancel := context.WithCancel(t.Context())

	defer cancel()

	go Watchdog(ctx, nil, 20*time.Millisecond)

	buf := make([]byte, 1024)

	if err := pc.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	n, _, err := pc.ReadFrom(buf)
	if err != nil {
		t.Fatalf("did not receive watchdog notification: %v", err)
	}

	msg := string(buf[:n])
	if !strings.Contains(msg, "WATCHDOG=1") {
		t.Errorf("expected WATCHDOG=1 notification, got %q", msg)
	}

	if got := ReadNotifyCounters().WatchdogPings; got != pingsBefore+1 {
		t.Errorf("WatchdogPings = %d, want %d", got, pingsBefore+1)
	}
}

func TestWatchdogInterval(t *testing.T) {
	// Cannot use t.Parallel: t.Setenv is not allowed in parallel tests.
	tests := []struct {
		name         string
		watchdogUsec string
		watchdogPid  string
		want         time.Duration
		wantEnabled  bool
	}{
		{
			name:        "unset WATCHDOG_USEC disables the watchdog",
			want:        0,
			wantEnabled: false,
		},
		{
			name:         "half of WatchdogSec is the recommended interval",
			watchdogUsec: "10000000",
			want:         5 * time.Second,
			wantEnabled:  true,
		},
		{
			name:         "WATCHDOG_PID for another process disables the watchdog",
			watchdogUsec: "10000000",
			watchdogPid:  "999999",
			want:         0,
			wantEnabled:  false,
		},
		{
			name:         "invalid WATCHDOG_USEC logs and disables",
			watchdogUsec: "not-a-number",
			want:         0,
			wantEnabled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.watchdogUsec != "" {
				t.Setenv("WATCHDOG_USEC", tt.watchdogUsec)
			}

			if tt.watchdogPid != "" {
				t.Setenv("WATCHDOG_PID", tt.watchdogPid)
			}

			interval, enabled := WatchdogInterval(nil)

			if enabled != tt.wantEnabled {
				t.Errorf("enabled = %v, want %v", enabled, tt.wantEnabled)
			}

			if interval != tt.want {
				t.Errorf("interval = %v, want %v", interval, tt.want)
			}
		})
	}
}

// TestNotifyWithoutSystemdIsSilent pins what "silent no-op" means off
// systemd: with the notify environment unset, READY and STOPPING change no
// counter and no watchdog is configured.
func TestNotifyWithoutSystemdIsSilent(t *testing.T) {
	// Cannot use t.Parallel: t.Setenv is not allowed in parallel tests.
	t.Setenv("NOTIFY_SOCKET", "")
	t.Setenv("WATCHDOG_USEC", "")
	t.Setenv("WATCHDOG_PID", "")

	before := ReadNotifyCounters()

	NotifyReady(nil)
	NotifyStopping(nil)

	if after := ReadNotifyCounters(); after != before {
		t.Errorf("counters moved without systemd: before %+v, after %+v", before, after)
	}

	if interval, enabled := WatchdogInterval(nil); enabled {
		t.Errorf("watchdog enabled at %v without systemd, want disabled", interval)
	}
}

func TestWatchdogInterval_CurrentPidHonored(t *testing.T) {
	// Cannot use t.Parallel: t.Setenv is not allowed in parallel tests.
	t.Setenv("WATCHDOG_USEC", "10000000")
	t.Setenv("WATCHDOG_PID", strconv.Itoa(os.Getpid()))

	interval, enabled := WatchdogInterval(nil)
	if !enabled {
		t.Fatal("enabled = false, want true for current-pid WATCHDOG_PID")
	}

	if interval != 5*time.Second {
		t.Errorf("interval = %v, want %v", interval, 5*time.Second)
	}
}
