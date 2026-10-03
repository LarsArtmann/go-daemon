# Decision — v0.4.0 notify API shape

- **Date:** 2026-10-03
- **Status:** accepted (session: bank-sync sd_notify adoption, plan T3/T5)
- **Driver:** bank-sync needs sd_notify READY/STOPPING + watchdog keepalive for a **TCP** http.Server; the capability existed only inside the unix-socket-only `SocketServer`.

## Decision

Standalone package-level functions in `notify.go`:

- `NotifyReady(logger *slog.Logger)` — sends `READY=1`
- `NotifyStopping(logger *slog.Logger)` — sends `STOPPING=1`
- `Watchdog(ctx context.Context, logger *slog.Logger, interval time.Duration)` — sends `WATCHDOG=1` every `interval` until `ctx` is done; no-op on non-positive interval (library code never panics)
- `WatchdogInterval(logger *slog.Logger) (time.Duration, bool)` — wraps `SdWatchdogEnabled`, returns the recommended ping interval = **WatchdogSec/2**, per `sd_watchdog_enabled(3)`: _"It is recommended that a daemon sends a keep-alive notification message to the service manager every half of the time returned here"_ (systemd man source, fetched 2026-10-03)
- `ReadNotifyCounters() NotifyCounters` — process-global atomic counters (`ReadySent`, `StoppingSent`, `WatchdogPings`, `NotifyFailures`) as the observability seam

`SocketServer` delegates to these (behavior, log wording, and ordering unchanged); its private `watchdog`/`notifyStopping` methods are deleted.

## Rationale (5 lines)

1. sd_notify is **process-global state** (`NOTIFY_SOCKET`, `WATCHDOG_USEC` env set once by systemd) — a per-instance `Notifier` type would add ceremony without any per-instance reality to model.
2. The only parameter is the logger; void returns + log + counters keep call sites one-liners (errors are advisory by contract: a daemon systemd cannot hear must still run).
3. Counters (not injected hooks) because every consumer path (SocketServer, future TCP runner, standalone) flows through the same process-global funcs — one seam, zero wiring; bank-sync reads them straight into its `/metrics` scrape.
4. `WatchdogInterval` moves the WatchdogSec/2 policy into the library so no consumer re-derives it (and gets it wrong) — the systemd recommendation is now encoded once with its citation.
5. Rejected alternatives: a `Notifier` interface ( ceremony, no variants exist today); returning `error` (forces every consumer to handle the normal no-systemd case); hook-based metrics (per-instance plumbing for a process-global fact).

## Consequences

- New exported API is additive; `SocketServer` behavior is byte-for-byte the same (same sd calls, same order in the lifecycle contract).
- Counters are process-global by design — documented, not accidental.
- Consumers on networks other than unix can adopt readiness/watchdog without `SocketServer` at all (unlocks bank-sync's TCP server).
