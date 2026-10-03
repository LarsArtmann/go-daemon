package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// healthMux returns a mux answering GET /health with 200.
func healthMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	return mux
}

// getOverSocket issues a context-carrying GET against the socket and returns
// the response. The caller must close the body.
func getOverSocket(t *testing.T, socketPath, path string, timeout time.Duration) *http.Response {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost"+path, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	resp, err := NewHTTPClient(socketPath, timeout).Do(req)
	if err != nil {
		t.Fatalf("GET %s over socket: %v", path, err)
	}

	return resp
}

// runSocketServer starts srv with the given lifecycle and blocks until a live
// socket exists or Start exits early. The caller must cancel and drain the
// returned context/error channel (stopSocketServer does both).
func runSocketServer(
	t *testing.T,
	socketPath string,
	lifecycle Lifecycle,
	opts ...SocketServerOption,
) (*SocketServer, context.CancelFunc, <-chan error) {
	t.Helper()

	opts = append([]SocketServerOption{WithHealthPath("/health")}, opts...)

	srv := NewSocketServer(socketPath, opts...)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() { errCh <- srv.Start(ctx, lifecycle) }()

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		select {
		case err := <-errCh:
			t.Fatalf("Start exited before the socket appeared: %v", err)
		default:
		}

		// Wait for a live socket, not just any file: a pre-existing stale
		// plain file must not be mistaken for a started server.
		if info, err := os.Stat(socketPath); err == nil && info.Mode()&os.ModeSocket != 0 {
			return srv, cancel, errCh
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("server socket did not appear within timeout")

	return nil, nil, nil // unreachable
}

// stopSocketServer cancels the lifetime context and waits for Start to
// return, failing the test on error or timeout.
func stopSocketServer(t *testing.T, cancel context.CancelFunc, errCh <-chan error) {
	t.Helper()

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Start returned error during shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Start did not return within 5s of context cancel")
	}
}

func TestSocketServer_ServesAndShutsDownGracefully(t *testing.T) {
	t.Parallel()

	mux := healthMux()
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("hi")) })

	socketPath := filepath.Join(t.TempDir(), "serve.sock")

	_, cancel, errCh := runSocketServer(t, socketPath, Lifecycle{Handler: mux})

	for _, path := range []string{"/health", "/hello"} {
		resp := getOverSocket(t, socketPath, path, 2*time.Second)

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: status %d", path, resp.StatusCode)
		}

		if path == "/hello" && string(body) != "hi" {
			t.Fatalf("GET /hello body: %q", body)
		}
	}

	stopSocketServer(t, cancel, errCh)

	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket file should be removed after shutdown, stat err: %v", err)
	}
}

func TestSocketServer_CreatesSocketDirWithMode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	socketPath := filepath.Join(root, "nested", "deeper", "daemon.sock")

	_, cancel, errCh := runSocketServer(t, socketPath, Lifecycle{Handler: healthMux()})

	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}

	if info.Mode()&os.ModeSocket == 0 {
		t.Fatalf("%s is not a socket", socketPath)
	}

	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("socket permissions: got %o, want 600", perm)
	}

	stopSocketServer(t, cancel, errCh)
}

func TestSocketServer_CustomSocketMode(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "mode.sock")

	_, cancel, errCh := runSocketServer(
		t,
		socketPath,
		Lifecycle{Handler: healthMux()},
		WithSocketMode(0o660),
	)

	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o660 {
		t.Fatalf("socket permissions: got %o, want 660", perm)
	}

	stopSocketServer(t, cancel, errCh)
}

func TestSocketServer_HookOrdering(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "hooks.sock")

	var mu sync.Mutex

	order := make([]string, 0, 4)

	record := func(name string) {
		mu.Lock()
		defer mu.Unlock()

		order = append(order, name)
	}

	srv := NewSocketServer(socketPath, WithHealthPath("/health"))

	ctx, ctxCancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.Start(ctx, Lifecycle{
			Handler: healthMux(),
			Startup: func() (func(), error) {
				record("startup")

				return func() { record("startup-cleanup") }, nil
			},
			OnListen: func(context.Context) { record("on-listen") },
			OnShutdown: func() {
				record("on-shutdown")

				// The socket file must already be removed when OnShutdown runs.
				if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
					record("on-shutdown-before-socket-removal")
				}
			},
		})
	}()

	// Wait for a live socket, then cancel from the test: the shutdown
	// sequence (drain → socket removal → OnShutdown → startup cleanup) runs
	// inside Start before it returns, so the recorded order is stable by then.
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		if info, err := os.Stat(socketPath); err == nil && info.Mode()&os.ModeSocket != 0 {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	ctxCancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after cancel")
	}

	want := []string{"startup", "on-listen", "on-shutdown", "startup-cleanup"}

	if fmt.Sprint(order) != fmt.Sprint(want) {
		t.Fatalf("hook order: got %v, want %v", order, want)
	}

	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatal("socket should be removed after shutdown")
	}
}

