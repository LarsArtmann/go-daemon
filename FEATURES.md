# Features

Honest inventory of what this library provides. Statuses answer one question:
does working code exist, and if not, why not?

**Verification basis:** re-audited 2026-11-27 against Go 1.27.3 locally
(CI builds with 1.27.4) with the full gate green — `go build`, `go vet`,
`go test ./... -race -count=1`, `golangci-lint run` (strict config),
`nix build .#default`, `nix run .#check-readme`, `nix run .#check-docs`,
`docs-gates run .`. Every FULLY_FUNCTIONAL row below is backed by tests in
that suite; the coverage numbers below were reproduced with CI's exact
flags (`-covermode=atomic -coverprofile`).

**Docs pages stay out of this inventory.** `docs/supervision.md`,
`docs/DOMAIN_LANGUAGE.md`, and the decision records are process
documentation, not library features; adding them as rows here would inflate
the counts without changing what a consumer can call. (Decision recorded
2026-11-27; the 2026-10-08 report asked.)

**Coverage (measured 2026-11-27, `go test -covermode=atomic`):** 96.7% of
statements total. The CI floor (80%, `ci.yml`) is real and passing, so it
stays — no blind raise. The weakest remaining functions are
`FlightRecorder.Start` and `SnapshotToFile` (75.0%, error branches), SSE
`consumeCR` (83.3%), `TraceHandler` (85.2%), and `isSocketAlive` (88.9%).

## Socket lifecycle

| Feature                                                                                                                                                                                                                           | Status           | Evidence                                                                                                                                                                                                                                                                     |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Unix-socket daemon server: stale-socket cleanup, socket-dir creation, `0600` default permissions                                                                                                                                  | FULLY_FUNCTIONAL | `socket.go:244-350,358-416`; tests `TestSocketServer_CreatesSocketDirWithMode`, `TestSocketServer_CustomSocketMode`, `TestSocketServer_StaleSocketRemoved`, `TestSocketServer_LiveSocketKept`                                                                                |
| Listener-agnostic server: `NewServer(network, addr)` (TCP or any network) and `WithListener` (pre-built listener, zero filesystem management) run the same lifecycle on every network; `SocketServer` is a type alias of `Server` | FULLY_FUNCTIONAL | `socket.go:60,156-188,244`; tests `TestServer_TCP_ServesAndShutsDownGracefully`, `TestServer_TCP_HookOrdering`, `TestServer_TCP_ListenErrorSurfaces`, `TestServer_TCP_RestartAfterShutdown`, `TestServer_WithListener_ServesWithoutSocketManagement`                         |
| Standalone sd_notify API: `NotifyReady`/`NotifyStopping`/`Watchdog`/`WatchdogInterval` (WatchdogSec/2 per sd_watchdog_enabled(3)) plus `ReadNotifyCounters` process-global counters — usable without `SocketServer`               | FULLY_FUNCTIONAL | `notify.go:68-146`; tests `TestNotifyReady`, `TestNotifyStopping`, `TestWatchdog_ExitsOnContextCancel`, `TestWatchdog_SendsWatchdogNotifications`, `TestWatchdogInterval`, `TestNotifyWithoutSystemdIsSilent`; decision `docs/decisions/2026-10-03_v040-notify-api-shape.md` |
| systemd integration: `sd_notify` READY/STOPPING, watchdog keepalive at `WatchdogSec/2`, and a deferrable READY (`Lifecycle.DeferReady` + `Server.MarkReady`) for daemons whose readiness is more than a listening socket          | FULLY_FUNCTIONAL | `notify.go:68-111` + wiring `socket.go:255`, `socket.go:316`; tests `TestNotifyReady`, `TestWatchdog_SendsWatchdogNotifications`, `TestSocketServer_DeferReady`, `TestSocketServer_ReadyByDefault`                                                                           |
| Graceful drain with configurable grace period; socket removed on every exit path                                                                                                                                                  | FULLY_FUNCTIONAL | `socket.go:313-347`; tests `TestSocketServer_ServesAndShutsDownGracefully`, `TestSocketServer_DrainTimeoutStillCleansUp`                                                                                                                                                     |
| Lifecycle hook contract: `Startup` → `OnListen` → `OnShutdown` → startup cleanup, test-enforced                                                                                                                                   | FULLY_FUNCTIONAL | `socket.go:204-244`; tests `TestSocketServer_HookOrdering`, `TestServer_TCP_HookOrdering`, `TestSocketServer_StartupErrorContinues`                                                                                                                                          |
| Stale-socket liveness probe (`IsSocketAlive`/`CleanStaleSocket`, configurable health path/timeout)                                                                                                                                | FULLY_FUNCTIONAL | `socket.go:401-425`; tests `TestIsSocketAlive`, `TestSocketServer_StaleSocketRemoved`, `TestSocketServer_LiveSocketKept`                                                                                                                                                     |
| Wait helpers: `WaitForSocket(ctx, path)` returns once a listener accepts on the path; `ProbeUntilAlive(ctx, path, healthPath)` once a daemon answers the health probe                                                             | FULLY_FUNCTIONAL | `wait.go`; tests `TestWaitForSocket`, `TestProbeUntilAlive`                                                                                                                                                                                                                  |

