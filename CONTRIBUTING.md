# Contributing

Thanks for your interest in contributing! This is the daemon
building-blocks library: mechanism, not policy — sockets, lifecycle
ordering, codecs, framing. Changes that pull in domain logic belong in the
consumer, not here.

## Development setup (nix-first)

The nix flake is the source of truth for the toolchain (`go 1.27`) and every
gate. From a checkout with nix installed:

```bash
nix develop          # shell with go, golangci-lint, dprint
nix build .#default  # hermetic build (also catches a stale vendorHash)
nix run .#test       # go test ./... -race -count=1
nix run .#lint       # golangci-lint run ./...
nix fmt              # format nix files
nix develop -c dprint fmt  # format markdown/json/yaml
nix run .#check-readme     # compile the README quickstart snippet
```

Plain-Go equivalent (if you have go 1.27 + golangci-lint installed):

```bash
go build ./...
go test ./... -race -count=1
go vet ./...
golangci-lint fmt && golangci-lint run ./...
```

## Before you open a PR

- Every gate above must pass; CI runs the same checks (build, vet, race
  suite with an 80% coverage floor, vulnerability scan, lint, README
  compile check, nix build + flake check).
- Lifecycle ordering is a contract — see the `Lifecycle` doc comment in
  `socket.go`. Do not reorder stages; the hook-ordering test pins it.
- Content-negotiation behavior, doc comments, and
  `TestResolveResponseCodec`'s table move together.
- Table-driven tests lead with a `name` field; rows must match the struct
  arity.
- Add tests for behavior changes; a red test is either a broken test or a
  found bug — analyze before "fixing" it green.

## Reporting issues

Use GitHub Issues with the provided templates. Security issues never go
into public issues — see [SECURITY.md](SECURITY.md).

## License

By contributing you agree that your contributions are licensed under the
MIT License that covers this repository.