// errRecorderBusy simulates a startup hook failing (e.g. flight recorder
// already active in the process).
var errRecorderBusy = errors.New("recorder busy")

func TestSocketServer_StartupErrorContinues(t *testing.T) {
	t.Parallel()

	onListenRan := false

	socketPath := filepath.Join(t.TempDir(), "startup-err.sock")

	srv := NewSocketServer(socketPath, WithHealthPath("/health"))

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.Start(ctx, Lifecycle{
			Handler:  healthMux(),
			Startup:  func() (func(), error) { return nil, errRecorderBusy },
			OnListen: func(context.Context) { onListenRan = true },
		})
	}()

	deadline := time.Now().Add(5 * time.Second)

	served := false

	for time.Now().Before(deadline) && !served {
		resp, err := getOverSocketQuiet(t, socketPath, "/health", time.Second)
		if err == nil {
			_ = resp.Body.Close()
			served = true
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}

	if !served {
		t.Fatal("server must serve despite startup-hook error")
	}

	if !onListenRan {
		t.Fatal("OnListen must run despite startup-hook error")
	}

	stopSocketServer(t, cancel, errCh)
}

// getOverSocketQuiet probes a socket without failing the test on dial
// errors (used while waiting for a server to come up).
func getOverSocketQuiet(_ *testing.T, socketPath, path string, timeout time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost"+path, nil)
	if err != nil {
		return nil, err
	}

	return NewHTTPClient(socketPath, timeout).Do(req)
}

func TestSocketServer_StaleSocketRemoved(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "stale.sock")

	// A leftover plain file (e.g. after a crash) blocks listen if not removed.
	if err := os.WriteFile(socketPath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	_, cancel, errCh := runSocketServer(t, socketPath, Lifecycle{Handler: healthMux()})

	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}

	if info.Mode()&os.ModeSocket == 0 {
		t.Fatal("stale plain file was not replaced by a live socket")
	}

	stopSocketServer(t, cancel, errCh)
}

func TestSocketServer_LiveSocketKept(t *testing.T) {
	t.Parallel()

	mux := healthMux()

	socketPath := filepath.Join(t.TempDir(), "live.sock")

	_, cancel, errCh := runSocketServer(t, socketPath, Lifecycle{Handler: mux})

	defer stopSocketServer(t, cancel, errCh)

	// A second server on the same path must NOT steal the live socket; its
	// listen fails with "address already in use" instead.
	second := NewSocketServer(socketPath, WithHealthPath("/health"))

	err := second.Start(context.Background(), Lifecycle{Handler: mux})
	if err == nil {
		t.Fatal("expected listen error when the socket is owned by a live daemon")
	}

	if _, statErr := os.Stat(socketPath); statErr != nil {
		t.Fatalf("live daemon's socket must be preserved, stat: %v", statErr)
	}
}

func TestSocketServer_HandlerRequired(t *testing.T) {
	t.Parallel()

	srv := NewSocketServer(filepath.Join(t.TempDir(), "nil-handler.sock"))

	err := srv.Start(context.Background(), Lifecycle{})
	if !errors.Is(err, ErrHandlerRequired) {
		t.Fatalf("expected ErrHandlerRequired, got %v", err)
	}
}

func TestSocketServer_StartupSocketDirErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		skipIf    func(t *testing.T) bool
		wantErrIs error
		wantErrIn string
	}{
		{
			name: "socket directory path is a regular file",
			setup: func(t *testing.T) string {
				t.Helper()

				blocker := filepath.Join(t.TempDir(), "blocker")
				if err := os.WriteFile(blocker, []byte("not a dir"), 0o600); err != nil {
					t.Fatalf("write blocker file: %v", err)
				}

				return filepath.Join(blocker, "daemon.sock")
			},
			wantErrIn: "create socket directory",
		},
		{
			name: "socket directory is not writable",
			setup: func(t *testing.T) string {
				t.Helper()

				dir := t.TempDir()
				if err := os.Chmod(dir, 0o555); err != nil {
					t.Fatalf("chmod dir: %v", err)
				}

				t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

				return filepath.Join(dir, "sub", "daemon.sock")
			},
			skipIf: func(t *testing.T) bool {
				t.Helper()

				return os.Geteuid() == 0
			},
			wantErrIs: fs.ErrPermission,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.skipIf != nil && tt.skipIf(t) {
				t.Skip("cannot test permission denial as root")
			}

			srv := NewSocketServer(tt.setup(t))

			err := srv.Start(t.Context(), Lifecycle{Handler: healthMux()})
			if err == nil {
				t.Fatal("expected startup error")
			}

			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("error = %v, want wrapped %v", err, tt.wantErrIs)
			}

			if tt.wantErrIn != "" && !strings.Contains(err.Error(), tt.wantErrIn) {
				t.Errorf("error = %v, want it to contain %q", err, tt.wantErrIn)
			}
		})
	}
}