## Client transport

| Feature                                                                                         | Status           | Evidence                                                                                                                                         |
| ----------------------------------------------------------------------------------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Unix-socket HTTP clients: `NewHTTPClient` (timeout) and `NewHTTPClientNoTimeout` (SSE streams), | FULLY_FUNCTIONAL | `transport.go:25-51`; test `TestTransportOverSocket`                                                                                             |
| pooled keep-alive tuned for socket traffic                                                      |                  |                                                                                                                                                  |
| Transport-error classification: `ClassifyTransportError(ctx, err)` → `TransportErrorKind`       | FULLY_FUNCTIONAL | `transport_error.go`; tests `TestClassifyTransportError`, `TestClassifyTransportError_ClientErrors`, `TestGoStillUsesServerClosedIdleConnString` |

## Content negotiation

| Feature                                                                                                                                                                                                                                  | Status           | Evidence                                                                                                                                                                                                |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Request codec from `Content-Type`: JSON default, CBOR via exact or RFC 6839 structured-suffix type                                                                                                                                       | FULLY_FUNCTIONAL | `negotiation.go` (`ResolveRequestCodec`, `matchesEncoding`, `CodecForContentType`); tests `TestResolveRequestCodec`, `TestCodecForContentType`, `TestMatchesEncoding`                                   |
| Response codec from `Accept`: q-value ranking, tie prefers CBOR, `q=0` refusals never served,                                                                                                                                            | FULLY_FUNCTIONAL | `negotiation.go` (`ResolveResponseCodec`, `acceptQuality`, `qualityParam`); golden matrix + refusal property + fuzz; live                                                                               |
| wildcards echo the request codec                                                                                                                                                                                                         |                  | consumer API — project-discovery-daemon `content_negotiation.go:15-19` calls both resolvers                                                                                                             |
| Pooled-buffer response writers: `WriteEncoded`/`WriteJSON`/`WriteErrorf`, plus codec-aware `WriteJSONWithCodec`/`WriteErrorfWithCodec` so a negotiated codec carries error bodies too                                                    | FULLY_FUNCTIONAL | `write.go` (`bufPool`, `WriteEncoded`, `WriteJSON`, `WriteErrorf`, codec-aware variants); tests `TestWriteEncoded_JSONAndCBOR`, `TestWriteErrorf`, `TestWriteJSONWithCodec`, `TestWriteErrorfWithCodec` |
| `Vary: Accept` emission from every `WriteEncoded` response (cache correctness for Accept-negotiated encodings)                                                                                                                           | FULLY_FUNCTIONAL | `write.go` (`WriteEncoded` doc contract + `Header().Add`); tests `TestWriteEncoded_VaryAccept` (unconditional, combines with handler Vary, error path, negotiated round trip)                           |
| Injectable negotiation codecs: `WithJSONCodec`/`WithCBORCodec` substitute what `ResolveRequestCodec`/`ResolveResponseCodec`/`CodecForContentType` resolve to (defaults and ranking contract unchanged)                                   | FULLY_FUNCTIONAL | `negotiation.go` (`NegotiationOption`, `resolveCodecs`); tests `TestResolveResponseCodec_InjectedCodecs`, `TestCodecForContentType_InjectedCodecs`                                                      |
| `JSONv2Codec`: `encoding/json/v2` adapter implementing `codec.Codec` + `codec.BufferEncoder` with caller-chosen marshal/unmarshal options, pinning either json/v2 semantics or v1 wire shapes independent of the default codec's backend | FULLY_FUNCTIONAL | `jsonv2.go` (`JSONv2Codec`); tests `TestJSONv2Codec_Encode`, `TestJSONv2Codec_EncodeToBuffer`, `TestJSONv2Codec_Decode`, `TestJSONv2Codec_DurationRoundTrip`, `TestJSONv2Codec_NegotiatedWrite`         |
| 406-capable response-negotiation variant (current signature cannot express "nothing acceptable")                                                                                                                                         | ON_HOLD          | API-shape decision pending (both-refused currently echoes the request codec — `substituteForRefused` + golden matrix rows)                                                                              |

