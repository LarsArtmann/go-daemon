# Owner decision memo — the five flip gates

**Date:** 2026-10-01
**Status:** awaiting owner answers; every technical prerequisite is done
**Companion:** `docs/runbooks/post-flip.md` (pre-written mechanical tail)

Everything below the fold is data gathered this session so each decision is
a one-sitting answer. Nothing here is code work; the five gates are the
owner's.

## 1. API stability policy (TODO: v0.x no-promises vs v1 freeze)

**Data:** v0.2.0 is tagged; the only untagged API surface is `Vary: Accept`
on `WriteEncoded` (in CHANGELOG Unreleased, demand-gated for v0.3.0).
Consumers today: project-discovery-daemon (floor-bumped, rev-pinned).
Coverage 92%; negotiation contract is table-pinned; lifecycle ordering is
test-enforced.

**Recommendation:** stay v0.x no-promises. The library's one consumer is
in-repo-family; a v1 freeze buys an obligation (compatibility promise) that
outweighs discoverability gains until a stranger depends on it. Revisit
when a second external consumer exists.

## 2. Name (TODO: keep `go-daemon` vs rename pre-flip)

**Data (2026-10-01 collision check):** `github.com/sevlyar/go-daemon` — the
dominant library for this exact phrase — is MIT, 2,309 stars, actively
maintained (pushed 2026-07-19), package name `daemon`. pkg.go.dev search
shows 25+ modules named `go-daemon`; most have ≤12 importers, sevlyar has
619. No hard collision: `github.com/LarsArtmann/go-daemon` is unclaimed and
Go import paths are fully qualified, so nothing breaks technically.

**Cost of keeping:** search results are dominated by sevlyar; this repo
will never rank first for "go-daemon". Soft collision on the *package
name* `daemon` — a consumer importing both libraries must alias one.
**Cost of renaming:** module path change post-flip = a v2 migration for
every consumer; pdd re-pins; docs/README/memo paths change.

**Recommendation:** keep `go-daemon`. The audience for this library is
people who already know it exists (pdd's README, this owner's other
repos), not search traffic; and rename-after-flip is the single most
expensive mistake available here. If search differentiation ever matters,
differentiate the *package* name or the description, not the module path.

## 3. Docs visibility (TODO: scrub vs keep `docs/status/`, `docs/planning/`, `reports/`)

**Data:** `docs/status/` (6 annotated reports), `docs/planning/` (1 active
plan + archived/), and `reports/` contain session narratives: internal repo
names (pdd, buildflow), tooling opinions, owner-facing questions. No
secrets (full-history scan clean 2026-10-01), but real opinionated
narrative. Scrubbing = rewriting history (filter-repo) because the scans
are in past commits; keeping = zero work, some transparency cost.

**Recommendation:** keep, don't scrub. The narratives are honest
engineering process records; nothing in them is damaging; the alternative
(history rewrite) invalidates the trust scans by manufacturing new history.
If any single file feels too personal, deleting it going forward is enough
— old commits stay, and that is acceptable.

## 4. GitHub Actions billing (TODO: fix payments; CI never ran on v0.2.0)

**Data:** run 36645862196 failed with "payments failed"; the v0.2.0 tag
commit and the never-executed checkout/setup-go v7/v9.3 bumps are untested
by CI. As of 2026-10-01 ci.yml carries two NEW gates (check-readme, nix
job) that are config-ready but unverifiable until billing works. Private
repo minutes are the paid resource; after the flip, minutes are free.

**Recommendation:** fix billing (or flip visibility first — public repos
run free). Order of operations: if the flip happens first, CI validates
the new gates for free on the first push; billing fix becomes optional
unless private-master CI is wanted in the interim.

## 5. The visibility flip itself (TODO: after the pre-flip cluster is green)

**Data — pre-flip cluster status 2026-10-01:**
- History secret scan: CLEAN (gitleaks 53 commits + manual pass).
- License inventory: all-permissive build closure, NOTICE decision
  documented (`docs/dependency-licenses.md`).
- Name collision: data above; no hard collision.
- SECURITY.md + issue templates + description/topics: DONE.
- CI: check-readme + nix gates wired, actionlint-clean, no secret
  assumptions.
- `nix flake check --all-systems`: PASS (x86_64-darwin filtered, upstream
  dropped it).
- README first-impression pass + Non-goals: DONE.
- `scripts/pre-publish-audit.sh`: exists; exits 1 on GOPRIVATE refs until
  the post-flip runbook executes (expected, documented).
- Remaining flip-tail: `docs/runbooks/post-flip.md` (~30 min mechanical).

**Recommendation:** flip is one word away. Do 3 (keep docs), skim 1-2,
then flip; billing (4) becomes optional afterwards.

## How to answer

Reply per number (e.g. "1 keep-v0x, 2 keep-name, 3 keep-docs, 4 skip, 5
flip"). On "5 flip": execute `docs/runbooks/post-flip.md` top to bottom.
