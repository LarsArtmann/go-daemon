# Contributing

Thanks for your interest in contributing! This is the daemon
building-blocks library: mechanism, not policy — sockets, lifecycle
ordering, codecs, framing. Changes that pull in domain logic belong in the
consumer, not here.

## Development setup (nix-first)

The nix flake is the source of truth for the toolchain (`go 1.27.2` floor;
CI builds with 1.27.4 because earlier patches carry CVE-2026-97054) and
every gate. From a checkout with nix installed:

```bash
nix develop                          # shell with go, golangci-lint, dprint, govulncheck, actionlint
nix develop -c release-check         # every CI gate in order, one verdict (run before a push or a tag)
nix build .#default  # hermetic build (also catches a stale vendorHash)
nix flake check --no-build --all-systems  # evaluate every listed system (CI runs this)
nix run .#test       # go test ./... -race -count=1
nix run .#lint       # golangci-lint run ./...
nix run .#check-readme     # compile the README quickstart snippet
nix run .#check-docs       # dprint, row completeness, strikes, FEATURES counts
nix run github:Artmann-Technologies/docs-gates/bfdefbaf73d668fd4b2aab787ddecee7dc046d56 -- run .  # full docs-health battery, the rev CI pins
nix fmt              # format nix files
nix develop -c dprint fmt  # format markdown/json/yaml
```

Plain-Go equivalent (if you have go 1.27.2 + golangci-lint installed):

```bash
go build ./...
go test ./... -race -count=1
go vet ./...
golangci-lint fmt && golangci-lint run ./...
```

## Before you open a PR

- Every gate above must pass; `nix develop -c release-check` runs exactly
  the CI list, and CI runs the same checks (build, vet, race suite with an
  80% coverage floor, vulnerability scan, lint, README compile check, nix
  build + `nix flake check --no-build --all-systems`, both docs gates).

## Releases

A release is an annotated tag on a `master` commit whose CI run is green:

```bash
nix develop -c release-check     # the same gates CI runs, on this tree
git tag -a v0.X.Y -m "v0.X.Y"
git push origin v0.X.Y
```

Pushing a `v*` tag runs the full CI workflow on the tagged tree; the Go
module proxy serves the version once that run is green. Never move or
re-tag a published version: a released wire shape is a contract, so a fix
goes into the next version.

- Lifecycle ordering is a contract — see the `Lifecycle` doc comment in
  `socket.go`. Do not reorder stages; the hook-ordering test pins it.
- Content-negotiation behavior, doc comments, and
  `TestResolveResponseCodec`'s table move together.
- The default JSON codec emits json/v2 wire shapes (nil slice `[]`, no
  `time.Duration` encoding, unordered map keys). If a wire shape matters,
  pass an explicit codec (`JSONv2Codec` with `FormatNilSliceAsNull(true)` /
  `FormatDurationAsNano(true)`, or `encoding/json`-backed code) through
  `WriteJSONWithCodec`, `WriteErrorfWithCodec`, or `WithJSONCodec` — never
  rely on the backend go-codec happened to compile.
- Table-driven tests lead with a `name` field; rows must match the struct
  arity.
- Add tests for behavior changes; a red test is either a broken test or a
  found bug — analyze before "fixing" it green.

## Documentation

The vocabulary is in [docs/DOMAIN_LANGUAGE.md](docs/DOMAIN_LANGUAGE.md);
supervisor settings that pair with the library's contracts are in
[docs/supervision.md](docs/supervision.md). Public API changes add or update
their glossary term in the same change.

## Reporting issues

Use GitHub Issues with the provided templates. Security issues never go
into public issues — see [SECURITY.md](SECURITY.md).

## License

By contributing you agree that your contributions are licensed under the
MIT License that covers this repository.
