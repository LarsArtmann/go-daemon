# Roadmap

Long-term direction and raw ideas. Bounded, actionable work lives in
[TODO_LIST.md](TODO_LIST.md); shipped work lives in [FEATURES.md](FEATURES.md)
and [CHANGELOG.md](CHANGELOG.md). Vocabulary is in
[docs/DOMAIN_LANGUAGE.md](docs/DOMAIN_LANGUAGE.md).

## Themes

### 1. Negotiation as a full HTTP contract

Content negotiation covers media types with q-values. The raw direction:
make response encoding a complete, cache-correct HTTP contract — an
error-signaling variant (406-capable resolution instead of silently echoing
the request codec when a client refuses everything), and a stated policy for
the other `Accept*` headers. A golden corpus of wire headers — the same
philosophy as the SSE conformance corpus — would make the whole behavior
matrix reviewable.

Refinement candidates: emit `Vary: Accept` only when the `Accept` header
actually influenced the encoding, instead of unconditionally (the current
blanket emission is conservative but over-broad for fixed-JSON endpoints);
pin `Vary` on the `q=0` refusal-substitution path; document that handlers
bypassing `WriteEncoded` get no `Vary` (a golden test would keep both
statements true).

### 2. More daemon building blocks, same discipline

This library exists by extraction: blocks graduate here from
project-discovery-daemon once they prove themselves in production. Future
candidates follow the same mechanism-not-policy line — lifecycle plumbing,
socket concerns, observability hooks — each arriving with its regression
tests and its ordering/behavior contracts already test-enforced.

Raw ideas: a `JSONv1Codec` convenience adapter (explicit `encoding/json` v1)
for consumers that need v1 shapes without v2 option gymnastics; an
`EACCES`-mapping `TransportErrorKind` (`PermissionDenied`) so a socket-mode
mistake is nameable instead of falling into "other"; a golden test
for the `sd_notify` message sequence (the lifecycle ordering already has
one); a one-shot benchmark of the SSE line reader (it allocates a 32 KiB
chunk per parser instance; a number beats a hunch); a benchmark for the
pooled-buffer write path so a future "remove the pool" proposal is
decidable; systemd socket activation (`LISTEN_FDS`) support, documented as
unsupported today in `docs/supervision.md`; and a documented nil-codec
contract for the exported write helpers (tracked in TODO_LIST).

### 3. Fleet coherence

go-daemon sits mid-chain: project-discovery-daemon consumes it; it consumes
go-codec and go-flightrecorder; the SSE parser is pinned to go-sse's corpus.
Raw ideas: a single consumer-chain picture with current pins across all five
repos; extracting the pure media-type ↔ codec mapping into go-codec when a
second transport consumer appears (the pre-agreed trigger); go-cqrs-lite's
transport deciding whether it wants Accept-based negotiation; feeding the
negotiation patterns documented here back into project-discovery-daemon's
docs, where the code originated.

Cross-repo pointers parked here rather than in TODO_LIST: pdd adopting
`Vary: Accept` for its dual-encoding responses (cache-unsafe without it);
pdd replacing its private `classifyTransportError` with
`ClassifyTransportError` and its hand-rolled wait loops with
`WaitForSocket`/`ProbeUntilAlive`; pdd switching error writes to
`WriteErrorfWithCodec` so error bodies match the negotiated codec; landing
the own+consumer vendorHash lesson in crush-config's `references/lessons.md`;
reporting buildflow's nix-hash-fix findings-gate false alarm (intermediate
builds shown as "failed"); go-codec's dangling `testdata_test.go` gopls
`unusedfunc` diagnostic; go-datastar's v0.6.2 re-cut decision (frozen
tag-CI); and an art-dupl accepted-clone baseline gate (assert exactly the
negotiation `default:` pair, fail on new clones) so clone detection becomes
regression-catching instead of advisory.

Pointers harvested 2026-11-28 from the archived reports: go-sse's status
archive hygiene (auditing the pre-09-29 fully resolved reports into its
archive), a `GOTOOLCHAIN=auto` pinning policy doc, the remaining
coverage-floor, tag-CI and fuzz-soak items, the flake-update weekly
workflow's golangci cross-check against the directive-floor policy, and a
sweep for stale git worktree entries; go-datastar's smoke app caller in CI
once it gains runnable examples and a check that `datastartest` still pins
ssetest v0.4.0; pdd's post-push status entry for the v0.5.0 bump per its own
reporting convention and its copy of the same-shape test pattern; the
crush-config docs-health workflow note that annotate-tool output must
survive `dprint fmt`; and an art-dupl accepted-clone baseline gate that also
surfaces filtered-suppressed groups on review runs.

Also from the 2026-11-28 session: nextd's tests resolving `$HOME` when no
state directory is given (a `TestMain` default would make the whole suite
hermetic); the pdd admin-package timing flakes (`TestSelfMetricsScrape`,
`TestBridgeStopEndsInFlightHandlerFetches`, 1 in 20 runs); archive manifest
provenance and the ledger convention in this repo's AGENTS; and a
`release-check --require-clean` mode plus a binary overlay or cached build
for the Go toolchain, both under theme 5.

### 4. Public presence

The repository is public since 2026-11-27. The store front is the minimum
today (description, topics, README badges). Raw ideas: a homepage, a docs
site built with the sibling-project website-launch pattern, a launch note
in the consumer's README, and a recorded v0.x release-presentation policy
(prerelease vs "Latest", signed tags). All owner-taste calls; none block
adoption.

The release presentation policy is undecided: whether a pre-release may be
the GitHub Latest release, and how long v0.4.x receives fixes once the
consumers are on a newer line.

### 5. Post-flip security posture

Security is now a public-repository obligation: CodeQL default setup,
branch/tag protection, the remaining secret-scanning patterns, and a
documented response to `go.mod`'s 1.27.2 floor versus the CVE fix in
1.27.4. The floor decision is deliberately deferred (raising it breaks
consumers whose nixpkgs still ships 1.27.3); the README and SECURITY.md
carry the build-with-1.27.4 requirement until the toolchain supply catches
up.

## Open questions

- When a client refuses every mentioned encoding (`q=0` across the board),
  should resolution signal 406 instead of echoing the request codec? The
  current signature cannot express "nothing acceptable" — this is an
  API-shape decision, not an implementation detail (FEATURES row ON_HOLD).
- Is tie-preference-for-CBOR the intended permanent server preference, or a
  transitional default before request-echo wins ties?
- What cadence should regenerate the SSE conformance corpus from upstream
  WPT vectors (the ingestion generator exists in go-sse; no scheduled
  trigger)?
- Should the module floor move to `go 1.27.4` (security) even though it
  breaks consumers whose nixpkgs still ships 1.27.3, and if so is that a
  v0.6.0? (Owner decision; TODO_LIST tracks the flake-side bump that would
  make the local toolchain match.)
- What is the v1.0 gate? The 2026-11-27 decision keeps v0.x no-promises
  until a second external consumer exists; a v1 freeze is a compatibility
  obligation, not a discoverability win.

## Explicit non-goals

- Domain logic: caching, business handlers, payload decoding — policy stays
  with consuming daemons; this library is mechanism only.
- TLS termination and service-manager integration: unix sockets are the
  default and TCP works through `NewServer`, but securing the transport is
  the operator's job.
- Payload decoding inside `ParseSSEData`: framing only, by design.
- Extracting negotiation into go-codec before a second real consumer exists.
- Performance work on negotiation without a profiling signal (it is not the
  hot path at this size).