func TestIsSocketAlive(t *testing.T) {
	t.Parallel()

	if IsSocketAlive(context.Background(), filepath.Join(t.TempDir(), "missing.sock"), "/health") {
		t.Fatal("nonexistent socket must not be alive")
	}

	socketPath := filepath.Join(t.TempDir(), "alive.sock")

	srv, cancel, errCh := runSocketServer(t, socketPath, Lifecycle{Handler: healthMux()})

	defer stopSocketServer(t, cancel, errCh)

	if !srv.IsSocketAlive(context.Background()) {
		t.Fatal("listening daemon's socket must be alive")
	}
}

func TestTransportOverSocket(t *testing.T) {
	t.Parallel()

	mux := healthMux()

	socketPath := filepath.Join(t.TempDir(), "transport.sock")

	_, cancel, errCh := runSocketServer(t, socketPath, Lifecycle{Handler: mux})

	defer stopSocketServer(t, cancel, errCh)

	ctx, ctxCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer ctxCancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := NewHTTPClientNoTimeout(socketPath).Do(req)
	if err != nil {
		t.Fatalf("no-timeout client GET: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestSocketServer_Options(t *testing.T) {
	t.Parallel()

	customLogger := slog.New(slog.DiscardHandler)

	tests := []struct {
		name    string
		opts    []SocketServerOption
		want    func(*SocketServer) bool
		wantMsg string
	}{
		{
			name:    "WithSocketDirMode",
			opts:    []SocketServerOption{WithSocketDirMode(0o750)},
			want:    func(s *SocketServer) bool { return s.socketDirMode == 0o750 },
			wantMsg: "socketDirMode",
		},
		{
			name:    "WithLogger",
			opts:    []SocketServerOption{WithLogger(customLogger)},
			want:    func(s *SocketServer) bool { return s.logger == customLogger },
			wantMsg: "logger",
		},
		{
			name:    "WithVersion",
			opts:    []SocketServerOption{WithVersion("v9.9.9-test")},
			want:    func(s *SocketServer) bool { return s.version == "v9.9.9-test" },
			wantMsg: "version",
		},
		{
			name:    "WithReadHeaderTimeout",
			opts:    []SocketServerOption{WithReadHeaderTimeout(3 * time.Second)},
			want:    func(s *SocketServer) bool { return s.readHeaderTimeout == 3*time.Second },
			wantMsg: "readHeaderTimeout",
		},
		{
			name:    "WithIdleTimeout",
			opts:    []SocketServerOption{WithIdleTimeout(7 * time.Second)},
			want:    func(s *SocketServer) bool { return s.idleTimeout == 7*time.Second },
			wantMsg: "idleTimeout",
		},
		{
			name:    "WithShutdownGracePeriod",
			opts:    []SocketServerOption{WithShutdownGracePeriod(2 * time.Second)},
			want:    func(s *SocketServer) bool { return s.shutdownGracePeriod == 2*time.Second },
			wantMsg: "shutdownGracePeriod",
		},
		{
			name:    "WithProbeTimeout",
			opts:    []SocketServerOption{WithProbeTimeout(50 * time.Millisecond)},
			want:    func(s *SocketServer) bool { return s.probeTimeout == 50*time.Millisecond },
			wantMsg: "probeTimeout",
		},
		{
			name: "defaults with no options",
			opts: nil,
			want: func(s *SocketServer) bool {
				return s.socketMode == DefaultSocketMode &&
					s.socketDirMode == DefaultSocketDirMode &&
					s.logger == slog.Default() &&
					s.version == "" &&
					s.readHeaderTimeout == DefaultReadHeaderTimeout &&
					s.idleTimeout == DefaultIdleTimeout &&
					s.shutdownGracePeriod == DefaultShutdownGracePeriod &&
					s.healthPath == DefaultHealthPath &&
					s.probeTimeout == DefaultProbeTimeout
			},
			wantMsg: "defaults",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := NewSocketServer("/run/test/options.sock", tt.opts...)

			if !tt.want(srv) {
				t.Errorf("option state mismatch on %s", tt.wantMsg)
			}

			if got := srv.SocketPath(); got != "/run/test/options.sock" {
				t.Errorf("SocketPath() = %q, want %q", got, "/run/test/options.sock")
			}
		})
	}
}
