# Status Report: art-dupl notify.go Dedup Session

> **Live remainder:** §a done; §b/c/e/f items are closed by later work (nil-logger tests exist in notify_test.go) or routed to TODO_LIST/ROADMAP. Live work: TODO_LIST.md.

- **Timestamp:** 2026-10-05 12:32 CEST
- **Scope:** Single session, single trigger: user-supplied `art-dupl --sort total-tokens -t 1` output showing 2 clone findings.
- **Session products:** `notify.go` refactor (7be4443), AGENTS.md note + 2 doc formatting repairs (1072d6a).
- **Format note:** Written as `.md` per explicit user instruction (status-report skill default is HTML; user instruction wins).

## Self-Review (asked directly: forgotten / could do better / could still improve)

### What did I forget?

1. **Nil-path test coverage was never verified.** I extracted `loggerOrDefault` and the full suite passed, but I never checked whether ANY test actually exercises the nil-logger branch. "Tests pass" is not "nil path is tested". Still unverified (task f1).
2. **The commit-policy tension.** AGENTS.md gotchas explicitly say: "commit descriptively yourself when the change matters (heuristic chore: auto-commit messages flatten history)". The harness rule says NEVER COMMIT without an explicit user request. I silently picked the harness rule, and the daemon promptly flattened this session into `chore: auto-commit 1 changed file(s) (heuristic)` commits (7be4443, 1072d6a). I forgot to surface this conflict when it mattered.
3. **Exact-flag verification.** My final art-dupl run dropped the user's `--suggest-generics --rich-text` flags. Almost certainly immaterial (threshold identical), but "verified" should mean "verified with your command".
4. **Hermetic gates.** No `nix build` / `nix flake check` this session. Defensible (zero dependency changes, so vendorHash cannot drift) but it means the nix-side gate claim in this report is "not run", not "green".

### What could I have done better?

1. Fixed-on-sight is good, but I fixed the 2 pre-existing unformatted docs files **after** discovering them via a red gate, mid-flow. A `dprint check` earlier (before the AGENTS.md edit) would have surfaced the pre-existing drift with a cleaner separation of "my change" vs "inherited drift".
2. I should have looked at the complete session diff (`git diff HEAD~2`) as a final review step instead of verifying file-by-file.
3. When re-running art-dupl, use the exact user invocation, not a reduced one.

### What could we still improve?

1. **art-dupl has no enforcement.** The `-t 1` finding count is now exactly 1 known finding (negotiation `default:` pair). Nothing fails if a new clone lands tomorrow. A pin-the-baseline gate script would convert the tool from advisory to regression-catching.
2. **Unformatted docs landed on master** (2026-10-04 files) because nothing runs `dprint check` at commit time; the auto-commit daemon commits regardless of format state. Recurrence is guaranteed without a hook.
3. **Every session re-litigates fix-vs-document for lint findings.** A one-line policy ("fix structural duplication; document idioms") in AGENTS.md would end the debate.

## a) FULLY DONE

| #                                        | Item                                                                                                                                                                                                                                                                   | Evidence                                                                                                                                                           |
| ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| ~~1~~ done (docs-health pass 2026-11-27) | ~~Triaged BOTH art-dupl findings: negotiation.go `default:` pair confirmed as documented accepted idiom (AGENTS.md), left untouched; notify.go `if logger == nil` pattern identified as 3 sites (Watchdog, WatchdogInterval, sendNotify), not 2 as the tool reported~~ | ~~grep found exactly 3 occurrences; AGENTS.md conventions bullet~~                                                                                                 |
| ~~2~~ done (docs-health pass 2026-11-27) | ~~Extracted `loggerOrDefault` helper (notify.go:137): one implementation of the "nil logger defaults to slog.Default()" contract; replaced all 3 inline conditionals~~                                                                                                 | ~~commit 7be4443 (daemon heuristic, notify.go)~~                                                                                                                   |
| ~~3~~ done (docs-health pass 2026-11-27) | ~~Go gates green post-change: `go build`, `go vet`, `go test -race -count=1` (1.542s ok), `golangci-lint run` (0 issues)~~                                                                                                                                             | ~~session command output~~                                                                                                                                         |
| ~~4~~ done (docs-health pass 2026-11-27) | ~~art-dupl `-t 1` re-run: notify.go finding GONE; only the documented negotiation `default:` pair remains ("1 shown")~~                                                                                                                                                | ~~art-dupl output: `Detected 95 clone groups, 1 shown`~~                                                                                                           |
| ~~5~~ done (docs-health pass 2026-11-27) | ~~AGENTS.md updated: art-dupl bullet now records that the `default:` pair is the ONLY `-t 1` finding and that new notify entry points must call `loggerOrDefault`, not re-inline the nil check; edit grep-verified on disk~~                                           | ~~commit 1072d6a (AGENTS.md)~~                                                                                                                                     |
| ~~6~~ done (docs-health pass 2026-11-27) | ~~Fixed pre-existing docs drift on sight: 2 files unformatted since 2026-10-04 (table padding) — `dprint fmt`; dprint sub-check clean afterward~~                                                                                                                      | ~~commit 1072d6a. CORRECTION: "check-docs exit 0" was a false claim — the exit code was captured after a tail pipe and belonged to `tail`, not the gate (see d4)~~ |

