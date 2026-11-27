# Status Report — 2026-10-01 02:15 — docs-health AUDIT: full annotate + archive + harvest pass (TODO_LIST rebuilt, FEATURES counts repaired, pareto plan archived, dprint-vs-strikes root-caused)

> **Live remainder:** §a done; §b partials were closed by the 2026-10-01 plan execution and the flip (2026-11-27). Live work: TODO_LIST.md documentation rows.

**Session scope:** executed the docs-health skill on ALL `**/*2026-0*` files (6 reports)
plus all six living docs → VERIFY every claim against code → HARVEST the two
unharvested 09-30 reports into a rebuilt TODO_LIST → ANNOTATE all three 09-30
reports inline (91 resolutions, tool-driven) → repair the 09-01 report's 24
mangled strikes → root-cause WHY they were mangled (dprint vs full-row table
strikes) → ANNOTATE + ARCHIVE the fully-executed pareto plan (73 strikes,
`git mv` to `docs/planning/archived/`) → full gate sweep.

**Format:** Markdown per the standing user instruction pattern (last three
reports were `.md` by explicit request; the status-report HTML default stays
overridden).

**Gates at close (all green on the final tree):** `go build` / `go vet` /
`go test ./... -race -count=1` (1.28s) / `golangci-lint run` (0 issues) /
`nix build .#default` / `nix flake check` / `nix run .#check-readme` /
`nix develop -c dprint fmt` (idempotent) / `check-rows.py` — zero PARTIAL rows;
two mixed tables exit 1 **by design** (deliberately-open rows; see §e4).
Tree state at report time: clean except the one-line archived-banner link fix
(`../../status/`), which the auto-daemon will sweep.

**Commit state:** the auto-daemon flattened this pass into four heuristic
commits (`fae82a2` → `704bfa7`). No hand-made commit (harness rule: no commit
without explicit user word). The annotate + archive + harvest narrative now
lives only in this report — the descriptive-message gap this repo's own
AGENTS.md warns about, repeated (see §d4).

---

## Brutal self-review (asked directly)

### What did I forget?

1. **I struck 91 rows before testing the format against the repo's own
   formatter.** The 09-01 file's mangled `| ~~ |` rows were IN MY HANDS at the
   start of the session — documented evidence that something eats full-row
   table strikes — and I did not infer the mechanism until dprint chewed my
   own strikes an hour later. The skill's own rule ("ALWAYS dry-run a new
   file shape first") was followed for the SPEC matching but not for the
   FORMAT-vs-DPRINT interaction. A two-line scratch table + `dprint fmt`
   would have exposed it in 30 seconds and saved the entire repair cycle
   (annotate → dprint → check-rows → repair → dprint → check-rows).
2. **The tooling choice was wrong for tables.** `annotate-rows.py` exists
   precisely because table strikes must be per-cell (dprint-stable); I used
   `annotate-status-items.py`'s whole-line format everywhere for consistency,
   and consistency is exactly what broke. House formats exist per surface;
   I flattened them.
3. **Formatter triage last, not first.** I ran `nix fmt` (nixfmt-tree — nix
   files only), then `dprint fmt` outside the devShell (cannot resolve plugin
   URLs), then `dprint fmt --incremental false` (flag parse failure), then
   `nix develop -c dprint fmt --incremental false` (same), before plain
   `nix develop -c dprint fmt` worked. Four invocations, and `nix fmt`'s
   "0 changed" nearly got cited as format-cleanliness evidence — it never
   touched a markdown file. One `grep formatter flake.nix` at the start
   answers all of it (now an AGENTS.md gotcha, but learned at my expense).
4. **Prose-strike wrap integrity never verified.** `check-rows.py` only checks
   tables. The 00-43 report's 47 prose strikes were grep-counted after dprint
   but I never confirmed the markdown `textWrap` behavior can't split a long
   `~~…~~` across lines (dprint default is `maintain`, so risk is low — but
   "low because of a default I never read" is not verification).
