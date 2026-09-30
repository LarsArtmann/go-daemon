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
golangci-lint run ./...     # strict config mirrored from project-discovery-daemon (locally pruned: no pdd-only types here)
nix build .#default         # hermetic build (all deps public — no mkPreparedSource)
nix run .#test
nix run .#check-readme      # compiles the README quickstart snippet (run from repo root)
nix flake check
nix fmt
```

The toolchain is `go 1.27.1`; `encoding/json/v2` is GA there (no
GOEXPERIMENT needed). Ambient go on this machine may be older — run
`go version` (or trust the editor's gopls diagnostics) BEFORE writing code;
bare `go` that refuses means use `nix shell nixpkgs#go_1_27` or
`nix develop`.

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
- Table-driven tests lead with a `name` field; positional row literals must
  match the struct arity — mismatched rows fail the build, so gate before
  finishing (see Gotchas: auto-commit).
- Content-negotiation policy (exact matching, RFC 6839 suffixes, q-value
  ranking, tie → CBOR, `q=0` refusal substitution, wildcard neutrality) is a
  documented contract in the `negotiation.go` doc comments — change behavior,
  doc comment, and the `TestResolveResponseCodec` table together. Both
  resolvers are live consumer-facing API (project-discovery-daemon calls
  them from its content_negotiation.go).
- The `.golangci.yml` mirrors project-discovery-daemon's but is locally
  pruned: entries for types that only exist in the consumer
  (project-discovery-sdk, samber/ro) were removed — re-check the diff
  before syncing from pdd again.

## Dependencies (all public — resolves via Go proxy)

- `github.com/larsartmann/go-codec` — JSON/CBOR codecs (pinned to versions
  the primary consumer uses; check before bumping)
- `github.com/larsartmann/go-flightrecorder` — execution-trace buffering
- `github.com/coreos/go-systemd/v22` — sd_notify + watchdog
- `github.com/prometheus/client_golang` — flight-recorder metrics
- `github.com/larsartmann/go-sse/sseparse` — ADOPTED as a test-only
  dependency (v0.1.0; public module, so no GOPRIVATE/vendor cost):
  `sse_conformance_test.go` asserts `ParseSSEData` against
  `sseparse.MustCorpus(t)` — the WPT/Chromium wire-format vectors, projected
  onto the framing surface (dispatched data payloads). The corpus is the
  oracle: a vector disagreement means the parser is wrong. The go-daemon-
  specific maxEventBytes/large-event tests stay in `sse_test.go`.

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
- Adding or bumping ANY dependency (even a test-only one) changes the
  flake's `vendorHash`, and only `nix build` catches the mismatch —
  build/test/vet/lint all stay green. Repair with
  `buildflow -s nix-hash-fix --fix` (works here despite no .buildflow.yml);
  never hand-paste the `got:` hash. This bit the sseparse adoption, which
  shipped with a stale hash and needed a follow-up repair commit.
- **The auto-commit daemon commits regardless of compile state.** Run a
  gate (build or gopls diagnostics) SYNCHRONOUSLY immediately after writing
  code — a verification that is backgrounded, deferred to "after this
  analysis", or interrupted ships a broken commit to master (the
  negotiation overhaul landed a test table that never compiled).
- `nix develop` can take minutes on cold nix evaluation before the command
  inside even starts — a slow start is not a hang; run long gates in the
  background and keep verification in the foreground of the session.
- Timestamped status reports land in `docs/status/` (auto-committed). They
  are snapshots, not backlogs: the next docs pass must harvest each
  report's section-f items into TODO_LIST/ROADMAP (verify against code
  first — many are already done by later sessions).
- **Atomic edits under the auto-commit daemon.** The daemon snapshots the
  tree roughly every minute, so a multi-step edit sequence (write, then
  fix, then gate) can ship a broken intermediate to master — it did once
  with a non-compiling README example (`caf1f0c`, fixed `378607f`). Make
  each edit self-consistent and compile-clean on its own, gate
  synchronously between steps, and commit descriptively yourself when the
  change matters (heuristic `chore: auto-commit` messages flatten history).
- **Re-read files after any formatter run.** `nix fmt` / dprint /
  `golangci-lint fmt` rewrite files in place, and edit-tool stale-read
  checks will bounce the next edit until the file is re-read. Budget a
  re-view after every formatting pass.
- **Stale LSP diagnostics: trust the CLI gates.** `golangci_lint_ls` can
  keep showing phantom errors for lines that were already fixed (it did
  for a whole session after the negotiation test fix). When LSP output
  contradicts a green CLI gate, `lsp_restart` clears the cache; the CLI
  (`golangci-lint run`, `go build`) is authoritative.
- **Fuzz corpora are seed-only by policy** (`FuzzResolveResponseCodec`,
  `FuzzParseSSEData`): the `f.Add` seeds are committed, crash corpora are
  not. If a fuzz run ever finds a crash, commit the crashing input to the
  target's corpus so the fix stays regression-pinned.
- **Dependabot + vendorHash policy: close-and-adopt.** Dependabot's
  dependency PRs update go.mod/go.sum but never the flake's `vendorHash`,
  so their CI nix jobs fail by design. The agreed handling: merge the
  bump, then immediately repair with `buildflow -s nix-hash-fix --fix`
  (here or in the consuming repo) and commit the hash — do not wait for
  or ask the bot to fix nix. (2026-09-30: the go-codec v0.3.0 bump landed
  this way and the hash repaired cleanly.)
- dprint plugin pins are versioned URLs in `dprint.json` (checked
  2026-09-30: json 0.25.0, pretty_yaml 0.6.0, markdown 0.24.0,
  dockerfile 0.6.0). `dprint config update` refreshes versions but also
  rewrites the URL scheme to `npm:` — restore the `plugins.dprint.dev`
  URL scheme when using it, and confirm zero churn with `nix fmt` after
  any pin bump.
