# Decision — repository visibility flip

**Date:** 2026-11-27
**Status:** executed
**Supersedes:** the open gates in `2026-10-01_owner-decision-memo.md`

## Context

The library was created private on 2026-09-28 while being extracted from
the production daemon that still consumes it. It stayed private to finish
a pre-flip hardening cluster (history secret scan, license closure,
security policy, issue templates, README polish, a written post-flip
runbook). That cluster finished on 2026-10-01 and the memo closed with
"the flip is one word away"; nobody said the word for eight weeks.

Three facts forced the decision:

1. **CI never once ran.** Every workflow run on the private repo ended in
   about three seconds with zero steps and no log — GitHub Actions
   billing, not code. The v0.5.0 tag push included. Public repositories
   run Actions on the free tier, so visibility was the switch that turns
   the gates on.
2. **The only consumer paid a private-dependency tax.** Its flake pinned
   this repo by rev through a private transport, wrapped it in a
   source-preparation helper, kept a dedicated CI deploy key, and had to
   move that pin in lockstep with `go.mod` — a documented drift hazard
   that had already caused one broken build.
3. **The sibling libraries are already public.** `go-codec`,
   `go-flightrecorder`, and `go-sse` are public and resolve through the
   module proxy; this module depends only on public modules, so it was
   the odd private one out.

## Decisions

| Gate            | Decision                                              | Reason                                                                                                                                                         |
| --------------- | ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Visibility      | **Public**                                            | The two structural costs above only exist while private. Trust scans were clean and had been for eight weeks.                                                  |
| API stability   | **Stay v0.x, no compatibility promise**               | One in-family consumer; a v1 freeze buys an obligation worth less than the freedom to fix the lifecycle contract while adoption is still small.                |
| Name            | **Keep `go-daemon`**                                  | `sevlyar/go-daemon` owns the search phrase either way; no import-path collision exists, and renaming after publication is a v2 migration for every consumer.   |
| Snapshot docs   | **Keep `docs/status/`, `docs/planning/`, `reports/`** | Honest process records with no secrets. Rewriting history would invalidate the very scans the flip is based on, and old commits keep their content regardless. |
| Actions billing | **Moot**                                              | Public repositories run the workflows for free; the first public run executed both jobs for real.                                                              |

## What the flip changed

- `README.md` — no longer links to a private repository (that link would
  404 in public), and states the patched toolchain requirement.
- `flake.nix` — the private-module environment variable and the vendored
  source notes are gone; every dependency resolves through the proxy.
- `.github/workflows/ci.yml` — Go pinned to 1.27.4 (1.27.3 and earlier
  carry CVE-2026-97054), and the flake gate is now `nix build .#default`
  followed by `nix flake check --no-build --all-systems`, because
  `--all-systems` without `--no-build` tries to build other platforms'
  checks and fails with a platform mismatch (measured in the first public
  run).
- `SECURITY.md` — supported-version table moved to 0.5.x, the
  private-repository reporting channel removed in favour of GitHub's
  private vulnerability reporting (now enabled on the repository).
- Test fixtures — absolute home paths replaced with neutral ones so the
  pre-publish scan can be green.
- `docs/runbooks/post-flip.md` — retired after execution.
- Consumer repo — the private flake input, the prepare-source wrapping,
  and the go-daemon deploy key are gone; the module now comes from
  `go.mod` through the proxy and the vendor hash is re-derived.

## Verification

| Check                                                      | Result                                                           |
| ---------------------------------------------------------- | ---------------------------------------------------------------- |
| Visibility                                                 | `gh repo view` → `PUBLIC`                                        |
| Proxy resolution without private transport                 | `go list -m github.com/LarsArtmann/go-daemon@v0.5.0` → v0.5.0    |
| pkg.go.dev                                                 | 200; renders `SocketServer` and `ParseSSEData`                   |
| Private vulnerability reporting                            | enabled                                                          |
| Secret scanning + push protection                          | enabled (free on public repositories; unavailable while private) |
| `scripts/pre-publish-audit.sh`                             | exit 0, CLEAN                                                    |
| Local gates (build, vet, race tests, lint, README snippet) | green with Go 1.27.4; `govulncheck` clean                        |
| Hermetic builds, both repos                                | green after re-deriving the consumer's vendor hash               |
| CI on the public repository                                | run 46022249899 — both jobs green                                |

## Consequences

- The repository is now a normal public Go module: proxy-resolvable,
  indexable, and free to CI.
- **Toolchain floor for security:** the `go` directive stays at 1.27.2
  (raising it would break consumers whose toolchain is 1.27.3), so the
  CVE-2026-97054 requirement lives in the README, the security policy,
  and the CI pin instead of the module contract.
- nixpkgs still ships Go 1.27.3, so the devShell's vulnerability scan
  reports that standard-library finding until the flake's Go moves to
  1.27.4 — tracked in the TODO list.
- Five historical status reports still lack strikethrough annotations and
  keep the docs-health gate red; that is a separate harvest task, not a
  side effect of publishing.
