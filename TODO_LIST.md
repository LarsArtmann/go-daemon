# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md); long-term ideas live in
> [ROADMAP.md](ROADMAP.md). Items are ranked by impact; status is verified, not assumed.
> Last harvested 2026-09-30 from `docs/status/2026-09-29_09-01_*`,
> `docs/status/2026-09-29_11-35_*`, and `docs/status/2026-09-30_00-43_*`
> (sections f), each item re-verified against code. All bounded items closed
> 2026-09-30 by the release-hardening pass (see CHANGELOG [Unreleased] and
> the v0.2.0 tag); both 09-29 reports are now annotated inline.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | External dependency or decision needed before it moves. |

## Standing (event-driven)

| Task                                                                                        | Status    | Impact | Effort | Evidence                                                                                                                 |
| ------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------------------------------------------------------------------------ |
| Re-run `TestParseSSEData_CorpusConformance` after the next sseparse tag: bump the test-only | 🔴 `TODO` | Med    | 15min  | Oracle at `sse_conformance_test.go:19` vs `sseparse.MustCorpus`; keeps the framing parser pinned to the WPT/Chromium     |
| dependency and re-run conformance                                                           |           |        |        | corpus (source: go-sse 07:03 report §f24). sseparse is still at v0.1.0 as of 2026-09-30 — verified via the module proxy. |
