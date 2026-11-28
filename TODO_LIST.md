# TODO List

> Short-term, actionable, bounded work items, verified against the actual
> code. Completed work lives in [CHANGELOG.md](CHANGELOG.md); long-term ideas
> live in [ROADMAP.md](ROADMAP.md). Items are ranked by impact; status is
> verified, not assumed.
>
> **Last harvested 2026-11-28** from
> `docs/status/2026-11-27_16-49_docs-health-audit-corpus-archive-and-living-doc-rebuild.md`
> and the 89 open items the `docs-gates` `audit-all` replay found across the
> archived reports; each carries an inline verdict there (done, decided, or
> routed here or to [ROADMAP.md](ROADMAP.md)). Rows landed by the 2026-11-28
> session (both consumer bumps, the release gate, Go 1.27.4 in the flake, the
> wire-shape and nil-codec tests, examples, fuzz) are removed; they live in
> [CHANGELOG.md](CHANGELOG.md). Every row was checked against the code before
> it landed. Cross-repo items carry a `(repo: …)` tag and route through
> [ROADMAP.md](ROADMAP.md) theme 3.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | External dependency or decision needed before it moves. |

## Release follow-ups

| Task                                                                                                                            | Status    | Impact        | Effort | Evidence                                                                                                         |
| ------------------------------------------------------------------------------------------------------------------------------- | --------- | ------------- | ------ | ---------------------------------------------------------------------------------------------------------------- |
| Add message-wording stability to the release checklist: consumers substring-match `Validate()` errors, so error strings are API | 🔴 `TODO` | release trust | XS     | `docs/status/archived/2026-10-01_05-51` #10; project-discovery-daemon asserts on `Validate()` message substrings |

## Consumer adoption (DiscordSync)

| Task                                                                                                                         | Status    | Impact            | Effort | Evidence                                                                                                                                                 |
| ---------------------------------------------------------------------------------------------------------------------------- | --------- | ----------------- | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Adopt `Lifecycle.DeferReady` + `Server.MarkReady` in DiscordSync and delete `cmd/discordsync/systemd.go` (repo: DiscordSync) | 🔴 `TODO` | consumer adoption | S      | `docs/feedback/new/2026-10-04_discordsync-consumer-feedback.md` F3 resolved 2026-11-28; both adoption blockers (F1 public, F3 deferrable READY) are gone |

## Documentation and gates

| Task                                                                                                                                                                    | Status    | Impact          | Effort | Evidence                                                                                                                                                      |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | --------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Add a nightly `-count=15` timing run for the socket/notify tests, and `-shuffle=on` to CI                                                                               | 🔴 `TODO` | flake detection | S      | `docs/status/2026-11-27_15-32` §f27-28; two timing fixes in PR 5 hint at order dependencies                                                                   |
| Add `scripts/consumer-smoke.sh` that runs `go get` against the newest tag in a scratch module, so a release is verified as a consumer would use it                      | 🔴 `TODO` | release trust   | S      | `docs/status/2026-11-27_16-49_docs-health-audit-corpus-archive-and-living-doc-rebuild.md` §f33; the v0.5.0 verification was a manual scratch module in `/tmp` |
| Pin the erraudit known-dismissed baseline in a durable artifact so audits diff against it instead of re-litigating the same findings                                    | 🔴 `TODO` | quality         | XS     | `docs/status/2026-11-27_16-49_docs-health-audit-corpus-archive-and-living-doc-rebuild.md` §f40; the current baseline lives only in `AGENTS.md` prose          |
| Add shellcheck and `bash -n` gates for `scripts/*.sh`, as a devShell command, a `release-check.sh` step and a CI step                                                   | 🔴 `TODO` | gate parity     | S      | `docs/status/archived/2026-10-01_03-52` #9; three shell scripts carry the README, docs and release gates with no syntax gate                                  |
| Add a negative corpus or property test for `scripts/check-rows.py`                                                                                                      | 🔴 `TODO` | gate parity     | S      | `docs/status/archived/2026-10-01_03-52` #46                                                                                                                   |
| Make the FEATURES count gate in `scripts/check-docs.sh` pin the row total, not only the per-label counts                                                                | 🔴 `TODO` | gate parity     | XS     | `docs/status/archived/2026-10-01_03-52` #16                                                                                                                   |
| Cite `sd_notify(3)` in `docs/supervision.md` next to its `systemd.exec(5)` reference                                                                                    | 🔴 `TODO` | docs            | XS     | `docs/status/archived/2026-10-08_22-46` #21                                                                                                                   |
| Add an AGENTS.md gotcha: lint tools can under-report, so verify a dismissal by grep, not by a clean run                                                                 | 🔴 `TODO` | docs            | XS     | `docs/status/archived/2026-10-05_12-32` #9                                                                                                                    |
| Add a pull request template with a ran-release-check checkbox                                                                                                           | 🔴 `TODO` | docs            | XS     | `docs/status/archived/2026-09-29_09-01` row 35; `.github` has no PR template                                                                                  |
| Register flight-recorder metrics only after `flightrecorder.New` succeeds, with a failure-path test                                                                     | 🔴 `TODO` | robustness      | S      | `docs/status/archived/2026-10-01_05-51` #1; `flightrecorder.go` still calls `reg.MustRegister` first                                                          |
| Pin the erraudit binary version and document the audit-vs-lint mode protocol                                                                                            | 🔴 `TODO` | quality         | S      | `docs/status/archived/2026-10-01_05-51` #9, #14                                                                                                               |
| Add a one-line cross-reference in `sse_conformance_test.go` to the hand-picked dispatch table in `sse_test.go`, and tag the table rows with the WHATWG section they pin | 🔴 `TODO` | docs            | XS     | `docs/status/archived/2026-10-01_05-46` #7, #37                                                                                                               |
| Document the `DeterministicCodec` limitation on `JSONv2Codec` (closed signing-safe interface) in its doc comment, or raise the upstream ask                             | 🔴 `TODO` | docs            | XS     | `docs/status/archived/2026-10-04_08-20` row 9; `jsonv2.go` documents deterministic map ordering only                                                          |
| Make the `check-docs` strike gate same-day-aware so a brand-new report does not fail a green build                                                                      | 🔴 `TODO` | gate parity     | S      | `docs/status/archived/2026-10-01_09-59` row 38; `scripts/check-docs.sh` has no date rule                                                                      |

