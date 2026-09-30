# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Nothing yet.

### Changed

- Nothing yet.

### Fixed

- Nothing yet.

## [0.2.0] - 2026-09-30

### Added

- `TestParseSSEData_CorpusConformance` — the framing parser is now asserted
  against the WPT/Chromium-derived conformance corpus exported by
  `github.com/larsartmann/go-sse/sseparse` (test-only dependency, v0.1.0):
  every vector's wire bytes run through `ParseSSEData` and the dispatched
  data payloads must match the corpus's expected events exactly. The corpus
  found three real spec gaps in the parser, all fixed: CR/LF/CRLF are all
  line terminators (CRLF counts as one — `bufio.Scanner` only split on LF),
  a bare `data` line with no colon contributes an empty payload line (it was
  silently dropped), and exactly one leading UTF-8 BOM is stripped (it used
  to poison the first field name). Over-cap lines now fail wrapping
  `bufio.ErrTooLong` (`errors.Is`-matchable).
- `TestResolveResponseCodec_GoldenMatrix` and
  `TestResolveResponseCodec_NeverServesRefusedEncoding`: the full
  negotiation contract (wildcards including `application/*`, duplicate-q
  max-wins, malformed `q` params, `Q=` case, refusal substitution, the
  unsatisfiable both-refused echo, browser headers) is now pinned in one
  reviewable table plus a refusal-property matrix, so contract drift fails
  tests instead of doc comments.
- `FuzzResolveResponseCodec`: a fuzz target for the hand-rolled `Accept`
  parser asserting, over both request codecs, that any input resolves to a
  supported encoding and never serves an explicitly refused one (except the
  pinned unsatisfiable both-refused case). Soaked for 5 minutes / 22M
  executions locally; seed corpus is the in-test `f.Add` set (browser
  headers, wildcards, malformed and NaN q-values, 100-entry lists), no
  crash corpus to commit.
- Direct unit tests for the negotiation helpers (`bareMediaType`,
  `qualityParam`, `matchesEncoding`, `acceptQuality`), previously exercised
  only transitively through the exported functions.

### Changed

- JSON/CBOR content negotiation now negotiates exactly. Media-type matching
  is exact after parameter stripping and lowercasing (`application/cbor-seq`
  and `application/cborx` no longer decode as CBOR), RFC 6839 structured
  suffixes resolve to their base encoding (`application/vnd.acme+cbor` is
  CBOR, `application/json-patch+json` is JSON), the `Accept` header's
  q-values rank JSON vs CBOR with a tie preferring CBOR, and a `q=0`
  refusal is never served: when the request codec is the refused encoding,
  the other supported encoding is served instead. Wildcards (`*/*` and
  `application/*`) still express no preference and echo the request codec.
  All exported signatures are unchanged, and both `ResolveRequestCodec` and
  `ResolveResponseCodec` remain live consumer-facing API
  (project-discovery-daemon calls both).
- Bumped `github.com/larsartmann/go-codec` v0.2.0 → v0.3.0, restoring pin
  parity with project-discovery-daemon. v0.3.0 is an error-contract release
  (stable machine-readable codes, behavioral families) with unchanged
  `Codec`/`BufferEncoder` signatures; go-daemon never matched codec error
  strings, so no source changes were needed.

### Fixed

- The content-negotiation test table from the q-value overhaul shipped with
  rows that did not compile (missing table field); fixed forward, and the
  full race suite is verified green.
- Quality parameter names in `Accept` headers are now matched
  case-insensitively (RFC 9110 §5.6.2.1): `application/cbor;Q=0` previously
  counted as a parameter-less mention with quality 1, so an uppercase `Q=`
  could not express a refusal or a lowered preference.
- `q=NaN` values are now skipped instead of parsed: Go's `ParseFloat`
  accepts `NaN`, which then propagated through quality clamping and made the
  poisoned mention behave as a silent refusal (all quality comparisons
  against NaN are false). NaN entries now fall back to the documented
  unparseable-q default (a valid q elsewhere in the entry still applies,
  otherwise 1).

## [0.1.1] - 2026-09-29

### Fixed

- `FlightRecorder.MaybeCaptureOnSlowRequest` no longer silently drops the
  async capture when the caller's context is cancelled: the capture goroutine
  races the caller returning (net/http cancels the request context at exactly
  that moment), and the underlying library's pre-write context check discarded
  the snapshot whenever the goroutine lost that race — precisely on loaded
  daemons where slow requests occur. The context is now detached from
  cancellation via `context.WithoutCancel` (values preserved), so the capture
  survives the caller. Regression-tested with a pre-cancelled context.
- Tests that hold an active flight recorder are serialized behind a mutex:
  Go's runtime/trace allows only one active recorder per process, so parallel
  recorder tests could fail with "another flight recorder is already active
  in this process" under load.

## [0.1.0] - 2026-09-29

### Added

- `SocketServer`: full unix-socket daemon lifecycle — stale-socket cleanup,
  socket directory creation, `0600` default permissions, `sd_notify`
  READY/STOPPING/WATCHDOG with systemd watchdog keepalive, graceful drain
  with grace period, socket removal on every exit path.
- `Lifecycle` hook contract (`Startup`/`OnListen`/`OnShutdown`) with a
  documented, test-enforced ordering.
- `IsSocketAlive`/`CleanStaleSocket` liveness probing over a configurable
  health path.
- `NewHTTPClient`/`NewHTTPClientNoTimeout`: unix-socket HTTP transports with
  pooled keep-alive connections tuned for socket traffic.
- JSON/CBOR content negotiation: `ResolveRequestCodec`,
  `ResolveResponseCodec`, `ContentTypeFor`, `CodecForContentType`, plus
  pooled-buffer response writers `WriteEncoded`/`WriteJSON`/`WriteErrorf`.
- `ParseSSEData`: WHATWG Server-Sent Events framing (multi-line data joins,
  comment/heartbeat handling, single leading-space stripping, size limits).
- `FlightRecorder`: execution-trace flight-recorder integration with
  slow-request one-shot auto-trigger, Prometheus metrics behind a
  configurable `MetricPrefix`, and a rate-limited `TraceHandler` streaming
  `go tool trace` files.

[Unreleased]: https://github.com/LarsArtmann/go-daemon/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/LarsArtmann/go-daemon/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/LarsArtmann/go-daemon/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/LarsArtmann/go-daemon/releases/tag/v0.1.0
