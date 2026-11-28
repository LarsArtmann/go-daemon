# Status Report — 2026-09-30 00:43 — docs-health AUDIT: negotiation-overhaul verification closeout, critical compile-break fix-forward, and doc-fleet buildout (FEATURES/ROADMAP built, TODO_LIST rebuilt, README/AGENTS/CHANGELOG updated)

> **Live remainder:** §a fully done; §f residuals are cross-repo queues (ROADMAP theme 3) or owner decisions since answered by the visibility-flip decision.

> **Annotated 2026-09-30 (docs-health, second pass):** struck items are resolved/routed (verdict inline, with evidence); unstruck items remain open — §b1-§b2 are historical process facts, §f14 is a fleet idea, and §f30 waits on the first grouped dependabot PR.

**Session scope:** docs-health AUDIT ("view ALL files, make all six living docs superb")
→ full-file read pass → discovered master's test suite **never compiled** since the
negotiation overhaul → fix-forward + lint fixes + complete gate sweep → BUILD
missing docs (FEATURES, ROADMAP) → HARVEST both status reports into a rebuilt
TODO_LIST → VERIFY every doc claim (incl. two cross-repo checks against
project-discovery-daemon) → README/AGENTS/CHANGELOG updates → inline health
report → this status report.

**Format override (flagged per skill):** status-report's canonical format is a
styled HTML dashboard; the user's instruction explicitly requested `.md` with
sections a–g, so this file is Markdown by explicit user instruction.

**Gates at close (all green, re-run on the final tree):** `go build` / `go vet` /
`go test ./... -race -count=1` (1.3s) / `golangci-lint fmt` + `run` (0 issues) /
`dprint fmt` (idempotent) / `nix build .#default` / `nix run .#test`.
**Health report (inline, first audit — no baseline):** Accuracy 7.75 → 10/10,
Fitness 6.5 → 10/10 after fixes.

**Commit state:** five auto-daemon commits this session (`06b3071` → `14dd7bf`).
Push state as measured at report time: origin/master is at `06b3071` (the
negotiation_test compile fix is pushed); the negotiation.go lint fixes, CHANGELOG,
and ALL doc work (`10619a9` → `14dd7bf`) are **local-only, unpushed** — including
`caf1f0c`, the one commit carrying a broken intermediate README (see §d1: no
broken state ever reached origin). Pushing is not mine to do; no commit was
hand-made this session (harness rule), so the six-heuristic-message narrative
gap stands, tracked in TODO_LIST.

---

## a) FULLY DONE

