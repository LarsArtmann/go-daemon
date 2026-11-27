# Security Policy

## Supported versions

| Version | Supported          |
| ------- | ------------------ |
| 0.5.x   | :white_check_mark: |
| < 0.5   | :x:                |

This library is pre-1.0: only the latest minor release receives security
fixes. Consumers should track the latest tag.

## Reporting a vulnerability

Please do **not** open a public issue for security reports. Use GitHub
private vulnerability reporting (Security → Report a vulnerability), which
keeps the report confidential until a fix ships.

Include what you can of: the affected version (`go list -m
github.com/LarsArtmann/go-daemon`), a minimal reproduction, and your
assessment of impact. You will get an acknowledgement within a week, and a
fix or a mitigation plan before any public disclosure.

## Security Practices

- The library terminates on a unix domain socket with owner-restricted
  permissions (`0o600` by default). Any process that can write to the socket
  executes requests with the daemon's privileges — permission-widening is a
  deliberate operator decision, not a bug report.
- The SSE parser is framing-only: it never evaluates payload bytes. Payload
  decoding bugs belong to the consumer that decodes them.
- Build with Go 1.27.4 or newer. Go 1.27.0–1.27.3 ship CVE-2026-97054
  (GO-2026-6859), a net/http/1 server panic on a MaxBytesReader header
  race; `govulncheck` reports it against every daemon that serves HTTP.
  The fix is in the toolchain, not in this library.
