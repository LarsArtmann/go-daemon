# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Nothing yet.

### Fixed

- Nothing yet.

## [0.5.0] - 2026-11-27

### Added

- `JSONv2Codec`: a `codec.Codec` + `codec.BufferEncoder` adapter over
  `encoding/json/v2` with caller-chosen marshal/unmarshal options, so a
  consumer can pin json/v2 semantics (deterministic map ordering,
  integer-nanos durations, case-insensitive field matching) or the v1 wire
  shapes (`FormatNilSliceAsNull`, `FormatDurationAsNano`) regardless of
  which go-codec JSON backend the build selected.
- Injectable negotiation codecs: `WithJSONCodec`/`WithCBORCodec` options let
  `ResolveRequestCodec`, `ResolveResponseCodec`, and `CodecForContentType`
  resolve consumer-supplied codecs; built-in defaults and the negotiation
  ranking contract are unchanged.
- `ClassifyTransportError` and `TransportErrorKind`: transport-error
  classification for daemon clients, promoted from
  project-discovery-daemon's `classifyTransportError`. A done caller
  context is `TransportErrorCanceled`; a deadline under a live context is
  `TransportErrorTimeout` (the client's own timeout: the daemon accepted
  and never answered); connection refused, socket missing, reset and EOF
  name the other ways a daemon stops answering, and
  `TransportErrorKind.Unavailable()` is the one verdict a consumer maps
  onto its own sentinel.
- `WaitForSocket` and `ProbeUntilAlive`: context-bounded wait helpers
  for tests and restart loops. `WaitForSocket` returns once a listener
  accepts on the socket path (a leftover socket file does not count);
  `ProbeUntilAlive` once a daemon answers the health probe. Both poll
  every `DefaultWaitPollInterval` (50ms, `WithPollInterval` overrides)
  and return an error wrapping `ctx.Err()` when the context ends first.
- `WriteJSONWithCodec` and `WriteErrorfWithCodec`: codec-aware counterparts
  of `WriteJSON`/`WriteErrorf`, so a handler that negotiated a codec (a
  `JSONv2Codec`, or CBOR) writes its payload and its error envelope through
  the same codec instead of falling back to the default. `WriteJSON` and
  `WriteErrorf` keep their signatures and behaviour.

### Changed

- The language floor moves from `go 1.27.1` to `go 1.27.2`, following
  go-codec v0.3.3's declared floor (a deliberate version contract in that
  module, not a formatting slip), so it cannot be lowered without
  downgrading the dependency. Consumers on an older Go patch get Go's
  automatic toolchain switch (or must upgrade under `GOTOOLCHAIN=local`).
  Dependency bumps since 0.4.0: go-codec v0.3.0 → v0.3.3, go-flightrecorder
  v0.2.0 → v0.2.1, go-sse/sseparse v0.2.0 → v0.2.1,
  prometheus/client_golang v1.24.1 → v1.25.0.
