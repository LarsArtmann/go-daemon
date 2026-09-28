package daemon

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	sd "github.com/coreos/go-systemd/v22/daemon"
)

const (
	// DefaultSocketMode restricts the unix socket to owner read/write.
	// Deliberately not group/world-accessible: any local process that can
	// write to the socket executes with the daemon user's privileges.
	DefaultSocketMode = 0o600

	// DefaultSocketDirMode makes the socket's parent directory traversable
	// so unprivileged clients can reach the socket inside e.g. /run/<name>.
	DefaultSocketDirMode = 0o755

	// DefaultReadHeaderTimeout and DefaultIdleTimeout bound per-request
	// header reads and idle keep-alive connections so slow or stuck clients
	// cannot pin server slots.
	DefaultReadHeaderTimeout = 10 * time.Second
	DefaultIdleTimeout       = 120 * time.Second

	// DefaultShutdownGracePeriod bounds how long graceful shutdown waits for
	// in-flight requests after the lifecycle context is canceled.
	DefaultShutdownGracePeriod = 5 * time.Second

	// DefaultHealthPath is the endpoint probed to decide whether an existing
	// socket file belongs to a live daemon before it is removed as stale.
	DefaultHealthPath = "/health"

	// DefaultProbeTimeout bounds the health liveness probe used to detect a
	// stale socket file before start.
	DefaultProbeTimeout = 500 * time.Millisecond
)

// ErrHandlerRequired is returned by SocketServer.Start when the Lifecycle
// carries no Handler.
var ErrHandlerRequired = errors.New("daemon: Lifecycle.Handler is required")

// SocketServer serves an HTTP handler on a unix domain socket with the full
// production daemon lifecycle: stale-socket cleanup, socket directory
// creation, restrictive permissions, systemd readiness/watchdog notification,
// graceful drain, and socket removal on every exit path.
//
// Construct with NewSocketServer.
type SocketServer struct {
	socketPath          string
	socketMode          fs.FileMode
	socketDirMode       fs.FileMode
	logger              *slog.Logger
	version             string
	readHeaderTimeout   time.Duration
	idleTimeout         time.Duration
	shutdownGracePeriod time.Duration
	healthPath          string
	probeTimeout        time.Duration
}

// SocketServerOption configures a SocketServer.
type SocketServerOption func(*SocketServer)

// WithSocketMode sets the filesystem permissions for the unix socket
// (default 0o600, owner-only). Widen deliberately — any local process that
// can write to the socket executes with the daemon user's privileges.
func WithSocketMode(mode fs.FileMode) SocketServerOption {
	return func(s *SocketServer) { s.socketMode = mode }
}

// WithSocketDirMode sets the permissions for the socket's parent directory
// (default 0o755), created on demand when it does not exist.
func WithSocketDirMode(mode fs.FileMode) SocketServerOption {
	return func(s *SocketServer) { s.socketDirMode = mode }
}

// WithLogger sets the structured logger (default slog.Default()).
func WithLogger(logger *slog.Logger) SocketServerOption {
	return func(s *SocketServer) { s.logger = logger }
}

// WithVersion sets the version string included in the startup log line.
func WithVersion(version string) SocketServerOption {
	return func(s *SocketServer) { s.version = version }
}

// WithReadHeaderTimeout bounds per-request header reads (default 10s) so
// slow or stuck clients cannot pin server slots.
func WithReadHeaderTimeout(d time.Duration) SocketServerOption {
	return func(s *SocketServer) { s.readHeaderTimeout = d }
}

// WithIdleTimeout bounds idle keep-alive connections (default 120s).
func WithIdleTimeout(d time.Duration) SocketServerOption {
	return func(s *SocketServer) { s.idleTimeout = d }
}

// WithShutdownGracePeriod bounds the graceful-drain window after the
// lifecycle context is canceled (default 5s).
func WithShutdownGracePeriod(d time.Duration) SocketServerOption {
	return func(s *SocketServer) { s.shutdownGracePeriod = d }
}

// WithHealthPath sets the endpoint probed by the stale-socket liveness check
// (default "/health").
func WithHealthPath(path string) SocketServerOption {
	return func(s *SocketServer) { s.healthPath = path }
}

// WithProbeTimeout bounds the stale-socket liveness probe (default 500ms).
func WithProbeTimeout(d time.Duration) SocketServerOption {
	return func(s *SocketServer) { s.probeTimeout = d }
}

