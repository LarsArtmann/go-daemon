# AGENTS.md

Enduring context for AI sessions working in this repository.

## What This Is

A Go library with the production-proven building blocks for long-lived
daemons over unix domain sockets. Extracted from
`project-discovery-daemon` (which is now also the primary consumer).
Mechanism, not policy: sockets, lifecycle ordering, codecs, framing — no
domain logic.

## Essential Commands

```bash
go build ./...
go test ./... -race -count=1
go vet ./...
golangci-lint run ./...     # strict config mirrored from project-discovery-daemon
nix build .#default         # hermetic build (all deps public — no mkPreparedSource)
nix run .#test
nix fmt
```

The toolchain is `go 1.27.1`; `encoding/json/v2` is GA there (no
GOEXPERIMENT needed). Ambient go on this machine may be older — use
`nix shell nixpkgs#go_1_27` or `nix develop` when bare `go` refuses.

## Conventions

- Flat single package `daemon` at the repo root (no `pkg/`/`internal/`).
- Tabs for Go; spaces for nix/json (`.editorconfig`, dprint.json).
- Functional options (`WithXxx`) for configuration; no panics in library code.
- Lifecycle ordering is a CONTRACT (see `Lifecycle` doc comment in
  socket.go): clean → Startup → listen/chmod → OnListen → READY → watchdog →
  serve → STOPPING → drain → remove socket → OnShutdown → startup cleanup.
  Do not reorder without updating the hook-ordering test.
- Metric names are parameterized (`FlightRecorderOptions.MetricPrefix`);
  `project-discovery-daemon` passes `project_discovery` to keep its metric
  names stable — do not rename the suffixes.
- `golangci-lint fmt` auto-fixes formatting-class findings (gci/golines);
  run it before `run`.

## Dependencies (all public — resolves via Go proxy)

- `github.com/larsartmann/go-codec` — JSON/CBOR codecs (pinned to versions
  the primary consumer uses; check before bumping)
- `github.com/larsartmann/go-flightrecorder` — execution-trace buffering
- `github.com/coreos/go-systemd/v22` — sd_notify + watchdog
- `github.com/prometheus/client_golang` — flight-recorder metrics
- `github.com/larsartmann/go-sse/sseparse` — PENDING (not yet adopted):
  zero-dep SSE parser + spec corpus. Blocked on its v0.1.0 tag (split from
  ssetest 2026-09-29, release tracked in go-sse TODO_LIST). Once tagged,
  migrate sse_test.go's framing tests to `sseparse.MustCorpus` vectors;
  keep the maxEventBytes/large-event tests (go-daemon-specific surface).

## Gotchas

- This repo is PRIVATE (`github.com/LarsArtmann/...`). Consumers resolve it
  direct via git/SSH (`GOPRIVATE=github.com/larsartmann/*,github.com/LarsArtmann/*`);
  hermetic nix consumers must pin it as a `git+ssh` flake input fed into
  `mkPreparedSource` (see project-discovery-daemon's flake.nix).
- The SSE parser is framing-only on purpose (mechanism vs policy); payload
  decoding lives with consumers.
- Version bumps should follow the primary consumer's needs — its nix build
  pins this repo by rev and re-derives vendorHash on every bump.
- **Go's runtime/trace allows only ONE active flight recorder per process.**
  Tests that start a recorder are serialized behind `flightRecorderMu`
  (flightrecorder_test.go): parallel recorder tests otherwise fail with
  "another flight recorder is already active in this process" under load.
  Keep new recorder tests inside that mutex (via `newTestFlightRecorder`
  or inline lock/unlock when a custom registry is needed).
- `MaybeCaptureOnSlowRequest` detaches the caller's context
  (`context.WithoutCancel`) before spawning the capture goroutine — that is
  the contract (v0.1.1): a request context cancelled at handler return must
  NOT drop the in-flight capture. The regression test
  (`TestFlightRecorder_SlowRequestTrigger_CancelledContext`) fails
  deterministically if someone removes the detachment; do not weaken it.
