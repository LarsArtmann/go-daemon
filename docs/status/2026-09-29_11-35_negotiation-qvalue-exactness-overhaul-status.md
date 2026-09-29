# Status Report — 2026-09-29 11:35 — go-daemon Content-Negotiation Correctness Overhaul (q-values, exact media-type matching, refusal honoring) — Code Done, Verification UNVERIFIED

**Session scope:** Q&A on go-codec negotiation design → cross-repo comparison (go-daemon) → judgment call ("worth extracting to go-codec?") → user-directed improvement of `go-daemon/negotiation.go` → implementation + test extension → **verification interrupted before completing**.

**Repos touched this session:** `go-daemon` (2 files changed), `go-codec` (read-only analysis), `go-cqrs-lite` (read-only duplication check).

**Commit state:** Auto-commit daemon already flattened the work into `f1d1fe8` ("chore: auto-commit 2 changed file(s) (heuristic)") at 10:27. Per the go-codec AGENTS.md lesson, this change has **no meaningful commit message anywhere in history** — needs a descriptive hand-made commit before any shared-branch push.

---

## Top-Line Honesty (brutal self-review)

### What did I forget?

1. **The tests were never confirmed green.** The single most important step — "TEST AFTER CHANGES" — did not complete. Ambient `go` is 1.26.7 vs required 1.27 (`GOTOOLCHAIN=local`), so the first `go test` attempt failed on toolchain, the `nix develop -c go test ./... -race -count=1` retry was backgrounded (shell 003), and the session was interrupted while it was still producing no output. **At report time: test status = UNKNOWN.**
2. **Lint never ran.** No `golangci-lint fmt` (gci/golines auto-fix), no `golangci-lint run`, no `buildflow --build-mode dev`. The strict mirrored config has never seen the new code. My hand-written vertical spacing (blank lines inside switch-case bodies) may violate the whitespace linters.
3. **The CHANGELOG entry I planned was never written.** I decided in-session to add an `Unreleased → Changed` entry documenting the behavior change (q-aware ranking, refusal honoring, `application/cbor-seq` fallback change) — not done. Behavior changes without CHANGELOG entries are how docs start lying.
4. **README/doc.go wording not re-checked** against the new semantics (README table line 19 describes the negotiation functions; doc.go line 5 mentions negotiation; neither mentions q-values — probably still accurate as summaries, but unverified).
5. **No `git status`/diff review after writing** — I never re-read my own diff as a reviewer before the interrupt (the auto-commit then baked it in).
6. **A prior-session suggestion was silently dropped:** the "second-consumer trigger" for extracting media-type mapping into go-codec was never recorded in TODO_LIST.md (mooted in practice by this session's improvement, but the decision trail is now only in chat history).

### What could I have done better?

1. **My earlier "fine as-is" judgment undersold real defects.** Two turns before improving it, I called the old implementation "fine as-is — the simplicity is a feature, not a gap." When actually asked to improve, I found four genuine correctness bugs (substring false positives like `application/json-patch+json`, prefix false positives like `application/cbor-seq`, total ignorance of q-values including `q=0` refusals, no suffix support). The layering judgment (don't move it to go-codec) was right; the "code is fine" framing was sloppy honesty. "Mechanism works for the happy path" ≠ "correct".
2. **I should have verified the toolchain before writing code.** AGENTS.md warns ambient go may be older; I wrote ~200 lines first and hit the wall after. Running `go version` upfront costs 1 second.
3. **I should have kept verification in the foreground.** Backgrounding the nix test and then getting interrupted left the session in its worst possible state: code committed, verification unknown. Should have run the buildflow dev-mode gate (5–30s, per the buildflow skill) immediately after writing, before any analysis chat.
4. **Caller inventory was incomplete.** I grepped callers but never registered the sharpest finding out loud: **`ResolveResponseCodec` has zero production callers inside go-daemon** — it is exported consumer-facing API only (write.go only uses `ContentTypeFor`). That status (ghost API vs deliberate consumer surface) was never resolved or reported until this report.

### What could I still improve?

1. Close the verification gap (test + lint + format), then write the CHANGELOG entry.
2. Pin the negotiation policy decisions (tie-break → CBOR, q=0 substitution, wildcard-neutral echo) as documented contracts — they currently live only in doc comments and test names.
3. Decide the fate of response-side negotiation: consumer-facing API, internal wiring (e.g. inside `WriteEncoded`), or shrink.
4. `Vary: Accept` is unhandled anywhere — a real HTTP-caching correctness gap for any consumer that serves both encodings through caches.

---

## a) FULLY DONE

1. **go-codec negotiation design answer** — established the library's stance: no runtime negotiation by design; codec choice is explicit and stamped (`Encoding()`), one-way `TranscodeToJSON` bridge, `AutoDetect` is diagnostics-only (doc.go, autodetect.go, codec.go). References delivered with file:line citations.
2. **go-daemon mechanism comparison** — located and explained the real negotiation implementation (`negotiation.go`): request codec from Content-Type, response codec from Accept with echo-request fallback, media-type helpers, JSON fallback everywhere.
3. **Duplication check across consumers** — confirmed go-cqrs-lite `transport/http` has **zero** Accept/Content-Type codec resolution (only `CBORToJSONTransform` payload bridge); go-daemon was the sole implementation. This grounded the "don't extract to go-codec" verdict.
4. **Extraction judgment** — recommended against moving negotiation into go-codec (transport-agnostic library, net/http would pollute it, policy-vs-mechanism split, single consumer, no duplication to centralize). Identified the only defensible extraction (pure media-type ↔ codec mapping) and its trigger (a second consumer).
5. **`negotiation.go` rewrite (committed in `f1d1fe8`)** — +134/−18 lines:
   - Exact media-type matching (`bareMediaType`): cut at `;`, trim, lowercase — kills the `application/cbor-seq` prefix false positive and the `strings.Contains` substring false positives.
   - RFC 6839 structured-suffix support: `application/vnd.acme+cbor` → CBOR, `application/vnd.acme+json` / `application/json-patch+json` → JSON.
   - q-value ranking (`acceptQuality` + `qualityParam`): highest q wins; tie prefers CBOR (preserves old preference order); q clamped to [0,1]; unparseable q counts as 1 (RFC 9110 default).
   - **q=0 refusal honoring** (`substituteForRefused`): a refused encoding is never returned — when the request codec is the refused one, the other supported encoding is served. Both-refused → echo request codec.
   - Wildcard neutrality: `*/*` (and `application/*` absence-of-preference) still echo the request codec — preserves documented behavior and the existing `*/*` tests.
   - **Dedup:** `ResolveRequestCodec` now delegates to `CodecForContentType` (removed the in-file split brain); `isCBORContentType` deleted; all exported signatures unchanged (no API break).
   - Doc comments rewritten as precise contracts (RFC 9110 §12.5.1 / RFC 6839 references).
6. **Test extension (committed in `f1d1fe8`)** — ~20 new table cases across `TestResolveRequestCodec` (suffixes, uppercase, `application/cbor-seq` and `application/cborx` near-misses), `TestResolveResponseCodec` (q-preference both directions, tie, q=0 refusal × request-codec combinations, both-refused, unparseable q, suffix selection, browser-style header, garbage), `TestCodecForContentType` (suffix, uppercase, near-miss fallback). All pre-existing cases kept passing-by-design (backward compatibility was an explicit design constraint).
7. **Toolchain obstacle handled per AGENTS.md** — ambient go 1.26.7 < required 1.27 correctly diagnosed from AGENTS.md guidance (`GOTOOLCHAIN=local`); `nix develop` route chosen instead of fighting the toolchain.

## b) PARTIALLY DONE

1. **Verification of the rewrite** — code written and committed, but `nix develop -c go test ./... -race -count=1` (background shell 003) had **still not produced output at report time**. The suite may be green, red, or stuck on nix evaluation. This is the session's biggest open risk: **the committed state is unverified.**
2. **Quality gate** — buildflow/golangci-lint not run at all. Per repo AGENTS.md the sequence is `golangci-lint fmt` then `run`; per the buildflow skill the delegated form is `buildflow --build-mode dev`. Neither started.
3. **CHANGELOG** — entry planned with drafted content (Changed: q-aware ranking, refusal honoring, cbor-seq behavior change, suffix support, exact matching); file not edited.
4. **Documentation consistency** — README negotiation row and doc.go mention not audited against new semantics; likely fine, unverified.
5. **History hygiene** — the change exists only as an auto-commit heuristic message; a descriptive commit (or amend-into-meaningful-message strategy) is pending, and MUST happen before pushing any shared branch (go-codec AGENTS.md: the error-contract overhaul shipped with no meaningful commit message anywhere — repeat of that mistake in progress).

## c) NOT STARTED

1. `Vary: Accept` support (response caching correctness for dual-encoding consumers).
2. 406-response variant of `ResolveResponseCodec` (current signature `codec.Codec` cannot express "nothing acceptable"; both-refused silently echoes request codec today — see question 2).
3. Fuzz/property coverage for the Accept parser (`acceptQuality`/`qualityParam` are hand-rolled parsers; go-daemon has fuzz precedent in `transform_fuzz_test.go`'s sibling convention).
4. Direct unit tests for `bareMediaType`/`qualityParam`/`matchesEncoding` (currently only tested transitively through the four exported functions).
5. `application/*` wildcard explicit test (behavior is defined — treated as no-preference — but no test pins it).
6. Ghost-API resolution: confirm whether `project-discovery-daemon` (primary consumer) actually calls `ResolveResponseCodec`/`ResolveRequestCodec`.
7. go-cqrs-lite `transport/http` adoption decision (it still has no negotiation; if it gains one, that's the extraction trigger for go-codec).
8. go-codec media-type mapping extraction (deliberately deferred until trigger fires — correctly not started).
9. Negotiation benchmark (irrelevant at this size unless profiling says otherwise — intentionally not started).
10. TODO_LIST.md harvest of this report's section f.

## d) TOTALLY FUCKED UP

**Nothing destroyed; no data loss; no revert needed.** Honestly assessing the category:

1. **Committed-but-unverified code is the closest thing to a fuck-up.** The auto-commit daemon captured `f1d1fe8` mid-verification-gap: if the race test fails or the linter finds real issues, the broken state is now in history on `master` and will need a fix-forward commit. Not data loss (old implementation recoverable via git), but process debt: the daemon committed what I never confirmed.
2. **The interrupted-turn state.** The session ended its last action waiting on a background shell with the user given no status — this report is the first complete accounting. The lesson: verification must complete (or explicitly fail) _before_ any turn ends, not be backgrounded across the finish line.
3. Explicitly checked and cleared: no force-push, no revert of others' work, no dependency changes (so no vendorHash risk per AGENTS.md — `nix build` unaffected by pure-Go edits), no secrets, no API breaks (all four exported functions kept signatures).

## e) WHAT WE SHOULD IMPROVE

1. **Verify-before-committing discipline in auto-commit fleets:** run the buildflow dev gate synchronously the moment code is written; treat "backgrounded verification" as "unverified".
2. **Judgment honesty:** separate "wrong layer for this code" verdicts (which were correct) from "code is correct" verdicts (which were not checked) — one sentence of audit, not vibes, before calling something fine.
3. **Negotiation policy should be a written contract:** three decisions (tie→CBOR, q=0 substitution, wildcard echo) are load-bearing for clients; they deserve doc-comment contracts (done) + CHANGELOG (pending) + tests (partially) + README (unaudited).
4. **Resolve the ghost-API question:** an exported, tested, internally-unused function pair is either the product (consumer API — then the primary consumer should prove it) or scope creep (then shrink). Find out which.
5. **Parser hardening:** hand-rolled header parsers get fuzz targets; the go-daemon repo already has the fuzz convention to copy.

## f) Up to 50 things we should get done next

_Sorted by impact. Items 1–8 are this session's direct closeout; 9–20 are negotiation-specific; 21–40 are repo-level items surfaced by this session; 41–50 are cross-repo. Most of 21+ are ROADMAP fuel, not commitments (per status-report skill: large N is a brainstorm)._

**Closeout of this change (blocking):**

1. Read background shell 003 output → confirm `go test ./... -race -count=1` green under Go 1.27 (nix develop).
2. If red: fix forward, re-test, then write the CHANGELOG entry with the real outcome.
3. Run `buildflow --build-mode dev` (lint + race tests, delegated gate).
4. `golangci-lint fmt` then `golangci-lint run ./...` per repo AGENTS.md if running tools directly instead.
5. Write `CHANGELOG.md` Unreleased → `Changed` entry (q-aware ranking; tie prefers CBOR; q=0 never returned; RFC 6839 suffixes; exact matching — `application/cbor-seq` now JSON-fallback, `application/json-patch+json` now matched as JSON by suffix; wildcards still echo; JSON remains the universal fallback).
6. Hand-make one descriptive commit covering the negotiation overhaul (supersede the heuristic auto-commit narrative) before any shared-branch push.
7. Audit README negotiation row + doc.go line 5 against new semantics; update if they overpromise/understate.
8. Harvest this section into `TODO_LIST.md` (items 9–50 route to TODO vs ROADMAP).

**Negotiation-specific:**
9. Add `application/*` wildcard test (pin no-preference echo behavior).
10. Add `Vary: Accept` emission (probably in `WriteEncoded`/write.go, not negotiation.go) when response encoding depends on Accept.
11. Decide + document both-refused semantics: echo (today) vs 406-capable API variant (`ResolveResponseCodecErr` or similar).
12. Fuzz `ResolveResponseCodec`/`acceptQuality` (seed with browser headers, curl `*/*`, junk q-values, huge lists).
13. Unit-test `bareMediaType`, `qualityParam`, `matchesEncoding` directly (whitespace variants, `q` with spaces, `Q=` uppercase param name — RFC says q is case-insensitive? verify and pin).
14. Pin first-q-wins vs last-q-wins for duplicate entries (`application/json;q=0.5, application/json;q=0.9`).
15. Decide whether `text/*`, `application/*` should ever count as JSON-adjacent (today: never; document).
16. Property test: response encoding is always one of {JSON, CBOR, reqCodec.Encoding} and never a q=0-refused one.
17. Confirm `qualityParam` on `;q=` (empty value) and `;q` (no `=`) behaves as documented (falls to 1).
18. Consider Accept-Charset / Accept-Encoding pass-through policy (out of scope today — document the boundary).
19. Snapshot the full behavior matrix as a golden test table (tiny, but makes policy regressions reviewable in PRs).
20. If project-discovery-daemon needs strict RFC behavior later: add q-precedence (specific > `application/*` > `*/*`) behind an option, not by default.

**Repo-level (surfaced this session):**
21. Resolve ghost-API status of `ResolveResponseCodec` (see question 1) — wire it into `WriteEncoded`'s optional path, keep as consumer API with proof, or shrink.
22. Verify sseparse corpus conformance re-run tracking (commit `eb52a75` mentioned tracking; confirm the TODO items exist and are current).
23. Confirm `docs/status/` reports are harvested into TODO_LIST (prior report 09-01 + this one).
24. Add "run toolchain check (`go version` vs go.mod) before writing code" to AGENTS.md Essential Commands preamble (bit this session).
25. Document in AGENTS.md that `nix develop` test runs can take minutes on cold evaluation (background-job expectations).
26. Consider committing status reports promptly (auto-commit daemon handles it, but the meaningful-message gap applies to reports too).
27. Review `write.go` for the same exactness class of bugs (it writes Content-Type — does anything parse it back with loose matching? `CodecForContentType` is the only reader — already fixed).
28. Audit `transport.go`/`sse.go` for other `strings.Contains`-style header matching (same bug class as the old negotiation code).
29. Check whether `socket.go` or lifecycle code assumes a single codec/encoding anywhere (mechanism/policy boundary review).
30. Add a negotiation example to doc.go or README (ResolveRequestCodec → handler → WriteEncoded round-trip).
31. Consider a `negotiation_example_test.go` (godoc examples run as tests — go-codec convention; go-daemon could adopt for this API).
32. Review test naming convention (arrows in names are established style — keep consistent for new cases; done this session).
33. Evaluate golangci-lint's `wsl`/vertical-spacing findings on the new switch style; conform or suppress deliberately.
34. Ensure `min`/`max` builtins don't trip the mirrored lint config (Go 1.21+ builtins; config predating them may complain — verify).
35. Check `.golangci.yml` for a `gorm`/`nilnil`-style linter that might object to returning concrete codecs (unlikely; verify once).
36. Add negotiation policy to a short ADR or doc comment block if the team treats doc comments as insufficient contract.
37. Track go-codec dependency pin policy: negotiation now leans harder on `Encoding` string tags — confirm the pin (AGENTS.md: "pinned to versions the primary consumer uses").
38. Version the behavior change: decide whether this lands in 0.1.2 or 0.2.0 (behavior change vs additive — SemVer call, see question 3).
39. After any future dependency bump: `buildflow -s nix-hash-fix --fix` (AGENTS.md gotcha — not needed this session, zero dep changes).
40. Re-run the flight-recorder mutex discipline check if any new tests touch recorders (none do this session — noted for future negotiation-over-SSE tests).

**Cross-repo:**
41. Verify project-discovery-daemon compiles against the new negotiation.go (exported signatures unchanged, so expected green — verify anyway; it pins go-daemon by rev per AGENTS.md).
42. Confirm project-discovery-daemon's actual call sites for the ghost-API question (feeds item 21).
43. go-cqrs-lite: decide if `transport/http` should adopt Accept-based negotiation or stay transform-only (drives the go-codec extraction trigger).
44. go-codec: record the extraction trigger (second consumer) in its TODO_LIST/ROADMAP when it fires — not before.
45. go-codec: the dangling `testdata_test.go` gopls `unusedfunc` diagnostic observed this session — route to that repo's queue (not this repo's job).
46. Coordinate go-daemon version bump with the primary consumer's nix rev pin (AGENTS.md: every bump re-derives vendorHash there).
47. Share the "auto-commit flattens meaningful history" mitigation: descriptive commit habit (item 6) is a fleet-wide lesson from go-codec AGENTS.md.
48. If a second transport (e.g. WebSocket in go-cqrs-lite) needs codec resolution, extract media-type mapping to go-codec — the pre-agreed trigger.
49. Consider publishing go-daemon negotiation patterns back into project-discovery-daemon docs (it's the extracted origin).
50. Post-closeout: re-run this session's cross-repo grep (`negotiat` across the fleet) to confirm no third implementation appears.

## g) Questions I CANNOT figure out myself

1. **Is `ResolveResponseCodec` load-bearing?** Does project-discovery-daemon (or any other consumer) actually call it — i.e., is response-side negotiation the product, or an internally-unused export we should wire into `WriteEncoded` or shrink? I can grep the repo if you point me at it being in scope, but whether unused-but-exported is "API surface by intent" for this library is your policy call.
2. **When a client refuses everything (all mentioned types `q=0`), should we serve 406 instead of silently echoing the request codec?** The current signature (`codec.Codec`, no error) cannot express "nothing acceptable." This is a product/API-shape decision, not an implementation detail.
3. **Version and contract call:** does the behavior change (q-aware responses, `application/cbor-seq` no longer decoded as CBOR, refused encodings never returned) ship as a patch (0.1.2, "bugfix" framing) or minor (0.2.0, "behavior contract change"), and is tie-break→CBOR the intended permanent server preference vs echo-the-request-codec?

---

_Verification status at write time: code committed (`f1d1fe8`), tests UNKNOWN (background nix develop run, shell 003, no output yet), lint NOT run, CHANGELOG NOT written. Do not treat this change as done until items 1–5 of section f are closed._
