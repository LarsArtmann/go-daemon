# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/LarsArtmann/go-daemon/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/LarsArtmann/go-daemon/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/LarsArtmann/go-daemon/releases/tag/v0.1.0
