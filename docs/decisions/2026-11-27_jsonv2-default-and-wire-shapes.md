# Decision — the default JSON codec's wire shapes are unconditional json/v2

**Date:** 2026-11-27
**Status:** accepted (recorded after the v0.5.0 release; behaviour was
already shipped and documented in `CHANGELOG.md` §0.5.0)
**Driver:** consumers asked what bytes the default codec emits. The answer
was previously an inference ("go-codec compiles the v1 flavor on Go 1.27"),
repeated in `AGENTS.md` and the release notes, and it was wrong.

## Context

`codec.JSONCodec{}` is the default for `WriteJSON`, `WriteErrorf`,
`WriteEncoded` (default codec), and negotiation's JSON arm. go-codec v0.3.3
selects its `encoding/json/v2` backend when the `jsonv2` experiment is on,
which Go 1.27 enables by default. A direct probe in `/tmp/jsonprobe` on
go1.27.3 with go-codec v0.3.3 measured:

- a nil slice encodes as `[]`, not `null`;
- `time.Duration` fails to encode (it has no default v2 representation);
- map key order is unordered.

`GOEXPERIMENT=none go build ./...` does not select a v1 backend — it fails
to build at all, because `jsonv2.go` imports `encoding/json/v2` and
`prometheus/common` imports `encoding/json/jsontext`, both excluded when
the experiment is off. There is no build configuration of this module in
which the default codec is v1.

## Decision

The default codec's json/v2 wire shapes are **the module's actual contract**,
not a per-build coin flip. Documentation states the shapes directly and does
not promise a v1 default.

Consumers that need v1 wire shapes pass an explicit codec:

- `JSONv2Codec{MarshalOptions: []json.Options{json.FormatNilSliceAsNull(true), json.FormatDurationAsNano(true)}}`
  for the v1 shapes through the v2 engine;
- any `encoding/json`-v1-backed `codec.Codec` implementation;
- routed through `WriteJSONWithCodec` / `WriteErrorfWithCodec` or
  negotiation's `WithJSONCodec`.

## Consequences

- Existing consumers on v0.5.0 see the v2 shapes. project-discovery-daemon
  pins v0.5.0 and must audit DTOs for `time.Duration` and nil-slice fields
  before its next bump (tracked in `TODO_LIST.md`).
- A future go-codec release that changes its default backend cannot silently
  flip this module: `AGENTS.md` requires probing behaviour, and the
  `TestJSONv2Codec_*` tables pin the explicit adapter. A named
  default-codec wire-shape test is still owed (`TODO_LIST.md`).
- `nextd` at `go 1.27.1` must raise its floor to 1.27.2 to consume v0.5.0
  (tracked in `TODO_LIST.md`).

## Alternatives rejected

- **Pin go-codec v0.3.1 for a `go 1.27` floor.** Trades the json/v2 decode
  fix and the v0.3.3 version contract for a lower floor; rejected as the
  default because v0.5.0 is already tagged and consumers can stay on v0.4.x.
- **Add a `GOEXPERIMENT=none` escape hatch.** Impossible: the module does
  not build with the experiment off.
- **Document only in `AGENTS.md`.** The 2026-11-27 session showed the facts
  are load-bearing and easy to lose in a rewrite; they now have an ADR and
  a CHANGELOG consequence.
