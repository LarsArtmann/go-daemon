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

## Go-public cluster (pre-flip)

| Task                                                                                           | Status       | Impact     | Effort | Evidence                                                                             |
| ---------------------------------------------------------------------------------------------- | ------------ | ---------- | ------ | ------------------------------------------------------------------------------------ |
| Post-flip: update AGENTS.md Gotchas (GOPRIVATE/`mkPreparedSource` notes become obsolete)       | 🔵 `BLOCKED` | doc truth  | XS     | 10-33 §f10 — gated on the flip; AGENTS.md:88-91 carries the private-repo notes today |
| Post-flip: pdd drops GOPRIVATE entry + `git+ssh` flake pin → plain input, re-derive vendorHash | 🔵 `BLOCKED` | the payoff | M      | 10-33 §f11 — pdd flake.nix:38 pins `git+ssh` today                                   |

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

| Task                                                                                                                                                                                                                                                                     | Status       | Impact | Effort | Evidence                                                                                                                                                                                                                         |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------ | ------ | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Verify pkg.go.dev rendering after proxy pickup                                                                                                                                                                                                                           | 🔵 `BLOCKED` | Low    | XS     | 2026-10-01: pkg.go.dev returns 404 — it cannot index a private module; verify right after the visibility flip (part of the post-flip runbook)                                                                                    |
| Bump `nextd` to go-daemon v0.5.0 and raise its `go` floor to 1.27.2 (v0.5.0 requires it via go-codec v0.3.3) | 🔴 `TODO`    | fleet  | S      | v0.5.0 cut 2026-11-27; `nextd/go.mod` still pins go-daemon v0.4.0 + `go 1.27.1` (checked 2026-11-27). The v0.4.1 re-tag plan is dropped: a `go 1.27` floor is impossible while go-codec declares `go 1.27.2` as its version contract, and v0.5.0 supersedes v0.4.x |

## Hygiene

| Task                                                                                                                                                                | Status    | Impact      | Effort | Evidence                                                       |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ----------- | ------ | -------------------------------------------------------------- |
| Harvest the five unannotated status reports (2026-10-01 05-46, 05-51, 07-05; 2026-10-04 08-20; 2026-10-05 12-28): pull §f items here, then annotate — they leave `nix run .#check-docs` failing on strike presence | 🔴 `TODO` | gate health | S      | check-docs exits 1 on strike presence for exactly those five files (verified 2026-11-27); dprint and the FEATURES counts pass |

## Consumer feedback (DiscordSync 2026-10-04)

| Task                                                                                                                                             | Status    | Impact                 | Effort | Evidence                                                                                                                                                                                               |
| ------------------------------------------------------------------------------------------------------------------------------------------------ | --------- | ---------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Deferrable READY: an option so `SocketServer.Run` skips its built-in `NotifyReady` and the consumer gates it (e.g. on a health-check round trip) | 🔴 `TODO` | consumer adoption (F3) | S      | `docs/feedback/new/2026-10-04_discordsync-consumer-feedback.md` F3 — note the lifecycle already notifies after `listen(2)` (socket.go:284), so the ask is a suppress/defer option, not moving the call |
