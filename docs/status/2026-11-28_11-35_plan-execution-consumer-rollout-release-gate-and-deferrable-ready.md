# Status — 2026-11-28 11:35 — plan execution: consumer rollout, one release gate, deferrable READY

> **Session scope:** executed the first half of
> `docs/planning/2026-11-27_22-49_release-trust-gate-parity-and-consumer-rollout.md`
> (T1, T2, T3, T5, T6, T8, T9, T11, most of T13) plus the archive annotation
> pass the docs battery demanded. Every claim below was produced in this
> session; commands and their exit codes are in the verification log.

## TL;DR

- **Both consumers run on go-daemon v0.5.0**: project-discovery-daemon
  (`5cbc99c`, `vendorHash` repaired, suite green after a wire audit) and
  nextd (`fe57b39`, `go 1.27.2`).
- **One command runs every CI gate**: `nix develop -c release-check`
  (`scripts/release-check.sh`). On the clean tree at `de01414` it reports
  ALL GATES GREEN, 97.0% statement coverage, `govulncheck` clean.
- **The flake builds Go 1.27.4**, so CVE-2026-97054 is out of the devShell,
  the release gate and the nix build; cross-compile sweep green on 11 targets.
- **Deferrable READY shipped**: `Lifecycle.DeferReady` + `Server.MarkReady`
  (the DiscordSync adoption driver, feedback F3), default behavior unchanged.
- **The docs corpus is annotated**: 117 open items across the archived
  reports now carry a verdict; `docs-gates run .` is 10/10 locally.
- **Master CI is red right now, and that is mine**: the docs battery I added
  to CI fails on the runner because `actions/checkout` is shallow (no tags,
  no old commits), so three verdicts cannot resolve there. One-line fix
  pending (`fetch-depth: 0`); see section d).

## Brutal self-review (asked first)

**What did I forget?** I added a gate to CI without asking what that gate
needs on a runner. Locally the docs battery reads the full git history; a
shallow checkout has neither tags nor the commits some verdicts cite. The
result is a CI step that is red on every push until `fetch-depth: 0` lands.
I also committed `27399f0` while the corpus was still red under that same
battery (the annotations were still being written), so two consecutive
master runs are red for reasons I knew about.

**What could I have done better?** I trusted the previous session's
"docs-gates 10/10" instead of re-running the battery first. It reported 85
unharvested archived items on the first run. Re-running every inherited
claim before building on it would have put the annotation pass at the start
of the session, before the gate went into CI.

**What could I still improve?** The flake's Go override builds Go from
source, which makes a cold `nix build` minutes slower and has no cache in
CI; a binary overlay or a store cache would cut that. `release-check` runs
the Go tests and then `nix build` builds the module again; acceptable, but
it is the slowest path. The tag trigger in CI is untested (no tag was
pushed). The nightly `-count=15` workflow is not written.

## a) FULLY DONE

