package daemon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// errServerClosedIdle carries the message of net/http's unexported
// errServerClosedIdle, which ClassifyTransportError matches by text.
var errServerClosedIdle = errors.New("http: server closed idle connection")

// errDecode stands in for a failure that is not a transport error.
var errDecode = errors.New("decode response")

// outOfRangeKind is not one of the declared kinds. A var on purpose: a typed
// const would count as an enum member.
var outOfRangeKind = TransportErrorKind(-1)

// timeoutError is a net.Error whose Timeout reports true, the shape net/http
// returns for a Client.Timeout.
type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestClassifyTransportError(t *testing.T) {
	t.Parallel()

	alive := t.Context()

	canceled, cancelCanceled := context.WithCancel(t.Context())
	cancelCanceled()

	expired, cancelExpired := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	t.Cleanup(cancelExpired)

	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want TransportErrorKind
	}{
		{"nil error", alive, nil, TransportErrorNone},
		{"connection refused", alive, syscall.ECONNREFUSED, TransportErrorConnectionRefused},
		{"socket missing", alive, syscall.ENOENT, TransportErrorSocketMissing},
		{"connection reset", alive, syscall.ECONNRESET, TransportErrorReset},
		{"broken pipe", alive, syscall.EPIPE, TransportErrorReset},
		{"closed connection", alive, net.ErrClosed, TransportErrorReset},
		{"server closed idle connection", alive, errServerClosedIdle, TransportErrorReset},
		{"EOF", alive, io.EOF, TransportErrorEOF},
		{"unexpected EOF", alive, io.ErrUnexpectedEOF, TransportErrorEOF},
		{"net.Error timeout", alive, &net.OpError{Op: "dial", Err: timeoutError{}}, TransportErrorTimeout},
		{"context deadline exceeded", alive, context.DeadlineExceeded, TransportErrorTimeout},
		{"syscall timed out", alive, syscall.ETIMEDOUT, TransportErrorTimeout},
		{"context canceled", alive, context.Canceled, TransportErrorCanceled},
		{"not a transport error", alive, errDecode, TransportErrorOther},
		{
			"wrapped client error keeps its kind",
			alive,
			fmt.Errorf("daemon request: %w", &url.Error{
				Op:  "Post",
				URL: "http://localhost/v1/discover",
				Err: &net.OpError{
					Op: "dial", Net: "unix", Err: &os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED},
				},
			}),
			TransportErrorConnectionRefused,
		},
		{"canceled context wins over a refused connection", canceled, syscall.ECONNREFUSED, TransportErrorCanceled},
		{"expired context wins over a refused connection", expired, syscall.ECONNREFUSED, TransportErrorCanceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ClassifyTransportError(tt.ctx, tt.err); got != tt.want {
				t.Errorf("ClassifyTransportError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// requestError issues one GET, or POST with body when body is set, to
// http://localhost/health over client and returns the client's error.
func requestError(ctx context.Context, client *http.Client, body io.Reader) error {
	method := http.MethodGet
	if body != nil {
		method = http.MethodPost
	}

	req, err := http.NewRequestWithContext(ctx, method, "http://localhost"+DefaultHealthPath, body)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	_ = resp.Body.Close()

	return nil
}

// deadSocketFile leaves a socket file behind with no listener: what a crashed
// daemon leaves when its socket was not cleaned up.
func deadSocketFile(t *testing.T) string {
	t.Helper()

	socketPath := filepath.Join(socketTempDir(t), "dead.sock")

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

// abandonRequests serves a raw unix socket that reads the start of one
// request and closes the connection without answering, like a daemon dying
// mid-request.
func abandonRequests(t *testing.T) string {
	t.Helper()

	socketPath := filepath.Join(socketTempDir(t), "abandon.sock")

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socketPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}

		buf := make([]byte, 4096)

		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, _ = conn.Read(buf)
		_ = conn.Close()
	}()

	return socketPath
}

// hangForever serves a handler that never answers until its connection goes
// away, so every request ends by the client's own timeout or cancellation.
func hangForever(t *testing.T) string {
	t.Helper()

	socketPath := filepath.Join(socketTempDir(t), "hang.sock")

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socketPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := &http.Server{
		Handler: http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() }),
	}

	served := make(chan struct{})

	go func() {
		defer close(served)

		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		_ = server.Close()

		<-served
	})

	return socketPath
}

