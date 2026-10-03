package daemon

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// ExampleNotifyReady shows the minimal type=notify wiring for any
// http.Server (not just SocketServer): report readiness right before
// serving, keep the watchdog pinged on a background goroutine, and report
// STOPPING on the way down. Without systemd (unset NOTIFY_SOCKET) every
// call is a silent no-op, so the same binary runs foreground unchanged.
func ExampleNotifyReady() {
	logger := slog.New(slog.DiscardHandler)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello from go-daemon"))
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}

	NotifyReady(logger)

	if interval, ok := WatchdogInterval(logger); ok {
		go Watchdog(ctx, logger, interval)
	}

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	defer func() { _ = ln.Close() }()

	go func() { _ = server.Serve(ln) }()

	// ... serve until shutdown ...
	_ = server.Shutdown(context.Background())

	NotifyStopping(logger)

	// Output:
}
