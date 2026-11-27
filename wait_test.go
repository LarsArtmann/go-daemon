package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// shortWait is how long a "never comes up" row waits before its context
// expires. Short on purpose: those rows only exercise the timeout path.
const shortWait = 150 * time.Millisecond

// shortPoll makes the timeout rows re-check often enough to be meaningful
// within shortWait.
const shortPoll = 5 * time.Millisecond

// briefContext returns a context that expires after shortWait.
func briefContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), shortWait)
	t.Cleanup(cancel)

	return ctx
}

// doneContext returns an already-canceled context.
func doneContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	return ctx
}

// listeningSocket is a path with a listener accepting on it.
func listeningSocket(t *testing.T) string {
	t.Helper()

	socketPath := filepath.Join(t.TempDir(), "live.sock")

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socketPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	t.Cleanup(func() { _ = listener.Close() })

	return socketPath
}

// leftoverSocketFile leaves a socket file behind with no listener: what a
// crashed daemon leaves when its socket was not cleaned up.
func leftoverSocketFile(t *testing.T) string {
	t.Helper()

	socketPath := filepath.Join(t.TempDir(), "leftover.sock")

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socketPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	// Keep the file when the listener closes; that file is the point here.
	listener.(*net.UnixListener).SetUnlinkOnClose(false)

	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	return socketPath
}

// plainFilePath is a path occupied by a regular file, not a socket.
func plainFilePath(t *testing.T) string {
	t.Helper()

	socketPath := filepath.Join(t.TempDir(), "plain.sock")

	if err := os.WriteFile(socketPath, []byte("not a socket"), 0o600); err != nil {
		t.Fatalf("write plain file: %v", err)
	}

	return socketPath
}

// missingSocketPath is a path nothing will ever listen on.
func missingSocketPath(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), "missing.sock")
}

// serveInBackground runs a SocketServer with handler on a fresh socket,
// waits for the socket to be live, and stops the server when the test ends.
func serveInBackground(t *testing.T, handler http.Handler) string {
	t.Helper()

	socketPath := filepath.Join(t.TempDir(), "daemon.sock")

	_, cancel, errCh := runSocketServer(t, socketPath, Lifecycle{Handler: handler})

	t.Cleanup(func() { stopSocketServer(t, cancel, errCh) })

	return socketPath
}

// healthDaemon serves a daemon answering GET /health with 200.
func healthDaemon(t *testing.T) string {
	t.Helper()

	return serveInBackground(t, healthMux())
}

// teapotDaemon answers every request with 418: ProbeUntilAlive only needs a
// served socket, any status code counts.
func teapotDaemon(t *testing.T) string {
	t.Helper()

	return serveInBackground(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
}

// hangingDaemon accepts requests and never answers them.
func hangingDaemon(t *testing.T) string {
	t.Helper()

	return serveInBackground(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
}

func TestWaitForSocket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ctx       func(t *testing.T) context.Context
		socket    func(t *testing.T) string
		wantErrIs error
	}{
		{"listener already accepting", briefContext, listeningSocket, nil},
		{"socket file nobody accepts on", briefContext, leftoverSocketFile, context.DeadlineExceeded},
		{"plain file in the way", briefContext, plainFilePath, context.DeadlineExceeded},
		{"no socket file", briefContext, missingSocketPath, context.DeadlineExceeded},
		{"context already done", doneContext, listeningSocket, context.Canceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := WaitForSocket(tt.ctx(t), tt.socket(t), WithPollInterval(shortPoll))
			if !errors.Is(err, tt.wantErrIs) {
				t.Errorf("WaitForSocket error = %v, want %v", err, tt.wantErrIs)
			}
		})
	}
}

func TestWaitForSocket_ReturnsWhenListenerAppears(t *testing.T) {
	t.Parallel()

	socketPath := missingSocketPath(t)

	// The listener shows up well inside the wait window.
	listeners := make(chan net.Listener, 1)

	listenCtx := t.Context()

	go func() {
		time.Sleep(3 * shortWait)

		listener, err := (&net.ListenConfig{}).Listen(listenCtx, "unix", socketPath)
		if err != nil {
			listeners <- nil

			return
		}

		listeners <- listener
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)

	start := time.Now()

	err := WaitForSocket(ctx, socketPath, WithPollInterval(shortPoll))
	if err != nil {
		t.Fatalf("WaitForSocket: %v", err)
	}

	// It returned when the listener appeared, not at the end of some fixed
	// pause.
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("WaitForSocket took %v; it should return as soon as the listener appears", elapsed)
	}

	listener := <-listeners
	if listener == nil {
		t.Fatal("the test listener never started")
	}

	_ = listener.Close()
}