// TestClassifyTransportError_ClientErrors classifies the errors Go 1.27's
// net/http really returns over a unix socket, not hand-built look-alikes:
// each row produces its error through NewHTTPClient or NewHTTPClientNoTimeout
// against a socket in the state its name describes.
func TestClassifyTransportError_ClientErrors(t *testing.T) {
	t.Parallel()

	hanging := hangForever(t)

	tests := []struct {
		name string
		do   func(t *testing.T) (context.Context, error)
		want TransportErrorKind
	}{
		{
			name: "socket file does not exist",
			do: func(t *testing.T) (context.Context, error) {
				t.Helper()

				missing := filepath.Join(socketTempDir(t), "missing.sock")

				return t.Context(), requestError(t.Context(), NewHTTPClient(missing, time.Second), nil)
			},
			want: TransportErrorSocketMissing,
		},
		{
			name: "socket file without a listener",
			do: func(t *testing.T) (context.Context, error) {
				t.Helper()

				dead := deadSocketFile(t)

				return t.Context(), requestError(t.Context(), NewHTTPClient(dead, time.Second), nil)
			},
			want: TransportErrorConnectionRefused,
		},
		{
			name: "daemon closes the connection without a response",
			do: func(t *testing.T) (context.Context, error) {
				t.Helper()

				abandoning := abandonRequests(t)

				return t.Context(), requestError(t.Context(), NewHTTPClientNoTimeout(abandoning), nil)
			},
			want: TransportErrorEOF,
		},
		{
			name: "daemon closes the connection with the body unread",
			do: func(t *testing.T) (context.Context, error) {
				t.Helper()

				abandoning := abandonRequests(t)
				// Far larger than the socket buffers, so the write is still
				// in progress when the daemon side closes.
				body := io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("x"), 2<<20)))

				return t.Context(), requestError(t.Context(), NewHTTPClientNoTimeout(abandoning), body)
			},
			want: TransportErrorReset,
		},
		{
			name: "client timeout while the daemon hangs",
			do: func(t *testing.T) (context.Context, error) {
				t.Helper()

				return t.Context(), requestError(t.Context(), NewHTTPClient(hanging, 100*time.Millisecond), nil)
			},
			want: TransportErrorTimeout,
		},
		{
			name: "caller deadline while the daemon hangs",
			do: func(t *testing.T) (context.Context, error) {
				t.Helper()

				ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
				defer cancel()

				return ctx, requestError(ctx, NewHTTPClientNoTimeout(hanging), nil)
			},
			want: TransportErrorCanceled,
		},
		{
			name: "caller cancel while the daemon hangs",
			do: func(t *testing.T) (context.Context, error) {
				t.Helper()

				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				timer := time.AfterFunc(50*time.Millisecond, cancel)
				defer timer.Stop()

				return ctx, requestError(ctx, NewHTTPClientNoTimeout(hanging), nil)
			},
			want: TransportErrorCanceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, err := tt.do(t)
			if err == nil {
				t.Fatal("expected a client error")
			}

			if got := ClassifyTransportError(ctx, err); got != tt.want {
				t.Errorf("ClassifyTransportError(%v) = %v, want %v", err, got, tt.want)
			}
		})
	}
}

func TestTransportErrorKind_Unavailable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind TransportErrorKind
		want bool
	}{
		{"none", TransportErrorNone, false},
		{"canceled", TransportErrorCanceled, false},
		{"connection refused", TransportErrorConnectionRefused, true},
		{"socket missing", TransportErrorSocketMissing, true},
		{"reset", TransportErrorReset, true},
		{"eof", TransportErrorEOF, true},
		{"timeout", TransportErrorTimeout, true},
		{"other", TransportErrorOther, false},
		{"out of range", outOfRangeKind, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.kind.Unavailable(); got != tt.want {
				t.Errorf("(%v).Unavailable() = %v, want %v", tt.kind, got, tt.want)
			}
		})
	}
}

func TestTransportErrorKind_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind TransportErrorKind
		want string
	}{
		{"none", TransportErrorNone, "none"},
		{"canceled", TransportErrorCanceled, "canceled"},
		{"connection refused", TransportErrorConnectionRefused, "connection refused"},
		{"socket missing", TransportErrorSocketMissing, "socket missing"},
		{"reset", TransportErrorReset, "reset"},
		{"eof", TransportErrorEOF, "eof"},
		{"timeout", TransportErrorTimeout, "timeout"},
		{"other", TransportErrorOther, "other"},
		{"out of range", outOfRangeKind, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.kind.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGoStillUsesServerClosedIdleConnString pins the one text match in
// ClassifyTransportError: net/http's errServerClosedIdle is unexported, so
// only its message can be recognized. When this fails, a Go release renamed
// the error and isReset must follow.
func TestGoStillUsesServerClosedIdleConnString(t *testing.T) {
	t.Parallel()

	root := goRoot(t.Context())
	if root == "" {
		t.Skip("cannot locate the Go source tree: GOROOT is unset and `go env GOROOT` failed")
	}

	source, err := os.ReadFile(filepath.Join(root, "src", "net", "http", "transport.go"))
	if err != nil {
		t.Skipf("cannot read Go's net/http source: %v", err)
	}

	if !bytes.Contains(source, []byte(serverClosedIdleFragment)) {
		t.Fatalf("net/http/transport.go no longer contains %q: update isReset", serverClosedIdleFragment)
	}
}

// goRoot returns the Go source tree of the toolchain running the tests, or
// "" when it cannot be found.
func goRoot(ctx context.Context) string {
	if root := os.Getenv("GOROOT"); root != "" {
		return root
	}

	out, err := exec.CommandContext(ctx, "go", "env", "GOROOT").Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

// ExampleClassifyTransportError maps a client error onto a caller's policy:
// every kind but a canceled or unrelated failure means the daemon could not
// serve the request.
func ExampleClassifyTransportError() {
	dir, err := os.MkdirTemp("", "godmn-example")
	if err != nil {
		panic(err)
	}

	defer func() { _ = os.RemoveAll(dir) }()

	socketPath := filepath.Join(dir, "daemon.sock") // nothing ever listens here

	ctx := context.Background()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost"+DefaultHealthPath, nil)

	resp, err := NewHTTPClient(socketPath, time.Second).Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	switch kind := ClassifyTransportError(ctx, err); {
	case kind == TransportErrorNone:
		fmt.Println("served")
	case kind == TransportErrorCanceled:
		fmt.Println("caller gave up:", kind)
	case kind.Unavailable():
		fmt.Println("daemon unavailable:", kind)
	default:
		fmt.Println("request failed:", kind)
	}

	// Output:
	// daemon unavailable: socket missing
}
