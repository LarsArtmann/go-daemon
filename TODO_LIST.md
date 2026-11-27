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

## Go-public cluster (flip executed 2026-11-27)

The repository is public as of 2026-11-27. The four owner gates (API
stability, name, docs visibility, the flip) were answered by that
instruction and are recorded with their evidence in
`docs/decisions/2026-11-27_visibility-flip.md`; the pre-flip and post-flip
tail items are closed, so this section carries no open work.

## CI gate completeness

| Task | Status | Impact | Effort | Evidence |
| ---- | ------ | ------ | ------ | -------- |

## Release follow-ups

| Task                                                                                                         | Status       | Impact | Effort | Evidence                                                                                                                                                                                                                                                           |
| ------------------------------------------------------------------------------------------------------------ | ------------ | ------ | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Verify pkg.go.dev rendering after proxy pickup                                                               | 🔵 `BLOCKED` | Low    | XS     | 2026-10-01: pkg.go.dev returns 404 — it cannot index a private module; verify right after the visibility flip (part of the post-flip runbook)                                                                                                                      |
| Bump `nextd` to go-daemon v0.5.0 and raise its `go` floor to 1.27.2 (v0.5.0 requires it via go-codec v0.3.3) | 🔴 `TODO`    | fleet  | S      | v0.5.0 cut 2026-11-27; `nextd/go.mod` still pins go-daemon v0.4.0 + `go 1.27.1` (checked 2026-11-27). The v0.4.1 re-tag plan is dropped: a `go 1.27` floor is impossible while go-codec declares `go 1.27.2` as its version contract, and v0.5.0 supersedes v0.4.x |

## Hygiene

| Task                                                                                                                                                                                                                                                                   | Status    | Impact      | Effort | Evidence                                                                                                                                                                                         |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ----------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Harvest the five unannotated historical status reports (2026-10-01 05-46, 05-51, 07-05; 2026-10-04 08-20; 2026-10-05 12-28) and this session's 2026-11-27 report §f: pull items here, then annotate — the five leave `nix run .#check-docs` failing on strike presence | 🔴 `TODO` | gate health | S      | check-docs exits 1 on strike presence for exactly those five files (verified 2026-11-27); the new report is self-struck, so it does not add a sixth failure; dprint and the FEATURES counts pass |

## Consumer feedback (DiscordSync 2026-10-04)

| Task                                                                                                                                             | Status    | Impact                 | Effort | Evidence                                                                                                                                                                                               |
| ------------------------------------------------------------------------------------------------------------------------------------------------ | --------- | ---------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Deferrable READY: an option so `SocketServer.Run` skips its built-in `NotifyReady` and the consumer gates it (e.g. on a health-check round trip) | 🔴 `TODO` | consumer adoption (F3) | S      | `docs/feedback/new/2026-10-04_discordsync-consumer-feedback.md` F3 — note the lifecycle already notifies after `listen(2)` (socket.go:284), so the ask is a suppress/defer option, not moving the call |