// NewSocketServer creates a SocketServer listening on the given unix socket
// path. The path is the bare filesystem path (e.g. "/run/mydaemon/daemon.sock"),
// without a "unix://" prefix. Serve /run-derived paths: /tmp is world-writable
// and invites symlink attacks.
func NewSocketServer(socketPath string, opts ...SocketServerOption) *SocketServer {
	s := &SocketServer{
		socketPath:          socketPath,
		socketMode:          DefaultSocketMode,
		socketDirMode:       DefaultSocketDirMode,
		logger:              slog.Default(),
		readHeaderTimeout:   DefaultReadHeaderTimeout,
		idleTimeout:         DefaultIdleTimeout,
		shutdownGracePeriod: DefaultShutdownGracePeriod,
		healthPath:          DefaultHealthPath,
		probeTimeout:        DefaultProbeTimeout,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// SocketPath returns the configured socket path.
func (s *SocketServer) SocketPath() string { return s.socketPath }

// Lifecycle wires a domain handler and optional hooks into the SocketServer
// lifecycle. The ordering below is the contract, proven by the
// project-discovery-daemon shutdown-regression tests — do not reorder:
//
//  1. CleanStaleSocket: probe healthPath; remove the socket file if dead.
//  2. Startup (if set): run before the socket exists. An error is logged as
//     a warning and startup continues (flight-recorder semantics: degrade,
//     never abort). A returned cleanup runs last, after OnShutdown.
//  3. Build the http.Server (BaseContext = Start's ctx) and create the
//     socket directory, listen, and chmod the socket.
//  4. OnListen (if set): start background work (refresh loops, watchers).
//  5. sd_notify READY=1 (no-op without NOTIFY_SOCKET); watchdog keepalive at
//     WatchdogSec/2 when systemd requests one.
//  6. Serve until the context is canceled or Serve fails.
//  7. sd_notify STOPPING=1, graceful Shutdown within the grace period,
//     remove the socket file.
//  8. OnShutdown (if set): drain background work (runs on BOTH exit paths,
//     after socket removal).
//  9. Startup cleanup (if any): stop the startup-only machinery, e.g. the
//     flight recorder.
type Lifecycle struct {
	// Handler is the HTTP handler to serve. Required.
	Handler http.Handler

	// Startup runs before the socket is created. Errors are logged as
	// warnings and do not abort startup. Return a cleanup function to have
	// it run last on shutdown (after OnShutdown).
	Startup func() (cleanup func(), err error)

	// OnListen runs after the socket is listening but before sd_notify
	// READY. Start background loops here; ctx is Start's lifetime context.
	OnListen func(ctx context.Context)

	// OnShutdown drains background work. Runs on every exit path, after the
	// HTTP server has drained and the socket file was removed.
	OnShutdown func()
}

// Start runs the full daemon lifecycle (see Lifecycle for the ordering
// contract) and blocks until the context is canceled or Serve fails. On
// context cancellation the socket file is removed and nil is returned; a
// Serve error is returned after the same cleanup.
//
//nolint:cyclop // lifecycle composition root: one branch per lifecycle stage
func (s *SocketServer) Start(ctx context.Context, lifecycle Lifecycle) error {
	if lifecycle.Handler == nil {
		return ErrHandlerRequired
	}

	s.CleanStaleSocket(ctx)

	// Run the startup hook before accepting requests so e.g. trace buffers
	// capture startup activity. Non-fatal: log and continue without it.
	var startupCleanup func()

	if lifecycle.Startup != nil {
		cleanup, err := lifecycle.Startup()
		if err != nil {
			s.logger.Warn("startup hook failed; continuing without", "error", err)
		} else {
			startupCleanup = cleanup
		}
	}

	if startupCleanup != nil {
		defer startupCleanup() // runs last: after OnShutdown (matched pairs)
	}

	server := &http.Server{
		Handler: lifecycle.Handler,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		ReadHeaderTimeout: s.readHeaderTimeout,
		IdleTimeout:       s.idleTimeout,
	}

	// Ensure the socket's parent directory exists. The default path lives under
	// /run which is created by systemd/tmpfiles in production, but standalone
	// and test deployments may use custom paths whose parent doesn't exist yet.
	socketDir := filepath.Dir(s.socketPath)
	if err := os.MkdirAll(socketDir, s.socketDirMode); err != nil {
		return fmt.Errorf("create socket directory %s: %w", socketDir, err)
	}

	listenConfig := net.ListenConfig{}

	ln, err := listenConfig.Listen(ctx, "unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on unix socket %s: %w", s.socketPath, err)
	}

	if err := os.Chmod(s.socketPath, s.socketMode); err != nil {
		_ = ln.Close()

		return fmt.Errorf("restrict socket permissions: %w", err)
	}

	s.logger.Info("daemon listening", "socket", s.socketPath, "version", s.version)

	if lifecycle.OnListen != nil {
		lifecycle.OnListen(ctx)
	}

	// Notify systemd that the daemon is ready. No-op when NOTIFY_SOCKET is unset.
	if sent, err := sd.SdNotify(false, sd.SdNotifyReady); err != nil {
		s.logger.Warn("sd_notify READY failed", "error", err)
	} else if sent {
		s.logger.Debug("sd_notify READY=1 sent")
	}

	// Start watchdog keepalive if systemd configured WatchdogSec.
	watchdogInterval, watchdogErr := sd.SdWatchdogEnabled(false)
	if watchdogErr != nil {
		s.logger.Warn("sd_watchdog check failed", "error", watchdogErr)
	}

	if watchdogInterval > 0 {
		go s.watchdog(ctx, watchdogInterval/2)
	}

	errCh := make(chan error, 1)

	go func() {
		err := server.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("serve: %w", err)

			return
		}

		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		s.notifyStopping()

		s.logger.Info("daemon shutting down", "reason", ctx.Err())

		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.shutdownGracePeriod)
		defer cancel()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			s.logger.Error("daemon shutdown", "error", err)
		}

		_ = os.Remove(s.socketPath)

		s.runOnShutdown(lifecycle)

		return nil
	case err := <-errCh:
		s.notifyStopping()

		removeErr := os.Remove(s.socketPath)
		if removeErr != nil {
			s.logger.Warn("failed to remove socket on error", "path", s.socketPath, "error", removeErr)
		}

		s.runOnShutdown(lifecycle)

		return err
	}
}

