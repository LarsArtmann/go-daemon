# Session Status — SSE Test Deduplication (art-dupl Review)

> **Live remainder:** §a done; §b/c clone-hygiene and gate-chain items are recorded in AGENTS.md and ROADMAP theme 3. Live work: TODO_LIST.md.

- **Date:** 2026-10-01 05:46 CEST
- **Session window:** ~05:30–05:55 CEST, single session
- **Scope:** per owner instruction, this report covers THIS session's run only
  (the `review, deduplicate` request on pasted `art-dupl -t 1` output). No
  project-wide research was done; section-f items tagged "context" come from
  AGENTS.md memory, not fresh verification.
- **Format note:** owner explicitly requested `.md`; the status-report skill's
  canonical format is styled HTML. Override honored, flagged here per skill.

## TL;DR

Collapses 9 copy-paste-shaped `ParseSSEData` tests into one table-driven test;
accepted the idiomatic `default:` clone pair in negotiation.go. All gates green
AFTER a late catch: the first version shipped to master lint-failing (missed
`golangci-lint` in the gate chain), fixed during this report's prep.

## Self-Review (brutal, per skill questions)

**What did I forget?**

- `golangci-lint run` — a documented repo gate. I ran gofmt + build + full
  `-race` suite and called it verified. The linter's `paralleltest` rule fired
  on my new test (parent missing `t.Parallel()`), and the auto-commit daemon
  had already shipped the lint-failing state as `c19f1ec`. This is precisely
  the AGENTS.md gotcha ("a verification that is deferred ... ships a broken
  commit") — I satisfied its letter (build was synchronous) but not its spirit
  (the full gate chain). Fixed at 05:50: parent `t.Parallel()` added, lint
  0 issues, focused + full suite green, `go vet` green.
- The unshown clone groups. My re-run reported "69 groups, 1 shown (61
  non-actionable, 8 filtered suppressed)". I accepted the tool's filter
  verdict without inspecting a single one of the 69 unshown groups. See (d).
- I dropped the inline comment `// No trailing blank line: the accumulated
  data is NOT dispatched.` when `StreamEndsMidEvent` became a table row. The
  row name states WHAT the contract is; the WHY (dispatch fires only on the
  blank line, per SSE spec) is now implicit. Mitigated by the sseparse
  conformance corpus covering the same behavior, but the loss is real.

**What is something that's stupid that we do anyway?**