- **~~T1 pdd on v0.5.0.~~** done. Wire audit before the bump: no raw
  `time.Duration` crosses the wire (the SDK's `Duration` type marshals as milliseconds), no
  test asserts a `null` slice. Built and tested against v0.5.0 through an
  alternate modfile, then bumped for real, repaired `vendorHash` with
  `buildflow -s nix-hash-fix --fix`, `nix build` green, pushed `5cbc99c`.
- **~~T2 nextd on v0.5.0.~~** done. nextd uses go-daemon only for the lifecycle, the
  liveness probe and the HTTP client; every response is its own
  `encoding/json/v2` output. `go 1.27.1` -> `1.27.2`, suite green, AGENTS and
  CHANGELOG updated, pushed `fe57b39`.
- **~~nextd test isolation fix.~~** done. `TestRunUsageErrors` fell back to
  `$HOME/.local/state/nextd`; when the user daemon's probe timed out it
  parked its test keys in the real ledger (observed once under `-race`; the
  two `o/r#N` entries were removed by hand). Every row now passes
  `--state <tempdir>`.
- **~~T3 + T6 release gate.~~** done. `scripts/release-check.sh` runs actionlint,
  build, README quickstart, vet, race tests + 80% coverage floor,
  govulncheck, lint, `nix build`, the all-systems flake check, `check-docs`
  and the rev-pinned docs battery; reports every gate, exits non-zero.
  Negative-tested with a deliberately failing test (exit 1, three gates
  named). Ships in the devShell, not as a flake app, because `nix run`
  chdirs into the store copy.
- **~~govulncheck attribution.~~** done. `scripts/govulncheck-nonstdlib.py` reads the
  JSON stream and fails the gate only on findings outside `stdlib`; a
  toolchain finding is printed and attributed.
- **~~T5 docs battery in CI.~~** done. The nix job runs
  `nix run github:Artmann-Technologies/docs-gates/bfdefba… -- run .`
  after a `safe.directory` step. Not a flake input (it would triple
  `flake.lock` and publishes no aarch64-darwin package).
- **~~T4 release trigger.~~** done. `ci.yml` also runs on `v*` tags, so the tagged
  tree gets the identical gate set; CONTRIBUTING documents the tag flow.
- **~~T8 wire contract.~~** done. `WriteEncoded(..., nil, ...)` no longer panics (nil
  codec = default JSON, same rule as the wrappers); `ContentTypeFor(nil)`
  returns `application/json`. `TestDefaultJSONCodec_WireShapes` pins `[]`
  and `{}` for nil slice and map; `TestDefaultJSONCodec_DurationIsAnEncodeError`
  pins the 500 envelope.
- **~~T9 adoption surface.~~** done. Examples `ExampleJSONv2Codec`,
  `ExampleWriteJSONWithCodec`, `ExampleWriteErrorfWithCodec`;
  `TestJSONv2Codec_NilOptionElements` (nil option elements are ignored);
  `FuzzResolveResponseCodec` also runs with a `JSONv2Codec` request codec,
  60 s soak, 4.1M execs, no crasher.
- **~~T11 deferrable READY.~~** done. `Lifecycle.DeferReady` makes `Start` skip
  `sd_notify READY=1` and the watchdog; `Server.MarkReady(ctx)` performs
  both for the consumer (watchdog started once). Tests
  `TestSocketServer_DeferReady` (mutation-tested: fails when the skip is
  removed) and `TestSocketServer_ReadyByDefault`; example
  `ExampleServer_MarkReady`. FEATURES, glossary, AGENTS, CHANGELOG and the
  DiscordSync feedback doc updated.
- **~~T13 toolchain.~~** done. nixpkgs `go_1_27` is still 1.27.3, so the flake
  overrides it to the 1.27.4 source tarball (go.dev checksum). devShell
  reports `go1.27.4`, `govulncheck` reports "No vulnerabilities found".
  Cross-compile on 1.27.4: linux, darwin, windows, freebsd, openbsd,
  dragonfly, solaris, js/wasm, wasip1/wasm all build.
- **~~Socket tests under a nix devShell.~~** done. Long test names overflowed the
  107-byte socket path; `socketTempDir(t)` replaces `t.TempDir()` in the 29
  socket-path sites. The race suite is green inside `nix develop`.
- **~~Archive annotation pass.~~** done. A subagent resolved 89, then 28 more items
  the `audit-all` replay flagged: 14 done with evidence, 14 decided or
  moot, 17 routed to TODO_LIST, 46 routed to ROADMAP, the rest superseded
  or duplicate. TODO_LIST rebuilt (25 open, 5 blocked), ROADMAP carries the
  cross-repo pointers.

## b) PARTIALLY DONE

- **T10 security operations.** Secret scanning, push protection and
  Dependabot alerts are on. CodeQL default setup is `not-configured`: the
  API answers 404 with the current `gh` token (scopes: `repo`, `read:org`,
  `gist`, `admin:public_key`). Non-provider secret patterns stay `disabled`
  after the PATCH. Branch protection is not enabled (it would reject the
  auto-commit daemon's direct pushes to `master`). The deploy key
  `systemnix-ci-readonly` is kept: SystemNix's lock still fetches go-daemon
  over `ssh://` three times, so deleting it first breaks SystemNix CI.
- **T7 clean-tree verification.** Done on `de01414` for every local gate.
  Not done: a run against the tagged tree of a release (nothing new is
  tagged) and a clean-tree `nix build` store path recorded per hand (CI's
  nix job builds every push instead).
- **T14 test depth.** `-shuffle=on` and `-race -count=3` are green
  locally; neither is wired into CI yet, and there is no nightly
  `-count=15` workflow.
- **CI gate parity.** Local: all green. CI: the Go job is green (37 s), the
  nix job fails in the docs battery on a shallow checkout.

## c) NOT STARTED

- T12 store front (homepage, "public since" line, announcement decision).
- T15 SystemNix lock float, stale worktrees, Dependabot re-check,
  `scripts/consumer-smoke.sh`.
- T16 quality ledgers (art-dupl baseline gate, erraudit baseline artifact,
  bare-assertion sweep, branching-flow decision).
- T17 benchmarks and the `sd_notify` sequence golden.
- T18 archive manifest provenance and the ledger convention in AGENTS.
- T19 pdd ergonomics (`Vary: Accept`, `ClassifyTransportError`,
  `WriteErrorfWithCodec` adoption in pdd).
- T20 release presentation policy.
- A release cut for the unreleased work (`DeferReady`, the nil-codec fix,
  the wire-shape tests are on `master` only; both consumers pin v0.5.0).

## d) TOTALLY FUCKED UP

1. **Master CI is red on `de01414` (run 46134887171).** The docs battery
   step fails with three `annotate` findings that do not exist locally:
   `2026-09-30_10-33` line 106 "claims tag `v0.2.0`, which does not exist",
   `2026-10-01_03-52` line 156 "`bd14443` does not resolve to a commit here",
   `2026-10-01_09-59` line 152 "claims tag `v0.3.0`, which does not exist".
   The runner's checkout is shallow. Fix: `fetch-depth: 0` on the nix job's
   checkout (plus fetching tags), then a green run.
2. **Master CI was already red on `27399f0` (run 46105865582).** I wired
   the battery into CI while 85 archived items were still unharvested; the
   local gate showed exactly that failure and the commit went out anyway
   (pushed by the auto-commit daemon at 04:05).
3. **A nextd test wrote into the real nextd ledger** during my bump
   verification. Pre-existing bug, my trigger. Fixed at the root in nextd;
   worth a check that no other nextd test reaches `$HOME`.
4. **The inherited "docs-gates 10/10" was not true under `audit-all`.** Not
   mine, but I repeated it for an hour before running the battery.
5. **Sloppy editing on my side, all caught by gates:** a broken duplicate
   test function sat in the working tree (build broken for about a minute
   while the auto-commit daemon runs), an example imported a package that
   does not exist (`encoding/json/v1`), `strconv.FormatInt` with base 1,
   and a 670 KB `govulncheck` JSON dump printed into the session because I
   assumed one object per line. None reached a commit.

## e) WHAT WE SHOULD IMPROVE

- **A CI step is not done until it is green on the runner.** The local gate
  cannot see checkout depth, secrets, or runner ownership. Push, watch the
  run, then claim the step.
- **Run every inherited gate before building on its verdict.**
- **`release-check` should refuse to run on a dirty tree in release mode**
  (an opt-in `--require-clean`), so "green on a dirty tree" cannot be
  reported as release evidence again.
- **Pin the Go toolchain by binary, not by source build**, or cache the
  built Go in CI; the override costs minutes on every cold runner.
- **nextd tests must never resolve `$HOME`**: a `NEXTD_STATE` default in the
  test binary (TestMain) would make the whole suite hermetic, not one table.
- **pdd's admin package has timing flakes** (`TestBridgeStopEndsInFlightHandlerFetches`,
  `TestSelfMetricsScrape`, `TestSelfMetricsCountsFanOutPerSubscriber`):
  1/20 runs on v0.4.0, 3/15 on v0.5.0, each a different test. Not caused by
  the bump; worth a row in pdd.
- **`scripts/release-check.sh` cleans its own `mktemp -d` with `rm -rf`**
  in an EXIT trap. That is standard for a script's private temp dir, but the
  fleet rule says `trash`, never `rm`; decide whether scripts are in scope.

## f) Up to 50 things to get done next

1. Give the CI nix job a full checkout (`fetch-depth: 0`, tags included)
   so the docs battery can resolve the tags and commits its verdicts cite;
   watch the run go green. Routed to TODO_LIST.md (Documentation and
   gates: full checkout for the CI docs battery).
2. Re-run the docs battery on the runner after item 1 and compare with the
   local matrix; any finding that exists only on CI is a parity bug.
3. Cut the release that carries `DeferReady`, the nil-codec fix and the
   wire-shape tests; bump both consumers to it. Routed to TODO_LIST.md
   (Release follow-ups).
4. Push a real `v*` tag and confirm the tag-triggered CI run happens and
   is green; the tag trigger is untested. Routed to TODO_LIST.md (Release
   follow-ups: prove the tag trigger).
5. Add `-shuffle=on` to the CI test step and to `release-check.sh`. Routed
   to TODO_LIST.md (nightly `-count=15` and `-shuffle=on`).
6. Write the nightly `-count=15` timing workflow. Same TODO row.
7. Reproduce and fix the watchdog timing flake named in the archived
   2026-10-04 report. Routed to TODO_LIST.md.
8. Add shellcheck and `bash -n` gates for `scripts/*.sh`. Routed to
   TODO_LIST.md.
9. Add `scripts/pre-publish-audit.sh` as a `workflow_dispatch` job. Routed
   to TODO_LIST.md.
10. Add gitleaks as a scheduled CI job. Routed to TODO_LIST.md.
11. Add a store cache to the CI nix job (magic-nix-cache or cachix). Routed
    to TODO_LIST.md.
12. Enable CodeQL default setup and non-provider secret patterns; needs a
    token with the security scope or a click in the repository settings.
    Routed to TODO_LIST.md (blocked on the owner).
13. Decide on branch protection versus the auto-commit daemon. Section g.
14. Float SystemNix's lock off the `ssh://` go-daemon inputs, then delete
    the `systemnix-ci-readonly` deploy key. Routed to TODO_LIST.md (repo:
    SystemNix).
15. Add the pull request template with a ran-release-check checkbox. Routed
    to TODO_LIST.md.
16. Make the `check-docs` strike gate same-day-aware. Routed to TODO_LIST.md.
17. Pin the FEATURES row total in `check-docs.sh`. Routed to TODO_LIST.md.
18. Add a negative corpus or property test for `check-rows.py`. Routed to
    TODO_LIST.md.
19. Document the `DeterministicCodec` limitation on `JSONv2Codec`. Routed to
    TODO_LIST.md.
20. Add message-wording stability to the release checklist (`Validate()`
    error strings are consumer API). Routed to TODO_LIST.md.
21. Register flight-recorder metrics only after `flightrecorder.New`
    succeeds. Routed to TODO_LIST.md.
22. Pin the erraudit version and document the audit-vs-lint protocol.
    Routed to TODO_LIST.md.
23. Cross-reference `sse_conformance_test.go` and the dispatch table, and
    tag the table rows with the WHATWG section. Routed to TODO_LIST.md.
24. Cite `sd_notify(3)` in `docs/supervision.md`. Routed to TODO_LIST.md.
25. Add the AGENTS gotcha that lint tools can under-report. Routed to
    TODO_LIST.md.
26. Re-check that Dependabot opens PRs on the public repo. Routed to
    TODO_LIST.md.
27. Decide the branching-flow adopt/reject question. Routed to TODO_LIST.md
    (owner).
28. Decide the public store front. Routed to TODO_LIST.md (owner, blocked).
29. Adopt `DeferReady` in DiscordSync and delete its hand-rolled notify
    port. Routed to TODO_LIST.md (repo: DiscordSync).
30. Make every nextd test hermetic with a `TestMain` state default. Routed
    to ROADMAP.md theme 3 (nextd).
31. Triage the pdd admin timing flakes. Routed to ROADMAP.md theme 3 (pdd).
32. T19 pdd ergonomics. Routed to ROADMAP.md theme 3 (pdd).
33. Add the `release-check --require-clean` mode. Routed to ROADMAP.md
    theme 5.
34. Replace the Go source-build override with a binary overlay or a cached
    build. Routed to ROADMAP.md theme 5.
35. Benchmarks: pooled-buffer write, `JSONv2Codec`, SSE line reader. Routed
    to ROADMAP.md theme 2.
36. `sd_notify` message-sequence golden test. Routed to ROADMAP.md theme 2.
37. art-dupl accepted-clone baseline gate. Routed to ROADMAP.md theme 3.
38. Release presentation policy (prerelease versus Latest, v0.4.x support).
    Routed to ROADMAP.md theme 4 (release presentation policy).
39. Archive manifest provenance and the ledger convention in AGENTS.
    Routed to ROADMAP.md theme 3 (archive manifest provenance).
40. The go-sse, go-datastar and crush-config pointers harvested on
    2026-11-28. Routed to ROADMAP.md theme 3.

## g) Three questions I cannot answer myself

1. **Branch protection or auto-commit daemon?** Required status checks on
   `master` would reject the daemon's direct pushes, and it pushes broken
   intermediates (it pushed `27399f0` while I still considered it
   unfinished). Keep `master` unprotected, or protect it and change how the
   daemon lands commits?
2. **Cut a release now?** `DeferReady` and the nil-codec fix are unreleased
   on `master`; both consumers pin v0.5.0. New API argues for v0.6.0, a
   patch would hide the new option. Tag v0.6.0 now, or wait for the CI
   checkout fix and the tag-trigger proof?
3. **Which credential for the security settings?** CodeQL default setup
   answers 404 and the non-provider secret patterns stay disabled with the
   current `gh` token (`repo`, `read:org`). Should I use a token with the
   security scope, or do you enable both in the repository settings?

## Verification log

```text
date                                    -> 2026-11-28 11:35 CET
pdd: go test ./... -race (v0.5.0)       -> ok; admin package flakes 3/15 (pre-existing family, 1/20 on v0.4.0)
pdd: nix build .#default                -> ok after buildflow nix-hash-fix (5cbc99c, pushed)
nextd: go test ./... -race -count=1     -> ok (fe57b39, pushed)
go build ./... / go vet ./...           -> ok (devShell go1.27.4)
go test ./... -race -count=1            -> ok, coverage 97.0% of statements
go test -shuffle=on / -race -count=3    -> ok
golangci-lint run (nix run .#lint)      -> 0 issues
actionlint -no-color                    -> clean
govulncheck ./... (Go 1.27.4 devShell)  -> No vulnerabilities found
govulncheck ./... (Go 1.27.3, earlier)  -> 1 stdlib finding, GO-2026-6859
cross-compile sweep, 11 targets         -> all build (CGO_ENABLED=0)
go test -fuzz FuzzResolveResponseCodec  -> 60 s, 4,096,651 execs, PASS
nix build .#default                     -> ok (clean tree, de01414)
nix flake check --no-build --all-systems -> all checks passed
nix run .#check-docs                    -> all documentation gates pass
docs-gates run .                        -> 10/10 PASS (local, full checkout)
nix develop -c release-check            -> ALL GATES GREEN (clean tree, de01414)
release-check negative test             -> exit 1, three failing gates named
DeferReady mutation test                -> test fails when the skip is removed
CI run 46101923938 (f2c8bbc)            -> success
CI run 46105865582 (27399f0)            -> FAILURE (docs battery: unharvested items, since annotated)
CI run 46134887171 (de01414)            -> FAILURE (nix job: annotate on a shallow checkout; Go job green in 37 s)
```