| #                                         | Work                                                                                                                                                                                                                                                                                                                                                                                           | Evidence                                                                                   |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| ~~1~~ done (docs-health pass 2026-11-27)  | ~~**Full read pass over every file** before any edit: 7 Go sources, 5 test files, flake.nix, .golangci.yml, ci.yml, dprint.json, .editorconfig, dependabot.yml, go.mod, both `docs/status/` reports~~                                                                                                                                                                                          | ~~All viewed in-session; zero blind edits~~                                                |
| ~~2~~ done (docs-health pass 2026-11-27)  | ~~**Critical fix-forward: the test table from the q-value overhaul never compiled** — `TestCodecForContentType`'s struct had 2 fields, 3 rows passed a name (`negotiation_test.go:211-213` at HEAD~). Added the `name` field, named all 8 rows, kept full coverage. The 11:35 report's §f2 "if red: fix forward" is now executed~~                                                             | ~~`negotiation_test.go:199-221`; landed in `06b3071`, **pushed**~~                         |
| ~~3~~ done (docs-health pass 2026-11-27)  | ~~**Lint debt of the unverified overhaul cleared**: 2× modernize (`strings.SplitSeq` in `acceptQuality`/`qualityParam`), 1× nonamedreturns (acceptQuality → unnamed returns with locals) → golangci-lint 0 issues against the strict mirrored config~~                                                                                                                                         | ~~`negotiation.go:89-148`; landed in `10619a9` (unpushed)~~                                |
| ~~4~~ done (docs-health pass 2026-11-27)  | ~~**Full gate sweep, green, re-run on the final tree**: build, vet, race suite, lint fmt+run, dprint idempotency, hermetic `nix build`, packaged `nix run .#test`~~                                                                                                                                                                                                                            | ~~All in-session output; vendorHash untouched (zero dep changes — `nix build` proves it)~~ |
| ~~5~~ done (docs-health pass 2026-11-27)  | ~~**FEATURES.md BUILT** (was missing): 16 rows grouped by domain — 14 FULLY_FUNCTIONAL (every row test-backed), 0 PARTIAL, 0 BROKEN, 0 DISABLED, 1 PLANNED (`Vary: Accept`), 1 ON_HOLD (406-capable variant); dedicated Evidence column; verification basis date-stamped~~                                                                                                                     | ~~`FEATURES.md`; counts derived from the file itself~~                                     |
| ~~6~~ done (docs-health pass 2026-11-27)  | ~~**ROADMAP.md BUILT** (was missing): 3 themes (negotiation as full HTTP contract; more extraction-led building blocks; fleet coherence), open questions (406 semantics, tie-break permanence, corpus cadence), 5 explicit non-goals~~                                                                                                                                                         | ~~`ROADMAP.md`~~                                                                           |
| ~~7~~ done (docs-health pass 2026-11-27)  | ~~**TODO_LIST.md REBUILT** via HARVEST from BOTH status reports' §f (11:35 + 09-01): 10 open rows (1 🔵 decision, 9 🔴 TODO), every item re-verified against code before entering; verifiably-done items (ghost-API, golangci-mirror claim, header-matching audits f27-29, lint-config worries f33-35) resolved and dropped, not harvested~~                                                   | ~~`TODO_LIST.md`; routing rationale in the file header~~                                   |
| ~~8~~ done (docs-health pass 2026-11-27)  | ~~**CHANGELOG completed** (11:35 §f5): `Unreleased → Changed` entry documenting the negotiation contract (exact matching, RFC 6839 suffixes, q-value ranking, tie → CBOR, `q=0` refusal never served, wildcard neutrality, signatures unchanged) + `Fixed` line for the compile break~~                                                                                                        | ~~`CHANGELOG.md` `[Unreleased]`~~                                                          |
| ~~9~~ done (docs-health pass 2026-11-27)  | ~~**README sales page updated** (09-01 §f42 + 11:35 §f7): negotiation row rewritten (exact matching, suffixes, q-values, refusals), corpus-oracle claim added to the `ParseSSEData` row, quickstart extended with a full negotiation round-trip — **the snippet is compile-verified** (extracted into a throwaway package, `go build` green)~~                                                 | ~~`README.md`; compile-check harness deleted after use~~                                   |
| ~~10~~ done (docs-health pass 2026-11-27) | ~~**AGENTS.md hardened** (enduring context, 5.9 KB — within the 5-15 KB budget): auto-daemon commits regardless of compile state → gate synchronously; `nix develop` cold-eval latency; `docs/status/` §f harvest convention; test-table `name`-field convention; negotiation contract pointer; toolchain pre-check line; commit hashes stripped from the vendorHash gotcha (endurance rule)~~ | ~~`AGENTS.md`; temporal-pollution grep clean, zero hashes (verified)~~                     |
| ~~11~~ done (docs-health pass 2026-11-27) | ~~**Ghost-API question RESOLVED** (11:35 §g1): project-discovery-daemon calls BOTH `ResolveRequestCodec` and `ResolveResponseCodec` — response-side negotiation is live consumer-facing API, not scope creep. Recorded in FEATURES, AGENTS, CHANGELOG~~                                                                                                                                        | ~~pdd `content_negotiation.go:15-19` (this session's grep)~~                               |
| ~~12~~ done (docs-health pass 2026-11-27) | ~~**AGENTS.md golangci-mirror claim verified** (09-01 §f43): configs are byte-identical (`diff` exit 0)~~                                                                                                                                                                                                                                                                                      | ~~`diff .golangci.yml` pdd ↔ go-daemon~~                                                   |
| ~~13~~ done (docs-health pass 2026-11-27) | ~~**go-codec pin-parity audit** (11:35 §f37): go-daemon pins v0.2.0, pdd uses v0.3.0 → stale pin vs AGENTS.md's stated policy; repair path documented (dep bump + `buildflow -s nix-hash-fix --fix`) → TODO_LIST High row, deliberately NOT bumped in a docs pass~~                                                                                                                            | ~~`go.mod:7` vs pdd `go.mod:19`~~                                                          |
| ~~14~~ done (docs-health pass 2026-11-27) | ~~**pdd-safety by construction proven** (11:35 §f41): `git diff v0.1.1..HEAD -- negotiation.go` shows only unexported helpers changed; all exported signatures identical since v0.1.1~~                                                                                                                                                                                                        | ~~git diff evidence in-session~~                                                           |
| ~~15~~ done (docs-health pass 2026-11-27) | ~~**Inline health report delivered** per docs-health format (two independent scores, visible math, per-doc findings, first-audit no-baseline disclaimer)~~                                                                                                                                                                                                                                     | ~~In the closing message of the audit turn~~                                               |
| ~~16~~ done (docs-health pass 2026-11-27) | ~~All work committed by the auto-daemon (`06b3071` → `14dd7bf`); final tree dprint-clean and gate-green~~                                                                                                                                                                                                                                                                                      | ~~`git log`; `git status` clean at report time~~                                           |

## b) PARTIALLY DONE

1. **One broken intermediate reached master's history (not origin):** my first
   README quickstart edit referenced undefined `Item`/`body(r)` with broken
   indentation; the auto-daemon swept it into `caf1f0c` between my edit and my
   fix; the corrected version landed in the next daemon commit `378607f`.
   Living tree is correct (gates green on it); history carries ~1 daemon-cycle
   of a non-compiling docs example. `caf1f0c` is **local-only, unpushed** — the
   broken state never reached origin. Root cause and remedy in §d1/§e1.
2. **Stale golangci_lint_ls cache** kept reporting phantom compile errors at
   `negotiation_test.go:211-213` all session; CLI gates (compiler + golangci-lint
   CLI) are authoritative and green, but I never restarted the LSP server to
   clear the noise — a future session could misread it as a real failure.
   ~~3. **The "descriptive commit" TODO row needed three rewordings** as the daemon~~ superseded — the annotated v0.2.0 tag narrative plus descriptive per-task commits (85c82ef, 9975100, …) replaced the heuristic-message gap
   kept committing underneath it (its final wording is daemon-proof, but the
   underlying gap — six consecutive heuristic messages with no WHY — is open).
   ~~4. **ANNOTATE of the two prior status reports is not done**: their §f items are~~ done — 85c82ef (both 2026-09-29 reports annotated inline, 72 resolutions)
   now partially resolved by this session (compile fix, lint, CHANGELOG, README,
   harvest), but no inline `done at` markers exist in either report. A future
   docs-health ANNOTATE pass should strike each resolved item inline (HARVEST
   must never rewrite them; ANNOTATE resolves them on request).

## c) NOT STARTED

~~1. go-codec v0.2.0 → v0.3.0 bump (TODO High; dep change → own vendorHash repair + pdd-parity rationale).~~ done — v0.3.0 in go.mod (dependabot bump, verified 2026-09-30)
~~2. Release tag decision + cut (v0.1.2 vs v0.2.0; 🔵 BLOCKED on user decision; pdd floor bump + rev-pin coordination rides on it).~~ done — v0.2.0 (tag 3c66d39); pdd floor-bumped + rev-pinned (a92d625)
~~3. Descriptive closeout commit before any shared-branch push (TODO High).~~ superseded — annotated v0.2.0 tag + descriptive per-task commits
~~4. `Vary: Accept` emission from `WriteEncoded` (FEATURES PLANNED; TODO High).~~ done — 93f839c
~~5. Negotiation policy edge-case pins: `application/*` wildcard echo, duplicate-q precedence, empty/`=`-less `q` params, never-serve-refused property, golden behavior-matrix table (TODO High).~~ done — e0ee7b0
~~6. Fuzz target for the Accept parser (`ResolveResponseCodec`/`acceptQuality`/`qualityParam`; precedent `FuzzParseSSEData`).~~ done — fc818e5 (5m / 22.2M execs, 0 failures)
~~7. Direct unit tests for `bareMediaType`/`qualityParam`/`matchesEncoding`.~~ done — b9c98eb
~~8. Accept-Charset / Accept-Encoding boundary documentation in negotiation doc comments.~~ done — 8b62fe3
~~9. `negotiation_example_test.go` godoc example (go-codec convention).~~ done — 8b62fe3
~~10. 406-capable response-negotiation variant (ON_HOLD — API-shape decision; ROADMAP open question).~~ routed — FEATURES.md ON_HOLD row + ROADMAP open question 1
~~11. ANNOTATE + archive pass over both `docs/status/` reports (b4).~~ done — 85c82ef
~~12. All of ROADMAP.md's raw ideas (corpus regeneration cadence, go-codec extraction trigger, consumer-chain picture, negotiation patterns back into pdd docs, …).~~ routed — ROADMAP themes 1-3
~~13. sseparse corpus conformance re-run (waiting for sseparse's next tag; standing TODO row).~~ standing — TODO_LIST event row (sseparse still v0.1.0)
~~14. LSP cache hygiene habit (restart `golangci_lint_ls` when it contradicts CLI gates — b2).~~ done — AGENTS.md LSP-triage gotcha (33f44dd)

## d) TOTALLY FUCKED UP

**Nothing destroyed; no data loss; no reverts; no force-push; no dependency
changes (vendorHash safe — `nix build` green); no secrets; no API breaks.**
Honestly assessing the category:

1. **The auto-daemon committed a non-compiling README to master (`caf1f0c`).**
   My first quickstart edit was broken (`Item`, `body(r)`, wrong indentation);
   the daemon's sweep fired inside my edit→fix window and baked the broken
   state into history; my fix landed in the next daemon commit (`378607f`).
   The bitter irony: the new AGENTS.md gotcha I wrote in the same session —
   "the auto-commit daemon commits regardless of compile state; gate
   synchronously" — describes exactly this failure, and it still bit me at the
   docs layer. Mitigations that DID hold: the throwaway-package compile check
   caught it within two tool calls, `caf1f0c` is unpushed (verified via
   `git branch -r --contains`), and the fix-forward is the next commit in
   history. Lesson: for any file the daemon watches, "fix it in the next call"
   is not atomic enough — make replacements single self-consistent edits.
2. **Edit bounces after dprint (×2):** FEATURES and TODO_LIST edits failed with
   "file modified since last read" because I edited after running the formatter
   without re-reading — the 09-01 report's §d2 lesson ("don't hand-mutate files
   that a later structured edit will touch"), re-learned at my own expense.
3. **Wrong counts in the first-draft FEATURES.md** (wrote "12 FULLY_FUNCTIONAL";
   the file contains 14 rows). Caught and fixed BEFORE any commit landed
   (verified: `caf1f0c` already carries the correct 14), but an arithmetic slip
   in the document whose entire job is honest counts is exactly the
   hand-maintained-count rot the docs-health skill warns about.
4. **Five doc citations rotted mid-session:** my negotiation.go lint fixes
   shifted line numbers (+2) after I had already written negotiation.go:NN
   citations into FEATURES/TODO_LIST; required a re-grep and a 3-file
   citation-fix pass (`14dd7bf`). Sequencing error: all code edits should have
   finished before any line-citing doc was written.

## e) WHAT WE SHOULD IMPROVE

1. **Atomic-edit discipline under an auto-commit daemon:** any multi-step
   rewrite (edit → notice flaw → fix) must be compressed into ONE
   self-consistent edit, or the tree must be gate-clean between steps. The
   daemon's sweep interval is shorter than a "fix next call" habit assumes.
2. **Compile-verify example snippets as a standard step, immediately:** the
   throwaway-package `go build` trick worked; it should run BEFORE the first
   daemon sweep, not after.
3. **Code first, docs second:** every line-number citation written before the
   last code edit is a liability. Finish all `.go` changes, then write docs.
4. **dprint before edits, or re-read after fmt — never both blind.** Formatter
   runs invalidate the edit tool's file tracking (bounced twice).
5. **Derive counts, never hand-count:** `grep -c` the rows like the docs-health
   standing-gate principle demands; my own 12-vs-14 slip proves the point.
6. **Clear stale LSP caches when they contradict CLI gates** (`lsp_restart`)
   instead of leaving phantom errors that pollute the next session's context.
7. **Daemon-proof wording for time-sensitive doc claims:** anything saying
   "uncommitted at audit time" is stale before the ink dries; write against the
   daemon's behavior, not against a frozen tree.

## f) Up to 50 things we should get done next

_Sorted by impact. 1–10 are the verified TODO_LIST rows (already harvested —
done there first); 11–30 are repo-level follow-ups; 31–50 are cross-repo /
fleet ideas surfaced this session (mostly ROADMAP fuel and other repos'
queues, routed here only as pointers). Per the status-report skill: a large N
is a brainstorm, not a commitment list — items 11+ need docs-health HARVEST
routing rigor before anyone treats them as commitments._

**Closeout of this session (TODO_LIST rows, in impact order):**

~~1. Decide + cut the next release tag — v0.2.0 framing (negotiation is a behavior contract change) vs v0.1.2 (bugfix framing); 🔵 needs user decision; then floor-bump pdd's go.mod and coordinate its nix rev-pin re-derive.~~ done — v0.2.0 (tag 3c66d39)
~~2. Bump go-codec v0.2.0 → v0.3.0 for pin parity with pdd; repair own vendorHash via `buildflow -s nix-hash-fix --fix`; verify negotiation still compiles against v0.3.0's API.~~ done — v0.3.0 in go.mod (verified 2026-09-30)
~~3. Commit the closeout with ONE descriptive message before any shared-branch push (needs user go-ahead; harness forbids unprompted commits).~~ superseded — annotated v0.2.0 tag narrative
~~4. Emit `Vary: Accept` from `WriteEncoded` when response encoding depends on Accept (write.go; FEATURES PLANNED row).~~ done — 93f839c
~~5. Pin negotiation edge cases with tests: `application/*` wildcard echo, duplicate-q precedence, empty/`=`-less `q`, never-serve-refused property, golden behavior matrix.~~ done — e0ee7b0
~~6. Fuzz the Accept parser (browser headers, `*/*`, junk q-values, huge lists).~~ done — fc818e5
~~7. Unit-test `bareMediaType`, `qualityParam`, `matchesEncoding` directly (whitespace variants, `Q=` case, near-miss suffixes).~~ done — b9c98eb
~~8. Document the Accept-Charset / Accept-Encoding out-of-scope boundary in negotiation doc comments.~~ done — 8b62fe3
~~9. Add `negotiation_example_test.go` godoc example (request codec → handler → WriteEncoded).~~ done — 8b62fe3
~~10. Re-run `TestParseSSEData_CorpusConformance` when sseparse next tags (standing row).~~ standing — TODO_LIST event row

**Repo-level follow-ups (11–30):**

~~11. ANNOTATE both prior status reports inline (strike resolved §f items with evidence), then decide ARCHIVE per docs-health classification.~~ done — 85c82ef
~~12. Restart/clear stale golangci_lint_ls cache; note the "phantom vs CLI-gate" triage order in AGENTS.md if it recurs.~~ done — AGENTS.md LSP gotcha (33f44dd)
~~13. Add the atomic-edit-under-daemon lesson (§e1) to AGENTS.md gotchas — it complements the existing "gate synchronously" gotcha.~~ done — AGENTS.md atomic-edit gotcha (33f44dd)
14. Consider a pre-commit-sweep gate in the daemon's heuristic (if configurable): refuse to sweep when `go build ./...` fails — fleet-wide win.
~~15. Keep the README-snippet compile check: make it a tiny flake app (`nix run .#check-readme`) so examples can never rot silently.~~ done — 9975100
~~16. Coverage check: run `go tool cover` locally to see the negotiation rewrite's coverage; the CI gate is 80% — know the real number (11:35 report flagged trusting gates blind).~~ done — 742bf3a (86.1%; CI 80% floor kept, no blind raise)
~~17. Sweep for other unverified-committed work: `git log` has more heuristic messages — grep for TODO/FIXME in recent diffs.~~ done 2026-09-30 — docs-health sweep: no TODO/FIXME/XXX markers in non-test sources
~~18. Decide whether `ResolveResponseCodec`'s both-refused echo deserves a doc-comment warning label (it silently "works" — the 406 variant is the real fix, ROADMAP).~~ done — echo semantics documented in the doc comments and pinned by golden-matrix rows; the 406 variant remains the real fix (ROADMAP)
~~19. Extend the golden-matrix idea beyond negotiation: a lifecycle-ordering golden test already exists (hook-ordering); consider one for sd_notify message sequences.~~ routed — ROADMAP theme 2 (sd_notify golden sequence)
~~20. Benchmark the SSE line reader once (it allocates a 32 KiB chunk per parser instance; fine today, but a number beats a hunch — ROADMAP fuel).~~ routed — ROADMAP theme 2
~~21. Check `flightRecorderMu` documentation still matches `newTestFlightRecorder` behavior after any future test additions (AGENTS.md gotcha depends on it).~~ verified 2026-09-30 — discipline intact (recorder tests serialized; full suite green)
~~22. Add `nix flake check` to the AGENTS.md Essential Commands sweep (I ran build + test apps but never plain `nix flake check`; checks.build is covered indirectly, but the command is the canonical sweep).~~ done — 33f44dd
~~23. Re-verify dprint plugin pins quarterly (markdown-0.22.1 etc. — pins rot).~~ standing — pins refreshed 33f44dd (2026-09-30); quarterly re-check remains a discipline
~~24. Decide whether FEATURES.md counts should be grep-derived in CI (tiny script, standing gate) instead of hand-maintained prose.~~ routed — TODO_LIST hygiene (counts rotted once — stale "1 PLANNED" — and needed a hand fix 2026-09-30)
~~25. Consider marking the ON_HOLD 406 row and the PLANNED Vary row with the same target release once the tag decision lands (keeps FEATURES and TODO_LIST in lockstep).~~ moot — Vary flipped to FULLY_FUNCTIONAL; no PLANNED rows remain (counts fixed 2026-09-30)
~~26. README: consider a short "Non-goals" pointer to ROADMAP so the mechanism-not-policy promise is visible on the sales page.~~ routed — TODO_LIST hygiene
~~27. doc.go: mention q-value/refusal behavior in the package doc's negotiation sentence (currently says "content negotiation" without the contract summary; README is precise, doc.go is vaguer — acceptable, but one clause would align them).~~ done — 8b62fe3 (doc.go carries the negotiation + Vary contract sentence)
~~28. Prune `.golangci.yml` ireturn allowlist entries for project-discovery-sdk/ro (copied from pdd's mirror; unused here — mirror fidelity vs dead config).~~ done — 4e11590
~~29. After the tag decision: update CHANGELOG's compare links (bottom of file) for the new version.~~ done — compare links updated in the v0.2.0 CHANGELOG section
30. Consider dependabot grouping scope: gomod weekly + grouped minor/patch already configured — verify the first grouped PR behaves with the vendorHash gate (it will fail nix jobs by design; pre-agree the close-and-adopt policy — 09-01 §f18).

**Cross-repo / fleet (31–50 — pointers, mostly other repos' queues):**

31. pdd: verify content_negotiation.go tests still pass against go-daemon v0.1.1 + rev-pin (compile-safe by construction; tests are the remaining check).
32. ~~pdd: after the tag decision, re-pin its flake to the new go-daemon rev + re-derive vendorHash.~~ done (pdd commit 5cbc99c bumped go-daemon to v0.5.0 with the vendorHash repaired; go-daemon is public, so no flake rev pin is needed any more) (docs-health pass 2026-11-28)
33. go-codec: confirm v0.3.0's changelog for breaking changes before item 2's bump.
34. go-codec: the dangling `testdata_test.go` gopls `unusedfunc` diagnostic (11:35 §f45) — still their queue.
35. go-cqrs-lite: transport negotiation adoption decision (drives the go-codec extraction trigger; ROADMAP theme 3).
36. Fleet: one-page consumer-chain picture with current pins (pdd → go-daemon → go-codec/go-flightrecorder/go-sse → go-datastar).
37. Fleet: "own + consumer vendorHash" lesson into crush-config references/lessons.md (09-01 §e2, still uncommitted there).
38. ~~Fleet: auto-commit-daemon descriptive-message habit — share this session's caf1f0c/378607f incident as the docs-layer case study.~~ routed to ROADMAP.md (theme 3, fleet pointer) (docs-health pass 2026-11-28)
39. go-sse: sseparse corpus regeneration cadence (generator exists, no trigger) — pairs with ROADMAP open question 3.
40. ~~go-sse: remaining 07-03 §f items routed there (coverage floors, tag-CI verification, fuzz soak) — untouched here, by design.~~ routed to ROADMAP.md theme 3 (go-sse: coverage floors, tag-CI verification, fuzz soak) (docs-health pass 2026-11-28)
41. go-datastar: v0.6.2 re-cut decision (frozen red tag-CI) — untouched, their queue.
42. buildflow: nix-hash-fix UX (intermediate builds reported as "failed" — 09-01 §f45) — untouched, its queue.
43. crush-config: consider a status-report asset note that Markdown override is a recurring user pattern (two repos in a row now flagged it).
44. Fleet: document the "GOTOOLCHAIN=local + ambient 1.26 gopls" triage (go-daemon AGENTS.md now has the pre-check line; siblings could copy it).
45. ~~pdd: publish the negotiation-pattern docs back-port (ROADMAP theme 3) when its docs pass next runs.~~ routed to ROADMAP.md (theme 3 already carries the pdd docs back-port pointer) (docs-health pass 2026-11-28)
46. ~~Fleet: agree the dependabot-on-vendorHash close-and-adopt policy once, per repo (this repo's item 30 generalizes).~~ routed to ROADMAP.md (theme 3, fleet pointer; this repo's close-and-adopt policy is already in AGENTS.md) (docs-health pass 2026-11-28)
47. go-sse: consider corpus-oracle cross-links (their sseparse README ↔ go-daemon conformance test) — marketing-grade evidence both ways.
48. buildflow: "verify README examples compile" as an optional buildflow step (this session hand-rolled it; it generalizes).
49. ~~Fleet: session-handoff MEASURED-vs-CLAIMED template (07-03 §f23) — this report's gates-at-close header is that pattern; make it standard.~~ routed to ROADMAP.md (theme 3, fleet pointer) (docs-health pass 2026-11-28)
50. Celebrate: the doc fleet is complete (six living docs, all fresh, all claims verified), master is gate-green end-to-end, and the negotiation overhaul finally has its verification receipt.

## g) Questions I CANNOT figure out myself

~~1. **Which framing should the next release carry — v0.1.2 ("bugfixes", quietly changing negotiation behavior) or v0.2.0 (behavior-contract change, SemVer-honest)?** This gates items 1 and 29, the pdd floor bump + rev-pin re-derive, and how loudly consumers should be told about `application/cbor-seq`/q-value behavior changes. I can prepare either; I cannot decide the product framing.~~ decided — v0.2.0 behavior-contract framing (tag 3c66d39)
~~2. **Do you want one descriptive closeout commit now (and should I also draft the six-message-history narrative somewhere, e.g. a tag annotation), or does the auto-daemon's flat history stand?** I will not commit without your explicit word; the TODO row stays 🔴 until you say.~~ superseded — annotated v0.2.0 tag + descriptive per-task commits
~~3. **Should the go-codec v0.3.0 bump happen now (fresh dep change, vendorHash repair, pdd-parity restored) or ride the next pdd-driven sync?** Doing it now means one more dep-touching gate sweep today; waiting means AGENTS.md's pin policy stays contradicted until then. Your call on timing and risk appetite.~~ done — landed via dependabot close-and-adopt (go.mod v0.3.0, verified 2026-09-30)

---

_Verification status at write time: all gates green on the final tree (build,
vet, race suite, golangci-lint 0 issues, dprint idempotent, nix build, nix run
.#test); origin at `06b3071`; `10619a9`..`14dd7bf` local-only; broken-README
commit `caf1f0c` local-only and fix-forwarded in `378607f`; FEATURES/ROADMAP
built, TODO_LIST rebuilt and harvested, CHANGELOG/README/AGENTS updated and
verified. Awaiting instructions._
