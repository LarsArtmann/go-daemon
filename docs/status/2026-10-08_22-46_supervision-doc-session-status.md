# Session Status: supervision recommendations doc + README wiring

**Date:** 2026-10-08 22:46 (Thursday)
**Scope:** this session only — answering "does go-daemon support priority/niceness?",
then shipping `docs/supervision.md` + the README Non-goals link. No Go code changed.

## Context

User asked whether the library makes it easy to run a daemon at a lower
resource priority (nice vs other workloads), e.g. via flake.nix. Verified at
source level: the library has zero priority/niceness surface (mechanism, not
policy), and its flake.nix is a hermetic build/devShell, not a service
template. Correct answer: priority is supervisor (systemd/NixOS) policy. User
asked for a dedicated recommendations doc linked from README — delivered.

## a) FULLY DONE

- Answered the priority question with code-level evidence (no `Nice`/
  `CPUWeight`/`IOScheduling*` anywhere in repo or flake; README Non-goals
  already said "systemd owns that").
- Created `docs/supervision.md` with five sections:
  1. unit settings matching the library's contracts (`Type=notify`,
     `WatchdogSec` auto-paired at half-interval, `TimeoutStopSec` vs
     `WithShutdownGracePeriod` default 5s, `Restart` + stale-socket story),
  2. resource priority (`Nice`, `CPUSchedulingPolicy=idle`, I/O idle,
     `CPUWeight` — weakest-knob-first guidance, memory caveat),
  3. NixOS module example + tmpfiles rule for `/run/<dir>`,
  4. hardening starters, plus an intro stating everything is optional.
- Every claim about library behavior was verified against source before
  writing (`socket.go`: listen → OnListen → READY → watchdog → serve;
  `CleanStaleSocket` at Start; `MkdirAll` dir creation; options list).
- README Non-goals now links the doc exactly in the user's requested shape
  ("systemd owns that — but see docs/supervision.md ... lower resource
  priority").
- Gates green on every touched surface: `dprint fmt` + `dprint check`,
  `check-rows` (3 files complete), FEATURES counts, and `check-readme`
  (README quickstart still compiles).

## b) PARTIALLY DONE

- `check-docs` is green in every section EXCEPT the pre-existing
  strike-presence failure on 5 legacy reports (see d/e). My new/edited files
  pass; the repo-wide gate still exits 1 — unchanged from session start.
- `docs/supervision.md` snippets are prose-verified but not machine-verified:
  the ini/nix fences have no compile/eval gate (only README's first go fence
  is gated by `check-readme`).
- The doc change shipped to master, but the living-doc wiring around it
  (CHANGELOG, FEATURES policy, AGENTS pointer, ROADMAP ideas) was not done.

## c) NOT STARTED

- CHANGELOG "Unreleased" entry for the new recommendations doc.
- FEATURES.md row (or an explicit decision that docs pages stay out of the
  feature inventory — affects the counts gate).
- AGENTS.md convention bullet pointing future sessions at
  `docs/supervision.md` (so "priority?" questions get linked, not re-derived).
- ROADMAP raw ideas surfaced by writing the doc: systemd socket activation
  (`LISTEN_FDS`), `MemoryMax`/`OOMPolicy` guidance, reload/`notify-reload`.
- CONTRIBUTING.md cross-link for consumer authors.
- Any link-rot gate for the new relative link (README → docs/supervision.md).

## d) TOTALLY FUCKED UP!

Nothing destroyed — but two honest misses:

1. **Gate sequencing miss:** I edited README but ran only `check-docs`
   (dprint) in the gate call; `check-readme` — the gate that exists precisely
   for README edits — was forgotten and only run now (green). A README change
   shipped unverified for one gate cycle. Same family as the 2026-10-05
   false-green incident: the gate I didn't run is the gate that's wrong.
2. **Flattened history:** I did not self-commit the docs change; the
   auto-commit daemon swept it as `6e60e56 chore: auto-commit 2 changed
   file(s) (heuristic)`. Project AGENTS explicitly says to commit
   descriptively when a change matters — this one did (new public-facing doc).

## e) WHAT WE SHOULD IMPROVE! (session self-review)

- **What I forgot:** check-readme in the first gate pass; the descriptive
  self-commit; CHANGELOG/living-doc wiring; an AGENTS.md pointer.
- **What could have been better:** all affected gates in ONE synchronous bash
  call per edit (dprint + check-docs + check-readme together); commit the
  docs change descriptively in the same call that wrote it.
- **What can still improve:** a snippet gate for ini/nix fences (or an
  explicit "advisory-only" note in the doc); strike-presence gate semantics
  for legacy reports; harvest this report's section-f into TODO_LIST/ROADMAP.

### Session checklist (struck = resolved)

| # | Item                                                | Verdict                                   |
| - | --------------------------------------------------- | ----------------------------------------- |
| 1 | ~~Priority question answered with source evidence~~ | ~~Verified: no priority surface; policy~~ |
| 2 | ~~docs/supervision.md written, source-anchored~~    | ~~Five sections, all claims checked~~     |
| 3 | ~~README Non-goals link in user's requested shape~~ | ~~README.md Non-goals section~~           |
| 4 | ~~dprint + check-rows + FEATURES counts green~~     | ~~Pass for all touched files~~            |
| 5 | ~~check-readme green after README edit~~            | ~~Run belatedly; passes~~                 |
| 6 | Descriptive self-commit                             | Missed; daemon flattened it (6e60e56)     |
| 7 | CHANGELOG/FEATURES/AGENTS wiring                    | Not started (see section f)               |

## f) Next (brainstorm-grade, up to 50; ROUTE VIA docs-health HARVEST)

1. HIGH — Fix the standing red gate: annotate the 5 strike-less
   `docs/status/` reports (docs-health ANNOTATE) or amend the gate to a
   cutoff date. `check-docs` exit 1 since 2026-10-05.
2. HIGH — CHANGELOG "Unreleased" entry for docs/supervision.md.
3. HIGH — HARVEST this report's open items into TODO_LIST/ROADMAP (this file
   is a snapshot, not a backlog).
