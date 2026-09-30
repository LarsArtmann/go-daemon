# Features

Honest inventory of what this library provides. Statuses answer one question:
does working code exist, and if not, why not?

**Verification basis:** audited 2026-09-29 against Go 1.27.1 with the full
gate green — `go build`, `go vet`, `go test ./... -race -count=1`,
`golangci-lint run` (strict config), `nix build .#default`. Every
FULLY_FUNCTIONAL row below is backed by tests in that suite.

**Coverage (measured 2026-09-30, `go test -covermode=atomic`):** 86.1% of
statements total, 30 functions at 100%. The CI floor (80%, `ci.yml`) is real
and passing, so it stays — no blind raise. Known weak spots are trivial
option setters (`WithXxx`, 0%), `WriteEncoded`'s encode-error and
non-BufferEncoder paths (65.6%), and `Start`'s startup-error paths (76.2%).

## Socket lifecycle

| Feature                                                                                            | Status           | Evidence                                                        |
| -------------------------------------------------------------------------------------------------- | ---------------- | --------------------------------------------------------------- |
| Unix-socket daemon server: stale-socket cleanup, socket-dir creation, `0600` default permissions   | FULLY_FUNCTIONAL | `socket.go:195-247`; tests `socket_test.go:108,140,164,329,353` |
| systemd integration: `sd_notify` READY/STOPPING, watchdog keepalive at `WatchdogSec/2`             | FULLY_FUNCTIONAL | `socket.go:255-270,327-351`; tests `socket_test.go:438,461`     |
| Graceful drain with configurable grace period; socket removed on every exit path                   | FULLY_FUNCTIONAL | `socket.go:285-315`; test `socket_test.go:108`                  |
| Lifecycle hook contract: `Startup` → `OnListen` → `OnShutdown` → startup cleanup, test-enforced    | FULLY_FUNCTIONAL | `socket.go:151-187`; tests `socket_test.go:188,269`             |
| Stale-socket liveness probe (`IsSocketAlive`/`CleanStaleSocket`, configurable health path/timeout) | FULLY_FUNCTIONAL | `socket.go:353-399`; tests `socket_test.go:329,353,389`         |

## Client transport

| Feature                                                                                         | Status           | Evidence                                        |
| ----------------------------------------------------------------------------------------------- | ---------------- | ----------------------------------------------- |
| Unix-socket HTTP clients: `NewHTTPClient` (timeout) and `NewHTTPClientNoTimeout` (SSE streams), | FULLY_FUNCTIONAL | `transport.go:25-51`; test `socket_test.go:407` |
| pooled keep-alive tuned for socket traffic                                                      |                  |                                                 |

## Content negotiation

| Feature                                                                                            | Status           | Evidence                                                                                             |
| -------------------------------------------------------------------------------------------------- | ---------------- | ---------------------------------------------------------------------------------------------------- |
| Request codec from `Content-Type`: JSON default, CBOR via exact or RFC 6839 structured-suffix type | FULLY_FUNCTIONAL | `negotiation.go` (`ResolveRequestCodec`, `matchesEncoding`, `CodecForContentType`); tests `TestResolveRequestCodec`, `TestCodecForContentType`, `TestMatchesEncoding` |
| Response codec from `Accept`: q-value ranking, tie prefers CBOR, `q=0` refusals never served,      | FULLY_FUNCTIONAL | `negotiation.go` (`ResolveResponseCodec`, `acceptQuality`, `qualityParam`); golden matrix + refusal property + fuzz; live |
| wildcards echo the request codec                                                                   |                  | consumer API — project-discovery-daemon `content_negotiation.go:15-19` calls both resolvers                   |
| Pooled-buffer response writers: `WriteEncoded`/`WriteJSON`/`WriteErrorf`                           | FULLY_FUNCTIONAL | `write.go` (`bufPool`, `WriteEncoded`, `WriteJSON`, `WriteErrorf`); tests `TestWriteEncoded_JSONAndCBOR`, `TestWriteErrorf` |
| `Vary: Accept` emission from every `WriteEncoded` response (cache correctness for Accept-negotiated encodings) | FULLY_FUNCTIONAL | `write.go` (`WriteEncoded` doc contract + `Header().Add`); tests `TestWriteEncoded_VaryAccept` (unconditional, combines with handler Vary, error path, negotiated round trip) |
| 406-capable response-negotiation variant (current signature cannot express "nothing acceptable")   | ON_HOLD          | API-shape decision pending (both-refused currently echoes the request codec — `substituteForRefused` + golden matrix rows) |

## SSE framing

| Feature                                                                                         | Status           | Evidence                                                                                |
| ----------------------------------------------------------------------------------------------- | ---------------- | --------------------------------------------------------------------------------------- |
| `ParseSSEData`: WHATWG framing — CR/LF/CRLF terminators, BOM strip, multi-line joins, heartbeat | FULLY_FUNCTIONAL | `sse.go:42-95`; 13 tests + fuzz in `sse_test.go`; pinned to the WPT/Chromium corpus via |
| comments, one-space stripping, size limits (`bufio.ErrTooLong` on over-cap)                     |                  | `sse_conformance_test.go:19` (oracle: `sseparse.MustCorpus`)                            |

## Observability

| Feature                                                                                            | Status           | Evidence                                                                                     |
| -------------------------------------------------------------------------------------------------- | ---------------- | -------------------------------------------------------------------------------------------- |
| Flight recorder: in-memory execution-trace buffer, one-shot slow-request auto-trigger              | FULLY_FUNCTIONAL | `flightrecorder.go:155-358`; 18 tests in `flightrecorder_test.go` (recorder tests serialized |
| (cancellation-safe), options validation with sentinel errors                                       |                  | behind `flightRecorderMu:25`; cancellation regression test `flightrecorder_test.go:212`)     |
| Prometheus metrics: `<prefix>_flight_recorder_snapshots_total`, `..._snapshot_duration_seconds`    | FULLY_FUNCTIONAL | `flightrecorder.go:206-216`; metric-name test `flightrecorder_test.go:447`                   |
| Rate-limited `TraceHandler` streaming `go tool trace` files (404 when disabled, 429 + Retry-After, | FULLY_FUNCTIONAL | `flightrecorder.go:395-448`; tests `flightrecorder_test.go:505,520,547,583`                  |
| TOCTOU-safe atomic rate limit)                                                                     |                  |                                                                                              |
| Snapshot compression (gzip) and retention (`MaxSnapshots`)                                         | FULLY_FUNCTIONAL | `flightrecorder.go:257-263`; tests `flightrecorder_test.go:384` and MaxSnapshots via library |

## Counts

Computed from this file: 14 FULLY_FUNCTIONAL, 0 PARTIALLY_FUNCTIONAL, 0
BROKEN, 0 DISABLED, 1 PLANNED, 1 ON_HOLD (16 rows total).
