# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md); long-term ideas live in
> [ROADMAP.md](ROADMAP.md). Items are ranked by impact; status is verified, not assumed.
> Last harvested 2026-09-30 from `docs/status/2026-09-30_10-33_go-public-session-review.md`
> (§f, 22 items), `docs/status/2026-09-30_11-29_release-v0.2.0-and-full-todo-closeout.md`
> (§f, 28 items), and residuals of `docs/status/2026-09-30_00-43_*` (§f) — every item
> re-verified against code at harvest time; already-done items were dropped, not carried.
> Cross-repo items route to [ROADMAP.md](ROADMAP.md) theme 3, never here.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | External dependency or decision needed before it moves. |

## Standing (event-driven)

| Task                                                                                        | Status    | Impact | Effort | Evidence                                                                                                                      |
| ------------------------------------------------------------------------------------------- | --------- | ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------- |
| Re-run `TestParseSSEData_CorpusConformance` after the next sseparse tag: bump the test-only | 🔴 `TODO` | Med    | 15min  | Oracle at `sse_conformance_test.go:19` vs `sseparse.MustCorpus`; keeps the framing parser pinned to the WPT/Chromium          |
| dependency and re-run conformance                                                           |           |        |        | corpus (source: go-sse 07:03 report §f24). sseparse is still at v0.1.0 as of 2026-09-30 — verified via go.mod + module proxy. |

## Go-public cluster (pre-flip)

| Task                                                                                           | Status       | Impact           | Effort | Evidence                                                                             |
| ---------------------------------------------------------------------------------------------- | ------------ | ---------------- | ------ | ------------------------------------------------------------------------------------ |
| Read README end-to-end; polish for external audience                                           | 🔴 `TODO`    | first impression | S      | 10-33 §f8 — README skimmed to line 80 only during the go-public review               |
| Write the reusable pre-publish audit script (history secrets, dep licenses, private-path scan) | 🔴 `TODO`    | leverage         | M      | 10-33 §f18 — generalizes task 1 above into a command                                 |
| Post-flip: update AGENTS.md Gotchas (GOPRIVATE/`mkPreparedSource` notes become obsolete)       | 🔵 `BLOCKED` | doc truth        | XS     | 10-33 §f10 — gated on the flip; AGENTS.md:88-91 carries the private-repo notes today |
| Post-flip: pdd drops GOPRIVATE entry + `git+ssh` flake pin → plain input, re-derive vendorHash | 🔵 `BLOCKED` | the payoff       | M      | 10-33 §f11 — pdd flake.nix:38 pins `git+ssh` today                                   |

## Owner decisions (BLOCKED — nothing to code)

| Task                                                                                                                       | Status       | Impact            | Effort | Evidence                                                                                                                                                                                                                                                                                                                                   |
| -------------------------------------------------------------------------------------------------------------------------- | ------------ | ----------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| API stability policy: v0.x no-promises vs v1 freeze                                                                        | 🔵 `BLOCKED` | contract decision | owner  | 10-33 §f2/§g1                                                                                                                                                                                                                                                                                                                              |
| Name decision: keep `go-daemon` vs rename pre-flip                                                                         | 🔵 `BLOCKED` | hard to reverse   | owner  | 10-33 §f3/§g2; collision data 2026-10-01: sevlyar/go-daemon (MIT, 2.3k★, active) dominates the search; 25+ same-name modules on pkg.go.dev; no hard path collision (`github.com/LarsArtmann/go-daemon` unclaimed) but the `daemon` package name collides with sevlyar's — both facts in `docs/decisions/2026-10-01_owner-decision-memo.md` |
| Visibility of `docs/status/`, `docs/planning/`, `reports/` in a public repo (scrub vs keep)                                | 🔵 `BLOCKED` | privacy           | owner  | 10-33 §f4/§g3, 11-29 §f22                                                                                                                                                                                                                                                                                                                  |
| The visibility flip itself (after the pre-flip cluster is green)                                                           | 🔵 `BLOCKED` | the decision      | owner  | 10-33 §c                                                                                                                                                                                                                                                                                                                                   |
| Fix GitHub Actions billing; then confirm CI green on the v0.2.0 tag commit AND the never-executed actions v7/v9.3 workflow | 🔵 `BLOCKED` | blocking trust    | owner  | 11-29 §f1/§f5/§b — `gh run 36645862196`: "payments failed"; tag-CI never ran                                                                                                                                                                                                                                                               |

## CI gate completeness

| Task | Status | Impact | Effort | Evidence |
| ---- | ------ | ------ | ------ | -------- |

## Release follow-ups

| Task                                                                                                          | Status       | Impact | Effort | Evidence                                                                        |
| ------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ------ | ------------------------------------------------------------------------------- |
| Cut v0.3.0 when a consumer wants `Vary: Accept` (Vary + godoc example + boundary docs sit untagged on master) | 🔵 `BLOCKED` | High   | S      | 11-29 §f3 — demand-gated; v0.2.0 deliberately shipped without Vary              |
| Visual check of the v0.2.0 GitHub Release page rendering                                                      | 🔴 `TODO`    | Low    | XS     | 11-29 §f16 — release created, never visually inspected                          |
| Verify pkg.go.dev rendering after proxy pickup                                                                | 🔴 `TODO`    | Low    | XS     | 10-33 §f13 — `go list -m …@v0.2.0` resolution verified; rendering never checked |

## Hygiene

| Task                                                                                                                                        | Status    | Impact | Effort | Evidence                                                                                                 |
| ------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | -------------------------------------------------------------------------------------------------------- |
| Ratify CONTRIBUTING.md: provenance unknown (`90f86db`, auto-daemon); align its plain-go setup section with the nix-first AGENTS.md commands | 🔴 `TODO` | Med    | XS     | 11-29 §f6/§g2 — CONTRIBUTING.md verified present 2026-09-30; commands predate the check-readme/nix gates |
| Clarify `90f86db` vendorHash provenance (who ran the repair?)                                                                               | 🔴 `TODO` | Med    | XS     | 11-29 §f7 — one-line answer closes it                                                                    |
| Re-verify the pruned `.golangci.yml` still diffs sensibly vs pdd's (mirror-drift check)                                                     | 🔴 `TODO` | Low    | S      | 11-29 §f13 — last diff check was 2026-09-30 pre-prune                                                    |
| Add `ExampleParseSSEData` + a socket.go godoc example (pattern established by `ExampleResolveResponseCodec`)                                | 🔴 `TODO` | Low    | S      | 11-29 §f25 — `negotiation_example_test.go` is the template                                               |
| `nix flake check --all-systems` pass (darwin omitted so far)                                                                                | 🔴 `TODO` | Low    | M      | 11-29 §f15                                                                                               |
| Derive FEATURES.md counts mechanically (grep) instead of hand-maintained prose — they rotted once already                                   | 🔴 `TODO` | Low    | XS     | 00-43 §f24 — counts drifted (stale "1 PLANNED") and needed a hand fix on 2026-09-30                      |
| README: add a short "Non-goals" pointer to ROADMAP (mechanism-not-policy on the sales page)                                                 | 🔴 `TODO` | Low    | XS     | 00-43 §f26                                                                                               |
