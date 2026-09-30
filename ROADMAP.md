# Roadmap

Long-term direction and raw ideas. Bounded, actionable work lives in
[TODO_LIST.md](TODO_LIST.md); shipped work lives in [FEATURES.md](FEATURES.md)
and [CHANGELOG.md](CHANGELOG.md).

## Themes

### 1. Negotiation as a full HTTP contract

Content negotiation currently covers media types with q-values. The raw
direction: make response encoding a complete, cache-correct HTTP contract —
an error-signaling variant (406-capable resolution instead of silent
request-codec echo when a client refuses everything), explicit precedence
between specific types and wildcards for strict-RFC consumers (behind an
option, never by default), and a stated policy for the other `Accept*`
headers. A golden corpus of wire headers — the same philosophy as the SSE
conformance corpus — would make the whole behavior matrix reviewable.
Refinement candidate: emit `Vary: Accept` only when the Accept header
actually influenced the encoding, instead of unconditionally (the current
blanket emission is conservative but over-broad for fixed-JSON endpoints).

### 2. More daemon building blocks, same discipline

This library exists by extraction: blocks graduate here from
project-discovery-daemon once they prove themselves in production. Future
candidates follow the same mechanism-not-policy line — lifecycle plumbing,
socket concerns, observability hooks — each arriving with its regression
tests and its ordering/behavior contracts already test-enforced. Runnable
godoc examples as a normal part of every exported surface. Smaller raw
ideas: a golden test for the `sd_notify` message sequence (the lifecycle
ordering already has one), and a one-shot benchmark of the SSE line reader
(it allocates a 32 KiB chunk per parser instance; a number beats a hunch).

### 3. Fleet coherence

go-daemon sits mid-chain: project-discovery-daemon consumes it; it consumes
go-codec and go-flightrecorder; the SSE parser is pinned to go-sse's corpus.
Raw ideas: a single consumer-chain picture with current pins across all five
repos; extracting the pure media-type ↔ codec mapping into go-codec when a
second transport consumer appears (the pre-agreed trigger); go-cqrs-lite's
transport deciding whether it wants Accept-based negotiation; feeding the
negotiation patterns documented here back into project-discovery-daemon's
docs, where the code originated. Cross-repo pointers parked here rather
than in TODO_LIST: pdd adopting `Vary: Accept` for its dual-encoding
responses (cache-unsafe without it); landing the own+consumer vendorHash
lesson in crush-config's `references/lessons.md`; reporting buildflow's
nix-hash-fix findings-gate false-alarm (intermediate builds shown as
"failed"); a docs website via the website-launch pattern and a post-flip
announcement draft, both only if the go-public flip happens.

## Open questions

- When a client refuses every mentioned encoding (`q=0` across the board),
  should resolution signal 406 instead of echoing the request codec? The
  current signature cannot express "nothing acceptable" — this is an
  API-shape decision, not an implementation detail.
- Is tie-preference-for-CBOR the intended permanent server preference, or a
  transitional default before request-echo wins ties?
- What cadence should regenerate the SSE conformance corpus from upstream
  WPT vectors (the ingestion generator exists; no scheduled trigger)?

## Explicit non-goals

- Domain logic: caching, business handlers, payload decoding — policy stays
  with consuming daemons; this library is mechanism only.
- TCP/TLS serving: the focus is unix domain sockets between local processes.
- Payload decoding inside `ParseSSEData`: framing only, by design.
- Extracting negotiation into go-codec before a second real consumer exists.
- Performance work on negotiation without a profiling signal (it is not the
  hot path at this size).
