// Package daemon provides the production-proven building blocks for superb
// long-lived daemons: listener lifecycle (stale-socket cleanup, 0600 unix
// permissions, sd_notify readiness/watchdog, graceful drain) on unix domain
// sockets or TCP, unix-socket HTTP transports for clients, JSON/CBOR content
// negotiation (Accept-driven with "Vary: Accept" on every written response)
// with pooled response writing, WHATWG SSE framing, and an execution-trace
// flight-recorder HTTP integration.
//
// The package is mechanism, not policy: it owns sockets, lifecycle ordering,
// codecs, and framing. Domain handlers, caching, and business logic stay with
// the consuming daemon.
//
// No file in the package carries a build constraint. The sd_notify
// integration switches on NOTIFY_SOCKET at run time, so one binary runs
// unchanged in the foreground, under launchd, or under systemd, and
// without systemd every notify call is a silent no-op.
//
// Extracted from project-discovery-daemon, whose lifecycle ordering,
// shutdown-regression tests, and metric names this library preserves.
package daemon
