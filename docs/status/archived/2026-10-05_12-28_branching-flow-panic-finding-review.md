# Session Status Report: Branching-Flow Panic Finding Review

> **Live remainder:** §a done; branching-flow adoption and the nil-codec hardening are routed to ROADMAP theme 2 / TODO_LIST. Live work: TODO_LIST.md.

- **Date:** 2026-10-05 12:28 CEST
- **Repo:** `go-daemon` (branch `master`, working tree clean at report time)
- **Session scope:** Triage of one finding from the user-run `branching-flow panic` analyzer (experimental): type assertion without ok-check at `write.go:30`.
- **Scope discipline:** Per operator instruction, no fresh repo-wide research was performed; this report covers only this session's run plus observations already in context (files read this session: `write.go`, `.golangci.yml` via grep; carried context: `AGENTS.md`).

## What Happened This Session

1. Operator attached `branching-flow panic` output: 1 detection, medium severity, `write.go:30`, "Type assertion without ok check may panic".
2. Read `write.go` in full; grepped `bufPool` (5 matches, all in `write.go` or FEATURES.md prose) and `.golangci.yml` (no `branching-flow` linter; `forcetypeassert` present at config lines 36 and 232).
3. Verdict delivered: **false positive, dismissed, no code change.**
4. No edits were made; nothing was committed by this session; the auto-commit daemon had nothing new to sweep from this session's work.

## Self-Critique (asked first, answered honestly)

**What did I forget?**

1. **No repo-wide validation of the analyzer's claim.** The tool said "Detections: 1" and I triaged exactly that one row. A single `rg` sweep for bare type assertions would have either confirmed "1 of 1, all clean" or exposed upcoming noise. I never verified the tool's completeness claim against the codebase.
2. **No durable memory write.** Per the aggressive update protocol, the verdict (sync.Pool bare assertion = infallible idiom here, accepted) should have been recorded in `AGENTS.md` alongside the existing negotiation `default:` pair note. It currently lives only in chat and this timestamped report.
3. **No gate run for evidence.** I argued from code reading alone. One synchronous `golangci-lint run` (or even `go build`) would have made the "repo gate is green and `forcetypeassert` suppression works" claim evidence-grade instead of inferred.

**What could I have done better?**

1. **Tool-level verdict, not just finding-level.** I assessed the finding but stopped short of a staff-level adopt/reject recommendation for branching-flow itself. The honest recommendation is: reject as a standing gate for this repo (its own banner admits high false-positive rate; `forcetypeassert` already covers this class; golangci-style `nolint` directives appear unsupported, given it suggests its own directive syntax).
2. **State the auto-fix tradeoff explicitly.** I deferred adding `//nolint:branching-flow:panic` with "if you adopt it". Defensible (suppressing findings for a non-gate tool is speculative noise), but the tradeoff deserved one explicit sentence rather than a conditional.
3. **Verify directive semantics.** I did not check whether branching-flow honors golangci-style `//nolint:forcetypeassert` (it evidently does not treat the existing one as sufficient). One command or one doc lookup would have pinned its suppression model.

**What could I still improve (systemic)?**

1. An **analyzer-triage protocol**: when an experimental analyzer is pointed at a repo, sweep all findings in one pass, verify the tool's detection count, record verdicts durably, and end with an explicit adopt/reject.
2. An **accepted-idioms ledger** in `AGENTS.md` (like the negotiation `default:` pair entry) so identical future findings are dismissed by reference, not re-litigated.
3. **Cross-repo comparison** before adopting any experimental linter: run it on `project-discovery-daemon` too, since lint config is mirrored between the two.

---

## a) FULLY DONE

| Item                                                                                                                           | Evidence                                                                                                                                                                                                                                                                                           |
| ------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~Triaged the branching-flow finding at `write.go:30` as false positive~~ done (docs-health pass 2026-11-27)                   | ~~`write.go:17` (`New` returns `new(bytes.Buffer)`), `write.go:16` (unexported pool), `write.go:30`+`write.go:33` are the only `Get`/`Put` sites in the codebase; the dynamic type is provably always `*bytes.Buffer`. Existing `//nolint:forcetypeassert` justification covers the same ground.~~ |
| ~~Confirmed `branching-flow` is not part of the repo's lint gate~~ done (docs-health pass 2026-11-27)                          | ~~`.golangci.yml` contains no `branching-flow` entry; `forcetypeassert` is enabled (config lines 36, 232). No CI impact from the finding.~~                                                                                                                                                        |
| ~~Left the repo in a clean, gate-safe state~~ done (docs-health pass 2026-11-27)                                               | ~~Zero edits, zero commits; `git status` clean; no broken-intermediate risk shipped to the auto-commit daemon.~~                                                                                                                                                                                   |
| ~~Identified a follow-up nil-codec hazard while reading `write.go` (unreported until now)~~ done (docs-health pass 2026-11-27) | ~~`WriteEncoded` is exported; `WriteEncoded(w, 200, nil, v)` panics at `write.go:41` (`contentCodec.Encode` on a nil interface). Recorded in section (f) below.~~                                                                                                                                  |