func TestProbeUntilAlive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ctx       func(t *testing.T) context.Context
		socket    func(t *testing.T) string
		wantErrIs error
	}{
		{"daemon answers health", briefContext, healthDaemon, nil},
		{"any status code counts", briefContext, teapotDaemon, nil},
		{"daemon accepts but never answers", briefContext, hangingDaemon, context.DeadlineExceeded},
		{"socket file nobody accepts on", briefContext, leftoverSocketFile, context.DeadlineExceeded},
		{"no socket file", briefContext, missingSocketPath, context.DeadlineExceeded},
		{"context already done", doneContext, healthDaemon, context.Canceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ProbeUntilAlive(tt.ctx(t), tt.socket(t), DefaultHealthPath, WithPollInterval(shortPoll))
			if !errors.Is(err, tt.wantErrIs) {
				t.Errorf("ProbeUntilAlive error = %v, want %v", err, tt.wantErrIs)
			}
		})
	}
}

func TestProbeUntilAlive_ReturnsWhenDaemonAnswers(t *testing.T) {
	t.Parallel()

	// The daemon is up from the start but holds every request until release,
	// so only a later probe can succeed.
	release := make(chan struct{})

	socketPath := serveInBackground(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
		}
	}))

	timer := time.AfterFunc(3*shortWait, func() { close(release) })

	t.Cleanup(func() { timer.Stop() })

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)

	start := time.Now()

	err := ProbeUntilAlive(ctx, socketPath, DefaultHealthPath, WithPollInterval(shortPoll))
	if err != nil {
		t.Fatalf("ProbeUntilAlive: %v", err)
	}

	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("ProbeUntilAlive took %v; it should return as soon as the daemon answers", elapsed)
	}
}

func TestResolveWaitConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []WaitOption
		want time.Duration
	}{
		{"no options", nil, DefaultWaitPollInterval},
		{"positive interval", []WaitOption{WithPollInterval(time.Second)}, time.Second},
		{"zero keeps the default", []WaitOption{WithPollInterval(0)}, DefaultWaitPollInterval},
		{"negative keeps the default", []WaitOption{WithPollInterval(-time.Second)}, DefaultWaitPollInterval},
		{"nil option keeps the default", []WaitOption{nil}, DefaultWaitPollInterval},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := resolveWaitConfig(tt.opts).pollInterval; got != tt.want {
				t.Errorf("poll interval = %v, want %v", got, tt.want)
			}
		})
	}
}

// ExampleWaitForSocket waits for a listener and reports a socket that never
// shows up through the context's error.
func ExampleWaitForSocket() {
	dir, err := os.MkdirTemp("", "godmn-wait")
	if err != nil {
		panic(err)
	}

	defer func() { _ = os.RemoveAll(dir) }()

	socketPath := filepath.Join(dir, "daemon.sock")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	listener, err := (&net.ListenConfig{}).Listen(ctx, "unix", socketPath)
	if err != nil {
		panic(err)
	}

	defer func() { _ = listener.Close() }()

	fmt.Println(WaitForSocket(ctx, socketPath))

	missingCtx, cancelMissing := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelMissing()

	err = WaitForSocket(missingCtx, filepath.Join(dir, "missing.sock"))

	fmt.Println(err != nil, errors.Is(err, context.DeadlineExceeded))

	// Output:
	// <nil>
	// true true
}

// ExampleProbeUntilAlive waits for a daemon to answer its health endpoint.
func ExampleProbeUntilAlive() {
	dir, err := os.MkdirTemp("", "godmn-probe")
	if err != nil {
		panic(err)
	}

	defer func() { _ = os.RemoveAll(dir) }()

	socketPath := filepath.Join(dir, "daemon.sock")

	srv := NewSocketServer(socketPath, WithLogger(slog.New(slog.DiscardHandler)))

	go func() { _ = srv.Start(context.Background(), Lifecycle{Handler: healthMux()}) }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println(ProbeUntilAlive(ctx, socketPath, DefaultHealthPath))

	// Output:
	// <nil>
}