## Repository and security operations

| Task                                                                                                                                                           | Status       | Impact    | Effort | Evidence                                                                                                                                                                           |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | --------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Delete the `systemnix-ci-readonly` deploy key once SystemNix's lock no longer fetches go-daemon over `ssh://` (depends on the SystemNix row) (repo: SystemNix) | 🔵 `BLOCKED` | security  | XS     | 2026-11-28: the key (id 168109853) still backs the three `ssh://git@github.com/LarsArtmann/go-daemon` inputs in `SystemNix/flake.lock`; deleting it first would break SystemNix CI |
| Enable CodeQL default setup and the non-provider secret-scanning patterns (owner: the `gh` token lacks the scope; both come back unchanged through the API)    | 🔵 `BLOCKED` | security  | XS     | 2026-11-28: `gh api .../code-scanning/default-setup` answers 404 and the secret-pattern toggle stays `disabled`; secret scanning, push protection and Dependabot alerts are on     |
| Decide on branch/tag protection for `master`: it would also block the auto-commit daemon, which pushes straight to `master` (repo: owner)                      | 🔵 `BLOCKED` | security  | XS     | `docs/status/2026-11-27_16-16` §f20; no protection today (`branches/master/protection` answers 404)                                                                                |
| Float SystemNix's lock so the nested private-scheme go-daemon input disappears (repo: SystemNix)                                                               | 🔴 `TODO`    | coherence | S      | `docs/status/2026-11-27_16-16` §f5; SystemNix floats pdd by branch and its lock still carries the old nested input                                                                 |
| Decide the public store front: repository homepage, an announcement, a docs site (repo: owner)                                                                 | 🔵 `BLOCKED` | adoption  | XS     | `docs/status/2026-11-27_16-16` §g3; README badges landed 2026-11-27, the rest is an owner taste call                                                                               |
| Add `scripts/pre-publish-audit.sh` as a `workflow_dispatch` CI job                                                                                             | 🔴 `TODO`    | security  | S      | `docs/status/archived/2026-10-01_03-52` #11                                                                                                                                        |
| Add gitleaks as a scheduled CI job                                                                                                                             | 🔴 `TODO`    | security  | S      | `docs/status/archived/2026-10-01_03-52` #12                                                                                                                                        |
| Add nix store caching (magic-nix-cache or cachix) to the CI nix job                                                                                            | 🔴 `TODO`    | CI        | S      | `docs/status/archived/2026-10-01_03-52` #15                                                                                                                                        |
| Re-check that Dependabot opens PRs on the now-public repo (none since the 2026-11-27 flip)                                                                     | 🔴 `TODO`    | deps      | XS     | `docs/status/2026-11-27_16-49` #32                                                                                                                                                 |
| Decide the branching-flow adopt/reject question; it gates four archived review rows                                                                            | 🔴 `TODO`    | quality   | S      | `docs/status/archived/2026-10-05_12-28` #3, #6-#8 (repo: owner)                                                                                                                    |
