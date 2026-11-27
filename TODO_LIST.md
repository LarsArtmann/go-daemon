# TODO List

> Short-term, actionable, bounded work items, verified against the actual
> code. Completed work lives in [CHANGELOG.md](CHANGELOG.md); long-term ideas
> live in [ROADMAP.md](ROADMAP.md). Items are ranked by impact; status is
> verified, not assumed.
>
> **Last harvested 2026-11-27** from
> `docs/status/2026-11-27_15-32_v050-release-five-pr-merge-and-jsonv2-flavour-audit.md`
> and `docs/status/2026-11-27_16-16_visibility-flip-ci-revival-and-consumer-repin.md`
> (both §f), plus the 13 open items the `docs-gates freshness --audit-all`
> replay found across the reports archived the same day. Every surviving row
> was re-checked against code before it landed; already-done items were
> dropped, not carried. Cross-repo items carry a `(repo: …)` tag and route
> through [ROADMAP.md](ROADMAP.md) theme 3.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | External dependency or decision needed before it moves. |

## Release follow-ups

| Task                                                                                                                                                       | Status    | Impact        | Effort | Evidence                                                                                                                                                                                                             |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ------------- | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Bump `nextd` to go-daemon v0.5.0 and raise its `go` floor to 1.27.2 (v0.5.0 requires it via go-codec v0.3.3)                                               | 🔴 `TODO` | fleet         | S      | v0.5.0 cut 2026-11-27; `nextd/go.mod` still pins go-daemon v0.4.0 + `go 1.27.1` (checked 2026-11-27). The v0.4.1 re-tag plan is dropped: a `go 1.27` floor is impossible while go-codec declares `go 1.27.2`         |
| Bump `project-discovery-daemon` to v0.5.0: audit its DTOs for `time.Duration` and nil-slice JSON fields, re-derive `vendorHash`, run its suite (repo: pdd) | 🔴 `TODO` | consumer      | M      | `docs/decisions/2026-11-27_jsonv2-default-and-wire-shapes.md`; `CHANGELOG.md` §0.5.0 — the default codec now errors on `time.Duration` and emits `[]` for nil slices                                                 |
| Add a tag-triggered `.github/workflows/release.yml` so release gates run on every tag, not only when a session remembers them                              | 🔴 `TODO` | release trust | S      | `docs/status/2026-11-27_15-32` §f36; the v0.5.0 tag was pushed with CI suspended and the release notes' verification relies on local runs                                                                            |
| Move the flake's Go to 1.27.4 once nixpkgs ships it                                                                                                        | 🔴 `TODO` | security      | XS     | CVE-2026-97054 / GO-2026-6859 is fixed in 1.27.4; nixpkgs still ships `go_1_27 = 1.27.3` (checked 2026-11-27, nixos-unstable too), so the devShell's govulncheck reports it. CI already pins 1.27.4 and passes clean |

## Consumer adoption (DiscordSync)

| Task                                                                                                                                         | Status    | Impact                 | Effort | Evidence                                                                                                                                                                                            |
| -------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ---------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Deferrable READY: an option so `Server.Start` skips its built-in `NotifyReady` and the consumer gates it (e.g. on a health-check round trip) | 🔴 `TODO` | consumer adoption (F3) | S      | `docs/feedback/new/2026-10-04_discordsync-consumer-feedback.md` F3 — the lifecycle already notifies after `listen(2)` (`socket.go:291`), so the ask is a suppress/defer option, not moving the call |

## Documentation and gates