- I replicated the user's exact flag combo (`--type-aware
  --suggest-generics`) after the tool itself warned the combination is
  redundant (types erased from hash). Propagating a warned-against flag
  instead of cleaning it is small but exactly the compliance-over-thinking
  habit the global AGENTS.md bans.

**What could you have done better?**

- Gate chain as one synchronous command (`gofmt -l && go build &&
  golangci-lint run && go test ./... -race`) from the first edit — the lint
  failure would never have existed, let alone shipped.
- Verify the "zero harmful duplication" claim by enumerating what the filter
  hid, not just what it showed.
- Committed descriptively (repo AGENTS.md asks for it when a change matters);
  the refactor landed as a heuristic `chore: auto-commit` message. Harness
  forbids commits without explicit user request — conflict flagged as
  question 1 below instead of resolved unilaterally.

**What could you still improve?**

- Same as section (e); the highest-leverage items are the gate-chain fix and
  the clone-baseline codification.

**Did you lie to you?**

- Half-lie by scope creep: my closing line "Zero harmful duplication remains"
  was true only for the two SHOWN groups. 69 unshown groups at `-t 1` were
  never inspected. Corrected claim: "zero harmful duplication among shown
  groups; 61 non-actionable + 8 filtered groups unverified."

**Ghost systems?** None created. None found in session scope (test-only
refactor; no new code paths, no unwired surfaces).

**Split brains?** None created: old test names fully deleted (grep-verified
zero stale references repo-wide); AGENTS.md now points new dispatch-shape
cases at the single table. Pre-existing (acceptable, noted) overlap:
`TestParseSSEData_CorpusConformance` (sseparse oracle vectors) and the new
dispatch table intentionally cover some of the same wire→payload behavior
from different oracles.

**Did we remove something actually useful?** The per-test failure messages
survived verbatim as row names. The lost inline spec comment is the one real
loss (above). Nothing else.

**Tests?** Assertion strength preserved or improved 1:1 (`slices.Equal`
against full expected payload slice ≡ the old exact-match conditionals;
the 0-event and joined-multiline cases map exactly). Coverage spot-check not
run — item 19 in section f.

## a) FULLY DONE

1. ~~**art-dupl output reviewed with judgment** — both shown clone groups~~
   ~~classified extract/accept per the deduplicate-code skill.~~ done (session §a; re-verified 2026-11-27)
2. ~~**9 same-shape tests → 1 table** — `TestParseSSEData_Dispatch`~~
   ~~(sse_test.go:9): the 7 flagged groups + `MultipleEvents` +~~
   ~~`StreamEndsMidEvent` (identical prologue/call/assert, differing only in~~
   ~~wire input and expected payloads). Keyed rows, `name` field first per repo~~
   ~~convention, spec messages preserved as row names. Net −130 lines.~~ done (session §a; re-verified 2026-11-27)
3. ~~**Error/size-shaped cases kept separate** — `MaxEventBytesExceeded`,~~
   ~~`LargeEvent`, `ScannerErrorWrapped` (different assertion shapes, not~~
   ~~clones). Fuzz target and corpus-conformance test untouched.~~ done (session §a; re-verified 2026-11-27)
4. ~~**negotiation.go `default:` pair accepted** — idiomatic switch structure,~~
   ~~not extractable; fallback rationale already in `ResolveResponseCodec`'s~~
   ~~doc comment. No code change.~~ done (session §a; re-verified 2026-11-27)
5. ~~**All gates green (final state)** — gofmt clean; `go build`; `go vet`;~~
   ~~`golangci-lint run` 0 issues; `go test ./... -race -count=1` ok.~~ done (session §a; re-verified 2026-11-27)
6. ~~**Re-scan clean** — art-dupl re-run: sse_test.go groups gone; only the~~
   ~~accepted `default:` group remains shown.~~ done (session §a; re-verified 2026-11-27)
7. ~~**Stale-reference sweep** — zero repo references to the 9 removed test~~
   ~~names.~~ done (session §a; re-verified 2026-11-27)
8. ~~**Memory updated** — AGENTS.md Conventions: new dispatch-shape cases go~~
   ~~into the table; accepted-clone note for the `default:` pair.~~ done (session §a; re-verified 2026-11-27)

## b) PARTIALLY DONE

1. **Gate verification** — was partial (lint skipped, lint-failing commit
   `c19f1ec` on master); closed during this report's prep. The bad
   intermediate commit itself remains in history (see d).
2. **"Zero harmful duplication" verification** — shown groups verified;
   61 non-actionable + 8 filtered-suppressed groups at `-t 1` NOT inspected
   (section f item 1).
3. **Memory maintenance** — project AGENTS.md updated; no cross-project
   lesson recorded (this one is arguably session-specific; revisit if the
   "lint after build" miss recurs in another repo).

## c) NOT STARTED (explicitly, this session)

1. Inspecting the 61 non-actionable + 8 filtered clone groups.
2. Consumer-side sweep (project-discovery-daemon) for the same test shape.
3. docs-health HARVEST of this report's section f into TODO_LIST/ROADMAP.
4. Codifying an accepted-clone baseline for art-dupl in this repo.
5. Descriptive commit of the dedup refactor (pending owner decision, g1).

## d) TOTALLY FUCKED UP

1. **Lint-failing commit shipped to master** (`c19f1ec`, "chore: auto-commit
   1 changed file(s)") — my miss: build+tests ran, the repo's own lint gate
   did not, and the ~1-minute auto-daemon window did the rest. Severity:
   recovered (fix on disk, all gates green within the session), no data
   loss, but master history permanently carries a commit that fails
   `golangci-lint run`. Root cause and permanent fix in (e) items 1–2.
2. **Overstated completion claim** — "zero harmful duplication remains" was
   asserted beyond the verified scope (shown groups only). Honesty failure
   caught in self-review, corrected above; unverified surface is section f
   item 1.

Nothing else in scope is broken: API untouched (test-only diff), consumer
pinned by rev unaffected, vendorHash unaffected (no dependency change).

## e) WHAT WE SHOULD IMPROVE

1. **Gate chain must be atomic and complete**: run
   `gofmt -l . && go build ./... && golangci-lint run ./... && go test ./... -race -count=1`
   as ONE synchronous command after every edit batch. Build-green ≠
   lint-green (paralleltest proved it today).
2. **Tighten the AGENTS.md auto-commit gotcha**: "run a gate (build or gopls
   diagnostics)" → "run build AND lint". The current wording is what I
   satisfied while still shipping a lint failure.
3. **Claims must match verified scope** — enumerate what a tool filtered
   before declaring a clean bill.
4. **Drop warned-against flag combos** instead of replicating them.
5. **Commit policy needs a standing answer** (question g1) so meaningful
   refactors stop landing under heuristic messages.
6. **Clone hygiene needs a baseline artifact** — accepted clones documented
   once (like erraudit's known-dismissed list) so scans diff against intent,
   not memory.

## f) Up to 50 Things To Get Done Next (brainstorm, not commitments)

Owner asked for up to 50; per skill, most of these are ROADMAP fuel and
docs-health HARVEST must apply routing rigor. Impact/effort are session-level
guesses. Items 23–30 are known project context from AGENTS.md, not re-verified
today.

**Clone/dedup follow-ups**

1. Inspect the 61 non-actionable + 8 filtered-suppressed art-dupl groups at
   `-t 1` (unfiltered/HTML run) — closes the verification gap behind today's
   overstated claim. Impact high, effort low.
2. Codify an accepted-clone baseline (rationale per accepted group, in the
   spirit of the erraudit known-dismissed list). Med/low.
3. Re-baseline art-dupl at default `-t 5` and file the report under
   docs/status for trend tracking. Low/low.
4. Sweep project-discovery-daemon for the same same-shape-test pattern (its
   content_negotiation/sse tests likely share the lineage). Med/med.
5. Sweep remaining go-daemon test files at `-t 3` for near-miss same-shape
   prologues (flightrecorder_test, socket lifecycle tests). Med/med.
6. Add art-dupl to a gate script or buildflow step so same-shape test
   regrowth is caught at PR time, diffing against the baseline (item 2).
   Med/med.
7. Document the intentional overlap between `TestParseSSEData_CorpusConformance`
   (sseparse oracle) and the dispatch table (hand-picked spec cases) —
   one-line cross-reference, pre-empting a future "duplicate tests?" flag.
   Low/low.
8. Canonicalize the art-dupl flag combo (drop `--type-aware` when
   `--suggest-generics` is set) in any wrapper/alias so the tool's warning
   disappears. Low/low.

**Gate hygiene (today's misses → permanent fixes)**

9. Adopt the atomic four-step gate chain from (e)1 as standing session
   procedure here. High/low.
10. Edit the AGENTS.md auto-commit gotcha to "build AND lint" per (e)2.
    Med/low.
11. Confirm whether `.golangci.yml` here enables govet; if not, keep `go vet`
    in the chain (it ran green today, standalone). Low/low.
12. Verify the daemon committed today's post-lint-fix state (sse_test.go fix
    - AGENTS.md conventions + this report) with sane diffs. Low/low.
13. Resolve the descriptive-commit policy (question g1) and record the
    answer in AGENTS.md. Med/low.

**Docs & memory**

14. docs-health HARVEST: pull this report's section f plus older
    docs/status reports' section-f items into TODO_LIST.md/ROADMAP.md —
    verify against code first (AGENTS.md warns many are already done).
    High/med.
15. Run `nix run .#check-docs` after today's markdown edits (AGENTS.md
    conventions + this report). Low/low.
