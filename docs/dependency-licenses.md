# Dependency license inventory

Generated 2026-10-01 from the module graph (`go list -m all` + LICENSE files
in the module cache). Regenerate by re-running the method below; versions are
informational, the license identities are the durable facts.

**Method:** `go list -deps ./...` for the build closure (what consumers of
this library compile and ship), `go list -deps -test` for the dev closure;
license read from each module's `LICENSE`/`COPYING` file in
`$(go env GOMODCACHE)`.

## Build closure (15 modules — ships to consumers)

| Module                                     | Version                            | License      |
| ------------------------------------------ | ---------------------------------- | ------------ |
| `github.com/beorn7/perks`                  | v1.0.1                             | MIT          |
| `github.com/cespare/xxhash/v2`             | v2.3.0                             | MIT          |
| `github.com/coreos/go-systemd/v22`         | v22.7.0                            | Apache-2.0   |
| `github.com/fxamacker/cbor/v2`             | v2.9.4                             | MIT          |
| `github.com/larsartmann/go-codec`          | v0.3.0                             | MIT          |
| `github.com/larsartmann/go-error-family`   | v0.11.0                            | MIT          |
| `github.com/larsartmann/go-flightrecorder` | v0.2.0                             | MIT          |
| `github.com/munnerz/goautoneg`             | v0.0.0-20191010083416-a7dc8b61c822 | BSD-3-Clause |
| `github.com/prometheus/client_golang`      | v1.24.1                            | Apache-2.0   |
| `github.com/prometheus/client_model`       | v0.6.3                             | Apache-2.0   |
| `github.com/prometheus/common`             | v0.72.0                            | Apache-2.0   |
| `github.com/prometheus/procfs`             | v0.22.0                            | Apache-2.0   |
| `github.com/x448/float16`                  | v0.8.4                             | MIT          |
| `golang.org/x/sys`                         | v0.48.0                            | BSD-3-Clause |
| `google.golang.org/protobuf`               | v1.36.12                           | BSD-3-Clause |

## Test-only direct dependency

| Module                                   | Version | License |
| ---------------------------------------- | ------- | ------- |
| `github.com/larsartmann/go-sse/sseparse` | v0.1.0  | MIT     |

## Dev-transitive (test dependencies of dependencies — never shipped)

41 modules, all permissive: MIT (largest group — gomega, testify, goleak,
tidwall family, go-snaps, yaml, json-iterator, …), BSD-3-Clause (httprouter,
golang/protobuf legacy, go-cmp, golang.org/x/{net,oauth2,sync,text},
klauspost/compress, …), Apache-2.0 (modern-go pair, mwitkow/go-conntrack,
kylelemons/godebug, go.yaml.in pair, …), BSD-2-Clause (godbus/dbus), and one
MPL-2.0 (`pgregory.net/rapid`, file-level copyleft, test-only).

**No copyleft license appears anywhere in the build closure.**

## NOTICE-file decision

No `NOTICE` file is added to this repository. Rationale:

- This repo is a **library**: it ships no binaries, so the Apache-2.0 §4(d)
  NOTICE-preservation obligation falls on whoever compiles and distributes a
  binary, not on the library.
- The Go module cache carries every dependency's LICENSE (and the
  `prometheus/common` + `prometheus/client_golang` NOTICE files) verbatim to
  consumers, which is the ecosystem-standard attribution path.
- If a consumer ever distributes a compiled daemon built on this library,
  their tooling (or a generated THIRD-PARTY-NOTICES file) must preserve those
  two NOTICE files; that requirement is now visible here.