## SSE framing

| Feature                                                                                         | Status           | Evidence                                                                                                              |
| ----------------------------------------------------------------------------------------------- | ---------------- | --------------------------------------------------------------------------------------------------------------------- |
| `ParseSSEData`: WHATWG framing — CR/LF/CRLF terminators, BOM strip, multi-line joins, heartbeat | FULLY_FUNCTIONAL | `sse.go:42-95`; dispatch table + cap/scanner/EOF tests + fuzz in `sse_test.go`; pinned to the WPT/Chromium corpus via |
| comments, one-space stripping, size limits (`bufio.ErrTooLong` on over-cap)                     |                  | `sse_conformance_test.go:19` (oracle: `sseparse.MustCorpus`)                                                          |

## Observability

| Feature                                                                                            | Status           | Evidence                                                                                                                                                             |
| -------------------------------------------------------------------------------------------------- | ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Flight recorder: in-memory execution-trace buffer, one-shot slow-request auto-trigger              | FULLY_FUNCTIONAL | `flightrecorder.go:165-395`; 18 tests in `flightrecorder_test.go` (recorder tests serialized                                                                         |
| (cancellation-safe), options validation with sentinel errors                                       |                  | behind `flightRecorderMu:25`; cancellation regression test `flightrecorder_test.go:212`)                                                                             |
| Prometheus metrics: `<prefix>_flight_recorder_snapshots_total`, `..._snapshot_duration_seconds`    | FULLY_FUNCTIONAL | `flightrecorder.go:206-236`; metric-name test `TestFlightRecorder_MetricPrefix`                                                                                      |
| Rate-limited `TraceHandler` streaming `go tool trace` files (404 when disabled, 429 + Retry-After, | FULLY_FUNCTIONAL | `flightrecorder.go:395-448`; tests `TestTraceHandler_NotEnabled`, `TestTraceHandler_Enabled`, `TestTraceHandler_RateLimited`, `TestTraceHandler_ConcurrentRateLimit` |
| TOCTOU-safe atomic rate limit)                                                                     |                  |                                                                                                                                                                      |
| Snapshot compression (gzip) and retention (`MaxSnapshots`)                                         | FULLY_FUNCTIONAL | `flightrecorder.go:257-263`; tests `flightrecorder_test.go:384` and MaxSnapshots via library                                                                         |

## Counts

Computed from this file: 21 FULLY_FUNCTIONAL, 0 PARTIALLY_FUNCTIONAL, 0
BROKEN, 0 DISABLED, 0 PLANNED, 1 ON_HOLD (22 rows total).
