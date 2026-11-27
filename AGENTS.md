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
nix run .#check-docs        # documentation health: dprint check + row completeness + strike presence + FEATURES counts
bash scripts/pre-publish-audit.sh  # trust scan for repo flips: history secrets + private paths + licenses (exit 1 is EXPECTED until the post-flip runbook removes the GOPRIVATE refs)

The check-readme logic lives in `scripts/check-readme.sh` — the flake app
and ci.yml both invoke that script, so edit the script only (never inline
the awk in a second place).
nix flake check --no-build --all-systems  # evaluates every listed system (see Gotchas)
nix fmt
```

The language floor is `go 1.27` — no patch floor, because a `go 1.27.2`
directive forces every consumer onto 1.27.2+ (v0.4.x carried one by
accident; the flake pins nixpkgs `go_1_27`, 1.27.3 at the time of
writing). `encoding/json/v2` is GA there. Ambient go on this machine may
differ — run `go version` (or trust the editor's gopls diagnostics)
BEFORE writing code; bare `go` that refuses means use
`nix shell nixpkgs#go_1_27` or `nix develop`.

**Build with Go 1.27.4+, because 1.27.3 and earlier carry
CVE-2026-97054 / GO-2026-6859** (net/http MaxBytesReader header race →
server crash; it spans 87 reachable net/http symbols, so the
reachability trace is nearly all of `socket.go`). CI pin `1.27.4`;
nixpkgs was still shipping `go_1_27 = 1.27.3` on 2026-11-27, so the
devShell's govulncheck reports that stdlib finding until nixpkgs catches
up — bump the flake's Go then, and do not treat the finding as ours.

**The default JSON codec is the json/v2 backend, and the module cannot
build with it disabled.** go-codec v0.3.3 compiles `json_compat_v2.go`
when the `jsonv2` experiment is on, which Go 1.27 does by default;
re-probed 2026-11-27 on go1.27.3 — `codec.JSONCodec{}` yields nil slices
as `[]` (not null), an _error_ for `time.Duration`, and unordered map
keys. That is this module's actual wire behavior, not a per-build coin
flip: `GOEXPERIMENT=none` fails to build at all, because jsonv2.go
imports `encoding/json/v2` and prometheus/common imports
`encoding/json/jsontext`, both excluded without the experiment (verified
2026-11-27). Consumers needing v1 shapes must pass an explicit codec:
`JSONv2Codec` (jsonv2.go) with `FormatNilSliceAsNull(true)` /
`FormatDurationAsNano(true)`, routed through `WriteJSONWithCodec` /
`WriteErrorfWithCodec` or `WithJSONCodec`. The pre-v0.3.3 note here
claiming `codec.JSONCodec{}` was v1 on Go 1.27 does not hold with this
toolchain; probe behavior, not the tag.

## Conventions

- Flat single package `daemon` at the repo root (no `pkg/`/`internal/`).
- Tabs for Go; spaces for nix/json (`.editorconfig`, dprint.json).
- Functional options (`WithXxx`) for configuration; no panics in library code.
- Error construction is stdlib-only (`errors.New` sentinels, `fmt.Errorf` +
  `%w` wraps, explicit `_ =` for best-effort cleanup): no samber/oops, no
  coded-error wrappers. erraudit audit runs (`--no-suppress` plus the
  enforce flags) surface these as findings — known-dismissed (reviewed
  2026-10-01, 0/26 actionable). Do not migrate sentinels or wrap-style
  errors without a cross-repo decision: project-discovery-daemon asserts
  on `Validate()` error-message substrings (its metrics_test.go), making
  message wording a consumer-visible contract, and it pins this repo by
  rev.
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
- New `ParseSSEData` dispatch-shape cases (wire input → dispatched payloads)
  become rows in the `TestParseSSEData_Dispatch` table (sse_test.go) — do
  not add new same-shape top-level test functions; error/size-shaped cases
  (`MaxEventBytesExceeded`, `LargeEvent`, `ScannerErrorWrapped`) stay
  separate functions. art-dupl at `-t 1` flags the `default:` pair in
  negotiation.go — accepted idiom, not a clone to fix. (As of 2026-10-05 it
  is the ONLY finding shown at `-t 1`: the nil-logger defaulting that
  art-dupl previously flagged in notify.go now has a single implementation,
  `loggerOrDefault` — add new notify entry points by calling it, not by
  re-inlining `if logger == nil`.)
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
  dependency (v0.2.1; public module, so no vendor cost):
  `sse_conformance_test.go` asserts `ParseSSEData` against
  `sseparse.MustCorpus(t)` — the WPT/Chromium wire-format vectors, projected
  onto the framing surface (dispatched data payloads). The corpus is the
  oracle: a vector disagreement means the parser is wrong. The go-daemon-
  specific maxEventBytes/large-event tests stay in `sse_test.go`.

