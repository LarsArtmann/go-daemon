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

## Medium Impact

| Task                                                                                        | Status    | Impact | Effort | Evidence                                                                                                         |
| ------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ---------------------------------------------------------------------------------------------------------------- |
| Unit-test `bareMediaType`, `qualityParam`, `matchesEncoding` directly (whitespace variants, | 🔴 `TODO` | Med    | 1h     | Unexported helpers `negotiation.go:113-148` are only exercised transitively via the four exported functions      |
| `Q=` case, near-miss suffixes)                                                              |           |        |        |                                                                                                                  |
| ANNOTATE both 2026-09-29 status reports inline (strike each now-resolved §f item with its   | 🔴 `TODO` | Med    | 45min  | Their §f items are partially resolved by the 09-30 audit but carry no `done at` markers; docs-health ANNOTATE    |
| evidence), then decide ARCHIVE per docs-health classification                               |           |        |        | resolves inline, never rewrites; completeness gate: every archived file carries strikethroughs                   |
| Add a `nix run .#check-readme` flake app that compiles the README quickstart snippet        | 🔴 `TODO` | Med    | 1h     | The daemon committed a non-compiling README example (`caf1f0c`, fixed in `378607f`); a standing gate turns       |
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
| Measure real coverage (`go tool cover -func`) vs the CI 80% gate; decide whether the floor  | 🔴 `TODO` | Low    | 30min  | CI gate is 80% (`ci.yml:45`); actual per-area numbers unknown — the go-sse fleet lesson was gates drifting from      |
| should track reality                                                                        |           |        |        | reality until information-free                                                                                       |
| Prune dead pdd-specific entries from the `.golangci.yml` ireturn allowlist (project-        | 🔴 `TODO` | Low    | 30min  | `.golangci.yml:157-159` allow project-discovery-sdk/ro types that never appear here; AGENTS.md's "mirrored from      |
| discovery-sdk, samber/ro), keeping the mirror claim honest                                  |           |        |        | pdd" wording needs "mirrored, locally pruned" if changed                                                             |
