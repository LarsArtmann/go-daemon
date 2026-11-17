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

| Task                                           | Status       | Impact | Effort | Evidence                                                                                                                                      |
| ---------------------------------------------- | ------------ | ------ | ------ | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Verify pkg.go.dev rendering after proxy pickup | 🔵 `BLOCKED` | Low    | XS     | 2026-10-01: pkg.go.dev returns 404 — it cannot index a private module; verify right after the visibility flip (part of the post-flip runbook) |
| Re-tag v0.4.1 with a minor-form floor: strip master `go 1.27.2` → `go 1.27` (run `go-version-auto-configure who-forces` first to name whatever forces 1.27.2), bump the pre-re-tag sibling pins (sseparse → v0.2.1, go-codec → v0.3.1), tag, then bump consumers (nextd) | 🔴 `TODO` | fleet | S | Published v0.4.0 `.mod` serves `go 1.27.1` (GOMODCACHE download cache, 2026-11-17); registered Active in gvac's POISONERS.md; nextd is held at parity `go 1.27.1` by go-daemon v0.4.0 + sseparse v0.2.0 (who-forces, 2026-11-17) |

## Hygiene

| Task                                                                                                                                                                | Status    | Impact      | Effort | Evidence                                                       |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ----------- | ------ | -------------------------------------------------------------- |
| Harvest the three 2026-10-01 status reports (05-46, 05-51, 07-05): pull §f items here, then annotate — they leave `nix run .#check-docs` failing on missing strikes | 🔴 `TODO` | gate health | S      | check-docs exits 1 on strike presence since the reports landed |

## Consumer feedback (DiscordSync 2026-10-04)

| Task                                                                                                                                             | Status    | Impact                 | Effort | Evidence                                                                                                                                                                                               |
| ------------------------------------------------------------------------------------------------------------------------------------------------ | --------- | ---------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Deferrable READY: an option so `SocketServer.Run` skips its built-in `NotifyReady` and the consumer gates it (e.g. on a health-check round trip) | 🔴 `TODO` | consumer adoption (F3) | S      | `docs/feedback/new/2026-10-04_discordsync-consumer-feedback.md` F3 — note the lifecycle already notifies after `listen(2)` (socket.go:284), so the ask is a suppress/defer option, not moving the call |