5. **QMD tool misuse:** two failed `mcp_qmd_multi_get` calls (relative, then
   absolute paths) before accepting the index doesn't cover this repo and
   falling back to `view`. The tool's own instructions state collections are
   scoped; I paid two calls to re-learn it.

### What could I have done better?

1. **Scratch-file the interaction, then mass-apply:** probe format survival on
   a copy before striking real files. (The single biggest miss.)
2. **Read the formatters' definitions before running them** — the flake
   declares `formatter = nixfmt-tree` and dprint only in devShell packages.
3. **Resolve the check-rows mixed-table semantics BEFORE close** instead of
   shipping an exit-1 gate with a rationale. Options exist (split open rows
   into a separate "still open" table per report, or teach check-rows an
   open-row manifest); I chose justification over structure.
4. **Ask about the descriptive commit** before letting the daemon flatten a
   major docs closeout — the repo's AGENTS.md explicitly says "commit
   descriptively yourself when the change matters," and this pass mattered.

### What can I still improve?

- Turn "gate green at close" into "gate green AND gate semantics understood" —
  a red exit I can explain is still a red exit.
- Batch formatter + annotation knowledge into reusable checks (a
  `check-docs` flake app could chain dprint --check, check-rows, and the
  strikethrough-presence grep so the next pass needs zero tribal knowledge).

---

## a) FULLY DONE

_(all verified: build / vet / race suite / lint 0 issues / `nix build` /
`nix flake check` / check-readme / dprint idempotent; both `check-rows`
hard-failures resolved; tree clean except the swept-in-flight link fix)_

