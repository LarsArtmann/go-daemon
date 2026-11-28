package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
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

// runServerUntilListening is runSocketServer for tests that assert on the
// socket file itself: bind(2) creates the file a moment before Start chmods
// it, so an immediate stat can still see the pre-chmod mode. OnListen runs
// after the chmod, and this helper returns only once it has.
func runServerUntilListening(
	t *testing.T,
	socketPath string,
	lifecycle Lifecycle,
	opts ...SocketServerOption,
) (*SocketServer, context.CancelFunc, <-chan error) {
	t.Helper()

	listening := make(chan struct{})

	onListen := lifecycle.OnListen
	lifecycle.OnListen = func(ctx context.Context) {
		if onListen != nil {
			onListen(ctx)
		}

		close(listening)
	}

	srv, cancel, errCh := runSocketServer(t, socketPath, lifecycle, opts...)

	select {
	case <-listening:
	case <-time.After(5 * time.Second):
		t.Fatal("OnListen did not run: the socket never finished listening")
	}

	return srv, cancel, errCh
}

func TestSocketServer_ServesAndShutsDownGracefully(t *testing.T) {
	t.Parallel()

	mux := healthMux()
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("hi")) })

	socketPath := filepath.Join(socketTempDir(t), "serve.sock")

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

	root := socketTempDir(t)
	socketPath := filepath.Join(root, "nested", "deeper", "daemon.sock")

	_, cancel, errCh := runServerUntilListening(t, socketPath, Lifecycle{Handler: healthMux()})

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

	socketPath := filepath.Join(socketTempDir(t), "mode.sock")

	_, cancel, errCh := runServerUntilListening(
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

	socketPath := filepath.Join(socketTempDir(t), "hooks.sock")

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

	socketPath := filepath.Join(socketTempDir(t), "startup-err.sock")

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

	socketPath := filepath.Join(socketTempDir(t), "stale.sock")

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

	socketPath := filepath.Join(socketTempDir(t), "live.sock")

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

	srv := NewSocketServer(filepath.Join(socketTempDir(t), "nil-handler.sock"))

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

// errChmodDenied stands in for an os.Chmod failure on the socket (e.g. a
// security module denying the call). No portable unprivileged filesystem
// state makes chmod(2) fail on a socket the process itself just created, so
// the test substitutes the server's chmod seam.
var errChmodDenied = errors.New("chmod denied")

func TestSocketServer_ChmodErrorSurfaces(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(socketTempDir(t), "chmod-fail.sock")

	srv := NewSocketServer(socketPath)
	srv.chmodSocket = func(string, fs.FileMode) error { return errChmodDenied }

	var onListenRan, onShutdownRan bool

	err := srv.Start(t.Context(), Lifecycle{
		Handler:    healthMux(),
		OnListen:   func(context.Context) { onListenRan = true },
		OnShutdown: func() { onShutdownRan = true },
	})
	if !errors.Is(err, errChmodDenied) {
		t.Fatalf("Start error = %v, want wrapped errChmodDenied", err)
	}

	if !strings.Contains(err.Error(), "restrict socket permissions") {
		t.Fatalf("Start error = %v, want it to contain %q", err, "restrict socket permissions")
	}

	if onListenRan || onShutdownRan {
		t.Fatal("OnListen and OnShutdown must not run when chmod fails before serving")
	}

	// The listener behind the failed chmod is closed, and Go's UnixListener
	// unlinks its socket file on Close: the path must be gone ...
	if _, statErr := os.Stat(socketPath); !os.IsNotExist(statErr) {
		t.Fatalf("socket file must be unlinked with the closed listener, stat err: %v", statErr)
	}

	// ... so a successor binds the same path immediately.
	ln, listenErr := (&net.ListenConfig{}).Listen(t.Context(), "unix", socketPath)
	if listenErr != nil {
		t.Fatalf("listen on the same path after a failed chmod: %v", listenErr)
	}

	_ = ln.Close()
}

func TestSocketServer_ServeErrorRunsShutdownHooks(t *testing.T) {
	t.Parallel()

	// WithListener hands Start a listener the test owns, so Serve itself can
	// be made to fail after it began serving (TestSocketServer_LiveSocketKept
	// only reaches the listen step): close the listener under the server.
	socketPath := filepath.Join(socketTempDir(t), "serve-error.sock")

	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socketPath)
	if err != nil {
		t.Fatalf("pre-build listener: %v", err)
	}

	// Keep the socket file on Close, same as
	// TestServer_WithListener_ServesWithoutSocketManagement.
	ln.(*net.UnixListener).SetUnlinkOnClose(false)

	srv := NewSocketServer(socketPath, WithListener(ln))

	var mu sync.Mutex

	hooks := make([]string, 0, 4)

	record := func(name string) {
		mu.Lock()
		defer mu.Unlock()

		hooks = append(hooks, name)
	}

	startErr := make(chan error, 1)

	go func() {
		startErr <- srv.Start(t.Context(), Lifecycle{
			Handler: healthMux(),
			Startup: func() (func(), error) {
				record("startup")

				return func() { record("startup-cleanup") }, nil
			},
			OnListen:   func(context.Context) { record("on-listen") },
			OnShutdown: func() { record("on-shutdown") },
		})
	}()

	// Serve must be running before the listener is pulled away.
	served := false
	deadline := time.Now().Add(5 * time.Second)

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
		t.Fatal("server never came up")
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	select {
	case err := <-startErr:
		if err == nil {
			t.Fatal("Start must return the Serve error")
		}

		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("Start error = %v, want wrapped net.ErrClosed", err)
		}

		if !strings.Contains(err.Error(), "serve:") {
			t.Fatalf("Start error = %v, want it to contain %q", err, "serve:")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after the listener was closed")
	}

	mu.Lock()
	got := slices.Clone(hooks)
	mu.Unlock()

	want := []string{"startup", "on-listen", "on-shutdown", "startup-cleanup"}

	if !slices.Equal(got, want) {
		t.Fatalf("hook order: got %v, want %v", got, want)
	}
}