// runOnShutdown invokes the OnShutdown hook exactly once per Start call.
// It must run on every exit path after the socket file was removed.
func (s *SocketServer) runOnShutdown(lifecycle Lifecycle) {
	if lifecycle.OnShutdown != nil {
		lifecycle.OnShutdown()
	}
}

// notifyStopping sends STOPPING=1 to systemd. No-op when not running under systemd.
func (s *SocketServer) notifyStopping() {
	if sent, err := sd.SdNotify(false, sd.SdNotifyStopping); err != nil {
		s.logger.Warn("sd_notify STOPPING failed", "error", err)
	} else if sent {
		s.logger.Debug("sd_notify STOPPING=1 sent")
	}
}

// watchdog sends periodic WATCHDOG=1 heartbeats to systemd at the given interval.
// Stops when the context is cancelled.
func (s *SocketServer) watchdog(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := sd.SdNotify(false, sd.SdNotifyWatchdog); err != nil {
				s.logger.Warn("sd_notify WATCHDOG failed", "error", err)
			}
		}
	}
}

// CleanStaleSocket removes the socket file when it exists but no daemon is
// answering the health probe on it. A live daemon's socket is left alone.
func (s *SocketServer) CleanStaleSocket(ctx context.Context) {
	if _, err := os.Stat(s.socketPath); err != nil {
		return
	}

	if !s.IsSocketAlive(ctx) {
		s.logger.Debug("removing stale socket", "path", s.socketPath)

		err := os.Remove(s.socketPath)
		if err != nil {
			s.logger.Warn("failed to remove stale socket", "path", s.socketPath, "error", err)
		}
	}
}

// IsSocketAlive probes whether a daemon is already answering on this
// SocketServer's socket path (GET healthPath with the configured probe timeout).
func (s *SocketServer) IsSocketAlive(ctx context.Context) bool {
	return isSocketAlive(ctx, s.socketPath, s.healthPath, s.probeTimeout)
}

// IsSocketAlive probes whether a daemon is already listening on the given
// socket path by issuing a GET for healthPath (e.g. "/health"). Uses
// DefaultProbeTimeout.
func IsSocketAlive(ctx context.Context, socketPath, healthPath string) bool {
	return isSocketAlive(ctx, socketPath, healthPath, DefaultProbeTimeout)
}

func isSocketAlive(ctx context.Context, socketPath, healthPath string, timeout time.Duration) bool {
	client := NewHTTPClient(socketPath, timeout)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost"+healthPath, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(request)
	if err != nil {
		return false
	}

	_ = resp.Body.Close()

	return true
}