## Gotchas

- **`nix flake check --all-systems` BUILDS every system's checks; add
  `--no-build` for eval-only.** Measured 2026-11-27 in the first public
  run (46020081576): `nix build .#default && nix flake check
  --all-systems` died with `Cannot build ... Reason: platform mismatch /
  Required system: 'aarch64-linux'`, because the check phase builds
  `checks.<system>.build` for each listed system. `--no-build` evaluates
  every system (~5s) and builds nothing, which is what CI runs. The plain
  check (no flags) still evaluates only the native system and reports the
  rest as `warning: The check omitted these incompatible systems` with
  exit 0 — also measured 2026-11-27 at nixpkgs `0bb8272318b4`, where with
  `x86_64-darwin` added to `systems` the plain check passed while the
  `--all-systems` form failed with `Nixpkgs 26.11 has dropped support for
  x86_64-darwin`.
- **No `//go:build` guards, by design — do not add any for the notify
  path.** `SdNotify` and `SdWatchdogEnabled` in go-systemd v22.7.0 are
  run-time switched (`NOTIFY_SOCKET` / `WATCHDOG_USEC` unset → not sent,
  no error, no log, no counter), the same on every platform. The only
  build-constrained pair there wraps `SdNotifyMonotonicUsec` (`unix` vs
  `!unix`), unused here — and that `unix` file is why the library does
  not compile on netbsd (`unix.CLOCK_MONOTONIC` is undefined in x/sys
  there). Cross-compile check 2026-11-27, Go 1.27.3, `CGO_ENABLED=0`:
  builds for linux, darwin, windows, freebsd, openbsd, dragonfly, solaris,
  js/wasm and wasip1/wasm.
- This repo is PUBLIC (flipped 2026-11-27): consumers resolve it through
  the Go module proxy and pin it as a plain flake input
  (`github:LarsArtmann/go-daemon/<rev-or-tag>`, see
  project-discovery-daemon's flake.nix). No vendored-source helper is
  needed for it any more.
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
- **Capture a gate's exit code BEFORE any pipe.** `nix run .#check-docs |
  tail; echo $?` reports `tail`'s status, not the gate's — on 2026-10-05
  this shipped a false "check-docs green" claim while the strike-presence
  check was failing on unannotated docs/status reports. Pattern that
  works: run the gate bare (`cmd; echo $?`) or with `set -o pipefail`.
  Same family as the stale-LSP gotcha below: the CLI is authoritative
  only when you actually read the CLI's own exit status.
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
- **Release cuts race the auto-commit daemon in BOTH repos.** During the
  v0.3.0 cut (2026-10-01) the daemon swept the CHANGELOG promotion into a
  heuristic commit between gate and release commit, and pdd's daemon split
  the consumer bump into two heuristic commits. Rules: commit release-
  critical edits (CHANGELOG promote, TODO gate-row removal) descriptively
  in the SAME bash call that makes them; in the consumer, squash
  daemon-split bump commits with `git reset --soft HEAD~N` + one
  descriptive commit before pushing (safe: local-only rewrite).
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
- **Verify row edits with an immediate grep, not the edit tool's success
  response.** Twice this session a TODO_LIST row removal reported success
  but the row was still on disk afterward (cause undiagnosed). Pattern
  that works: remove via script, `grep -c` in the SAME command, gate,
  then commit.
- dprint plugin pins are versioned URLs in `dprint.json` (checked
  2026-09-30: json 0.25.0, pretty_yaml 0.6.0, markdown 0.24.0,
  dockerfile 0.6.0). `dprint config update` refreshes versions but also
  rewrites the URL scheme to `npm:` — restore the `plugins.dprint.dev`
  URL scheme when using it, and confirm zero churn with `nix fmt` after
  any pin bump.
- **Never strike a whole markdown table row with surrounding tildes**
  (`~~| a | b |~~`): dprint's markdown plugin reparses the row and
  mangles it into a stray `~~` cell, dropping the closing marker (this
  destroyed the 09-01 report's annotations once and again during the
  2026-09-30 docs-health pass). Strike each cell's content individually
  (`| ~~a~~ verdict | ~~b~~ |`) — the `annotate-rows.py` format — which
  survives dprint realignment. Always re-run `check-rows.py` after any
  dprint pass over annotated reports.
- **`nix fmt` only formats nix files** (it is `nixfmt-tree`). Markdown,
  json, and yaml formatting is dprint: `nix develop -c dprint fmt`
  (bare `dprint` outside the devShell cannot resolve the plugin URLs).
