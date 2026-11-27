# Domain Language

> The ubiquitous language of this library. Every term used in code, tests,
> commits, and docs should mean exactly what this page says.

> **Verification:** verified 2026-11-27 against the cited files (read, not recalled).

This is a mechanism library, so its domain is deliberately small: a daemon's
lifecycle, the socket it lives on, the encodings it negotiates, and the
streams it frames. Business domains belong to the consuming daemon.

## Core terms

| Term                   | Definition                                                                                                                                                                                        | Where used in code                         |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| Daemon                 | A long-lived process serving a local client over a socket, supervised by systemd/launchd/another service manager. The library supplies the mechanism; the consumer supplies policy.               | `doc.go`, `socket.go`                      |
| Server                 | The listener-agnostic lifecycle runner: `NewServer(network, addr)` or `NewSocketServer(path)`. `SocketServer` is an alias, kept for the unix-first name.                                          | `socket.go:60`, `socket.go:87`             |
| Lifecycle              | The ordered start/stop contract a `Server` executes: stale-socket cleanup → Startup → listen/chmod → OnListen → READY → serve → STOPPING → drain → socket removal → OnShutdown → startup cleanup. | `socket.go:204-244`                        |
| Hook                   | A consumer callback in a `Lifecycle`: `Startup`, `OnListen`, `OnShutdown`. Hooks are ordering contracts, not event bus subscriptions.                                                             | `socket.go:220-242`                        |
| Startup cleanup        | The function returned by `Startup`; runs last, after `OnShutdown`, to stop startup-only machinery (for example a flight recorder).                                                                | `socket.go:230-236`                        |
| Stale socket           | A socket file left by a dead predecessor. `Start` probes it and removes it before listening; a live socket is never removed.                                                                      | `socket.go:244`, `socket.go:401`           |
| Liveness probe         | An HTTP request to the health path (default `GET /health`) used to decide whether a socket file belongs to a live daemon.                                                                         | `socket.go:418-429`                        |
| Graceful drain         | The shutdown sequence inside the grace period: stop accepting, let in-flight requests finish, then remove the socket file.                                                                        | `socket.go:244`, `WithShutdownGracePeriod` |
| Grace period           | The maximum time the drain may take (`WithShutdownGracePeriod`, default 5s). It must fit inside systemd's `TimeoutStopSec`.                                                                       | `socket.go:134`                            |
| Socket path            | The filesystem path a unix-domain server listens on. Owned and removed by the server when it manages the socket.                                                                                  | `socket.go:190-196`, `socket.go:351`       |
| Socket mode / dir mode | The permissions applied to the socket file (`0600` default) and its directory (`0755` default) — operator-visible security policy.                                                                | `socket.go:100-107`                        |
| READY                  | The `sd_notify` message (`READY=1`) sent after `listen(2)` succeeds and `OnListen` ran. Signals that the socket is accepting, not merely that the process exists.                                 | `notify.go:68`, `socket.go:244`            |
| STOPPING               | The `sd_notify` message (`STOPPING=1`) sent when shutdown begins, before the drain.                                                                                                               | `notify.go:75`                             |
| WATCHDOG ping          | The `sd_notify` keepalive (`WATCHDOG=1`) sent every `WatchdogSec/2`, the interval `sd_watchdog_enabled(3)` recommends.                                                                            | `notify.go:84-111`                         |
| Notify counters        | Process-global atomics (`ReadySent`, `StoppingSent`, `WatchdogPings`, `NotifyFailures`) that make sd_notify observable without a hook API.                                                        | `notify.go:37-55`                          |
| Watchdog interval      | The recommended ping interval derived from `WATCHDOG_USEC`; half the configured watchdog timeout.                                                                                                 | `notify.go:111`                            |
| Codec                  | A value that encodes and decodes one wire encoding (`Encoding()`), optionally into a pooled buffer. The library consumes go-codec's interface.                                                    | `negotiation.go:23`, `jsonv2.go:33`        |
| Encoding               | The wire format identity: JSON or CBOR. An encoding is not a media type; media types map onto it.                                                                                                 | `negotiation.go:12`, `jsonv2.go:50`        |
| Request codec          | The codec chosen from the request's `Content-Type`. Absent or unrecognized means JSON.                                                                                                            | `negotiation.go:73`                        |
| Response codec         | The codec chosen from the request's `Accept` header and the request codec. Wildcards echo the request codec.                                                                                      | `negotiation.go:98`                        |
| Structured suffix      | An RFC 6839 suffix (`+json`, `+cbor`) that resolves to its base encoding, so `application/vnd.acme+cbor` is CBOR.                                                                                 | `negotiation.go:178-190`                   |
| Exact match            | Media-type matching after parameter stripping and lowercasing. `application/cbor-seq` is not CBOR.                                                                                                | `negotiation.go:178`                       |
| q-value                | The RFC 9110 quality weight on an `Accept` mention. Higher wins; a tied JSON/CBOR pair prefers CBOR.                                                                                              | `negotiation.go:154`                       |
| Refusal (`q=0`)        | An explicit "never serve this" quality. A refused encoding is substituted with the other supported encoding, never served.                                                                        | `negotiation.go:137`                       |
| Wildcard               | `*/*` or `application/*`: no preference, so the response echoes the request codec.                                                                                                                | `negotiation.go:98`                        |
| `Vary: Accept`         | The cache-correctness header `WriteEncoded` adds to every written response, because the response encoding may depend on `Accept`.                                                                 | `write.go:29`                              |
| Wire shape             | The bytes an encoding produces: nil slice `[]` vs `null`, duration representation, map ordering. A codec choice, not an encoding choice.                                                          | `jsonv2.go:33-67`, `CHANGELOG.md`          |
| Pooled-buffer write    | Writing a response through a `sync.Pool`-recycled buffer instead of allocating per response.                                                                                                      | `write.go:16-59`                           |
| SSE framing            | The WHATWG event-stream framing layer: line terminators, BOM, comments, multi-line data joining, blank-line dispatch, size cap. Never payload decoding.                                           | `sse.go:42-95`                             |
| Event dispatch         | The call to the caller's `emit` with one event's joined data payload, made only on a blank line. An unterminated final frame never dispatches.                                                    | `sse.go:42`, `sse.go:196`                  |
| Heartbeat comment      | An SSE comment line (empty field name), ignored by the framer; used by servers to keep intermediaries open.                                                                                       | `sse.go:42-95`                             |
| Max event bytes        | The per-line size cap (`DefaultMaxSSEBytes`, 1 MiB). Over-cap input fails wrapping `bufio.ErrTooLong` rather than truncating.                                                                     | `sse.go:12-15`                             |
| Pooled transport       | The unix-socket `http.Client` with tuned keep-alive and idle limits; the no-timeout variant is for streaming responses.                                                                           | `transport.go:25-51`                       |
| Transport error kind   | The classification of a client-side failure against a daemon (`Canceled`, `Timeout`, `ConnectionRefused`, socket missing, reset, EOF, other).                                                     | `transport_error.go:16-118`                |
| Unavailable            | The one verdict a consumer maps onto its own sentinel: the daemon did not answer (as opposed to the caller giving up).                                                                            | `transport_error.go:92`                    |
| Wait helper            | A context-bounded poll that returns when a socket accepts (`WaitForSocket`) or a daemon answers its health probe (`ProbeUntilAlive`).                                                             | `wait.go:43-53`                            |
| Flight recorder        | An in-memory `runtime/trace` buffer with a one-shot slow-request trigger, Prometheus metrics, and a rate-limited trace download handler.                                                          | `flightrecorder.go:57-395`                 |
| Snapshot               | A written `go tool trace` capture (optionally gzip-compressed) produced by the flight recorder.                                                                                                   | `flightrecorder.go:298-329`                |
| Slow-request trigger   | `MaybeCaptureOnSlowRequest`: capture once when a request exceeds the threshold; the caller's cancellation must not drop the capture.                                                              | `flightrecorder.go:329`                    |
| Trace handler          | The HTTP handler that streams a trace snapshot, with a TOCTOU-safe rate limit and a 404 when disabled.                                                                                            | `flightrecorder.go:395`                    |
| Mechanism, not policy  | The library owns sockets, lifecycle ordering, codecs, and framing. Handlers, caching, business rules, packaging, and supervision stay with the consumer.                                          | `doc.go`, `README.md`                      |
| Consumer               | A daemon that imports this library. The primary consumer is `project-discovery-daemon`, whose metric names and lifecycle tests this library preserves.                                            | `AGENTS.md`, `README.md`                   |

