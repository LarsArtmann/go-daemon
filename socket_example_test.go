package daemon

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ExampleNewSocketServer shows the full daemon round trip: construct a
// server on a unix socket, serve a handler until the context is canceled,
// call it over that socket, and shut down gracefully (the socket file is
// removed on every exit path).
func ExampleNewSocketServer() {
	dir, err := os.MkdirTemp("", "go-daemon-example")
	if err != nil {
		panic(err)
	}

	defer func() { _ = os.RemoveAll(dir) }()

	socketPath := filepath.Join(dir, "example.sock")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello from go-daemon"))
	})

	srv := NewSocketServer(socketPath, WithVersion("vexample"))

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)

	go func() { errCh <- srv.Start(ctx, Lifecycle{Handler: mux}) }()

	for {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}

		select {
		case err := <-errCh:
			panic(err)
		case <-time.After(time.Second):
			panic("socket never appeared")
		default:
		}

		time.Sleep(time.Millisecond)
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer

				return d.DialContext(ctx, "unix", socketPath)
			},
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/hello", nil)
	if err != nil {
		panic(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 64)

	n, _ := resp.Body.Read(buf)
	_ = resp.Body.Close()

	fmt.Println(resp.StatusCode)
	fmt.Println(string(buf[:n]))

	cancel()

	fmt.Println(<-errCh)

	// Output:
	// 200
	// hello from go-daemon
	// <nil>
}
