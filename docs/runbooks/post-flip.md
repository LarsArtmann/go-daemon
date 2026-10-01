# Post-flip runbook — execute AFTER the owner flips visibility

**Precondition:** `github.com/LarsArtmann/go-daemon` is public. Nothing in
this file may run before that. Work top to bottom; every step ends in a
verifiable check. Total: ~30 minutes.

## 1. Verify the flip took effect

```bash
gh repo view LarsArtmann/go-daemon --json visibility --jq .visibility   # PUBLIC
```

## 2. Verify public resolution without SSH

From a directory OUTSIDE any GOPRIVATE-influenced env:

```bash
GONOSUMDB= GOPRIVATE= GONOSUMCHECK= go list -m github.com/LarsArtmann/go-daemon@v0.2.0
```

Expected: `github.com/LarsArtmann/go-daemon v0.2.0` resolved from the proxy
(no git/SSH fallback, no credentials prompt).

## 3. pkg.go.dev storefront check (closes the 🔵 row)

Fetch `https://pkg.go.dev/github.com/LarsArtmann/go-daemon` — it should
render within minutes of the flip (proxy pickup). Verify: README rendered,
godoc shows `SocketServer`, `ParseSSEData`, and the Example functions.
Close the "Verify pkg.go.dev rendering" row with the verdict.

## 4. Enable GitHub features that need a public repo

- Settings → Code security: enable private vulnerability reporting
  (Security → Report a vulnerability); it 404'd while the repo was private.
- Update SECURITY.md: delete the "while this repository is private"
  bullet, keep the private-vulnerability-reporting instruction.
- Optionally enable Discussions if issues alone get noisy.

## 5. This repo's AGENTS.md gotcha cleanup

Edit AGENTS.md:

- Delete the "This repo is PRIVATE" gotcha bullet (GOPRIVATE,
  `git+ssh` pin, `mkPreparedSource` notes) — obsolete the moment the repo
  is public.
- Update the Dependencies section note if it mentions private resolution.
- Commit: `docs: retire private-repo gotchas after the visibility flip`.

Then run `scripts/pre-publish-audit.sh` — the private-path scan (gate 2)
must now exit 0. If anything still matches `GOPRIVATE|git\+ssh`, it is a
leftover; fix it before continuing.

## 6. project-discovery-daemon re-pin (cross-repo; do it FROM pdd)

In `/home/lars/projects/project-discovery-daemon`:

1. `go.mod`: no change needed (module path is unchanged) — but drop the
   `GOPRIVATE=github.com/larsartmann/*,github.com/LarsArtmann/*` from any
   env shebangs / flake devShells / CI.
2. `flake.nix` (~line 36-38): replace the `git+ssh` go-daemon input

   ```nix
   # before
   go-daemon.url = "git+ssh://git@github.com/LarsArtmann/go-daemon?rev=c0aaf4c...";
   # after
   go-daemon.url = "github:LarsArtmann/go-daemon/<new-rev-or-tag>";
   ```

   and remove `mkPreparedSource` wrapping for it if that helper exists
   only for private inputs (keep it if other inputs still need it).
3. Re-derive the vendorHash: run `nix build .#<pdd-package>`; on hash
   mismatch run `buildflow -s nix-hash-fix --fix` (never hand-paste `got:`).
4. Commit pdd: `chore: pin go-daemon as a public flake input`.

## 7. Announcement decision (owner's call, not mechanical)

Options, in ascending effort: none (proxy + pkg.go.dev is the store front)
→ a line in the pdd README ("powered by go-daemon") → a docs website via
the sibling-project website pattern. Decide later; nothing in this runbook
depends on it.

## 8. Flip checklist (tick all)

- [ ] visibility = PUBLIC
- [ ] proxy-only resolution works without SSH
- [ ] pkg.go.dev renders; row closed
- [ ] private vulnerability reporting enabled; SECURITY.md updated
- [ ] AGENTS.md private-repo gotchas deleted
- [ ] `scripts/pre-publish-audit.sh` exits 0
- [ ] pdd re-pinned plain, vendorHash re-derived, pdd `nix build` green
- [ ] CI green on first post-flip push (also validates billing question 4)
