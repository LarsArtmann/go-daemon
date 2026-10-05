# Session Status — F2 (json/v2 write path + injectable negotiation codecs)

**Date:** 2026-10-04 08:20 (session ran ~07:45–08:20)
**Scope of this report:** this session only — the DiscordSync feedback review
and the F2 implementation (`c9bcc7d`). No new research beyond what the session
touched. Format is `.md` at the operator's explicit path demand (skill default
is HTML; override noted in the closing message).

**Session summary in one line:** reviewed DiscordSync's rejection feedback,
found F3's premise half-wrong, implemented F2 end-to-end (new codec adapter +
negotiation injection, additive and default-preserving), all gates green, one
descriptive squashed commit — with a real integration gap left open (see b1)
and several smaller misses owned below.

---

## a) FULLY DONE

1. **Feedback review** (`docs/feedback/new/2026-10-04_discordsync-consumer-feedback.md`):
   verdict per item against code. F1 confirmed as the true blocker; F2 verified
   accurate (`write.go` → go-codec, negotiation hardcodes built-ins); F3's
   premise corrected — `NotifyReady` fires after `listen(2)` (socket.go:284),
   and `connect()` succeeds once listen completes, so DiscordSync's dial loop
   gates the same point we already gate on. The real F3 ask is a
   suppress/defer option.
2. **F2 implementation, part 1 — `JSONv2Codec`** (`jsonv2.go`): `codec.Codec` +
   `codec.BufferEncoder` adapter over stdlib `encoding/json/v2` with
   caller-chosen `MarshalOptions`/`UnmarshalOptions` (functional-options GA
   API). Doc comment enumerates every observable v1/v2 divergence: duration
   SemanticError without `FormatDurationAsNano(true)`, non-deterministic maps
   without `Deterministic(true)`, nil map/slice as `{}`/`[]` without
   `FormatNil*AsNull(true)`, case-sensitive decode without
   `MatchCaseInsensitiveNames(true)`.