## b) PARTIALLY DONE

| #                                                                            | Item                              | Works now                                        | Open gap                                                                                                                                                                                         | Effort |
| ---------------------------------------------------------------------------- | --------------------------------- | ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------ |
| 1 **→ residual carried in §f below; live work tracked in TODO_LIST/ROADMAP** | Nil-logger contract verification  | Implementation extracted; full race suite passes | No evidence any test exercises the nil branch (helper or entry points). If the nil path is production-dead, a dedicated test is nice-to-have; if consumers rely on it, it is a contract test gap | S      |
| 2 **→ residual carried in §f below; live work tracked in TODO_LIST/ROADMAP** | Clone-detection regression safety | AGENTS.md documents the single accepted finding  | No automated gate; compliance relies on future sessions reading AGENTS.md                                                                                                                        | M      |
| 3 **→ residual carried in §f below; live work tracked in TODO_LIST/ROADMAP** | Session history quality           | Changes are committed and gated                  | Both commits carry heuristic daemon messages; whether squashing is possible depends on push state (not checked — remote research was out of session scope)                                       | S      |

## c) NOT STARTED

| #                                                           | Item                                                                                            | Why not started                                                                                                              | Still wanted?                                                                 |
| ----------------------------------------------------------- | ----------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| 1 **→ not started; live work tracked in TODO_LIST/ROADMAP** | Harvest prior status reports' section-f items into TODO_LIST/ROADMAP                            | Out of scope per user instruction ("do not research unrelated"); AGENTS.md flags this as standing debt from earlier sessions | Yes — AGENTS.md says many items are already done and need verify-then-harvest |
| 2 **→ not started; live work tracked in TODO_LIST/ROADMAP** | art-dupl baseline gate script                                                                   | New idea from this session's reflection                                                                                      | Yes — cheap, high leverage                                                    |
| 3 **→ not started; live work tracked in TODO_LIST/ROADMAP** | Commit-time `dprint check` (hook or daemon pre-step)                                            | Needs a decision on where it lives (crushrc hook vs flake app vs CI)                                                         | Yes — the 2026-10-04 drift proves the gap recurs                              |
| 4 **→ not started; live work tracked in TODO_LIST/ROADMAP** | Consumer-side check: does project-discovery-daemon pass nil loggers to the notify entry points? | Cross-repo research ruled out of scope this session                                                                          | Yes — determines whether nil path is hot contract or defensive-only           |
| 5 **→ not started; live work tracked in TODO_LIST/ROADMAP** | `nix build` + `nix flake check` for this session's tree                                         | Skipped: zero dependency changes, vendorHash cannot drift                                                                    | Low priority; run at next dependency change or release cut                    |
| 6 **→ not started; live work tracked in TODO_LIST/ROADMAP** | Annotate the 2026-10-04 status/feedback reports (docs-health ANNOTATE)                          | Out of scope; both remain unannotated per check-docs output                                                                  | Yes — docs-health standing work                                               |

## d) TOTALLY FUCKED UP

**Go gates green; docs gate red.** Nothing broken by this session's code edits (build/vet/race/lint/art-dupl all pass). The documentation gate `check-docs` FAILS (exit 1) — and, corrected late: it was already failing before this session started. What IS fucked up, honestly assessed:

1. **Commit history for meaningful work is now noise.** This session's only real refactor plus its documentation live inside `chore: auto-commit 1/3 changed file(s) (heuristic)` messages. A future reader of `git log` cannot tell that the nil-logger contract got consolidated. Root cause: unresolved conflict between harness never-commit rule and AGENTS.md self-commit guidance. Mitigation: none yet (question g1).
2. **Master carried unformatted docs for ~a day.** The two 2026-10-04 files passed the daemon straight onto master; only this session's unrelated gate run caught them. Root cause: no format gate between edit and commit. Mitigation: fixed now, recurrence unprevented (task f8).
3. **The tool under-reports.** art-dupl reported 2 sites for the notify pattern; there were 3 (it missed Watchdog:80). Root cause unknown (detection heuristic at 1-token threshold). Impact: trusting tool output alone would have left one duplicated conditional in place. Lesson: grep-level verification of tool findings is mandatory, and this session's grep verification is what caught it.
4. **I verified a gate with a masked exit code and reported it green when it was red.** The "check-docs EXIT=0" earlier this session was `tail`'s exit status (`nix run .#check-docs 2>&1 \| tail -6; echo $?`), not the gate's. The gate's strike-presence check requires every `docs/status/*.md` to carry `~~` annotations; the 4 pre-existing reports (3× 2026-10-01, 1× 2026-10-04) have none, so the gate was red all along — this session then added two more unannotated reports (the parallel session's 12-28 file and this one). Root cause: piping a gate before capturing `$?`. Impact: a false "gate green" signal in this session's handoff. Caught during report writing when the unpiped run returned 1; the fix is discipline (capture exit before any pipe), recorded in e7. The underlying strike debt is real standing work (f10) or a gate-design question (g3).