## Bounded contexts

Terms that carry a narrower meaning inside one subsystem:

| Term     | Lifecycle context                          | Client/stream context                                        |
| -------- | ------------------------------------------ | ------------------------------------------------------------ |
| Start    | `Server.Start` runs the whole lifecycle    | A trace or SSE stream "starting" is not a lifecycle start    |
| Probe    | `IsSocketAlive` probes a socket file       | `ProbeUntilAlive` polls until the health probe answers       |
| Snapshot | Flight-recorder trace capture              | A test snapshot of a value is unrelated                      |
| Stream   | SSE event stream framed by `ParseSSEData`  | Trace file bytes streamed by `TraceHandler`                  |
| Timeout  | `WithProbeTimeout` bounds a liveness probe | `TransportErrorTimeout` is the client's own request deadline |

## Anti-language (terms we avoid)

| Avoid                               | Use instead                               | Why                                                                |
| ----------------------------------- | ----------------------------------------- | ------------------------------------------------------------------ |
| "notifier" (as a type)              | `NotifyReady`/`NotifyStopping`/`Watchdog` | sd_notify is process-global state; a per-instance type would lie   |
| "server started" before `listen(2)` | "READY"                                   | The contract is readiness, not process existence                   |
| "SSE parser" (for payloads)         | "SSE framer"                              | `ParseSSEData` never decodes payload bytes                         |
| "JSON vs CBOR negotiation"          | "encoding negotiation"                    | Media types map onto encodings; codecs are injectable per encoding |
| "handler framework"                 | "mechanism"                               | This library does not own routing or business handlers             |

## Deprecated terms

| Old term                               | Current term                    | Why changed                                                                                                           |
| -------------------------------------- | ------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| "the socket server" as the only server | `Server` / `SocketServer` alias | v0.4.0 made the lifecycle listener-agnostic (`NewServer`, `WithListener`); unix is the default, not the limit         |
| "json/v1 default"                      | "default codec wire shape"      | On Go 1.27 + go-codec v0.3.3 the default codec emits json/v2 shapes; the shape is a codec choice, not a backend label |
| "private-module pin"                   | "proxy pin"                     | The repository is public since 2026-11-27; the flake input and source-preparation wrapper are gone                    |
