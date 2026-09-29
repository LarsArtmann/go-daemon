# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md), not here.

## Open items

| Status    | Item                                                                    | Notes                                                                                                                                                                                                                                                                                                                                                                                               |
| --------- | ----------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Re-run `TestParseSSEData_CorpusConformance` after the next sseparse tag | Keeps the consumer-oracle chain warm: when `github.com/larsartmann/go-sse/sseparse` cuts its next tag (new or extended corpus vectors), bump the test-only dependency and re-run conformance so the framing parser stays pinned to the reference corpus. Source: §f24 of go-sse's 2026-09-29 07:03 status report (`go-sse/docs/status/2026-09-29_07-03_ci-health-vendorhash-repair-pr-closure.md`). |