3. **F2 implementation, part 2 — injection** (`negotiation.go`):
   `NegotiationOption`, `WithJSONCodec`, `WithCBORCodec`, `resolveCodecs`;
   all three resolvers (`ResolveRequestCodec`, `ResolveResponseCodec`,
   `CodecForContentType`) now take additive variadic options. `substituteForRefused`
   refactored to take the refused encoding directly. Source-compatible; defaults
   byte-identical (pdd's `content_negotiation.go:15-19` calls compile unchanged).
4. **Key discovery, verified empirically:** go-codec v0.3.1 compiles its **v1
   flavor on Go 1.27** — its json/v2 backend is gated behind
   `goexperiment.jsonv2`, and that build tag is NOT satisfied on 1.27 (experiment
   removed at GA). Probed in-repo via the v2-only `signingSafe()` marker:
   absent. So `codec.JSONCodec{}` here is `encoding/json` v1 — DiscordSync's F2
   criticism was fully accurate. Recorded as an AGENTS.md gotcha.
5. **Tests (6 new functions, all green):** `TestJSONv2Codec_Encode` (6 rows:
   omitzero, nil-slice `[]` vs `null`, deterministic map sort, duration nanos,
   duration error), `TestJSONv2Codec_EncodeToBuffer` (buffer == Marshal),
   `TestJSONv2Codec_Decode` (case sensitivity default + option),
   `TestJSONv2Codec_DurationRoundTrip`, `TestJSONv2Codec_NegotiatedWrite`
   (end-to-end: inject → negotiate → WriteEncoded, asserts a byte shape v1
   cannot produce, plus Content-Type and Vary), and the injected-codec contract
   tables `TestResolveResponseCodec_InjectedCodecs` (9 rows: ranking arms,
   refusal substitution, echo arms, nil-codec guard, nil-option guard) and
   `TestCodecForContentType_InjectedCodecs` (4 rows, both Content-Type
   resolvers).
6. **Docs:** FEATURES +2 rows (counts 19 FULLY_FUNCTIONAL / 20 rows, count gate
   green), CHANGELOG Unreleased (2 entries), README API table +1 row,
   TODO_LIST new "Consumer feedback (DiscordSync)" section with the reframed
   F3 row, AGENTS.md gotcha (go-codec build flavor + probe technique +
   "don't assume a bump flips it").
7. **Gate hygiene:** the feedback doc itself was dprint-unformatted (broke the
   docs gate) — formatted; realigned, no struck rows involved.
8. **Gates all green:** `go build`, `go vet`, `golangci-lint run` (0 issues),
   `go test -race -count=1`, `nix build .#default` (vendorHash intact — stdlib
   import only, no go.mod change), `check-readme` compiles, `check-docs` passes
   everything except the three pre-existing unharvested 2026-10-01 reports.
9. **History discipline:** the auto-commit daemon swept the work into 4
   heuristic commits; squashed locally into one descriptive commit `c9bcc7d`
   (soft reset, nothing pushed) per the release-race playbook.

## b) PARTIALLY DONE

1. **F2's "write path" is only the NEGOTIATED write path.** `WriteJSON` and
   `WriteErrorf` (write.go:60-70) remain hardwired to `codec.JSONCodec{}`.
   A consumer who injects `JSONv2Codec` into negotiation gets v2 bytes on the
   happy path but **v1 bytes on error responses** (`WriteErrorf` →
   `WriteJSON` → built-in codec). Mixed-engine responses. Not a regression
   (defaults unchanged) and arguably out of the literal F2 ask, but it is the
   biggest honest gap against the feedback's spirit. Fix needs an API decision
   (see f1 and g2).
2. **Injection is documented-but-not-enforced on `Encoding()`**: a consumer
   passing `WithJSONCodec` a codec reporting `EncodingCBOR` gets incoherent
   behavior (wrong Content-Type, wrong ranking arm). Only the doc comment
   constrains it. No guard, no test pinning the (undefined) behavior.
3. **Fuzz target not extended:** `FuzzResolveResponseCodec` still fuzzes only
   default codecs; the injected dimension (ranking/refusal invariants under
   substitution) is table-tested but not fuzzed.
4. **pdd source-compat asserted, not executed:** read pdd's
   `content_negotiation.go` call sites and concluded source-compatibility;
   never built pdd against the new commit (replace-directive build). High
   confidence, zero empirical proof.
5. **Godoc examples missing:** repo has a strong `*_example_test.go` convention
   (`notify_example_test.go`, `negotiation_example_test.go`, SSE examples);
   `JSONv2Codec` / `WithJSONCodec` got none. For a library, examples ARE
   adoption documentation.

## c) NOT STARTED

1. **F1 (go public)** — owner-blocked; already tracked in the Go-public cluster.
2. **F3 (deferrable READY)** — TODO row added (reframed correctly); not
   implemented.
3. **Watchdog flake fix** — `TestWatchdog_SendsWatchdogNotifications`
   (notify_test.go:204) flakes ~1/8 under `-race` + load (20ms ping budget).
   Diagnosed this session, mentioned in chat, **written nowhere durable** —
   no TODO row exists (my miss, see e4).
4. **Stale LSP diagnostics cleanup** — `golangci_lint_ls` still shows phantom
   golines/gci warnings on `jsonv2.go` after a clean CLI gate and `golangci-lint
   fmt`; the documented remedy (`lsp_restart`) was never run.
5. **Feedback-doc annotation** — F2's row still reads as an open ask; the
   docs-health ANNOTATE pass (strike cells individually, never whole rows) to
   mark it addressed by `c9bcc7d` was left for HARVEST.
6. **erraudit pass over new code** — repo policy runs erraudit; not run
   explicitly over `jsonv2.go` (golangci-lint is green, erraudit is separate).
7. **Upstream go-codec report** — the dead build-tag (v2 flavor unreachable on
   Go 1.27) is an upstream bug worth filing (verify-before-filing first).

## d) TOTALLY FUCKED UP

Nothing shipped broken — every gate is green and the commit is clean. Two
process errors to own:

1. **Squashed before reading all gate outputs.** The descriptive commit was
   made while `check-docs`, `check-readme`, and `nix build` were still running
   in the background. They all passed afterwards (working tree identical to
   what was committed, so the risk was reputational, not real), but the
   ordering violated my own gate-then-commit discipline.
2. **First test failure was masked by `tail -3`.** The run right after the
   negotiation edit FAILED and I never saw which test — the tail swallowed
   the name. I re-ran, saw green, then chased flakiness and only later caught
   `TestWatchdog` by looping with `-v`. If that first failure had been my
   code, the gate would have waved it through on a lucky re-run. Gates must
   print failures, not their last three lines.

(Honorable mention: three lint round-trips on `jsonv2.go` nolint placement —
the directive must sit on the line the linter reports, which the formatter's
line-splitting had moved. Caught by gates, cost two cycles.)

## e) WHAT WE SHOULD IMPROVE (self-review answers)

**What did I forget?**

- The error-path gap (b1) — I designed injection around the resolvers and never
  re-checked the whole write surface (`WriteErrorf`) against the new capability.
- Durable registration of the watchdog flake (noticed → chat → nowhere).
- Example tests against an established repo convention.
- `lsp_restart` after diagnosing stale-LSP output as stale.

**What could I have done better?**

- Gate output discipline (d2): always grep `--- FAIL`, never `tail`.
- Commit ordering (d1): read every background gate before writing history.
- Prove consumer compat empirically (b4): a 30-second pdd replace-directive
  build beats reading call sites.
- The `/tmp/codecprobe` scratch module from the flavor investigation is still
  sitting in /tmp — trivial, but tidy as you go.

**What could I still improve?**

- Close b1 (error-path codec) properly — it's the one item a consumer would
  actually hit.
- Enforcement over documentation for the `Encoding()` constraint (b2): either
  a guard in the option or a pinned test.
- Upstream engagement (c7): the go-codec build-tag finding helps every
  consumer of that library, not just us.

**Did I lie to you?** No. One imprecision owned: I reported `DOCS_EXIT=0`
from a pipeline where `$?` measured `grep`, not `nix` — the real check-docs
exit is non-zero due to the three pre-existing unharvested reports (called out
as pre-existing at the time, but the exit-code reporting was sloppy).

**Ghost systems?** None created — `JSONv2Codec` is wired (public API, tests,
docs, README). Inverse risk noted: if go-codec ever fixes its build tag, TWO
v2 paths exist (their `JSONCodec` v2 flavor vs our `JSONv2Codec`) — the AGENTS
gotcha warns about this explicitly.

**Split brains?** One latent: `JSONv2Codec` cannot implement go-codec's
`DeterministicCodec` (unexported `signingSafe()` marker — closed interface),
so "deterministic JSON for signing" and "json/v2 with options" are two
incompatible doors. Unresolved, undocumented as a limitation.

**Scope creep?** No — F2 only, plus the F3 TODO row I had recommended in the
same conversation. Docs edits stayed proportionate.

**Tests?** Strong on behavior tables and the end-to-end path; weak on fuzz
(injection dimension), examples, benchmarks, and the nil-options edge
(`json.Marshal(v, nilOptionInsideSlice)` — untested behavior).

## f) Next — prioritized (session-derived, not padded)

| #  | Task                                                                                                                          | Impact | Effort |
| -- | ----------------------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| 1  | Close the error-path gap: `WriteErrorf`/`WriteJSON` on a v2-negotiated connection emit v1 bytes — new API or doc'd limitation | High   | S/M    |
| 2  | Godoc examples: `ExampleJSONv2Codec`, injected `ExampleResolveResponseCodec` (repo convention is example files)               | High   | XS     |
| 3  | pdd build+test against master via replace directive — empirically prove source-compat                                         | High   | XS     |
| 4  | TODO row + fix for the watchdog flake (poll-based assertion or looser budget)                                                 | Med    | S      |
| 5  | Enforce or pin the injected-codec `Encoding()` constraint (guard in option, or table rows documenting current behavior)       | Med    | XS     |
| 6  | File the go-codec upstream report (dead `goexperiment.jsonv2` tag on Go 1.27) after verify-before-filing                      | Med    | S      |
| 7  | docs-health HARVEST the feedback inbox: annotate F2 as done (`c9bcc7d`), confirm F3 row, drop nothing silently                | Med    | S      |
| 8  | Extend `FuzzResolveResponseCodec` with an injected-codec pass (refusal/ranking invariants under substitution)                 | Med    | S      |
| 9  | Document the `DeterministicCodec` limitation on `JSONv2Codec` (closed signing-safe interface) — or upstream ask               | Med    | XS     |
| 10 | Harvest the three 2026-10-01 status reports (pre-existing check-docs strike failure — TODO row exists)                        | Med    | S      |
| 11 | `lsp_restart` to clear phantom golines/gci warnings on `jsonv2.go`; trust-CLI rule already documented                         | Low    | XS     |
| 12 | Test `JSONv2Codec` with nil elements inside the options slices (undefined edge today)                                         | Low    | XS     |
| 13 | Benchmark `JSONv2Codec.EncodeToBuffer` vs the v1 buffer path (hot-path parity evidence)                                       | Low    | S      |
| 14 | Track go-codec > v0.3.1 for the build-tag fix; bump + `buildflow -s nix-hash-fix --fix` when it lands                         | Low    | XS     |
| 15 | erraudit pass over `jsonv2.go`/`negotiation.go` additions (repo error-policy audit)                                           | Low    | XS     |
| 16 | HTTP-body integration test: `ResolveRequestCodec` + injected v2 codec + `FormatDurationAsNano` decode round trip              | Low    | XS     |
| 17 | README: short json/v2 adoption paragraph under the API table (beyond the one table row)                                       | Low    | XS     |
| 18 | Process rule into AGENTS: gates print failures (`grep -- --- FAIL`), never `tail`; read background gates before committing    | Low    | XS     |
| 19 | Decide the Negotiator bound-resolver question only IF per-call options prove noisy in a real consumer (YAGNI watch)           | Low    | —      |
| 20 | rm -rf /tmp/codecprobe scratch module                                                                                         | Triv.  | XS     |

## g) Questions I cannot answer myself

1. **Default flip policy:** should the default JSON codec ever become
   `JSONv2Codec` (a behavior change for all consumers — map ordering, nil
   shapes, duration errors), or is v1-default a permanent compatibility
   contract with v2 always opt-in? (Owner policy; affects whether f1's fix
   designs for one engine or two.)
2. **Error-path API shape:** to close b1 — a `WriteErrorEncoded(w, status,
   contentCodec, format, args...)`, a `WriteJSONf` variant, or leave
   `WriteErrorf` v1-backed and document? (Consumer-visible API addition; the
   406-variant row shows this repo defers API-shape decisions to you.)
3. **Release cadence:** DiscordSync's revisit bar needs F1+F2+F3 — should F3
   (deferrable READY) be the next session's implementation target so a v0.5.0
   cut is ready the moment the go-public flip happens?
