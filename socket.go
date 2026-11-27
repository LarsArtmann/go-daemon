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

// Server serves an HTTP handler with the full production daemon lifecycle:
// stale-endpoint cleanup, systemd readiness/watchdog notification, graceful
// drain, and listener cleanup on every exit path. The lifecycle ordering is
// a contract (see Lifecycle).
//
// The constructor picks the network: NewSocketServer serves a unix domain
// socket (filesystem permissions, stale-socket cleanup, unlink on exit);
// NewServer serves any other network (e.g. "tcp"); WithListener serves a
// pre-built listener with no filesystem management at all.
//
// Construct with NewSocketServer or NewServer.
type Server struct {
	network             string
	addr                string
	listener            net.Listener
	socketMode          fs.FileMode
	socketDirMode       fs.FileMode
	logger              *slog.Logger
	version             string
	readHeaderTimeout   time.Duration
	idleTimeout         time.Duration
	shutdownGracePeriod time.Duration
	healthPath          string
	probeTimeout        time.Duration

	// chmodSocket applies socketMode to the socket file: os.Chmod outside
	// tests. Tests substitute a failing function to reach the error branch,
	// because no portable unprivileged filesystem state makes chmod(2) fail
	// on a socket the process itself just created.
	chmodSocket func(string, fs.FileMode) error
}

// SocketServer is the unix-domain-socket form of Server: a type alias, so
// every SocketServer option and method works on Server and vice versa. The
// unix-specific extras (socket mode, directory creation, stale-socket
// cleanup, unlink on exit) apply only when the server itself listens on a
// unix socket and owns it (NewSocketServer or NewServer("unix", ...)), not
// for WithListener.
type SocketServer = Server

// ServerOption configures a Server.
type ServerOption func(*Server)

// SocketServerOption configures a Server; retained name for the
// unix-domain-socket form (alias of ServerOption).
type SocketServerOption = ServerOption

// WithSocketMode sets the filesystem permissions for the unix socket
// (default 0o600, owner-only). Widen deliberately — any local process that
// can write to the socket executes with the daemon user's privileges. Unix
// sockets only; ignored for other networks and WithListener.
func WithSocketMode(mode fs.FileMode) ServerOption {
	return func(s *SocketServer) { s.socketMode = mode }
}

// WithSocketDirMode sets the permissions for the socket's parent directory
// (default 0o755), created on demand when it does not exist. Unix sockets
// only; ignored for other networks and WithListener.
func WithSocketDirMode(mode fs.FileMode) ServerOption {
	return func(s *SocketServer) { s.socketDirMode = mode }
}

// WithLogger sets the structured logger (default slog.Default()).
func WithLogger(logger *slog.Logger) ServerOption {
	return func(s *SocketServer) { s.logger = logger }
}

// WithVersion sets the version string included in the startup log line.
func WithVersion(version string) ServerOption {
	return func(s *SocketServer) { s.version = version }
}

// WithReadHeaderTimeout bounds per-request header reads (default 10s) so
// slow or stuck clients cannot pin server slots.
func WithReadHeaderTimeout(d time.Duration) ServerOption {
	return func(s *SocketServer) { s.readHeaderTimeout = d }
}

// WithIdleTimeout bounds idle keep-alive connections (default 120s).
func WithIdleTimeout(d time.Duration) ServerOption {
	return func(s *SocketServer) { s.idleTimeout = d }
}

// WithShutdownGracePeriod bounds the graceful-drain window after the
// lifecycle context is canceled (default 5s).
func WithShutdownGracePeriod(d time.Duration) ServerOption {
	return func(s *SocketServer) { s.shutdownGracePeriod = d }
}

// WithHealthPath sets the endpoint probed by the stale-socket liveness check
// (default "/health"). Unix sockets only; ignored for other networks and
// WithListener.
func WithHealthPath(path string) ServerOption {
	return func(s *SocketServer) { s.healthPath = path }
}

// WithProbeTimeout bounds the stale-socket liveness probe (default 500ms).
// Unix sockets only; ignored for other networks and WithListener.
func WithProbeTimeout(d time.Duration) ServerOption {
	return func(s *SocketServer) { s.probeTimeout = d }
}

// WithListener supplies a pre-built net.Listener. Start serves on it as-is
// and skips all filesystem socket management (stale-socket cleanup,
// directory creation, chmod, unlink): the caller owns the listener's full
// lifecycle, including unlinking a unix socket after shutdown. The
// constructor's address is ignored for serving.
func WithListener(ln net.Listener) ServerOption {
	return func(s *Server) { s.listener = ln }
}

