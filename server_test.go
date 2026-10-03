package daemon

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// tcpHealthGet fetches path over TCP from 127.0.0.1:port.
func tcpHealthGet(t *testing.T, port int, path string) (*http.Response, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	return client.Do(req)
}

// freeTCPPort returns a kernel-assigned free port (closed listener; small
// reuse race accepted, standard for port-picking tests).
func freeTCPPort(t *testing.T) int {
	t.Helper()

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}

	defer func() { _ = ln.Close() }()

	return ln.Addr().(*net.TCPAddr).Port
}

func TestServer_TCP_ServesAndShutsDownGracefully(t *testing.T) {
	t.Parallel()

	mux := healthMux()
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("hi")) })

	port := freeTCPPort(t)

	srv := NewServer("tcp", fmt.Sprintf("127.0.0.1:%d", port), WithVersion("vtcp"))

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() { errCh <- srv.Start(ctx, Lifecycle{Handler: mux}) }()

	deadline := time.Now().Add(5 * time.Second)

	served := false

	for time.Now().Before(deadline) && !served {
		resp, err := tcpHealthGet(t, port, "/health")
		if err == nil {
			_ = resp.Body.Close()
			served = true

			continue
		}

		select {
		case err := <-errCh:
			t.Fatalf("Start: %v", err)
		default:
		}

		time.Sleep(10 * time.Millisecond)
	}

	if !served {
		t.Fatal("TCP server never came up")
	}

	resp, err := tcpHealthGet(t, port, "/hello")
	if err != nil {
		t.Fatalf("GET /hello: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if string(body) != "hi" {
		t.Fatalf("GET /hello body: %q", body)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after cancel")
	}
}

func TestServer_TCP_HookOrdering(t *testing.T) {
	t.Parallel()

	port := freeTCPPort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	var order []string

	record := func(name string) { order = append(order, name) }

	srv := NewServer("tcp", addr)

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

				// Contract: OnShutdown runs only after the listener is
				// gone, so a new dial must be refused.
				dialer := net.Dialer{Timeout: 100 * time.Millisecond}

				conn, dialErr := dialer.DialContext(context.Background(), "tcp", addr)
				if dialErr == nil {
					_ = conn.Close()

					record("listener-still-open-at-on-shutdown")
				}
			},
		})
	}()

	deadline := time.Now().Add(5 * time.Second)

	served := false

	for time.Now().Before(deadline) && !served {
		resp, err := tcpHealthGet(t, port, "/health")
		if err == nil {
			_ = resp.Body.Close()
			served = true
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}

	if !served {
		t.Fatal("TCP server never came up")
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
}

func TestServer_TCP_ListenErrorSurfaces(t *testing.T) {
	t.Parallel()

	// Hold a port, then ask a Server for the same one: the listen error must
	// surface from Start (not hang, not be swallowed).
	holder, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("hold port: %v", err)
	}

	defer func() { _ = holder.Close() }()

	port := holder.Addr().(*net.TCPAddr).Port

	onShutdownRan := false

	srv := NewServer("tcp", fmt.Sprintf("127.0.0.1:%d", port))

	startErr := srv.Start(t.Context(), Lifecycle{
		Handler:    healthMux(),
		OnShutdown: func() { onShutdownRan = true },
	})
	if startErr == nil {
		t.Fatal("Start must fail when the port is taken")
	}

	if !strings.Contains(startErr.Error(), "listen on tcp") {
		t.Fatalf("Start error should be a listen error, got: %v", startErr)
	}

	if onShutdownRan {
		t.Fatal("OnShutdown must not run when listen fails before serving")
	}
}

func TestServer_TCP_RestartAfterShutdown(t *testing.T) {
	t.Parallel()

	// TCP analog of the stale-socket cleanup tests: a crashed predecessor
	// must not block the successor on the same port.
	port := freeTCPPort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	first := NewServer("tcp", addr)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() { errCh <- first.Start(ctx, Lifecycle{Handler: healthMux()}) }()

	deadline := time.Now().Add(5 * time.Second)

	served := false

	for time.Now().Before(deadline) && !served {
		resp, err := tcpHealthGet(t, port, "/health")
		if err == nil {
			_ = resp.Body.Close()
			served = true
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}

	if !served {
		t.Fatal("first server never came up")
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("first Start: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first Start did not return after cancel")
	}

	second := NewServer("tcp", addr)

	ctx2 := t.Context()

	errCh2 := make(chan error, 1)

	go func() { errCh2 <- second.Start(ctx2, Lifecycle{Handler: healthMux()}) }()

	deadline = time.Now().Add(5 * time.Second)

	served = false

	for time.Now().Before(deadline) && !served {
		resp, err := tcpHealthGet(t, port, "/health")
		if err == nil {
			_ = resp.Body.Close()
			served = true
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}

	if !served {
		t.Fatal("successor server never came up on the reused port")
	}
}

func TestServer_WithListener_ServesWithoutSocketManagement(t *testing.T) {
	t.Parallel()

	socketPath := filepath.Join(t.TempDir(), "caller-owned.sock")

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "unix", socketPath)
	if err != nil {
		t.Fatalf("pre-build listener: %v", err)
	}

	// Go's UnixListener unlinks its socket file when closed (including via
	// server.Shutdown). Disable that so the assertion below isolates what
	// Start itself does: with WithListener it must never touch the
	// filesystem. (A socket-activation listener from a raw fd has no unlink
	// behavior at all — this is the closest deterministic equivalent.)
	ln.(*net.UnixListener).SetUnlinkOnClose(false)

	srv := NewSocketServer(socketPath, WithListener(ln))

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() { errCh <- srv.Start(ctx, Lifecycle{Handler: healthMux()}) }()

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
		t.Fatal("WithListener server never came up")
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after cancel")
	}

	// The caller owns unlink: the socket file must still exist after
	// shutdown (with a live listener, Start never touches the filesystem).
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("caller-owned socket file must survive shutdown, stat err: %v", err)
	}
}