## b) PARTIALLY DONE

| Item                                                                                                          | What works                                                            | What remains                                                                                                                       | Effort                          |
| ------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | ------------------------------- |
| Branching-flow finding review **→ residual carried in §f below; live work tracked in TODO_LIST/ROADMAP**      | The single reported detection is fully triaged (dismissed with proof) | Adopt/reject decision for the tool itself; suppression policy if adopted; durable record of the verdict                            | S (blocked on section g, Q1/Q2) |
| Type-assertion hygiene of the repo **→ residual carried in §f below; live work tracked in TODO_LIST/ROADMAP** | The one flagged site is proven safe                                   | Unknown whether `write.go:30` is the only bare assertion; no sweep performed (scope-discipline choice, not oversight of execution) | S/M                             |

## c) NOT STARTED

| Item                                                                                                                                                              | Why not started                                                                                                        | Still wanted?                                 |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | --------------------------------------------- |
| Repo-wide bare type-assertion sweep (pre-triage future analyzer noise) **→ not started; live work tracked in TODO_LIST/ROADMAP**                                  | Session scope was one finding; operator forbade unrelated research                                                     | Yes, S effort, high noise-reduction value     |
| `AGENTS.md` entry: sync.Pool bare-assertion idiom + branching-flow verdict **→ not started; live work tracked in TODO_LIST/ROADMAP**                              | Deferred pending the tool adoption decision (would otherwise record a verdict for a tool that may never return)        | Yes, S                                        |
| Nil-codec hardening of `WriteEncoded`/`ContentTypeFor` **→ not started; live work tracked in TODO_LIST/ROADMAP**                                                  | Noticed at report time; no code changed this session by design                                                         | Yes, S                                        |
| docs-health HARVEST of the five existing `docs/status/` reports' section-f items into TODO_LIST/ROADMAP **→ not started; live work tracked in TODO_LIST/ROADMAP** | Standing demand in `AGENTS.md` (carried context, not re-verified this session); operator ordered wait-for-instructions | Yes, per AGENTS.md it is overdue              |
| Adopt/reject write-up for branching-flow as a gate (this repo and/or project-discovery-daemon) **→ not started; live work tracked in TODO_LIST/ROADMAP**          | Blocked on operator intent (section g, Q1)                                                                             | Only if the operator intends to keep the tool |

## d) TOTALLY FUCKED UP

**Nothing is fucked up from this session.** Stated with evidence, not comfort:

- No files were edited, so no broken intermediate could reach the auto-commit daemon (the repo's known failure mode; did not occur).
- The working tree was clean before and after; the only new commits (`1072d6a`, `7be4443`) predate this session's work.
- The analyzer finding itself is the only defect claim on the table, and it is a false positive (proof in section a) — a dismissed finding is not a fucked-up thing.

One standing, pre-existing risk is worth naming so silence is not mistaken for ignorance: the auto-commit daemon commits regardless of compile state (documented in `AGENTS.md`; prior incidents `caf1f0c`/`378607f`). It did not bite this session because no edits were made.

## e) WHAT WE SHOULD IMPROVE

1. **Analyzer triage protocol.** Pattern: tool runs, one finding gets full treatment, tool's completeness claim goes unverified, verdict evaporates with the chat. Fix: one-pass sweep + durable verdict + explicit adopt/reject, every time an experimental analyzer touches the repo. Impact: prevents recurring re-litigation and comment churn.
2. **Accepted-idioms ledger.** `AGENTS.md` already documents one accepted idiom (art-dupl `default:` pair in negotiation.go). The sync.Pool bare assertion belongs there too. Impact: future sessions (and future analyzer runs) dismiss by reference in seconds.
3. **Evidence-grade reviews.** Code-reading proofs are good; a synchronous gate run makes them verifiable. One command, near-zero cost. Impact: review conclusions become checkable claims.
4. **Tool adoption discipline.** Experimental analyzers with self-declared high false-positive rates should get an explicit adopt/reject before their suggested `nolint` syntax starts accumulating in source. Impact: avoids one-off suppression comments that outlive the tool.
5. **Nil-guard policy for exported wrappers.** `WriteEncoded`/`WriteJSON`/`WriteErrorf` are the package's public surface; a nil codec panics deep in the hot path instead of failing fast with a clear message. Impact: turns a latent panic into either a compile-time-visible contract or a one-line guard with a test.

## f) Things To Get Done Next

Ranked by impact. Items 1-11 are grounded in this session; items 12+ are carried from in-context `AGENTS.md` standing demands (flagged, not re-researched per scope instruction). The operator asked for "up to 50"; the honest count grounded in this session's evidence is 20. Per the status-report skill, extras beyond ~10 are brainstorm/ROADMAP fuel for HARVEST routing, and inventing 30 filler rows from un-researched backlog would violate both the no-research instruction and the report-quality guide.

> **§f disposition (2026-11-27):** this brainstorm table is a ledger of candidate work, not a commitment list. Done items are struck in place; the surviving work lives in `TODO_LIST.md`, `ROADMAP.md`, or the named sibling repo. Rows stay unstruck by design (absence of a marker is the open signal).

<!-- check-rows:ledger -->

| #  | Task                                                                                                                                                    | Impact   | Effort | Category      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | Decide branching-flow adopt/reject as a standing gate (this repo, and whether pdd mirrors it)                                                           | Critical | S      | Quality       |
| 2  | Repo-wide sweep of bare type assertions; triage every hit in one pass, record verdicts                                                                  | High     | S/M    | Quality       |
| 3  | Record the sync.Pool bare-assertion idiom + branching-flow verdict in `AGENTS.md`                                                                       | High     | S      | Documentation |
| 4  | Harden `WriteEncoded` against a nil `contentCodec` (guard or documented contract) + test                                                                | High     | S      | Bug (latent)  |
| 5  | Check `ContentTypeFor` behavior on nil/unknown codec; align with item 4's policy                                                                        | Medium   | S      | Bug (latent)  |
| 6  | If adopted: define branching-flow suppression policy (per-line justified `nolint` vs config exclusion for the pool idiom)                               | Medium   | S      | Quality       |
| 7  | Verify whether branching-flow honors golangci-style `nolint` directives (determines comment churn cost)                                                 | Medium   | S      | Quality       |
| 8  | Run branching-flow on `project-discovery-daemon` for a comparative verdict before any adoption                                                          | Medium   | M      | Quality       |
| 9  | One synchronous baseline gate run (`go build`, `go test -race`, `golangci-lint run`) to pin the repo green at review time                               | Medium   | S      | Quality       |
| 10 | Verify `TestWriteEncoded_JSONAndCBOR` actually exercises pool reuse across both codec paths; add a pool-lifecycle assertion if thin                     | Medium   | S      | Quality       |
| 11 | Add a benchmark for the pooled-buffer write path so future "remove the pool" proposals are decidable                                                    | Low      | M      | Quality       |
| 12 | (Carried) docs-health HARVEST: pull section-f items from the five existing `docs/status/` reports into TODO_LIST/ROADMAP, verifying against code first  | High     | M      | Documentation |
| 13 | (Carried) Harvest this report's section-f items in the same pass                                                                                        | Medium   | S      | Documentation |
| 14 | (Carried) Post-flip runbook: remove GOPRIVATE refs so `scripts/pre-publish-audit.sh` exits 0                                                            | Medium   | M      | Cleanup       |
| 15 | (Carried) Re-verify `nix run .#check-readme` and `nix run .#check-docs` at the next docs pass                                                           | Medium   | S      | Quality       |
| 16 | (Carried) Keep the erraudit known-dismissed list current at the next toolchain bump (sentinels/wraps stay per cross-repo contract)                      | Low      | L      | Quality       |
| 17 | (Carried) If go-codec is ever bumped: re-probe its build-tag flavor before assuming json/v2 availability                                                | Low      | S      | Quality       |
| 18 | (Carried) Next Dependabot dep PR: merge then `buildflow -s nix-hash-fix --fix` per standing policy (no action pending now)                              | Low      | S      | Cleanup       |
| 19 | (Carried) If an experimental-analyzer protocol is adopted (item e1), consider promoting it to `references/lessons.md` in crush-config if it generalizes | Low      | M      | Process       |
| 20 | (Carried) Annotate (do not rewrite) any of the five old status reports that later work has overtaken, using docs-health ANNOTATE cell-level strikes     | Low      | M      | Documentation |

## g) Questions I Cannot Answer Myself

1. **Is `branching-flow` intended to become a standing gate in this repo (and mirrored into project-discovery-daemon's config like `forcetypeassert` was), or was today an ad-hoc evaluation?** I checked `.golangci.yml` (absent) and the tool's own banner (self-declared experimental, high false-positive rate), but intent is not in the repo. This gates items f1, f3, f6, f7.
2. **If it becomes a gate: do you prefer per-line suppressions with justifications (matching the existing `//nolint:forcetypeassert` style) or a tool-level exclusion for the sync.Pool idiom?** Both are defensible; the repo currently leans per-line-with-justification, but this is a policy call I should not make unilaterally.
3. **What prompted today's run — a broader static-analysis evaluation across your repos (in which case the triage protocol in e1 is worth building once and reusing), or a one-off check?** This decides whether the protocol becomes a reusable checklist/skill or stays a paragraph in `AGENTS.md`.

## Handoff

Per the status-report skill: section (f) is the primary input for a `docs-health` HARVEST pass into `TODO_LIST.md`/`ROADMAP.md`. Not run now, per the operator's explicit "then wait for instructions". The five existing reports (2026-10-01 through 2026-10-04) still owe their harvest, per standing `AGENTS.md` policy.