// NewServer creates a Server listening on the given network and address
// (e.g. NewServer("tcp", "127.0.0.1:8080")). The network is passed straight
// to net.Listen at Start; an invalid network surfaces as a Start error, not
// a construction error. Unix-specific options apply only for network "unix".
func NewServer(network, addr string, opts ...ServerOption) *Server {
	s := &Server{
		network:             network,
		addr:                addr,
		socketMode:          DefaultSocketMode,
		socketDirMode:       DefaultSocketDirMode,
		chmodSocket:         os.Chmod,
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

// NewSocketServer creates a Server listening on the unix socket at the given
// path. The path is the bare filesystem path (e.g. "/run/mydaemon/daemon.sock"),
// without a "unix://" prefix. Serve /run-derived paths: /tmp is world-writable
// and invites symlink attacks.
func NewSocketServer(socketPath string, opts ...SocketServerOption) *SocketServer {
	return NewServer("unix", socketPath, opts...)
}

// SocketPath returns the configured unix socket path (or address string for
// other networks).
func (s *Server) SocketPath() string { return s.addr }

// Lifecycle wires a domain handler and optional hooks into the Server
// lifecycle. The ordering below is the contract, proven by the
// project-discovery-daemon shutdown-regression tests and the TCP
// hook-ordering test — do not reorder:
//
//  1. CleanStaleSocket: probe healthPath; remove the socket file if dead
//     (unix-owned sockets only).
//  2. Startup (if set): run before the socket exists. An error is logged as
//     a warning and startup continues (flight-recorder semantics: degrade,
//     never abort). A returned cleanup runs last, after OnShutdown.
//  3. Build the http.Server (BaseContext = Start's ctx), listen, and — for
//     unix-owned sockets — create the socket directory and chmod the socket.
//  4. OnListen (if set): start background work (refresh loops, watchers).
//  5. sd_notify READY=1 (no-op without NOTIFY_SOCKET); watchdog keepalive at
//     WatchdogSec/2 when systemd requests one.
//  6. Serve until the context is canceled or Serve fails.
//  7. sd_notify STOPPING=1, graceful Shutdown within the grace period,
//     remove the socket file (unix-owned sockets only).
//  8. OnShutdown (if set): drain background work (runs on BOTH exit paths,
//     after the listener is gone and the socket file was removed).
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
// context cancellation the unix socket file is removed (when the server owns
// it) and nil is returned; a Serve error is returned after the same cleanup.
//
//nolint:cyclop // lifecycle composition root: one branch per lifecycle stage
func (s *Server) Start(ctx context.Context, lifecycle Lifecycle) error {
	if lifecycle.Handler == nil {
		return ErrHandlerRequired
	}

	if s.managesUnixSocket() {
		s.CleanStaleSocket(ctx)
	}

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

	ln, err := s.listen(ctx)
	if err != nil {
		return err
	}

	s.logger.Info("daemon listening", "network", s.network, "addr", ln.Addr().String(), "version", s.version)

	if lifecycle.OnListen != nil {
		lifecycle.OnListen(ctx)
	}

	// Notify systemd that the daemon is ready. No-op when NOTIFY_SOCKET is unset.
	NotifyReady(s.logger)

	// Start watchdog keepalive if systemd configured WatchdogSec.
	if interval, ok := WatchdogInterval(s.logger); ok {
		go Watchdog(ctx, s.logger, interval)
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
		NotifyStopping(s.logger)

		s.logger.Info("daemon shutting down", "reason", ctx.Err())

		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.shutdownGracePeriod)
		defer cancel()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			s.logger.Error("daemon shutdown", "error", err)
		}

		if s.managesUnixSocket() {
			_ = os.Remove(s.addr)
		}

		s.runOnShutdown(lifecycle)

		return nil
	case err := <-errCh:
		NotifyStopping(s.logger)

		if s.managesUnixSocket() {
			removeErr := os.Remove(s.addr)
			if removeErr != nil {
				s.logger.Warn("failed to remove socket on error", "path", s.addr, "error", removeErr)
			}
		}

		s.runOnShutdown(lifecycle)

		return err
	}
}

// managesUnixSocket reports whether Start owns the unix socket file
// lifecycle (stale cleanup, mkdir, chmod, unlink): only when the server
// listens on "unix" AND the listener was not supplied by the caller.
func (s *Server) managesUnixSocket() bool {
	return s.listener == nil && s.network == "unix"
}

// listen returns the listener Start serves on: a caller-supplied one
// (WithListener) as-is, or a freshly created one. Unix-owned sockets get
// their parent directory created and permissions restricted here.
func (s *Server) listen(ctx context.Context) (net.Listener, error) {
	if s.listener != nil {
		return s.listener, nil
	}

	if s.managesUnixSocket() {
		// Ensure the socket's parent directory exists. The default path lives under
		// /run which is created by systemd/tmpfiles in production, but standalone
		// and test deployments may use custom paths whose parent doesn't exist yet.
		socketDir := filepath.Dir(s.addr)
		if err := os.MkdirAll(socketDir, s.socketDirMode); err != nil {
			return nil, fmt.Errorf("create socket directory %s: %w", socketDir, err)
		}
	}

	listenConfig := net.ListenConfig{}

	ln, err := listenConfig.Listen(ctx, s.network, s.addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s %s: %w", s.network, s.addr, err)
	}

	if s.managesUnixSocket() {
		if err := s.chmodSocket(s.addr, s.socketMode); err != nil {
			_ = ln.Close()

			return nil, fmt.Errorf("restrict socket permissions: %w", err)
		}
	}

	return ln, nil
}

// runOnShutdown invokes the OnShutdown hook exactly once per Start call.
// It must run on every exit path after the socket file was removed.
func (s *Server) runOnShutdown(lifecycle Lifecycle) {
	if lifecycle.OnShutdown != nil {
		lifecycle.OnShutdown()
	}
}

// CleanStaleSocket removes the socket file when it exists but no daemon is
// answering the health probe on it. A live daemon's socket is left alone.
func (s *Server) CleanStaleSocket(ctx context.Context) {
	if _, err := os.Stat(s.addr); err != nil {
		return
	}

	if !s.IsSocketAlive(ctx) {
		s.logger.Debug("removing stale socket", "path", s.addr)

		err := os.Remove(s.addr)
		if err != nil {
			s.logger.Warn("failed to remove stale socket", "path", s.addr, "error", err)
		}
	}
}

// IsSocketAlive probes whether a daemon is already answering on this
// SocketServer's socket path (GET healthPath with the configured probe timeout).
func (s *Server) IsSocketAlive(ctx context.Context) bool {
	return isSocketAlive(ctx, s.addr, s.healthPath, s.probeTimeout)
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