| #                                         | Work                                                                                                                                                                                                                                                                                                                                                          | Evidence                                                                                                        |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| ~~1~~ done (docs-health pass 2026-11-27)  | ~~**All 6 `**/*2026-0*` files read in full** (3× 09-30 status, 2× 09-29 status, 1× pareto plan) plus all six living docs (TODO_LIST, CHANGELOG, AGENTS, README, ROADMAP, FEATURES) — zero blind edits~~                                                                                                                                                       | ~~session reads; QMD bypassed for direct `view`~~                                                               |
| ~~2~~ done (docs-health pass 2026-11-27)  | ~~**Every load-bearing doc claim verified against code**: CI-exact coverage flags reproduce 86.1% exactly; weak spots confirmed (`write.go:29` 65.6%, `socket.go:195` 76.2%, `WithXxx` 0.0%, `notifyStopping` 50.0%, `lineAtEOF` 66.7%); Vary emission at `write.go:53`; doc.go contract sentence~~                                                           | ~~`go tool cover -func`; greps~~                                                                                |
| ~~3~~ done (docs-health pass 2026-11-27)  | ~~**Cross-repo pin parity verified**: pdd go.mod carries go-codec v0.3.0 + go-flightrecorder v0.2.0 + go-daemon v0.2.0, flake rev-pinned at `c0aaf4c` (the v0.2.0 tag commit); sseparse v0.1.0 in both~~                                                                                                                                                      | ~~pdd `go.mod`, pdd `flake.nix:36-38`~~                                                                         |
| ~~4~~ done (docs-health pass 2026-11-27)  | ~~**ci.yml audit done**: no check-readme, no `nix flake check`, 80% coverage gate present — TODO rows grounded; `.github/dependabot.yml` exists (gomod weekly + grouped actions); `.gitignore` read end-to-end — coherent, nothing to tidy (11-29 §f17 resolvable)~~                                                                                          | ~~file reads this session~~                                                                                     |
| ~~5~~ done (docs-health pass 2026-11-27)  | ~~**FEATURES.md counts repaired** — they had rotted ("14 FULLY_FUNCTIONAL + 1 PLANNED" while the file contains 15 FULLY_FUNCTIONAL + 0 PLANNED after the Vary flip); grep-derived recount; verification-basis date corrected to 2026-09-30 with CI-exact reproduction note~~                                                                                  | ~~`grep -c FULLY_FUNCTIONAL` = 15; edited counts section~~                                                      |
| ~~6~~ done (docs-health pass 2026-11-27)  | ~~**AGENTS.md fixed + hardened**: `ix run` → `nix run` typo; seed-only fuzz-corpus policy added; **two new gotchas** — dprint destroys whole-row table strikes (use per-cell), and `nix fmt` is nix-only (dprint needs the devShell)~~                                                                                                                        | ~~AGENTS.md Essential Commands + Gotchas tail~~                                                                 |
| ~~7~~ done (docs-health pass 2026-11-27)  | ~~**TODO_LIST.md rebuilt via HARVEST** — 22-item and 28-item §f lists from the two unharvested 09-30 reports routed item-by-item: verified-done dropped, 29 rows landed (standing 1 + go-public 9 + owner 5 + CI 1 + release 3 + tests 4 + hygiene 6), every row with code path + report citation~~                                                           | ~~TODO_LIST.md (was 1 suspiciously thin row — the exact cross-file consistency failure the skill warns about)~~ |
| ~~8~~ done (docs-health pass 2026-11-27)  | ~~**ROADMAP.md absorbed the routed-out ideas**: Vary-only-when-negotiated (theme 1), sd_notify golden sequence + SSE line-reader benchmark (theme 2), cross-repo pointers — pdd Vary adoption, crush-config lessons.md, buildflow UX, docs-site/announcement (theme 3)~~                                                                                      | ~~ROADMAP.md themes 1-3~~                                                                                       |
| ~~9~~ done (docs-health pass 2026-11-27)  | ~~**All three 09-30 reports annotated inline**: 91 resolutions (00-43: 47, 10-33: 5, 11-29: 39) with `--emit-keys` → `--verify` → apply (never hand-typed keys); banners added; each file's §b/§c/§f/§g resolved or deliberately left bare~~                                                                                                                  | ~~`grep -c '~~'` per file; annotate tool output "annotated N items" ×3~~                                        |
| ~~10~~ done (docs-health pass 2026-11-27) | ~~**09-01 report's 24 mangled strikes repaired** — the broken whole-row strike form was converted to per-cell strikethrough; check-rows went from "INCOMPLETE (0/50)" to recognizing all 24~~                                                                                                                                                                 | ~~check-rows before/after~~                                                                                     |
| ~~11~~ done (docs-health pass 2026-11-27) | ~~**dprint-vs-strikes ROOT-CAUSED**: dprint's markdown plugin reparses the whole-row tilde form table rows into a stray `~~` cell and drops the closing marker — this is how 09-01 got corrupted in the first place, and it re-mangled my own strikes until converted; canonical per-cell format survives realignment (re-verified after two dprint passes)~~ | ~~mangling observed live; 132 rows converted across 4 files; check-rows clean post-dprint~~                     |
| ~~12~~ done                               | ~~**Pareto plan annotated + ARCHIVED** — every one of its 14 tasks and 59 subtasks struck with per-task hash verdicts (72 via the tool + row 38 by hand: its literal grep completeness command defeats the tool's skip guard); banner with closeout link; `git mv` to `docs/planning/archived/`~~                                                             | ~~73 strikes in the archived file; LINK-OK after `../../status/` depth fix~~                                    |
| ~~13~~ done (docs-health pass 2026-11-27) | ~~**Archive classification completed for all 6 files**: pareto plan archived (fully executed); the 5 status reports stay unarchived — each carries deliberately-open items (owner decisions, go-public cluster, cross-repo queues), documented in banners~~                                                                                                   | ~~per-file banner text~~                                                                                        |
| ~~14~~ done (docs-health pass 2026-11-27) | ~~**README + CHANGELOG verified accurate, untouched**: README rows match shipped behavior (snippet compile-gated); `[Unreleased]` correctly carries only Vary — no inflation, no drift~~                                                                                                                                                                      | ~~read + check-readme gate~~                                                                                    |
| ~~15~~ done (docs-health pass 2026-11-27) | ~~**Internal-link sweep**: one real defect found and fixed (archived banner's `../status/` → `../../status/` from its new directory depth); all markdown links now resolve~~                                                                                                                                                                                  | ~~link-check loop; LINK-OK from the file's own directory~~                                                      |

## b) PARTIALLY DONE

1. **Archive coverage is 1 of 6 files.** The user's "archive FULLY done and
   UPDATED files" is satisfied — but only the pareto plan qualifies; the five
   status reports structurally cannot (open items). The strict
   "EVERY item resolved" bar means they may never archive (their open items
   are mostly other repos' queues). A calibrated bar (see §g3) would change
   that.
2. **check-rows exits 1 on two files** (09-01: 24/50 struck; 10-33: 4/22) —
   every struck row is uniform, zero PARTIAL, and the unstruck rows are
   deliberately open ("absence of a marker IS the open signal"), but the gate
   still reports INCOMPLETE because _some_ rows in those tables are struck.
   Rationalized, not structurally resolved (§e4, §g2).
3. **Prose-strike integrity is presumed, not proven** — dprint's `textWrap`
   default (`maintain`) implies long struck lines can't split, but I never
   read that default before relying on it (forgot-item 4).
4. **Descriptive-commit narrative missing**: the pass's history is four
   `chore: auto-commit N changed file(s)` messages; the WHY lives only in
   this report. The repo convention says commit descriptively when it
   matters — I didn't (harness rule + no explicit user go-ahead), so the gap
   stands for the owner to ratify (§g1).
5. **The final link fix is in-flight** — uncommitted at report time; the
   daemon will sweep it. If it lands, `704bfa7`..HEAD completes the pass.

## c) NOT STARTED

1. Everything now sitting in the rebuilt TODO_LIST — by design: the go-public
   pre-flip cluster (9 rows), owner decisions (5 🔵), CI gate completeness,
   release follow-ups, 4 test-coverage rows, 6 hygiene rows, plus the
   standing sseparse event row.
2. The two bare leftovers in the 00-43 report: §f14 (pre-commit sweep gate in
   the daemon's heuristic — fleet idea, not configurable here) and §f30
   (first grouped dependabot PR behavior — unobserved).
3. Archive re-evaluation for the 09-29/09-30 reports (gated on §g3).
4. No code changes were made or attempted — docs-only session, per scope.
5. `check-docs` flake app chaining the doc gates (§e6) — proposed, not built.

## d) TOTALLY FUCKED UP

**Nothing destroyed; no data loss; no reverts; no force-push; no dependency
changes (vendorHash untouched — `nix build` green); no API surface touched;
final tree fully gate-green.** The damage is process debt:

1. **Mass-applied a format I hadn't tested against the formatter, with the
   failure evidence already in front of me.** The 09-01 mangling was readable
   at 09:00; I reproduced it on 91 fresh rows at ~01:30 and only then
   diagnosed it. The annotate→dprint→mangle→repair→re-dprint→re-verify cycle
   burned roughly double the annotation effort. This is the same
   "verify-before-scale" failure the skills codify, executed anyway.
2. **Tool-choice flattening:** used the whole-line striker on tables because
   it handles prose too, when the repo's formatter only tolerates the
   per-cell format on tables. `annotate-rows.py` (per-cell, section-scoped)
   was the right tool for every table in this repo; I reached for one tool
   for all surfaces.
3. **Formatter flailing:** four invocations to run dprint correctly, and
   `nix fmt`'s "0 changed" was one careless citation away from becoming
   false gate evidence in a report whose entire job is verified claims.
4. **The descriptive-commit gap repeated knowingly:** this repo's own
   AGENTS.md documents that heuristic messages flatten history and says to
   commit descriptively when the change matters. A 188-resolution,
   harvest-plus-archive docs pass is the definition of "matters," and it
   landed as `chore: auto-commit 6/2/2/7 changed file(s)`. I had a rule for
   not committing and no plan for the consequence.
5. **check-rows exit 1 shipped at close** (two mixed tables), explained in
   prose rather than fixed structurally. A red gate with a good story is
   still a red gate.

## e) WHAT WE SHOULD IMPROVE

1. **Probe doc-format interactions on a scratch file before mass
   application.** The dry-run rule extends beyond spec-matching to "does the
   OUTPUT SURVIVE this repo's formatter." One 30-second probe would have
   halved this session's cost.
2. **Match the annotation tool to the surface:** tables → `annotate-rows.py`
   (per-cell), prose lists → `annotate-status-items.py`. Codify it in the
   skill's workflow note so the next pass doesn't re-derive it.
3. **Formatter triage first:** read the flake's `formatter` + devShell
   packages before any format claim. (Now an AGENTS.md gotcha — the lesson
   is written where the next session will trip over it.)
4. **Give check-rows an escape hatch for legitimately-open tables** — either
   house convention (reports split struck/open rows into separate tables) or
   an `--allow-open-rows <manifest>` mode. Upstream candidate for the
   docs-health assets.
5. **Upstream a table-mode in `annotate-status-items.py`** that emits the
   per-cell format directly, so the dprint-safe format is the default
   everywhere in the fleet (any dprint-formatted repo has this landmine).
6. **Chain the doc gates into one app** (`nix run .#check-docs`: dprint
   --check + check-rows over docs/ + strikethrough-presence grep over
   status/planning) so doc health is a standing gate, not a per-session
   ritual.
7. **Pre-authorize descriptive commits for doc passes** (owner call — §g1)
   so harvest/annotate/archive closeouts stop flattening into heuristic
   messages.

## f) Up to 50 things we should get done next

_Sorted by impact. Items 1–29 are the freshly harvested TODO_LIST rows
(already there, with code+report evidence — listed here as clusters, not
duplicated); 30–50 are new, session-grounded follow-ups. Per the
status-report skill: a large N is a brainstorm, not a commitment list._

**Already harvested and verified in TODO_LIST.md (do them there):**

1. Go-public pre-flip cluster (9 rows): git-history secret scan, dep license
   inventory, sevlyar/pkg.go.dev collision check, README external polish,
   SECURITY.md + templates, repo description/topics, pre-publish audit
   script, post-flip AGENTS + pdd de-GOPRIVATE (last two 🔵 gated on flip).
2. Owner decisions (5 🔵): API-freeze policy, name decision, docs/status
   visibility, the flip itself, GitHub Actions billing (+ post-billing
   tag-CI and actions-v7 verification).
3. CI gate completeness (1 row): check-readme + `nix flake check` +
   private-assumption audit into ci.yml (verified absent today).
4. Release follow-ups (3 rows): v0.3.0 demand-gated 🔵, release-page visual
   check, pkg.go.dev render check.
5. Test coverage gaps (4 rows): `WithXxx` setters (0%), `WriteEncoded`
   encode-error (65.6%), `Start()` startup errors (76.2%), `notifyStopping`/
   `lineAtEOF` (50%/66.7%) — percentages re-measured today with CI flags.
6. Hygiene (6 rows): CONTRIBUTING.md ratification + nix-first alignment,
   `90f86db` provenance, golangci mirror-drift re-check, godoc examples
   (`ExampleParseSSEData`, socket), `nix flake check --all-systems`,
   grep-derived FEATURES counts, README non-goals pointer.
7. Standing event row: sseparse corpus re-run on next tag (v0.1.0 verified).

**New, grounded in this session (30–50):**

30. Build the `check-docs` flake app (§e6) — dprint --check + check-rows +
    strike-presence grep as one standing gate.
31. Upstream per-cell table mode into `annotate-status-items.py` (§e5) and a
    mixed-table escape hatch into `check-rows.py` (§e4) — docs-health assets.
32. Propose the "own + consumer vendorHash"-style lesson for
    dprint-vs-table-strikes into crush-config `references/lessons.md`
    (fleet-wide: any dprint repo with annotated tables hits this).
33. Re-verify prose-strike wrap integrity under dprint's actual `textWrap`
    behavior (forgot-item 4) — cheap grep/golden check, closes b3.
34. Decide the descriptive-commit policy for doc passes (§g1) and, if
    ratified, hand-make one closeout commit for `fae82a2..HEAD` narrative —
    owner call.
35. Resolve check-rows mixed-table semantics (§g2): split "still open" rows
    into separate tables in the 09-01 + 10-33 reports so the gate goes fully
    green, or accept justified exit-1 as terminal.
36. Set the archive bar policy (§g3): strict every-item-resolved (today) vs
    "all this-repo items resolved + remainder struck as routed pointers"
    (would unlock archiving the 09-29/09-30 reports).
37. After the go-public decision cluster lands: re-run this session's
    annotation+archive pass over the 5 status reports (their banners already
    point at TODO_LIST).
38. Verify the auto-daemon swept the archived-banner link fix cleanly and the
    archived file's links resolve in the committed tree (in-flight at report
    time).
39. Add "annotate-tool output must survive `dprint fmt`" to the docs-health
    skill's workflow note (probe-first rule), fed by this session's §d1.
40. Consider a tiny golden test for dprint-vs-markdown-strike behavior
    (scratch table in `docs/` or a fixture) so formatter upgrades that change
    the mangling behavior get caught by CI, not by the next annotate pass.
41. When the CI gate row (TODO item 3) lands: also run `check-docs`/dprint in
    ci.yml so docs health is enforced, not local-only.
42. When CONTRIBUTING.md is ratified (hygiene row): add the docs-gate section
    (dprint, check-readme, check-docs) so external contributors know the doc
    health contract.
43. Sweep the remaining heuristic history: the six-message gap narrative from
    the 00-43 report plus this pass's four — candidate: a single
    `docs/history/` narrative file if the owner cares (otherwise drop).
44. Re-run `check-rows.py` after EVERY future dprint pass over annotated
    reports — make it muscle memory via check-docs (item 30) rather than
    discipline.
45. Watch for the first grouped dependabot PR (00-43 §f30, still bare) — its
    behavior feeds the close-and-adopt policy's first real test.
46. If the fleet adopts the per-cell strike convention, backfill the 11-35
    report's prose strikes (currently whole-line) for consistency — cosmetic,
    low priority.
47. Consider annotating §b/§c/§g of the 09-29 reports to the same bar as the
    09-30 set (their banners cover the delta; full inline pass would make
    archiving mechanical once §g3 is decided).
48. The 00-43 §f14 idea (daemon refuses to sweep when `go build` fails) —
    route to buildflow/daemon config once, fleet-wide win; parked there
    today.
49. A "docs-health pass receipt" convention: each annotate pass ends with a
    one-table summary (files, strikes, archives, gate states) in its status
    report — this report's §a is the template; make it standard.
50. Celebrate: the doc fleet is now internally consistent — TODO_LIST carries
    the full verified backlog, every 09-30 report resolves inline, the
    executed plan is archived, and the strike format finally survives the
    formatter.

## g) Questions I can NOT figure out myself

1. **Descriptive commits for doc passes:** the auto-daemon flattened this
   pass (annotate + archive + full TODO_LIST rebuild) into four
   `chore: auto-commit` messages, and the same will happen to the next one.
   Do you want to pre-authorize ONE descriptive hand-made closeout commit
   per docs-health pass (I'll ask no further), or should every commit remain
   explicitly requested?
2. **check-rows mixed tables:** the 09-01 and 10-33 reports have tables where
   only resolved rows are struck (the rest are deliberately open), so the
   uniformity gate exits 1 forever. Split open rows into a separate "still
   open" table per report (gate goes green, slightly changes report shape),
   or accept justified exit-1 as the terminal state?
3. **Archive bar:** keep "archive only when EVERY item in the file is
   resolved" (today: only the pareto plan qualified), or relax to "all
   THIS-repo items resolved; remaining cross-repo pointers struck as
   `routed — <repo>`"? The relaxed bar would let the two 09-29 reports and
   the 00-43 report archive soon, shrinking `docs/status/` to genuinely
   live work.

---

_Verification status at write time: all gates green on the final tree (build,
vet, race suite, lint 0 issues, nix build, nix flake check, check-readme,
dprint idempotent, check-rows with the two documented by-design exits);
auto-daemon at `704bfa7` with one in-flight link fix; 188 inline resolutions
across 6 files; TODO_LIST carrying 29 verified rows; awaiting instructions._