- **The default JSON codec now emits json/v2 wire shapes, and it is not
  switchable off.** go-codec v0.3.3 compiles its `encoding/json/v2` backend
  when the `jsonv2` experiment is enabled, which Go 1.27 does by default:
  through `codec.JSONCodec{}` (and therefore `WriteJSON`, `WriteErrorf`, and
  `WriteEncoded` with the default codec, plus negotiation's JSON default) a
  nil slice now encodes as `[]` not `null`, `time.Duration` fails to encode
  rather than emitting integer nanos, and map key order is unordered. This
  module cannot be built with the experiment disabled (`GOEXPERIMENT=none`
  excludes `encoding/json/v2`, which jsonv2.go imports), so the v2 shapes
  are unconditional. Pass an explicit codec when the wire shape matters:
  `WriteJSONWithCodec`, `WriteErrorfWithCodec`, or negotiation's
  `WithJSONCodec`, with
  `JSONv2Codec{MarshalOptions: []json.Options{json.FormatNilSliceAsNull(true),
  json.FormatDurationAsNano(true)}}` for the v1 shapes.

### Fixed

- Nothing yet.

### Security

- Nothing yet.

## [0.4.0] - 2026-10-03

### Added

- Standalone sd_notify API, callable without `SocketServer`:
  `NotifyReady`/`NotifyStopping` send READY=1/STOPPING=1, `Watchdog` keeps
  the watchdog pinged, `WatchdogInterval` returns the recommended interval
  (WatchdogSec/2, per `sd_watchdog_enabled(3)`), and `ReadNotifyCounters`
  exposes process-global activity counters for `/metrics` scrapes. All are
  silent no-ops without systemd. Decided in
  `docs/decisions/2026-10-03_v040-notify-api-shape.md`.
- Listener-agnostic `Server`: `NewServer(network, addr)` serves the full
  production lifecycle on any network (e.g. TCP), `WithListener` serves a
  pre-built listener with zero filesystem management. `SocketServer` is now
  a type alias of `Server`, so unix behavior and every existing option are
  unchanged. Contract tests (hook ordering, graceful drain, listen-error,
  port restart) ported to TCP.
- First consumer: bank-sync (2026-10-04) adopted v0.4.0 for all three serve
  modes — shared lifecycle runner, opt-in `systemdNotify`/`watchdogSec`
  NixOS module, `bank_sync_sd_notify_*` metrics from `ReadNotifyCounters`
  (ADR-017 in that repo).

## [0.3.0] - 2026-10-01

### Added

- Packaged the five owner decisions into
  `docs/decisions/2026-10-01_owner-decision-memo.md` (data +
  recommendation each: API policy, name, docs visibility, CI billing, the
  flip), pre-written the ~30-minute post-flip mechanical tail as
  `docs/runbooks/post-flip.md` (apply only after the flip), and drafted
  the demand-gated v0.3.0 tag annotation and CHANGELOG compare-link diff
  in `docs/release/v0.3.0-draft.md`. The v0.2.0 GitHub release page
  rendering is verified; the pkg.go.dev check is owner-gated on the flip
  (private modules cannot be indexed).
- Added `scripts/pre-publish-audit.sh`, generalizing the pre-flip trust
  scans into one command for any repo flip: gitleaks history scan plus a
  manual credential-pattern pass, a working-tree private-path scan
  (GOPRIVATE/git+ssh/home paths — expected to fail until the post-flip
  runbook lands), and a permissive-license check over the build closure.
- Added a `check-docs` flake app (and `scripts/check-docs.sh`) that turns
  documentation health into one command: dprint format check, per-table row
  completeness over the living docs, strikethrough presence in every
  historical report (dprint has destroyed full-row strikes before), and a
  mechanical FEATURES-count derivation that fails on drift. The row checker
  is vendored at `scripts/check-rows.py` so the gate is self-contained.
- Added Output-verified godoc examples for the two remaining flagship
  surfaces: `ExampleParseSSEData` (framing-only parse loop, multi-line data
  included) and `ExampleNewSocketServer` (full serve-over-unix-socket round
  trip ending in graceful shutdown). Examples run as tests.
- Added `SECURITY.md` (supported-versions table, private reporting channel,
  scope notes on socket permissions and framing-only SSE parsing), GitHub
  issue templates (bug + feature forms scoped to mechanism-not-policy), and
  the repository description and topics. Private vulnerability reporting
  cannot be enabled on a private repository, so the security contact path is
  documented for both the private and post-flip states.
- CI now enforces the two previously local-only gates: the README quickstart
  compile check (via `scripts/check-readme.sh`, the single source of truth
  that the flake's `check-readme` app also calls, so the gates cannot drift)
  and a new `nix` job running `nix build .#default && nix flake check` on a
  pinned installer — the only check that catches a stale `vendorHash`.
  Private-assumption audit of the workflow came back clean: no secrets, no
  GOPRIVATE, default `GITHUB_TOKEN` only.
- Closed every measured test-coverage gap: the seven `WithXxx` option
  setters, `SocketPath`, `WriteEncoded`'s encode-error and non-buffer-codec
  fallback paths, `Start()`'s socket-directory failure modes (blocked parent
  path, unwritable directory), `notifyStopping` (set/unset/unreachable
  systemd socket), and `lineAtEOF`'s EOF boundary branches. Total coverage
  rose from 86.1% to 92.0% (CI gate: 80%); every targeted function is now at
  100% except `Start` (77.1% — the remaining branches require
  non-portable failure injection: chmod-after-listen, mid-serve listener
  death).
- `WriteEncoded` (and through it `WriteJSON`/`WriteErrorf`) now emits
  `Vary: Accept` on every response: response encodings are commonly chosen
  from the request's Accept header, so caches must not reuse a stored
  response across different Accept preferences. The header is added rather
  than set, preserving handler-declared variance such as
  `Vary: Authorization`.

### Changed

- README now reads as a first-time visitor's page: install command and Go
  version requirement up front, a Non-goals section that states the
  mechanism-not-policy boundary (no domain logic, no packaging, unix
  sockets only) with a pointer to ROADMAP.md, and a Security section
  linking SECURITY.md.
- Rewrote CONTRIBUTING.md nix-first: the flake commands are the documented
  path (plain-Go equivalents listed), PR expectations now name the enforced
  gates (80% coverage floor, README compile check, lifecycle-contract and
  negotiation-table coupling), and security reports point at SECURITY.md.
  Also closed two provenance questions: `90f86db` is confirmed as the
  auto-commit daemon's heuristic sweep of the 2026-09-30 00:43 session
  (CONTRIBUTING creation, go.mod/go.sum alignment, vendorHash repair), and
  the local `.golangci.yml` still diffs against project-discovery-daemon's
  by exactly the three documented consumer-only prune entries — nothing to
  sync.

### Fixed

- `nix flake check --all-systems` now passes: x86_64-darwin is filtered
  from the flake's systems list because Nixpkgs 26.11 dropped support for
  it (evaluating any package there failed the check). aarch64-darwin and
  both linux systems remain; the library cross-compiles to darwin/arm64,
  verified with `GOOS=darwin go build`.

### Security

- Completed the pre-flip trust scan of the full git history: gitleaks across
  all 53 commits (~612 KB of blobs) plus a manual pass for PEM keys, cloud
  provider tokens, GitHub/GitHub-app tokens, high-entropy strings, internal
  hosts/paths, and `.netrc`-shaped content. Zero findings; the single manual
  hit was the scan plan describing its own patterns (self-referential false
  positive).
- Completed the dependency license inventory: all 15 build-closure modules
  are MIT, BSD-2/3-Clause, or Apache-2.0 (no copyleft); the dev-only graph
  adds nothing stricter than file-level MPL-2.0 (`pgregory.net/rapid`,
  test-only). Inventory and the NOTICE-file decision recorded in
  `docs/dependency-licenses.md`.

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

[Unreleased]: https://github.com/LarsArtmann/go-daemon/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/LarsArtmann/go-daemon/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/LarsArtmann/go-daemon/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/LarsArtmann/go-daemon/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/LarsArtmann/go-daemon/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/LarsArtmann/go-daemon/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/LarsArtmann/go-daemon/releases/tag/v0.1.0
