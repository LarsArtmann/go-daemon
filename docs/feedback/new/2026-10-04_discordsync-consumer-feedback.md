# Consumer Feedback — DiscordSync Adoption Delta

**Source:** DiscordSync's go-daemon evaluation (`/home/lars/projects/DiscordSync/docs/adr/ADR-070-godaemon-hardening-pattern-adoption.md`, 2026-10-03), re-verified against go-daemon `master` (`e2187e7`, v0.4.0 tag `0a5d806` + dep-sweep patch bumps only) on 2026-10-04.
**Question answered:** what would go-daemon need to change for DiscordSync to consider adopting it as a dependency?
**Verdict today:** not adopted — the three useful patterns were ported natively instead (sd_notify lifecycle, flight-recorder slow-request trigger, trace rate limit). The rejection is not static; the items below would reopen the decision.

## Actionable feedback

| #  | Ask                                                        | Evidence                                                                                                                                  | Consumer impact                                                                                                                                                                                                                                          |
| -- | ---------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| F1 | Go public                                                  | `gh repo view LarsArtmann/go-daemon` returns PRIVATE (verified 2026-10-04); the owner-blocked "Go-public cluster (pre-flip)" sits in TODO_LIST | As long as the repo is private, every consumer pays the full private-dep tax: flake input pinned to the tag commit, `mkPreparedSource`, `deps` map key, vendorHash churn, cache-busted hash-fix runs. Public drops adoption cost to normal public-dep level. This is the #1 barrier. |
| F2 | encoding/json/v2-native write path, or injectable marshal options | `write.go` routes response writing through `larsartmann/go-codec` (v0.3.1), not encoding/json/v2                                          | Consumers already on json/v2 (DiscordSync's `internal/api/write_json.go`: v1-style integer-nanos durations, `omitzero`, sorted map keys) would regress by adopting the negotiation. v2-native encoding or a marshal-options injection point removes the objection. |
| F3 | Gated READY (`NotifyReadyWhenServing` semantics)            | go-daemon `NotifyReady` notifies immediately; DiscordSync's local port (`cmd/discordsync/systemd.go`) dials the listener until it accepts (bounded loop) before sending READY | Parity would make the local port swappable for `go-daemon/notify`. Without it the local port is strictly stronger, so the swap gains nothing functionally.                                                                                               |

## Not actionable upstream (recorded for completeness)

| Piece                                   | Why no upstream change helps                                                                                                                                                           |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SocketServer` / `NewServer` (TCP since v0.4.0) | Capability already exists; DiscordSync simply has no second listener or control-plane need — its own API is the HTTP surface. Only relevant if DiscordSync grows an admin socket or sidecar. |
| `ParseSSEData`                          | DiscordSync only writes SSE (its ADR-050: SSE is browser-facing). It would need to become an SSE-consuming client first, which its architecture rules out.                              |

## Revisit bar

Public repo (F1) + v2-compatible codec (F2) + at least one functional delta over the natively ported patterns (F3 is the nearest candidate). Until all three hold, adopting go-daemon remains ecosystem dogfooding with zero functional payoff.