func TestSocketServer_DrainTimeoutStillCleansUp(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(socketTempDir(t), "drain-timeout.sock")

	handlerEntered := make(chan struct{})

	release := make(chan struct{})

	handlerReturned := make(chan struct{})

	mux := healthMux()
	// The slow handler ignores its request context on purpose: canceling the
	// lifecycle context alone unblocks ctx-aware handlers before the grace
	// period matters, which would never exercise the drain timeout.
	mux.HandleFunc("GET /slow", func(_ http.ResponseWriter, _ *http.Request) {
		close(handlerEntered)
		<-release
		close(handlerReturned)
	})

	var onShutdownRan bool

	// A grace period far shorter than the slow request: the drain must time
	// out while the handler still holds its connection.
	_, cancel, errCh := runSocketServer(
		t,
		socketPath,
		Lifecycle{
			Handler:    mux,
			OnShutdown: func() { onShutdownRan = true },
		},
		WithShutdownGracePeriod(50*time.Millisecond),
	)

	clientDone := make(chan struct{})

	go func() {
		defer close(clientDone)

		resp, err := getOverSocketQuiet(t, socketPath, "/slow", 2*time.Second)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	select {
	case <-handlerEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("slow handler was never reached")
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("a drain timeout must not fail Start: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after the drain window")
	}

	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket file must be removed even when the drain timed out, stat err: %v", err)
	}

	if !onShutdownRan {
		t.Fatal("OnShutdown must run after a timed-out drain")
	}

	// Let the abandoned handler return; its client is released with it.
	close(release)

	for _, ch := range []chan struct{}{handlerReturned, clientDone} {
		select {
		case <-ch:
		case <-time.After(5 * time.Second):
			t.Fatal("handler or client never finished after the drain timed out")
		}
	}
}

func TestSocketServer_StaleSocketRemovalFailureSurfaces(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("cannot test permission denial as root")
	}

	// A crashed predecessor's socket in a directory the daemon cannot write
	// to: CleanStaleSocket finds it dead but cannot remove it, so listen
	// must surface the leftover file instead of silently replacing it.
	dir := socketTempDir(t)
	socketPath := filepath.Join(dir, "daemon.sock")

	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socketPath)
	if err != nil {
		t.Fatalf("bind stale socket: %v", err)
	}

	// Without this the close below would unlink the socket being simulated.
	ln.(*net.UnixListener).SetUnlinkOnClose(false)

	if err := ln.Close(); err != nil {
		t.Fatalf("close stale listener: %v", err)
	}

	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}

	// Registered after t.TempDir's cleanup, so it runs first and the
	// directory is writable again when the temp dir is removed.
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	srv := NewSocketServer(socketPath)

	err = srv.Start(t.Context(), Lifecycle{Handler: healthMux()})
	if err == nil {
		t.Fatal("expected listen error: the un-removable stale socket still occupies the path")
	}

	if !strings.Contains(err.Error(), "listen on unix") {
		t.Fatalf("Start error = %v, want a listen error", err)
	}

	// The stale socket is left exactly as found.
	info, statErr := os.Stat(socketPath)
	if statErr != nil || info.Mode()&os.ModeSocket == 0 {
		t.Fatalf("stale socket must survive the failed removal, stat err: %v", statErr)
	}
}

func TestIsSocketAlive(t *testing.T) {
	t.Parallel()

	if IsSocketAlive(context.Background(), filepath.Join(socketTempDir(t), "missing.sock"), "/health") {
		t.Fatal("nonexistent socket must not be alive")
	}

	socketPath := filepath.Join(socketTempDir(t), "alive.sock")

	srv, cancel, errCh := runSocketServer(t, socketPath, Lifecycle{Handler: healthMux()})

	defer stopSocketServer(t, cancel, errCh)

	if !srv.IsSocketAlive(context.Background()) {
		t.Fatal("listening daemon's socket must be alive")
	}
}

func TestTransportOverSocket(t *testing.T) {
	t.Parallel()

	mux := healthMux()

	socketPath := filepath.Join(socketTempDir(t), "transport.sock")

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

// socketTempDir is t.TempDir for tests that bind a unix socket inside the
// directory. t.TempDir names the directory after the test, and a long test
// name under an equally long TMPDIR (a nix devShell's /tmp/nix-shell.XXXXXX)
// overflows the 107-byte sun_path limit: bind then fails with "invalid
// argument" before the code under test runs. This helper keeps the path short.
func socketTempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "gd") //nolint:usetesting // the short, test-name-free directory is the point
	if err != nil {
		t.Fatalf("create socket temp dir: %v", err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	return dir
}