| Task                                                                                                                                                                | Status    | Impact          | Effort | Evidence                                                                                                                                                                                         |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | --------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Run the `docs-gates run .` battery in CI as well (freshness, annotate, routing, cadence, glossary) — `nix run .#check-docs` now runs in the nix job                 | 🔴 `TODO` | gate health     | S      | `ci.yml` gained the check-docs step 2026-11-27; the fuller battery needs `docs-gates` available to CI (flake input or pinned `nix run`), the gap that let five unannotated reports rot invisibly |
| Add one local gate target that is exactly the CI gate set (build, vet, race, coverage, README snippet, actionlint, govulncheck, lint, nix build, flake check, docs) | 🔴 `TODO` | release trust   | M      | Two consecutive release reports record red pushes from gates the session forgot to run (`docs/status/2026-11-27_15-32` §f15-16, `…16-16` §e1); no single command exists today                    |
| Add `govulncheck` and `actionlint` as flake apps so local and CI invocations cannot drift                                                                           | 🔴 `TODO` | gate parity     | S      | Both tools are in the devShell but neither is a flake app; CI installs its own pins (`ci.yml`), which is the drift surface                                                                       |
| Add a nightly `-count=15` timing run for the socket/notify tests, and `-shuffle=on` to CI                                                                           | 🔴 `TODO` | flake detection | S      | `docs/status/2026-11-27_15-32` §f27-28; two timing fixes in PR 5 hint at order dependencies                                                                                                      |
| Pin the default codec's wire shapes with a named test (nil slice `[]`, `time.Duration` error, unordered map keys)                                                   | 🔴 `TODO` | wire contract   | S      | `docs/decisions/2026-11-27_jsonv2-default-and-wire-shapes.md`; today only the explicit `JSONv2Codec` adapter is pinned                                                                           |
| Harden `WriteEncoded` and `ContentTypeFor` against a nil codec (guard or documented contract) and test both                                                         | 🔴 `TODO` | latent panic    | S      | `write.go:29`; `WriteEncoded(w, 200, nil, v)` panics on the nil interface (`docs/status/archived/2026-10-05_12-28_branching-flow-panic-finding-review.md` §f4-5)                                 |
| Extend `FuzzResolveResponseCodec` with an injected-codec pass (ranking/refusal invariants under substitution)                                                       | 🔴 `TODO` | coverage        | S      | `negotiation_contract_test.go` fuzzes default codecs only; injection is table-tested but not fuzzed                                                                                              |
| Add `scripts/consumer-smoke.sh` that runs `go get` against the newest tag in a scratch module, so a release is verified as a consumer would use it                  | 🔴 `TODO` | release trust   | S      | `docs/status/2026-11-27_16-49_docs-health-audit-corpus-archive-and-living-doc-rebuild.md` §f33; the v0.5.0 verification was a manual scratch module in `/tmp`                                    |
| Pin the erraudit known-dismissed baseline in a durable artifact so audits diff against it instead of re-litigating the same findings                                | 🔴 `TODO` | quality         | XS     | `docs/status/2026-11-27_16-49_docs-health-audit-corpus-archive-and-living-doc-rebuild.md` §f40; the current baseline lives only in `AGENTS.md` prose                                             |
| Add godoc examples for `JSONv2Codec` and `WriteJSONWithCodec`/`WriteErrorfWithCodec`                                                                                | 🔴 `TODO` | adoption        | XS     | Repo convention is `*_example_test.go` for every flagship surface (`negotiation_example_test.go`); the v0.5.0 additions have none                                                                |
| Test `JSONv2Codec` with nil elements inside the marshal/unmarshal option slices                                                                                     | 🔴 `TODO` | edge            | XS     | Undefined behaviour today (`docs/status/archived/2026-10-04_08-20_f2-jsonv2-write-path-injection.md` §f12); the doc comment does not constrain it                                                |

## Repository and security operations

| Task                                                                                                                              | Status       | Impact    | Effort | Evidence                                                                                                                                                  |
| --------------------------------------------------------------------------------------------------------------------------------- | ------------ | --------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Delete the now-pointless `systemnix-ci-readonly` deploy key; verify and delete any stale consumer deploy-key secret (repo: owner) | 🔴 `TODO`    | security  | XS     | `docs/status/2026-11-27_16-16` §f6-7: the consumer resolves through the proxy now, and `gh secret list` returned empty (unverified, not confirmed absent) |
| Enable CodeQL default setup, branch/tag protection on `master`, and the remaining non-provider secret-scanning patterns           | 🔴 `TODO`    | security  | S      | `docs/status/2026-11-27_16-16` §f20-21,25; secret scanning and push protection are already on, CodeQL and branch protection are not                       |
| Float SystemNix's lock so the nested private-scheme go-daemon input disappears (repo: SystemNix)                                  | 🔴 `TODO`    | coherence | S      | `docs/status/2026-11-27_16-16` §f5; SystemNix floats pdd by branch and its lock still carries the old nested input                                        |
| Decide the public store front: repository homepage, an announcement, a docs site (repo: owner)                                    | 🔵 `BLOCKED` | adoption  | XS     | `docs/status/2026-11-27_16-16` §g3; README badges landed 2026-11-27, the rest is an owner taste call                                                      |
