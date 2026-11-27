# Session Status Report: v0.5.0 Release, Five PR Merges, and the json/v2 Flavour Audit

- **Date:** 2026-11-27 15:32 CET
- **Repo:** `go-daemon` (branch `master`, working tree clean at report time)
- **Session scope:** Operator pasted `gh issue list` (one open issue: #4, "Release tag for JSONv2Codec + injectable negotiation codecs") and `gh pr list` (five open PRs, #5–#9) with the instruction "get it all sorted and rebased into master".
- **Scope discipline:** No unrelated repo-wide research. Everything below comes from this session's commands and the files read for it. The one exception is deliberate and reported: consumer state (`nextd`, `project-discovery-daemon`) was only inspected to size the follow-ups, not audited.

## What Happened This Session

1. **Baseline.** `master` at `6b9ee1d`, clean, `v0.4.0` tag 46 commits behind. All five PR branches were based on `6b9ee1d` and ahead by 1–2 commits. All five had failing `build-and-test` and `nix` jobs with **empty step lists** — Actions is suspended on the account's billing, so no job actually executed.
2. **Reviewed every PR diff before touching a branch** (PR 5 socket error-path tests incl. the `chmodSocket` seam; PR 6 `transport_error.go`; PR 7 `wait.go`; PR 8 `--all-systems`; PR 9 notify-platform docs). Reviewed the surfaces the diffs touch (`write.go`, `jsonv2.go`, `negotiation.go`, `socket.go`, `isSocketAlive`, `DefaultProbeTimeout`) rather than trusting the PR bodies.
3. **Rebased and merged in dependency-free order**, fast-forward only: PR 9 → `9b684f6`, PR 8 → `81781c7`, PR 5 → `adac893`+`898e97e`, PR 6 → `e191a52`, PR 7 → `f73c081`. Two conflicts, both resolved by keeping both sides: `AGENTS.md` Gotchas (PR 8 vs PR 9) and `CHANGELOG.md` Added (PR 7 vs PR 6). `FEATURES.md` counts were corrected to 21 FULLY_FUNCTIONAL / 1 ON_HOLD (22 rows) after the two feature rows landed.
4. **Improved one PR while rebasing it:** `ExampleProbeUntilAlive` started a server on `context.Background()` and leaked it; it now stops the server and waits for `Start` to return before the temp dir is removed.
5. **Gate after every merge** (`go build`, `go vet`, `go test -race -count=1`, then `golangci-lint run` after each feature merge). Final: 0 lint issues, 96.7% statement coverage (CI floor 80%).
6. **Closed issue #4's remaining gap** with `WriteJSONWithCodec` / `WriteErrorfWithCodec` (codec-aware counterparts of `WriteJSON`/`WriteErrorf`, nil ⇒ default codec), `WriteJSON`/`WriteErrorf` kept as wrappers, table-driven tests for both plus CBOR envelope round-trip, and a wrapper test added after coverage showed `WriteJSON` at 0%.
7. **Discovered a wire-level truth while writing that test:** `codec.JSONCodec{}` on this toolchain is **not** v1. With go-codec v0.3.3 on go1.27.3: nil slice → `[]`, `time.Duration` → encode **error**, map keys unordered. Probed directly (`/tmp/jsonprobe`), then rewrote the stale `AGENTS.md` block, the `JSONv2Codec` doc comment, `FEATURES.md`, `README.md`, `CHANGELOG.md`, and the release notes to match.
8. **Found and corrected my own false claim:** the first draft of those docs said `GOEXPERIMENT=none` "selects the v1 backend again". It does not build this module at all — `jsonv2.go` imports `encoding/json/v2` and `prometheus/common` imports `encoding/json/jsontext`, both excluded with the experiment off (verified: `GOEXPERIMENT=none go build ./...` fails on both imports). Fixed in `AGENTS.md`, `CHANGELOG.md` (commit `beefcd7`) and in the already-published GitHub release notes (`gh release edit v0.5.0`).
9. **Release.** Promoted `CHANGELOG` `[Unreleased]` → `[0.5.0] - 2026-11-27` (Unreleased left with placeholders), added the missing `[0.4.0]`/`[0.5.0]` compare links, annotated tag `v0.5.0` on `d0b8259`, pushed `master` + tag, created a `--prerelease` GitHub release (v0.x convention), verified `go get github.com/LarsArtmann/go-daemon@v0.5.0` in a scratch module **and** built a consumer that calls `TransportErrorKind.Unavailable`, `DefaultWaitPollInterval`, `ClassifyTransportError`.
10. **Closed the five PRs.** #9 auto-detected as MERGED (its SHA landed unchanged). #5–#8 were rebased, so their original head SHAs are no longer ancestors; I declined a force-push and closed them with a pointer comment to the landing commit.
11. **Docs/health.** `dprint fmt` fixed a pre-existing `.config/metadata.yaml` formatting failure (tracked file) and realigned the edited tables. `TODO_LIST.md`: the stale "re-tag v0.4.1 with a `go 1.27` floor" row was replaced by the real remaining action (bump `nextd` to v0.5.0 + `go 1.27.2`), because a 1.27 floor is impossible while go-codec v0.3.3 declares `go 1.27.2`; the hygiene row now names all five unannotated reports instead of three.
12. **A flake.lock auto-update rode into my docs commit** via `git add -A`; caught before push and split into its own commit (`4e8ba04`), then re-ran `nix build` + `nix flake check --all-systems` against the new lock.

## Self-Critique (asked first, answered honestly)

**What did I forget?**

1. **I published a claim about a build configuration without executing it.** The `GOEXPERIMENT=none` escape hatch was reasoned from go-codec's build tags in isolation; running it fails to build the module at all. It was wrong in three places at once (AGENTS, CHANGELOG, GitHub release) for a few minutes.
2. **I did not run the CI gate set, only the fast four.** No `govulncheck`, no `actionlint`, no coverage-threshold comparison (I read 96.7% but did not run the gate's awk), no `nix run .#test` / `nix run .#lint` wrappers (bare tools instead, so `GOTOOLCHAIN=local` + `GOPRIVATE` from the flake env were not exercised).
3. **I checked no real consumer before releasing.** The synthetic consumer build came after the tag. `nextd` (pinned v0.4.0) and `project-discovery-daemon` were never built against v0.5.0, so the `time.Duration` hazard I documented is unverified against the structs that actually ship.
4. **I did not verify the "merged" semantics before choosing rebase.** Rebasing rewrote the SHAs, so four PRs read CLOSED. A `gh pr` state check before deciding would have surfaced that tradeoff up front.
5. **I let an unrelated `flake.lock` change into a docs commit** — `git add -A` without reading `git status`'s full diff first.
6. **I did not notice `WriteJSON` had gone to 0% coverage** until I measured coverage by hand for this report; the wrapper rewiring silently dropped it.
7. **I did not update the stale numbers I was already editing around**: `FEATURES.md` still claimed 86.1% coverage (2026-09-30) and `AGENTS.md` still pinned sseparse at v0.2.0. Both are now fixed, but only because I looked.
8. **I left the tag three commits behind `master`** (docs/formatting/lock follow-ups), so `git describe` reads `v0.5.0-3-g…` instead of `v0.5.0`.

**What could I have done better?**

1. **Fold the codec audit into the release decision, not the documentation.** The moment the nil-slice probe disagreed with the test expectation, the right move was to ask "is a default-codec wire change acceptable for v0.5.0, or does this need a v1 pin first?" before tagging. I documented the change instead of gating on it.
2. **Add the missing wrapper test in the same edit as the rewiring.** `WriteJSON` at 0% was a self-inflicted coverage regression in a commit that existed to raise quality.
3. **Use GitHub's own merge path when SHAs matter.** `gh pr merge --rebase` (or a merge commit) keeps the PR state honest; my local rebase + close-with-comment is auditable but reads as "abandoned" to anyone scanning the PR list.
4. **Run the consumer smoke test as part of pre-release verification**, not after.
5. **Split the "release" commit from the "docs truth" commit.** One commit carried the helpers, the flavour rewrite, the floor explanation and the TODO reshuffle; a reader of `git log` cannot tell which change shipped in the tag and which is post-tag documentation.
6. **State the go-codec downgrade option in the release notes.** `nextd` at `go 1.27.1` is a real cost of v0.5.0; the notes say "automatic toolchain switch" but not "or pin go-codec v0.3.1 and stay on 1.27".
7. **Verify the release notes' verification section by executing each listed command**, in order, at the tagged commit. I did run them, but across a dirty-tree window (`nix build` on `f73c081-dirty`), which is weaker evidence than a clean-tree run at the tag.

**What could I still improve (systemic)?**

1. **One `scripts/release-check.sh`** that runs the full CI set (build, vet, race, coverage gate, govulncheck, actionlint, lint, nix build, all-systems flake check, readme, dprint, counts) and prints a single verdict — the current split between "what CI runs" and "what I remember to run" is where the false claim came from.
2. **Make `check-docs` part of `nix flake check`** (or CI), so the five unannotated reports cannot rot silently for two months.
3. **A post-merge PR-state protocol:** after any external merge, query PR state; if CLOSED-not-MERGED is unacceptable, ask about `--force-with-lease` at that moment.
4. **Wire-shape tests that name the backend they observe** (e.g. `TestDefaultJSONCodec_DurationHasNoV1Representation`) so a future go-codec/Go change fails loudly on the wire contract instead of quietly changing it.
5. **A decision record for the codec policy** (`docs/decisions/…-jsonv2-default.md`): v2 default is now unconditional; consumers wanting v1 shapes must pass `JSONv2Codec` with explicit options. Right now that lives only in AGENTS/CHANGELOG.

---

## a) FULLY DONE

| Item                                                     | What was done                                                         | Evidence                                                                                                                                                  |
| -------------------------------------------------------- | --------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~1~~ done — five PRs integrated                         | ~~Rebase all open PRs onto master and merge~~                         | ~~Linear history `9b684f6`, `81781c7`, `adac893`, `898e97e`, `e191a52`, `f73c081`; `git log` on `master`~~                                                |
| ~~2~~ done — conflicts resolved on merit                 | ~~Resolve the two real conflicts without dropping either side~~       | ~~`AGENTS.md` keeps both Gotchas bullets; `CHANGELOG.md` keeps both Added bullets; FEATURES counts 21/1 (22)~~                                            |
| ~~3~~ done — every merge gated                           | ~~Build, vet, race tests, lint after each merge~~                     | ~~`go test ./... -race -count=1` green at every step; `golangci-lint run` 0 issues after PRs 5, 6, 7~~                                                    |
| ~~4~~ done — issue #4's named gap closed                 | ~~Make the write helpers accept a negotiated codec~~                  | ~~`WriteJSONWithCodec`, `WriteErrorfWithCodec` (`write.go`), tests `TestWriteJSONWithCodec`, `TestWriteErrorfWithCodec`, `TestWriteJSON_DefaultWrapper`~~ |
| ~~5~~ done — v0.5.0 released                             | ~~Promote CHANGELOG, tag, push, GitHub release~~                      | ~~Tag `v0.5.0` on `d0b8259`; release https://github.com/LarsArtmann/go-daemon/releases/tag/v0.5.0 (`prerelease`)~~                                        |
| ~~6~~ done — release verified as a consumer              | ~~Prove the tag resolves and compiles outside the repo~~              | ~~`go get …@v0.5.0` + build with `Unavailable`/`DefaultWaitPollInterval`/`ClassifyTransportError` in `/tmp/release-verify-050`~~                          |
| ~~7~~ done — issue #4 closed                             | ~~Close with the release and the corrected framing~~                  | ~~`gh issue view 4`: CLOSED 2026-11-27T14:28:14Z~~                                                                                                        |
| ~~8~~ done — default JSON codec behaviour established    | ~~Probe what `codec.JSONCodec{}` actually does on this toolchain~~    | ~~`/tmp/jsonprobe`: nil slice `[]`, duration encode error, unordered map keys~~                                                                           |
| ~~9~~ done — the false escape hatch removed              | ~~Delete the untested `GOEXPERIMENT=none` claim everywhere~~          | ~~`AGENTS.md` (rewritten block), `CHANGELOG.md` (`beefcd7`), release notes (`gh release edit`)~~                                                          |
| ~~10~~ done — docs gates that can pass, pass             | ~~dprint, row completeness, FEATURES counts~~                         | ~~`nix run .#check-docs`: dprint clean, `check-rows: 3 file(s) complete`, counts 21/22 match~~                                                            |
| ~~11~~ done — stale TODO rows made truthful              | ~~Replace the impossible v0.4.1 re-tag row; widen the hygiene row~~   | ~~`TODO_LIST.md` rows 47 and 53~~                                                                                                                         |
| ~~12~~ done — coverage regressions absent                | ~~Measure, find `WriteJSON` at 0%, fix it~~                           | ~~96.7% total; weakest function 75.0%; `TestWriteJSON_DefaultWrapper` added~~                                                                             |
| ~~13~~ done — stale measurement docs refreshed           | ~~Coverage paragraph, sseparse version~~                              | ~~`FEATURES.md:13`; `AGENTS.md:111-112`~~                                                                                                                 |
| ~~14~~ done — flake lock change isolated and re-verified | ~~Split the auto-update out of the docs commit and re-run nix gates~~ | ~~`4e8ba04`; `nix build` + `nix flake check --all-systems` run against the new lock~~                                                                     |
| ~~15~~ done — PR list cleared                            | ~~Every PR is out of OPEN state~~                                     | ~~`gh pr list --state open` empty~~                                                                                                                       |

## b) PARTIALLY DONE

| Item | What works | What remains | Effort |

> **Ledger:** rows carry their own verdict (done / routed / open); open rows are the live remainder named in the routing arrows.

<!-- check-rows:ledger -->

| --------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------- | ------ |
| PR merged-state semantics | PR #9 is MERGED; #5–#8 carry the landing SHA in a comment and their work is fully in `master` | GitHub shows them CLOSED; origin branch tips still point at pre-rebase SHAs; stale worktrees under `~/go-daemon-rows` | S |
| Tag vs `master` | Tag is correct and immutable on `d0b8259` | `master` is now 3+ commits ahead (formatting, docs truth, lock, this report); `git describe` shows `v0.5.0-N-g…` | XS |
| Release verification | Gates green at/around the tag; consumer build verified | CI never ran (billing); `nix build` ran on a dirty-tree window, not the exact tagged tree; govulncheck/actionlint not run | S |
| Consumer propagation | `nextd` gap identified and recorded as a TODO row | No consumer was actually bumped; `project-discovery-daemon` compatibility with the v2 default codec unverified | M |
| ~~Docs health~~ done — 2026-11-27: corpus annotated and archived; both docs gates green | ~~dprint, rows, counts green; the new report is self-struck~~ | ~~Five historical reports still lack strikes, so `nix run .#check-docs` still exits 1~~ | ~~M~~ |
| Codec policy documentation | AGENTS, CHANGELOG, README, release notes all now say "v2 shapes, unconditionally; pass a codec for v1" | No ADR; no test pinning the wire contract as a named invariant; no `JSONv1Codec` convenience adapter | S/M |

## c) NOT STARTED

| Item | Why not started | Still wanted? |

> **Ledger:** rows carry their own verdict (done / routed / open); open rows are the live remainder named in the routing arrows.

<!-- check-rows:ledger -->

| ----------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- | ---------------------------------------------- |
| CI/tag-CI green run for `v0.5.0` | GitHub Actions billing suspended (owner-level, external) | Yes — the release has no CI evidence |
| Bump `nextd` to v0.5.0 / `go 1.27.2` **→ TODO_LIST release follow-ups (nextd bump)** | Consumer repo, separate change; only the TODO row was written | Yes |
| Bump `project-discovery-daemon` to v0.5.0 + re-derive `vendorHash` **→ TODO_LIST release follow-ups (pdd bump + DTO audit)** | Same; also blocked on the `time.Duration` wire question | Yes |
| Audit consumers for `time.Duration`/`omitempty` JSON fields **→ TODO_LIST release follow-ups (consumer DTO audit)** | Scope discipline: no unrelated research this session | Yes — this is the release's real risk |
| ~~Annotate/archive the five unannotated status reports~~ done — 2026-11-27 (15 reports and 1 plan archived; docs-gates green) | ~~A docs-health pass, not this task~~ | ~~Yes~~ |
| ~~`docs/decisions/…-jsonv2-default.md`~~ done — docs/decisions/2026-11-27_jsonv2-default-and-wire-shapes.md | ~~Not asked; the facts are captured in AGENTS/CHANGELOG for now~~ | ~~Recommended~~ |
| `govulncheck`, `actionlint`, coverage-threshold run **→ TODO_LIST documentation and gates (govulncheck/actionlint parity)** | Not run this session | Yes (release hygiene) |
| `GOEXPERIMENT`/toolchain build matrix in CI **→ ROADMAP (GOEXPERIMENT matrix is moot: the module requires jsonv2)** | Discovered only at the end that the matrix is a single point (jsonv2 only) | Debatable — the module cannot build without it |

## d) TOTALLY FUCKED UP

Nothing catastrophic: no data loss, no re-tagged or moved release tag, no broken commit pushed to `master`, no PR content lost. The failures that did happen:

| # | What went wrong | Severity | Status |

> **Ledger:** rows carry their own verdict (done / routed / open); open rows are the live remainder named in the routing arrows.

<!-- check-rows:ledger -->

| --------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | ---------------------------------------------------------------- |
| 1 | ~~Published a false build claim (`GOEXPERIMENT=none` selects v1) in AGENTS, CHANGELOG and the live GitHub release notes~~ | High trust | Fixed within the session (`beefcd7`, `gh release edit`) |
| 2 | Four PRs read CLOSED instead of MERGED because rebasing changed the SHAs and I chose not to force-push | Low | Open — needs an operator decision (g2) |
| ~~3~~ done — fixed before push (split into 4e8ba04) | ~~An unrelated `flake.lock` auto-update was swept into a docs commit by `git add -A`~~ | ~~Low~~ | ~~Fixed before push (split into `4e8ba04`)~~ |
| ~~4~~ done — fixed (TestWriteJSON_DefaultWrapper) | ~~`WriteJSON` fell to 0% coverage because I rewired the wrappers without a test for the wrapper itself~~ | ~~Low~~ | ~~Fixed (`TestWriteJSON_DefaultWrapper`)~~ |
| 5 | Tag sits behind `master`; the release notes' "green on the tagged commit" evidence mixes a dirty-tree `nix build` with a clean-tree flake check | Low | Partially open — re-verification on the exact tag is a follow-up |

## e) WHAT WE SHOULD IMPROVE

1. **Never publish a build-configuration claim you have not executed.** One `GOEXPERIMENT=none go build` would have prevented a wrong sentence in a live release note.
2. **Prefer GitHub-side merges when PR state matters, or ask about force-push at decision time** — a local rebase plus close-with-comment is honest but ambiguous to future readers.
3. **Run the full CI gate set locally before tagging**, including govulncheck and actionlint, not just build/vet/test/lint.
4. **Gate the release on the consumer, not on the library's own tests.** The v2 default codec changes wire shapes; the consumers are where that lands.
5. **Measure coverage after any rewiring**, not only at release time. A 0% wrapper is invisible until the report is due.
6. **`git add -A` is a hazard in this repo.** The auto-commit daemon and flake auto-updaters write concurrently; stage explicit paths for release commits.
7. **Make docs gates executable by CI.** Two months of red `check-docs` is a policy hole, not an accident.
8. **Cut the tag from the final commit**, or accept and document that `master` moves ahead immediately (auto-commit daemon guarantees drift).
9. **One release-check script** with a single verdict, so "which gates did I run?" is not a memory test.
10. **Record the codec policy as an ADR**, because the AGENTS entry is now load-bearing and easy to lose in a rewrite.

## f) Up to 50 Things To Get Done Next

1. Push the trailing `master` commits (docs truth, lock, coverage, this report) after re-running the fast gate.
2. Re-run `nix build .#default` and `nix flake check --all-systems` on the exact final `master` (new nixpkgs lock) and record the result here or in the TODO row.
3. Decide g2 (force-with-lease the rebased branch tips so #5–#8 show MERGED, or keep CLOSED-with-pointer).
4. Decide g3 (keep go-codec v0.3.3 + `go 1.27.2`, or downgrade to v0.3.1 for a `go 1.27` floor).
5. Bump `nextd` to go-daemon v0.5.0 and `go 1.27.2`; re-derive any vendor/nix hash; run its suite.
6. Audit `project-discovery-daemon` DTOs for `time.Duration` and nil-slice fields before its v0.5.0 bump.
7. Bump `project-discovery-daemon` to v0.5.0, re-derive `vendorHash`, run its tests (it asserts on `Validate()` message substrings).
8. Collapse pdd's JSON v1/v2 split now that the library's default is v2: replace its private `classifyTransportError` with `ClassifyTransportError`, and its hand-rolled wait loops with `WaitForSocket`/`ProbeUntilAlive`.
9. Switch pdd's error writes to `WriteErrorfWithCodec` so its error bodies match its negotiated codec.
10. Restore GitHub Actions billing (owner) so CI can run for `v0.5.0` and future tags.
11. Re-run the `v0.5.0` tag workflow once billing is back; if it fails, fix forward in v0.5.1 (never move the tag).
12. Run `govulncheck ./...` locally and record the result.
13. Run `actionlint -no-color` locally and record the result.
14. Run the CI coverage gate exactly (awk comparison against 80.0) and record the value.
15. Add `govulncheck` and `actionlint` as flake apps so CI and humans run the same commands.
16. Write `scripts/release-check.sh` chaining every gate with one exit code.
17. Add a test pinning the default codec's duration behavior (`TestDefaultJSONCodec_DurationHasNoV1Representation`).
18. Add an ADR `docs/decisions/2026-11-27_jsonv2-default-and-wire-shapes.md`.
19. Add a `JSONv1Codec` adapter (explicit `encoding/json` v1) for consumers that need v1 shapes without v2 option gymnastics.
20. Decide whether `WriteEncoded(w, status, nil, v)` should default instead of panicking; cover whichever way with a test.
21. Add a godoc example for `WriteJSONWithCodec` / `WriteErrorfWithCodec` showing a negotiated-codec handler.
22. Add the same round trip to the README quickstart (it currently only shows default-codec writes).
23. Add a no-op-injection test proving `WriteErrorfWithCodec` never regresses to v1 when a JSONv2Codec is passed.
24. Consider `TransportErrorKind` gaining `PermissionDenied` (EACCES) for socket-mode mistakes. **→ ROADMAP theme 2 (TransportErrorKind PermissionDenied)**
25. Consider `ProbeUntilAlive` taking a per-probe timeout option instead of the fixed `DefaultProbeTimeout`.
26. Consider `WaitForSocket` taking a network argument (TCP parity with `NewServer`).
27. Add `-shuffle=on` to CI tests to catch order dependencies (two timing fixes in PR 5 hint at them).
28. Add a nightly `-count=15` run for the timing-sensitive socket/notify tests. **→ TODO_LIST documentation and gates (nightly timing run)**
29. Update the `Verify pkg.go.dev rendering` TODO row with the current v0.5.0 facts.
    ~~30. Annotate the five unannotated status reports (docs-health ANNOTATE) so `check-docs` can go green.~~ done — 2026-11-27 (five reports annotated; corpus archived)
    ~~31. Add `check-docs` to `nix flake check` or CI so it cannot rot again.~~ done — check-docs now runs in the CI nix job
    ~~32. Harvest this report's §f items into `TODO_LIST.md`/`ROADMAP.md` and strike/annotate this report.~~ done — this report's §f triaged into TODO_LIST/ROADMAP
30. Clean up the stale worktrees under `/home/lars/projects/go-daemon-rows/` and reconcile local vs origin branches.
31. Delete or refresh the origin branch tips for #5–#8 after g2 is decided.
32. Decide whether the four closed PRs should instead be re-opened and merged through GitHub for auditability.
33. Add a `.github/workflows/release.yml` (tag-triggered) so release gates are machine-enforced. **→ TODO_LIST release follow-ups (tag-triggered release workflow)**
34. Add a pre-push hook (build+vet+test) to defend against the auto-commit daemon shipping broken intermediate states.
    ~~38. Inspect the untracked/odd root artifacts (`coverage`, `reports/`, `result/`) and either track, ignore or remove them.~~ done — verified — coverage/reports/result and the Python caches are ignored
35. Confirm `.config/metadata.yaml` is tool-managed and belongs under dprint (it is now formatted and committed).
    ~~40. Re-check `AGENTS.md` against the tree once more (it grew three new gotchas this session; watch for length/duplication).~~ done — AGENTS re-checked 2026-11-27
    ~~41. Update `CONTRIBUTING.md` with the "pass an explicit codec" guidance added to README/AGENTS.~~ done — CONTRIBUTING gained the explicit-codec guidance
36. Add a release-notes template to the repo so the verification section is written from executed commands, not memory.
37. Set a policy for v0.x GitHub releases: `--prerelease` (current) vs "latest" badge; document it once.
38. Verify the new `v0.5.0` `go.mod` floor against `gvac`'s POISONERS registry and update that list if the floor stands.
39. Record the go-codec floor rationale in the go-codec repo (its version contract is now a downstream floor).
40. Add a consumer smoke test script — `scripts/consumer-smoke.sh` — that runs `go get` against the newest tag in a scratch module.
41. Decide whether `WriteJSONWithCodec`/`WriteErrorfWithCodec` should be flagged in FEATURES with their own row rather than sharing one.
    ~~48. Re-measure coverage after the consumer-driven changes and keep `FEATURES.md:13` honest.~~ done — 96.7% re-measured 2026-11-27
42. Clean the temp artifacts from this session (`/tmp/jsonprobe`, `/tmp/release-verify-050`, `/tmp/cover.out`, `/tmp/go-daemon-v0.5.0-notes.md`).
43. After the `nextd` bump lands, strike the TODO row with the evidence (and update the release follow-ups table).

## g) Questions I Cannot Answer Myself

1. **GitHub Actions billing:** do you want it restored (so v0.5.0 gets real CI evidence and future tags are gated), or is local-only verification the accepted norm for now? I cannot fix the account's billing state.
2. **PR state for #5–#8:** approve a `--force-with-lease` push of the four rebased branch tips so GitHub records them as MERGED (instead of CLOSED with a pointer comment), or keep the closed state and stale branches?
3. **The `go 1.27.2` floor:** keep go-codec v0.3.3 (floor 1.27.2, the json/v2 decode fix, `nextd` must upgrade its patch) or downgrade to v0.3.1 to regain a `go 1.27` floor at the cost of that fix? This is a consumer-facing trade-off only you can price.
