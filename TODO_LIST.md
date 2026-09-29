# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md); long-term ideas live in
> [ROADMAP.md](ROADMAP.md). Items are ranked by impact; status is verified, not assumed.
> Last harvested 2026-09-30 from `docs/status/2026-09-29_09-01_*`,
> `docs/status/2026-09-29_11-35_*`, and `docs/status/2026-09-30_00-43_*`
> (sections f), each item re-verified against code.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | External dependency or decision needed before it moves. |

## High Impact

| Task                                                                                         | Status       | Impact | Effort | Evidence                                                                                                             |
| -------------------------------------------------------------------------------------------- | ------------ | ------ | ------ | -------------------------------------------------------------------------------------------------------------------- |
| Cut the next release tag (v0.1.2 vs v0.2.0 — negotiation is a behavior change, SSE were bug  | 🔵 `BLOCKED` | High   | 30min  | 3 spec fixes + q-value overhaul sit untagged on master (`36c2ecc`, `f1d1fe8`); plain `go get` still resolves v0.1.1. |
| fixes): tag, then floor-bump pdd's go.mod and coordinate its nix rev-pin                     |              |        |        | Needs user decision; pdd uses a rev pin so nothing is broken meanwhile                                               |
| Bump `go-codec` v0.2.0 → v0.3.0 to restore pin parity with the primary consumer (AGENTS.md   | 🔴 `TODO`    | High   | 30min  | `go.mod:7` pins v0.2.0; pdd `go.mod:19` uses v0.3.0. Dependency change → repair vendorHash with                      |
| pins this dep "to versions the primary consumer uses")                                       |              |        |        | `buildflow -s nix-hash-fix --fix`; never hand-paste                                                                  |
| Emit `Vary: Accept` from `WriteEncoded` when the response encoding depends on `Accept`       | 🔴 `TODO`    | High   | 1h     | `write.go:47` sets Content-Type but nothing sets Vary; HTTP-caching correctness gap for dual-encoding consumers      |
| Commit the negotiation-overhaul closeout with a descriptive message before any shared-branch | 🔴 `TODO`    | High   | 5min   | Overhaul + closeout exist only as six consecutive heuristic auto-commit messages on master;                          |
| push                                                                                         |              |        |        | history nowhere explains WHY the negotiation contract changed                                                        |

## Medium Impact

| Task                                                                                        | Status    | Impact | Effort | Evidence                                                                                                         |
| ------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ---------------------------------------------------------------------------------------------------------------- |
| Pin negotiation policy edge cases with tests: `application/*` wildcard echo, duplicate-q    | 🔴 `TODO` | High   | 2h     | Behavior is defined in doc comments (`negotiation.go:25-33,84-110`) but untested: 18 table cases don't cover     |
| precedence (`;q=0.5, ;q=0.9`), empty/`=`-less `q` params, never-serve-refused property, and |           |        |        | duplicates/empty-q/wildcard-type; a golden matrix makes regressions reviewable                                   |
| a golden behavior-matrix table                                                              |           |        |        |                                                                                                                  |
| Fuzz the Accept parser: `ResolveResponseCodec`/`acceptQuality`/`qualityParam` (seed browser | 🔴 `TODO` | High   | 1h     | Hand-rolled header parser `negotiation.go:89-148`; fuzz precedent exists (`FuzzParseSSEData`, `sse_test.go:231`) |
| headers, `*/*`, junk q-values, huge lists)                                                  |           |        |        |                                                                                                                  |
| Unit-test `bareMediaType`, `qualityParam`, `matchesEncoding` directly (whitespace variants, | 🔴 `TODO` | Med    | 1h     | Unexported helpers `negotiation.go:113-148` are only exercised transitively via the four exported functions      |
| `Q=` case, near-miss suffixes)                                                              |           |        |        |                                                                                                                  |
| ANNOTATE both 2026-09-29 status reports inline (strike each now-resolved §f item with its   | 🔴 `TODO` | Med    | 45min  | Their §f items are partially resolved by the 09-30 audit but carry no `done at` markers; docs-health ANNOTATE    |
| evidence), then decide ARCHIVE per docs-health classification                               |           |        |        | resolves inline, never rewrites; completeness gate: every archived file carries strikethroughs                  |
| Add a `nix run .#check-readme` flake app that compiles the README quickstart snippet         | 🔴 `TODO` | Med    | 1h     | The daemon committed a non-compiling README example (`caf1f0c`, fixed in `378607f`); a standing gate turns       |
|                                                                                             |           |        |        | that incident class into a build failure instead of a history blemish                                            |

## Low Impact

| Task                                                                                        | Status    | Impact | Effort | Evidence                                                                                                             |
| ------------------------------------------------------------------------------------------- | --------- | ------ | ------ | -------------------------------------------------------------------------------------------------------------------- |
| Re-run `TestParseSSEData_CorpusConformance` after the next sseparse tag: bump the test-only | 🔴 `TODO` | Med    | 15min  | Oracle at `sse_conformance_test.go:19` vs `sseparse.MustCorpus`; keeps the framing parser pinned to the WPT/Chromium |
| dependency and re-run conformance                                                           |           |        |        | corpus (source: go-sse 07:03 report §f24)                                                                            |
| Document the Accept-Charset / Accept-Encoding pass-through boundary in the negotiation doc  | 🔴 `TODO` | Low    | 15min  | `negotiation.go` doc comments define the media-type contract but are silent on the other Accept* headers             |
| comments (out of scope today — say so explicitly)                                           |           |        |        |                                                                                                                      |
| Add a godoc example for the negotiation round-trip (`negotiation_example_test.go`: request  | 🔴 `TODO` | Low    | 30min  | No runnable example exists; go-codec uses the `*_example_test.go` convention — adopt it here                         |
| codec → handler → `WriteEncoded`)                                                           |           |        |        |                                                                                                                      |
| Measure real coverage (`go tool cover -func`) vs the CI 80% gate; decide whether the floor  | 🔴 `TODO` | Low    | 30min  | CI gate is 80% (`ci.yml:45`); actual per-area numbers unknown — the go-sse fleet lesson was gates drifting from    |
| should track reality                                                                       |           |        |        | reality until information-free                                                                                       |
| Prune dead pdd-specific entries from the `.golangci.yml` ireturn allowlist (project-        | 🔴 `TODO` | Low    | 30min  | `.golangci.yml:157-159` allow project-discovery-sdk/ro types that never appear here; AGENTS.md's "mirrored from    |
| discovery-sdk, samber/ro), keeping the mirror claim honest                                  |           |        |        | pdd" wording needs "mirrored, locally pruned" if changed                                                           |
| Extend AGENTS.md: atomic-edit-under-auto-daemon gotcha (the `caf1f0c` class) + add          | 🔴 `TODO` | Low    | 30min  | Incident + gap documented in the 2026-09-30 status report §d1/§e1; Essential Commands lack `nix flake check`        |
| `nix flake check` to the Essential Commands sweep                                           |           |        |        |                                                                                                                      |
| Hygiene notes: LSP-cache-vs-CLI-gate triage (`lsp_restart` when phantom errors contradict   | 🔴 `TODO` | Low    | 30min  | Stale golangci_lint_ls errors persisted all session (CLI gates authoritative); dependabot PRs fail nix jobs by    |
| gates) + pre-agreed dependabot-vendorHash policy                                            |           |        |        | design on dep bumps — decide close-and-adopt vs maintainer-fix once                                                 |