## e) WHAT WE SHOULD IMPROVE

| #                                     | Improvement                                               | Impact                                                                                                                | Concrete fix                                                                                                                                                                                                                  |
| ------------------------------------- | --------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 **→ ROADMAP (process improvement)** | Resolve the commit-policy conflict (harness vs AGENTS.md) | High: every future session either flattens history or violates the harness contract                                   | Decide (g1), write the outcome into AGENTS.md gotchas as one line                                                                                                                                                             |
| 2 **→ ROADMAP (process improvement)** | Pin the art-dupl baseline as a gate                       | High: converts clone detection from advisory to regression-catching                                                   | Script asserting "exactly 1 shown group = negotiation default pair", wire into `check-docs` or CI; also record the canonical invocation flags                                                                                 |
| 3 **→ ROADMAP (process improvement)** | Never trust tool counts; verify by grep                   | Medium: art-dupl missed 1 of 3 sites this session                                                                     | Already done this session; note it in AGENTS.md next to the existing "verify row edits with an immediate grep" gotcha (same failure class: reported-success/report-count != disk truth)                                       |
| 4 **→ ROADMAP (process improvement)** | Format gate at commit time                                | High: master should never carry unformatted docs                                                                      | dprint check in the auto-commit daemon path, or a pre-sweep hook; alternatively accept and rely on session-time repair (current de-facto)                                                                                     |
| 5 **→ ROADMAP (process improvement)** | Test the contract, not just the happy path                | Medium: nil-logger branch unverified by tests                                                                         | Table-driven test: each notify entry point with nil logger must not panic and must not send (NOTIFY_SOCKET unset)                                                                                                             |
| 6 **→ ROADMAP (process improvement)** | Watch for parallel sessions                               | Low: an untracked report from another agent (12:28, branching-flow-panic-finding-review) appeared during this session | Before bulk git operations, `git status` for files I did not author — did this, but only at report time; do it at session start                                                                                               |
| 7 **→ ROADMAP (process improvement)** | Capture gate exit codes before any pipe                   | High: a piped gate reports the pipe's status, not the gate's — this session shipped a false "check-docs green" claim  | Standing rule: `nix run .#check-docs; echo $?` with no pipe, or `set -o pipefail` — same failure class as AGENTS.md's "stale LSP diagnostics: trust the CLI gates" gotcha, but inverted: the CLI _read_ was the lie this time |

## f) TOP THINGS TO GET DONE NEXT (ranked by impact; capped at what this session observed — a full 50-item list requires the docs-health sweep that was ruled out of scope)

> **§f disposition (2026-11-27):** this brainstorm table is a ledger of candidate work, not a commitment list. Done items are struck in place; the surviving work lives in `TODO_LIST.md`, `ROADMAP.md`, or the named sibling repo. Rows stay unstruck by design (absence of a marker is the open signal).

<!-- check-rows:ledger -->

