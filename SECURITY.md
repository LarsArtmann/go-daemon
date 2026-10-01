# Security Policy

## Supported versions

| Version | Supported          |
| ------- | ------------------ |
| 0.2.x   | :white_check_mark: |
| < 0.2   | :x:                |

This library is pre-1.0: only the latest minor release receives security
fixes. Consumers should track the latest tag.

## Reporting a vulnerability

Please do **not** open a public issue for security reports.

- While this repository is private, contact the maintainer directly on
  GitHub (@LarsArtmann) — reports stay confidential by default.
- Once the repository is public, use GitHub private vulnerability reporting
  (Security → Report a vulnerability), which keeps the report confidential
  until a fix ships.

Include what you can of: the affected version (`go list -m
github.com/LarsArtmann/go-daemon`), a minimal reproduction, and your
assessment of impact. You will get an acknowledgement within a week, and a
fix or a mitigation plan before any public disclosure.

## Scope notes

- The library terminates on a unix domain socket with owner-restricted
  permissions (`0o600` by default). Any process that can write to the socket
  executes requests with the daemon's privileges — permission-widening is a
  deliberate operator decision, not a bug report.
- The SSE parser is framing-only: it never evaluates payload bytes. Payload
  decoding bugs belong to the consumer that decodes them.
