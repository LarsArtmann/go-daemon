// Package daemon provides the production-proven building blocks for superb
// long-lived daemons served over unix domain sockets: socket lifecycle
// (stale-socket cleanup, 0600 permissions, sd_notify readiness/watchdog,
// graceful drain), unix-socket HTTP transports for clients, JSON/CBOR content
// negotiation with pooled response writing, WHATWG SSE framing, and an
// execution-trace flight-recorder HTTP integration.
//
// The package is mechanism, not policy: it owns sockets, lifecycle ordering,
// codecs, and framing. Domain handlers, caching, and business logic stay with
// the consuming daemon.
//
// Extracted from project-discovery-daemon, whose lifecycle ordering,
// shutdown-regression tests, and metric names this library preserves.
package daemon
