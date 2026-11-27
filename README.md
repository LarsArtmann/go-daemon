# go-daemon

Production-proven building blocks for superb long-lived daemons served over
unix domain sockets — extracted from
[project-discovery-daemon](https://github.com/LarsArtmann/project-discovery-daemon),
whose lifecycle ordering, shutdown-regression tests, and metric names this
library preserves.

The package is mechanism, not policy: it owns sockets, lifecycle ordering,
codecs, and framing. Domain handlers, caching, and business logic stay with
your daemon.

## Install

```bash
go get github.com/LarsArtmann/go-daemon
```

Requires Go 1.27 or newer.

## What you get

| Piece                                                                                         | What it does                                                                                                                                                                                                                                                                                |
| --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SocketServer`                                                                                | Full socket lifecycle: stale-socket cleanup, directory creation, `0600` permissions, `sd_notify` READY/STOPPING/WATCHDOG, graceful drain, socket removal on every exit path — with an explicit, ordered `Lifecycle` hook contract (`Startup` → `OnListen` → `OnShutdown` → startup cleanup) |
| `NewServer` / `WithListener`                                                                  | The same lifecycle — hooks, graceful drain, sd_notify — on any network (e.g. `NewServer("tcp", "127.0.0.1:8080")`), or on a pre-built listener with zero filesystem management (`SocketServer` is a type alias of `Server`)                                                                 |
| `NotifyReady` / `NotifyStopping` / `Watchdog` / `ReadNotifyCounters`                          | Standalone sd_notify integration for any `http.Server`: readiness, shutdown, watchdog keepalive at `WatchdogSec/2` (per `sd_watchdog_enabled(3)`), and process-global observability counters — silent no-ops without systemd                                                                |
| `NewHTTPClient` / `NewHTTPClientNoTimeout`                                                    | Unix-socket HTTP transports for clients (pooled keep-alive connections tuned for socket traffic; no-timeout variant for SSE streams)                                                                                                                                                        |
| `ResolveRequestCodec` / `ResolveResponseCodec` / `WriteEncoded` / `WriteJSON` / `WriteErrorf` | JSON/CBOR content negotiation — exact media-type matching with RFC 6839 structured suffixes, Accept q-value ranking, `q=0` refusals never served, `Vary: Accept` on every written response — with pooled-buffer response writes                                                             |
| `WithJSONCodec` / `WithCBORCodec` / `JSONv2Codec`                                             | Injectable negotiation codecs plus an `encoding/json/v2` adapter with caller-chosen marshal options — adopt negotiation without leaving json/v2 semantics (omitzero, deterministic map ordering, integer-nanos durations)                                                                   |
| `ParseSSEData`                                                                                | WHATWG-spec SSE framing (multi-line data joins, heartbeat comments, one-space stripping, size limits) — pinned to the WPT/Chromium conformance corpus, so every dispatch decision is checked against the same vectors that test the reference parser — decode payloads your way             |
| `FlightRecorder`                                                                              | Execution-trace flight recorder with slow-request auto-trigger, Prometheus metrics, and a rate-limited `TraceHandler` that streams `go tool trace` files                                                                                                                                    |
| `IsSocketAlive`                                                                               | Liveness probe to detect whether a socket file belongs to a live daemon                                                                                                                                                                                                                     |
| `WaitForSocket` / `ProbeUntilAlive`                                                           | Context-bounded wait helpers for tests and restart loops: `WaitForSocket` returns once a listener accepts on the path (a leftover socket file does not count), `ProbeUntilAlive` once a daemon answers the health probe                                                                     |

## Quickstart

```go
package main

import (
	"context"
	"io"
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
	mux.HandleFunc("POST /items", func(w http.ResponseWriter, r *http.Request) {
		reqCodec := godmn.ResolveRequestCodec(r) // from Content-Type; JSON fallback

		body, err := io.ReadAll(r.Body)
		if err != nil {
			godmn.WriteErrorf(w, http.StatusBadRequest, "read request: %v", err)
			return
		}

		var item map[string]any
		if err := reqCodec.Decode(body, &item); err != nil {
			godmn.WriteErrorf(w, http.StatusBadRequest, "decode request: %v", err)
			return
		}

		// Respond in the encoding the client asked for (Accept),
		// falling back to the request's encoding.
		respCodec := godmn.ResolveResponseCodec(r, reqCodec)
		godmn.WriteEncoded(w, http.StatusCreated, respCodec, item)
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

## Non-goals

This library deliberately stops at the mechanism layer. It does not provide
domain handlers, caching, or business logic; it does not package, deploy, or
supervise your daemon (systemd owns that — but see
[docs/supervision.md](docs/supervision.md) for recommended unit settings,
including how to run the daemon at a lower resource priority); and it listens
on unix domain sockets only — no TCP listener, no TLS termination.
Longer-term ideas and raw directions live in [ROADMAP.md](ROADMAP.md).

## Security

See [SECURITY.md](SECURITY.md) for the supported-versions policy and how to
report vulnerabilities privately.

## License

MIT — see [LICENSE](LICENSE).
