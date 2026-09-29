# go-daemon

Production-proven building blocks for superb long-lived daemons served over
unix domain sockets — extracted from
[project-discovery-daemon](https://github.com/LarsArtmann/project-discovery-daemon),
whose lifecycle ordering, shutdown-regression tests, and metric names this
library preserves.

The package is mechanism, not policy: it owns sockets, lifecycle ordering,
codecs, and framing. Domain handlers, caching, and business logic stay with
your daemon.

## What you get

| Piece                                                                                         | What it does                                                                                                                                                                                                                                                                                |
| --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SocketServer`                                                                                | Full socket lifecycle: stale-socket cleanup, directory creation, `0600` permissions, `sd_notify` READY/STOPPING/WATCHDOG, graceful drain, socket removal on every exit path — with an explicit, ordered `Lifecycle` hook contract (`Startup` → `OnListen` → `OnShutdown` → startup cleanup) |
| `NewHTTPClient` / `NewHTTPClientNoTimeout`                                                    | Unix-socket HTTP transports for clients (pooled keep-alive connections tuned for socket traffic; no-timeout variant for SSE streams)                                                                                                                                                        |
| `ResolveRequestCodec` / `ResolveResponseCodec` / `WriteEncoded` / `WriteJSON` / `WriteErrorf` | JSON/CBOR content negotiation with pooled-buffer response writes                                                                                                                                                                                                                            |
| `ParseSSEData`                                                                                | WHATWG-spec SSE framing (multi-line data joins, heartbeat comments, one-space stripping, size limits) — decode payloads your way                                                                                                                                                            |
| `FlightRecorder`                                                                              | Execution-trace flight recorder with slow-request auto-trigger, Prometheus metrics, and a rate-limited `TraceHandler` that streams `go tool trace` files                                                                                                                                    |
| `IsSocketAlive`                                                                               | Liveness probe to detect whether a socket file belongs to a live daemon                                                                                                                                                                                                                     |

## Quickstart

```go
package main

import (
	"context"
	"log/slog"
	"net/http"

	godmn "github.com/LarsArtmann/go-daemon"
)

func main() {
	srv := godmn.NewSocketServer(
		"/run/mydaemon/daemon.sock",
		godmn.WithVersion("1.0.0"),
		godmn.WithLogger(slog.Default()),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		godmn.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	if err := srv.Start(context.Background(), godmn.Lifecycle{Handler: mux}); err != nil {
		slog.Error("daemon exited", "error", err)
	}
}
```

Client side:

```go
client := godmn.NewHTTPClient("/run/mydaemon/daemon.sock", 5*time.Second)
resp, err := client.Do(req) // dials the unix socket; host in URLs is ignored
```

## Why unix sockets (and `/run`)?

The daemon's socket is `0600` by default: only the owning user may talk to it.
`/run` is the FHS-standard location for runtime sockets — created by
systemd/tmpfiles at boot, cleared on reboot. `/tmp` is world-writable and
invites symlink attacks.

## License

MIT — see [LICENSE](LICENSE).
