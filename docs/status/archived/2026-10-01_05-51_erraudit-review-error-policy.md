# Status Report — 2026-10-01 05:51 CEST — erraudit review & stdlib-error policy

> **Live remainder:** §a done; §b/c residuals (erraudit automation, baseline artifact) are routed to ROADMAP theme 3. Live work: TODO_LIST.md.

Scope: this session only (erraudit audit-output review + policy recording +
verification follow-ups). No other research was done; everything below is
grounded in files read this session.

Session input (run by the user, output pasted):

```
erraudit ./... --type-aware --enforce-go-error-family --no-suppress \
  --enforce-samber-oops --enforce-generic-return --explain \
  --disable-extensions --enforce-coded-errors --enforce-deferred-close \
  --enforce-generic-return          # note: passed twice (noticed, see e)
```

Result: 26 violations (20 ERROR / 6 WARNING, 0 CRITICAL), exit code 2
(`[Rejection:violations.found]` — the only go-error-family classification in
the output is the tool's own exit condition, not a code finding). Hygiene
metrics were strong: 0 silent swallows, 0 panics-on-error, max wrap depth 3,
and `errors.Is(err, http.ErrServerClosed)` at socket.go:276 correctly NOT
flagged (type-aware working).

## a) FULLY DONE

1. ~~**Full review of all 26 erraudit findings** — classified into 4 clusters,~~
   ~~each verified against source (flightrecorder.go, socket.go, sse.go,~~
   ~~write.go read at every flagged line). Verdict: **0/26 actionable**.~~ done (session §a; re-verified 2026-11-27)
   - ~~`stdlib_constructor` ×15: flag-policy mismatch — go.mod has no~~
     ~~samber/oops, zero oops usage in repo; `errors.New` sentinels are~~
     ~~exported API; `%w` wraps carry real context (socket path, dir).~~ done (session §a; re-verified 2026-11-27)
   - ~~`ignored` ×4: intentional best-effort cleanup with explicit `_ =`~~
     ~~intent markers (socket.go:244,299,396; write.go:56) — exactly the~~
     ~~class `--no-suppress` is designed to surface.~~ done (session §a; re-verified 2026-11-27)
   - ~~`context_loss` ×1 (flightrecorder.go:267): the "lost" variables are~~
     ~~Prometheus collector objects, not diagnostic context — printing them~~
     ~~would put Go pointers in the message.~~ done (session §a; re-verified 2026-11-27)
   - ~~`generic_return` ×6: pure wraps, no structured fields; concrete types~~
     ~~would be ceremony (two of six are internal helpers).~~ done (session §a; re-verified 2026-11-27)
2. ~~**Dependency fact-check**: go.mod confirms no samber/oops;~~
   ~~go-error-family v0.11.0 is indirect only; toolchain go 1.27.~~ done (session §a; re-verified 2026-11-27)
3. ~~**AGENTS.md Conventions bullet recorded** (stdlib-error policy + the~~
   ~~2026-10-01 dismissal baseline so future audits pre-dismiss these~~
   ~~clusters). Grep-verified on disk; dprint check passes (exit 0).~~
   ~~Manual commit skipped per harness rule — the auto-commit daemon ships it.~~ done (session §a; re-verified 2026-11-27)
4. ~~**Self-caught and corrected an unverified claim** (see d): the original~~
   ~~bullet said the consumer "matches the sentinels"; verification showed it~~
   ~~asserts on `Validate()` error-message substrings~~
   ~~(project-discovery-daemon metrics_test.go:131-145), NOT `errors.Is`.~~
   ~~Bullet corrected in-session; message wording is the actual~~
   ~~consumer-visible contract — which strengthens the anti-oops argument.~~ done (session §a; re-verified 2026-11-27)

## b) PARTIALLY DONE

1. **erraudit integration into repo tooling** — reviewed and policy-recorded,
   but not wired: no pinned audit script, no CI job, no baseline artifact
   beyond the AGENTS.md sentence (see f #4, #5, #8).
2. **Post-review gates for docs** — dprint on AGENTS.md passes, but the
   full `nix run .#check-docs` was not run over the repo state that also
   contains the concurrent sse_test.go work (see note below).
3. **Metric-registration hazard found but not fixed** (discovered while
   reviewing flightrecorder.go): `NewFlightRecorder` registers both metrics
   via `reg.MustRegister` (flightrecorder.go:216) BEFORE
   `flightrecorder.New` (line 265). A failed `New` leaves orphan metrics on
   the caller's registry, and a retry with the same registry panics via
   MustRegister duplicate registration. Fix + regression test NOT started
   (f #1, #21) — flagged, not repaired, because it is a behavior change
   beyond this session's review mandate.

### Concurrent work observed (NOT this session's)

- sse_test.go: auto-commit `c19f1ec` (05:45:14) landed a −164/+68 refactor
  mid-session; +2 uncommitted lines remain. Another AGENTS.md bullet
  (ParseSSEData dispatch-shape test conventions + art-dupl `-t 1` note)
  appeared uncommitted at 05:50:05 from the same concurrent work. This
  session did not author it, did not touch it, and has not gated it.

## c) NOT STARTED

- Metric registration-order fix + regression test (flightrecorder.go:216
  vs :265) — the one real code issue found this session.
- erraudit as a reproducible, pinned invocation (script/CI).
- HARVEST of this report's section f into TODO_LIST.md / ROADMAP.md.
- Upstream erraudit feedback (context_loss false positive on metric
  collectors; generic_return noise on pure-wrap methods) — gated by
  verify-before-filing, nothing prepared.
- Org-wide samber/oops stance decision (question 2 below).
- errors.Join(ln.Close()) option on the chmod cleanup path
  (socket.go:243-247) — deliberately dismissed for now; reopening only on
  a strictness-policy change.

## d) TOTALLY FUCKED UP

Nothing shipped broken. The honest failure-mode entry:

- **I wrote an inference as fact into a durable doc.** "project-discovery-daemon
  matches the sentinels" went into AGENTS.md without a grep behind it; a
  post-hoc verification disproved it (consumer substring-matches messages
  instead). Caught and fixed within the session, and the corrected claim is
  stronger — but the rule is now explicit: AGENTS.md edits get the same
  evidence bar as code. `verify-external-claims` applies to my own bullets,
  not just third-party docs.

## e) WHAT WE SHOULD IMPROVE

1. **Verify before writing durable claims** — the sentinel-matching miss
   above. One `rg` would have taken ten seconds.
2. **Reproduce, don't just review pasted output** — running erraudit in-session
   pins the exact invocation. The pasted command passed
   `--enforce-generic-return` TWICE — harmless here, but it shows hand-typed
   audit invocations drift; a pinned script removes the class of error.
3. **Gate every edit synchronously, including docs** — the auto-commit daemon
   snapshots within ~a minute; my first AGENTS.md edit landed ungated
   (dprint ran after; clean, but by luck of style discipline, not process).
4. **Persist an erraudit baseline** so future audits diff against a recorded
   verdict instead of re-litigating the same 26 findings.
5. **Treat error message wording as API** — the consumer substring-matches
   Validate() messages; any rewording is consumer-breaking. This belongs in
   the release checklist, not just AGENTS.md.

## f) Next tasks (30, impact-sorted; TODO_LIST vs ROADMAP fuel marked)

**Fix now (code correctness)**

1. Register flight-recorder metrics only after `flightrecorder.New` succeeds
   (or `prometheus.Register` + unregister-on-failure) — flightrecorder.go:216
   vs :265. [TODO_LIST]
2. Add regression test: failed `NewFlightRecorder` leaves the registry clean;
   retry with the same registry does not panic. [TODO_LIST]
3. Gate the in-flight sse_test.go refactor (owner-dependent): `go test ./...
   -race` + golangci-lint once committed. [TODO_LIST, coordinate first]
4. Confirm oversize-line error contract has test coverage:
   `errors.Is(err, bufio.ErrTooLong)` for sse.go:171. [TODO_LIST]
5. Audit consumer call sites of `SnapshotToFile` for empty-path-on-error
   handling (observed signature this session; usage not audited). [TODO_LIST]

**Wire up (tooling)**
6. Add `scripts/erraudit-audit.sh` pinning the exact audit invocation (each
flag exactly once) and reference it from AGENTS.md Essential Commands.
[TODO_LIST]
7. Add erraudit normal-mode gate (`lint ./... --type-aware`) to CI per the
go-error-modernization recipe — pending question 1. [TODO_LIST]
8. Record the 2026-10-01 erraudit baseline (26 findings → 0 actionable;
hygiene 0/0/0, depth 3) in FEATURES.md quality notes for future diffing.
[TODO_LIST]
9. Pin the erraudit binary version in tooling (source of the binary is
unverified this session). [TODO_LIST]
10. Add message-wording stability to the release checklist (consumer
substring-matches Validate() errors). [TODO_LIST]
11. Investigate erraudit project-config support (per-cluster ignore rules) so
audit mode needs fewer blanket flags; capability unverified. [ROADMAP]
12. Add erraudit to the BuildFlow toolchain org-wide — pending question 1.
[ROADMAP]

**Document**
13. HARVEST this section into TODO_LIST.md / ROADMAP.md (docs-health HARVEST
mode; most ROADMAP-marked items go there, not to TODO_LIST).
[TODO_LIST, do first]
14. Document the audit-vs-lint mode protocol (who runs which, when, and who
may dismiss). [TODO_LIST]
15. Point README/FEATURES at the error-handling contract if user-facing docs
mention errors (unverified whether they do). [TODO_LIST]
16. Re-run the erraudit audit after the next Go toolchain bump and compare
against the 2026-10-01 baseline (date recorded in AGENTS.md).
[TODO_LIST, recurring]

**Decide (needs owner input — see section g)**
17. Org-wide samber/oops stance: permanent stdlib-errors for mechanism
libraries, or transitional before oops adoption? [ROADMAP, Q2]
18. CI gating policy for erraudit here (enforced vs advisory vs manual).
[ROADMAP, Q1]
19. go-error-family direct adoption for go-daemon (mechanism library may
legitimately stay family-free; the only Rejection classification seen is
the tool's own exit condition). [ROADMAP]

**Polish (noticed in passing, low priority)**
20. errors.Join(ln.Close()) on socket.go:243-247 if strictness policy ever
changes (currently a documented dismissal). [ROADMAP]
21. WriteEncoded control flow (write.go:41-56): early-return on encode error
instead of buf.Write guarded by `err == nil`. [ROADMAP]
22. isSocketAlive diagnosability (socket.go:383-399): bool-only; debug-log the
failure reason. [ROADMAP]
23. socket.go Start() `//nolint:cyclop` — fine today; extract per-stage
functions if lifecycle grows. [ROADMAP]
24. Consider extracting the metric-set (snapshotsTotal + snapshotDuration)
into its own type to make the #1 fix structural. [ROADMAP]
25. Upstream erraudit: context_loss false-positive on Prometheus collectors
(minimal repro: flightrecorder.go:265-267) — verify-before-filing first.
[ROADMAP]
26. Upstream erraudit: generic_return noise on pure-wrap methods — same gate.
[ROADMAP]
27. Ensure test for `NewFlightRecorder` failure path exists alongside #2.
[TODO_LIST, with #2]
28. Sanity-check placement of the new AGENTS.md bullets in the next
docs-health pass (Conventions vs Gotchas). [TODO_LIST]
29. Re-run `nix run .#check-docs` over the merged state of this report +
concurrent sse_test.go work once committed. [TODO_LIST]
30. Confirm the foreign AGENTS.md bullet (ParseSSEData dispatch-shape) and
this session's bullet don't overlap/contradict in the next docs pass.
[TODO_LIST]

## g) Questions I cannot answer myself

1. **CI gating**: should erraudit become an enforced CI gate in go-daemon
   (normal `--type-aware` mode), an advisory annotation, or an org-level
   BuildFlow step across all repos? This decides items 7 and 12.
2. **oops stance**: is stdlib-errors the _permanent_ stance for mechanism
   libraries, or is samber/oops the intended org direction that go-daemon
   should plan a cross-repo migration toward (consumer + this repo
   together)? This decides whether the AGENTS.md policy line is permanent
   or transitional, and item 17.
3. **sse_test.go refactor**: the −164/+68 sse_test.go change (auto-committed
   05:45:14) plus the uncommitted +2 and the new ParseSSEData AGENTS.md
   bullet — yours or another agent's? May I gate it (`go test -race`) or is
   it hands-off while in flight?

---

_Point-in-time snapshot; do not edit in place — annotate via docs-health
ANNOTATE mode if brought current. Section f is HARVEST input._