| #  | Task                                                                                                                                                                                                                                                                                                   | Impact | Effort | Category      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------ | ------ | ------------- |
| 1  | Add table-driven test: nil logger into NotifyReady/NotifyStopping/Watchdog/WatchdogInterval and loggerOrDefault must not panic, must not send when NOTIFY_SOCKET unset                                                                                                                                 | High   | S      | Quality       |
| 2  | Check project-discovery-daemon for nil-logger call sites; if found, add "consumers rely on this" to the doc comments                                                                                                                                                                                   | High   | S      | Documentation |
| 3  | Build the art-dupl baseline gate script (assert exactly the known finding; fail on new) and wire into check-docs or CI                                                                                                                                                                                 | High   | M      | Quality       |
| 4  | Decide commit policy (g1) and record it in AGENTS.md                                                                                                                                                                                                                                                   | High   | S      | Process       |
| 5  | Harvest section-f items from prior docs/status reports into TODO_LIST/ROADMAP (verify against code first, per AGENTS.md)                                                                                                                                                                               | High   | M      | Documentation |
| 6  | Add commit-time dprint check (hook, daemon pre-step, or CI)                                                                                                                                                                                                                                            | High   | M      | Process       |
| 7  | Check whether commits 7be4443/1072d6a are pushed; if local-only, squash into one descriptive commit                                                                                                                                                                                                    | Medium | S      | Cleanup       |
| 8  | Review the 82 non-actionable + 12 filtered-suppressed art-dupl groups once to confirm nothing real hides there                                                                                                                                                                                         | Medium | M      | Quality       |
| 9  | Record the "tool under-reports, verify by grep" lesson in AGENTS.md next to the existing grep-verification gotcha                                                                                                                                                                                      | Medium | S      | Documentation |
| 10 | Annotate (or harvest-then-annotate) the 6 unannotated docs/status reports — this is the ONLY thing keeping check-docs red: 3× 2026-10-01, 1× 2026-10-04, the parallel session's 12-28 report, and this report. Verify section-f items against code first per AGENTS.md; the 12-28 file needs its owner | High   | L      | Documentation |
| 11 | TODO_LIST row 20: remove `/tmp/codecprobe` scratch module                                                                                                                                                                                                                                              | Low    | S      | Cleanup       |
| 12 | TODO_LIST row 19: revisit Negotiator bound-resolver only if per-call options prove noisy in a real consumer (YAGNI watch — keep parked)                                                                                                                                                                | Low    | —      | Watch         |
| 13 | Record canonical art-dupl invocation (exact flags) in AGENTS.md essential commands so future runs are comparable                                                                                                                                                                                       | Low    | S      | Documentation |
| 14 | Decide whether this refactor warrants a CHANGELOG entry at next release (internal-only change; likely no)                                                                                                                                                                                              | Low    | S      | Documentation |
| 15 | Watch: if a second file ever needs the nil-logger contract, promote loggerOrDefault (export or move) — YAGNI now                                                                                                                                                                                       | Low    | S      | Watch         |
| 16 | Run nix build + nix flake check at next dependency change or release cut (standing gate, skipped this session with cause)                                                                                                                                                                              | Low    | M      | Quality       |
| 17 | Confirm the parallel session's report (12-28 branching-flow-panic-finding-review) gets committed/owned — it sat untracked at session end and is not mine to commit                                                                                                                                     | Medium | S      | Process       |
| 18 | After any future dprint pass over annotated reports, re-run check-rows (standing gotcha; no annotated reports were touched this session)                                                                                                                                                               | Low    | S      | Process       |

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Commit policy:** AGENTS.md tells me to descriptively commit meaningful changes myself and squash daemon splits; the harness forbids commits without an explicit "commit". This session's work is now buried in heuristic messages. Which rule wins — and if the harness rule wins, do you accept heuristic-flat history as the cost?
2. **Production exposure of the nil path:** Do project-discovery-daemon (or other consumers) ever pass `nil` loggers into NotifyReady/NotifyStopping/Watchdog/WatchdogInterval? This decides whether f1 is a contract test (must-have) or defensive-code hygiene (nice-to-have). I can grep the consumer repo next session, but you likely know offhand.
3. **Strike-gate intent:** check-docs fails while ANY docs/status report lacks `~~` annotations — including freshly written ones whose section-f rows are all legitimately open (like this one). Is the intended workflow "every report goes red-until-a-later-pass annotates it", or should the strike check exempt reports younger than their first docs pass? This decides whether the gate can ever be green between sessions or only right after an annotate sweep.

## Verification Log

```text
go version go1.27.0 linux/amd64
go build ./...                     -> ok
go vet ./...                       -> ok
go test ./... -race -count=1       -> ok (1.290s, then 1.542s re-run)
golangci-lint fmt + run ./...      -> 0 issues
art-dupl --sort total-tokens -t 1  -> 1 shown (negotiation.go default pair only)
grep "logger == nil" notify.go     -> 0 in functions, 1 in loggerOrDefault helper
grep -c loggerOrDefault AGENTS.md  -> 1 (edit verified on disk)
dprint fmt                         -> 2 files fixed (pre-existing 2026-10-04 drift)
nix run .#check-docs               -> dprint / check-rows / FEATURES sub-checks pass;
                                      gate exit 1 (UNPIPED, verified): strike-presence
                                      fails on 6 unannotated docs/status reports —
                                      4 pre-date this session, 2 are today's
                                      CORRECTION: an earlier "EXIT=0" in this session
                                      was `tail`'s exit code after a pipe, not the
                                      gate's — see d4 and e7
NOT RUN: nix build, nix flake check (no dependency changes; vendorHash cannot drift)
```

## Session Commit Map (daemon-swept)

- `7be4443` — notify.go (loggerOrDefault extraction)
- `1072d6a` — AGENTS.md (art-dupl bullet) + 2 docs formatting repairs
- Untracked, NOT mine: `docs/status/2026-10-05_12-28_branching-flow-panic-finding-review.md` (parallel session; left untouched)
