# Supervising a go-daemon daemon

This library is mechanism, not policy: it speaks `sd_notify` and drains
gracefully, but packaging, deployment, and supervision belong to your service
manager. This page collects the supervisor settings that pair with the
library's contracts — recommendations, not requirements. Everything here is
optional: without systemd the notify calls are silent no-ops and the daemon
runs in the foreground unchanged.

## The unit that matches the library's contracts

```ini
[Service]
Type=notify
WatchdogSec=30
TimeoutStopSec=10
Restart=on-failure
```

Why each line:

- `Type=notify` — `SocketServer` sends `READY=1` right after `listen(2)`
  succeeds and `STOPPING=1` when shutdown begins, so systemd considers the
  service started only once the socket is actually listening.
- `WatchdogSec=30` — `SocketServer` detects `WATCHDOG_USEC` and pings
  `WATCHDOG=1` at half that interval automatically (per
  `sd_watchdog_enabled(3)`). A hung process gets killed and restarted; no
  application code is needed.
- `TimeoutStopSec=10` — keep it above your `WithShutdownGracePeriod`
  (default 5s): the whole drain sequence (`STOPPING` → graceful shutdown →
  socket removal → `OnShutdown`) must fit inside systemd's stop timeout, or
  systemd sends `SIGKILL` mid-drain.
- `Restart=on-failure` — combined with the watchdog, this turns hangs into
  restarts. A restarted daemon never trips over a crashed predecessor's
  socket: `Start` probes and removes a stale socket file before listening
  (`CleanStaleSocket`), and removes its own on every exit path.

## Resource priority: making the daemon nice

The common reason to reach for these: a background daemon must never steal
CPU or disk from interactive work. All of it is supervisor policy — set it in
the unit, not in Go:

```ini
[Service]
Nice=10
CPUSchedulingPolicy=idle
IOSchedulingClass=idle
IOSchedulingPriority=7
CPUWeight=50
```

Pick the weakest knob that solves your problem (they are alternatives, not a
required set):

- `Nice=10` — classic niceness; only matters under CPU contention. Higher
  positive values defer more.
- `CPUSchedulingPolicy=idle` — stronger than nice: the daemon runs only when
  nothing else wants the CPU. For daemons whose latency never matters
  (scrapers, indexers, batch syncs).
- `IOSchedulingClass=idle` (+ `IOSchedulingPriority=7`) — the same idea for
  disk I/O; for daemons that can burst into heavy reads or writes.
- `CPUWeight=50` — cgroup-relative share, compared against other units'
  weights rather than global niceness. The right knob when you want "half as
  much CPU as my other services", not absolute deference.

## Memory: a separate knob with a separate failure mode

None of the CPU/I/O knobs bound memory — nice does nothing for RAM. Memory
limits are cgroup limits, and they fail _loudly_: when a unit hits its cap
the kernel OOM-killer picks a victim, so the limit must be paired with a
restart policy and a deliberate `OOMPolicy=`:

```ini
[Service]
MemoryHigh=256M
MemoryMax=512M
OOMPolicy=stop
Restart=on-failure
RestartSec=2s
```

- `MemoryHigh=` is the throttle: the kernel reclaims aggressively and slows
  the cgroup down. Use it when you want pressure before death.
- `MemoryMax=` is the hard wall: crossing it triggers the OOM killer. Leave
  headroom for the Go runtime's heap target and any flight-recorder snapshot
  buffer (`MaxSnapshots` × trace size), which is the usual surge.
- `OOMPolicy=stop` (the default is `continue`) stops the whole unit when a
  process is OOM-killed, so `Restart=on-failure` sees a failed unit instead
  of a half-dead process. `OOMPolicy=continue` is only right when one
  process dying must not take its siblings down.

A daemon with an unbounded flight recorder or an in-memory cache has no
natural memory ceiling; `MemoryMax=` is the only thing that turns a slow
leak into a restart instead of an OOM-killed host. Watch the unit's
`MemoryPeak` (systemd 254+) when choosing the numbers.

## NixOS module

```nix
{
  systemd.services.mydaemon = {
    wantedBy = [ "multi-user.target" ];
    serviceConfig = {
      ExecStart = "/path/to/mydaemon";
      Type = "notify";
      WatchdogSec = 30;
      TimeoutStopSec = 10;
      Restart = "on-failure";
      Nice = 10;
      IOSchedulingClass = "idle";
      User = "mydaemon";
    };
  };

  systemd.tmpfiles.rules = [
    "d /run/mydaemon 0750 mydaemon mydaemon -"
  ];
}
```

The tmpfiles rule pre-creates the runtime socket directory owned by the
daemon user. `/run` is cleared at boot and tmpfiles recreates the directory
before the daemon starts; without it, a non-root daemon cannot create its
socket directory under `/run`.

## Hardening starters

```ini
[Service]
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
ReadWritePaths=/run/mydaemon
```

The socket is `0600` by default, so only the owning user may connect; unit
hardening narrows what a compromised daemon process can otherwise reach. See
`systemd.exec(5)` for the full set of sandboxing directives.