4. MED — AGENTS.md bullet: supervisor recommendations live in
   docs/supervision.md; link it instead of re-deriving.
5. MED — Decide FEATURES.md policy for docs pages (row vs out of inventory).
6. MED — Verify the NixOS module snippet via `nix eval` scratch check or add
   a check-supervision gate (ini/nix fences currently ungated).
7. MED — Extend supervision.md: MemoryMax/OOMPolicy guidance (memory is not
   covered by nice; one sentence today, deserves a subsection).
8. MED — Add systemd socket activation (LISTEN_FDS) to ROADMAP as a raw
   idea; note in supervision.md that it is unsupported today.
9. LOW — CONTRIBUTING.md cross-link to supervision doc.
10. LOW — Link-rot gate for relative md links (README → docs/supervision.md).
11. LOW — docs/ index or README docs list (decisions, runbooks,
    supervision).
12. LOW — Consider `Type=notify-reload`/RELOAD support as a ROADMAP idea.
13. MED — Verify (via TODO_LIST, not assumed): watchdog flake fix row from
    the 2026-10-04 report (poll-based assertion) — still open?
14. MED — Verify: nil-logger contract test (f1, 2026-10-05 report) — done?
15. LOW — Post-flip runbook: remove GOPRIVATE refs so
    `pre-publish-audit.sh` exits 0 (currently expected-1).
16. HIGH (owner-blocked) — go-public flip (F1): makes DiscordSync adoption
    possible; repo currently private.
17. MED (owner-blocked) — Deferrable READY option (F3): the stated adoption
    driver for DiscordSync.
18. LOW — Re-check `.golangci.yml` drift vs project-discovery-daemon before
    the next sync.
19. LOW — Next Dependabot bump: remember close-and-adopt +
    `buildflow -s nix-hash-fix --fix` (vendorHash).
20. LOW — Consider a "docs snippets are advisory" note in CONTRIBUTING if no
    snippet gate is built.
21. LOW — Add `systemd.exec(5)`/`sd_notify(3)` man-page references to
    SECURITY or docs footer for discoverability.
22. LOW — ROADMAP: tmpfiles helper or NixOS module template generator?
    (probably YAGNI — decide at harvest).
23. LOW — Roadmap idea from session: expose socket path in $NOTIFY messages
    or docs example showing `systemctl status` readiness output.

## g) Questions (cannot be answered from the repo alone)

1. **FEATURES.md policy:** should `docs/supervision.md` (and future docs
   pages) appear as rows in FEATURES.md, or is the inventory library-API
   only? Decides f-item 5 and the counts-gate shape.
2. **Strike-gate semantics:** annotate the 5 legacy reports with verdict
   strikes (non-destructive, per docs-health ANNOTATE) or change
   `check-docs.sh` to only require strikes for reports written after a cutoff
   date? Both defensible — owner taste on history vs gate honesty.
3. **Snippet verification:** is it worth a gate that evaluates/compiles the
   ini/nix fences in supervision.md (like check-readme does for README's go
   fence), or should doc snippets stay advisory-only with a note?

## Gate state at close

- `nix run .#check-readme` — PASS
- `nix run .#check-docs` — exit 1, solely the 5 pre-existing strike-less
  legacy reports; dprint / check-rows / FEATURES counts all pass; the new
  report carries per-cell strikethrough (the annotate-rows.py format) so it
  does not become a 6th failure.
- Go build/test/vet/lint — not run (no Go code touched this session).

**WAITING FOR INSTRUCTIONS.**