16. Record the accepted-clone decision (negotiation.go `default:` pair) in
    TODO_LIST/DECISIONS so it survives AGENTS.md pruning. Low/low.
17. Confirm FEATURES.md doesn't enumerate removed test names (repo grep is
    already clean; only if FEATURES lists test scenarios). Low/low.
18. Annotate older docs/status reports per docs-health ANNOTATE mode —
    several predate today and are likely stale. Med/med.

**Test suite depth**

19. `go test -cover` baseline after consolidation; confirm dispatch-path
    coverage unchanged (assertions were preserved 1:1; verify, don't assume).
    Low/low.
20. Bounded fuzz runs (`-fuzz=FuzzParseSSEData -fuzztime=60s`, same for
    `FuzzResolveResponseCodec`) as periodic hygiene; commit any crash input
    per the seed-only corpus policy. Med/low.
    ~~21. Table-drive any additional same-shape groups the `-t 3` sweep (item 5)~~ done — no additional same-shape groups at -t 1 (re-run 2026-11-27)
    finds. Med/med.
21. If any external CI/script `-run` filter targeted the old test names,
    update it — repo-internal grep is clean, external references unknown.
    Low/low.
22. Restore the lost spec-WHY (dispatch fires only on blank line) either in
    the row name or `ParseSSEData`'s doc comment — owner's call on comment
    policy. Low/low.

**Known project context (from AGENTS.md, not re-verified this session)**

24. Post-flip runbook: `scripts/pre-publish-audit.sh` still exits 1 until
    GOPRIVATE refs are removed — finish the runbook steps.
25. Next Dependabot PR: merge + immediate `buildflow -s nix-hash-fix --fix`
    per the close-and-adopt policy; never wait on the bot for nix.
26. dprint pin hygiene: if `dprint config update` is ever run, restore the
    `plugins.dprint.dev` URL scheme and confirm zero churn via `nix fmt`.
27. Keep sseparse test-only (no non-test imports) — it was adopted that way
    deliberately to avoid GOPRIVATE/vendor cost.
28. `nix build` remains the only gate that catches vendorHash drift — none
    triggered today (no dependency change), keep the rule for next bump.
29. Any new flight-recorder tests must stay inside `flightRecorderMu` —
    untouched today, standing constraint.
30. `MaybeCaptureOnSlowRequest` context-detachment contract and its
    regression test: standing inventory note, no action.

**Tooling ideas**

31. If art-dupl is ours: add a flag to surface filtered-suppressed groups
    for review runs (today's blind spot is a tool UX gap too). Low/med.
32. Small `scripts/` wrapper (or buildflow step) that runs art-dupl against
    the accepted baseline and fails only on NEW harmful clones. Med/med.
33. Store future scan reports as `--html` under docs/status for trend
    comparison. Low/low.

**Consumer/ecosystem**

34. Next pdd bump of go-daemon: today's change is test-only — no API change,
    vendorHash unaffected; just confirm CI green. Low/low.
35. Check whether pdd carries the old copy-pasted test shape (same as item 4,
    consumer side). Med/med.
36. go-codec pin policy: untouched today; any bump follows the primary
    consumer's versions first. Low/low.

**Nice-to-have / brainstorm**

37. Tag dispatch-table rows with the spec section they pin (WHATWG
    eventstream processing) — in names or docs, owner's preference. Low/low.
38. `ExampleParseSSEData` for pkg.go.dev, seeded from the table rows.
    Low/med.
39. Extract a `collectEvents` test helper only if a 4th inline closure usage
    appears (rule of three — currently inline is clearer). Low/low.
40. Re-run art-dupl after the daemon commits today's remaining edits to
    confirm the final tree is clone-clean (expected yes). Low/low.

## g) Questions I Cannot Figure Out Myself

1. **Commit policy for meaningful refactors:** should I make descriptive
   commits immediately when a change matters (overriding the auto-daemon's
   `chore: auto-commit` heuristic messages), or does history stay
   daemon-owned? Today the refactor + a lint-failing intermediate both landed
   under heuristic messages; the harness forbids me committing without your
   explicit say-so.
2. **Was `-t 1 --type-aware --suggest-generics` a one-off probe, or should
   `-t 1` become a recurring gate here?** It decides whether I chase the 61
   non-actionable + 8 filtered groups (f1) and codify a baseline (f2), or
   treat `-t 1` as noise-floor exploration only.
3. **Scope of the dedup sweep:** should the same-shape-test cleanup extend to
   project-discovery-daemon (primary consumer, pins this repo by rev), or is
   this cleanup go-daemon-only for now?
