package daemon

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	flightrecorder "github.com/larsartmann/go-flightrecorder"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	defaultFRMinAge   = 30 * time.Second
	defaultFRMaxBytes = 1 << 20 // 1 MiB

	// defaultFRMetricPrefix namespaces the flight-recorder Prometheus metrics.
	defaultFRMetricPrefix = "daemon"

	// defaultFRTriggerKind labels slow-request auto-trigger snapshots in
	// flight-recorder logs and metric hooks.
	defaultFRTriggerKind = "request"
)

// Sentinel errors for FlightRecorderOptions.Validate.
var (
	ErrFRNegativeMinAge               = errors.New("flight recorder MinAge must not be negative")
	ErrFRNegativeSlowRequestThreshold = errors.New("flight recorder SlowRequestThreshold must not be negative")
	ErrFRNegativeMaxSnapshots         = errors.New("flight recorder MaxSnapshots must not be negative")
)

// FlightRecorderOptions configures the execution-trace flight recorder.
//
// The flight recorder buffers the last few seconds of Go runtime execution trace
// data in memory. On demand (via the trace HTTP handler) or automatically (when
// a request exceeds SlowRequestThreshold), the buffer can be snapshotted to
// capture the goroutine scheduling, blocking, and interaction state leading up
// to the capture moment.
//
// This is invaluable for diagnosing latency spikes, goroutine leaks, and
// contention issues in long-running daemon instances — problems that are
// invisible to logs and metrics alone.
//
// Analyze captured traces with:
//
//	go tool trace snapshot-<timestamp>.trace
//
// See: https://go.dev/blog/flight-recorder
type FlightRecorderOptions struct {
	// MinAge is the minimum duration of trace data retained in the buffer.
	// Set to ~2x the expected problem window (e.g., if debugging a 5s timeout,
	// set to 10s). Default: 30s.
	MinAge time.Duration

	// MaxBytes bounds the in-memory buffer size to prevent unbounded memory
	// growth. Average trace production is a few MB/s for a busy service.
	// Default: 1 MiB.
	MaxBytes uint64

	// SlowRequestThreshold triggers an automatic snapshot written to
	// SnapshotDir when a request exceeds this duration. Only the first slow
	// request per server lifetime triggers a capture (subsequent slow
	// requests are suppressed to avoid disk flooding). Use the manual trace
	// endpoint for additional captures. 0 disables auto-trigger. Default: 0.
	SlowRequestThreshold time.Duration

	// SnapshotDir is the directory where auto-triggered snapshots are written.
	// Empty defaults to os.TempDir(). The directory is created if it does not
	// exist.
	SnapshotDir string

	// CompressSnapshots enables gzip compression for auto-triggered snapshot
	// files. Compressed files use the .trace.gz extension and are typically
	// ~10x smaller (1 MiB → ~100 KB). Compatible with `go tool trace` (Go 1.19+).
	// Default: false.
	CompressSnapshots bool

	// MaxSnapshots limits the number of snapshot files retained in SnapshotDir.
	// When set (>0), the oldest files beyond this count are removed after each
	// capture and during startup. 0 disables retention (unlimited).
	// Default: 0.
	MaxSnapshots int

	// MetricPrefix namespaces the registered Prometheus metrics as
	// "<prefix>_flight_recorder_snapshots_total" and
	// "<prefix>_flight_recorder_snapshot_duration_seconds". Must be a valid
	// Prometheus name fragment (letters, digits, underscores). Empty defaults
	// to "daemon".
	MetricPrefix string

	// TriggerKind labels slow-request auto-trigger snapshots in
	// flight-recorder logs (TriggerContext.Kind). Empty defaults to "request".
	TriggerKind string
}

func (o FlightRecorderOptions) withDefaults() FlightRecorderOptions {
	if o.MinAge == 0 {
		o.MinAge = defaultFRMinAge
	}

	if o.MaxBytes == 0 {
		o.MaxBytes = defaultFRMaxBytes
	}

	if o.MetricPrefix == "" {
		o.MetricPrefix = defaultFRMetricPrefix
	}

	if o.TriggerKind == "" {
		o.TriggerKind = defaultFRTriggerKind
	}

	return o
}

// Validate checks the options for invalid values that would otherwise be
// silently corrected by withDefaults. Returns an error if any duration field
// is negative (which has no meaningful interpretation and may indicate a
// configuration mistake). Zero values are valid (treated as "use default").
//
// Call Validate before constructing the FlightRecorder to surface
// misconfiguration early.
func (o FlightRecorderOptions) Validate() error {
	if o.MinAge < 0 {
		return ErrFRNegativeMinAge
	}

	if o.SlowRequestThreshold < 0 {
		return ErrFRNegativeSlowRequestThreshold
	}

	if o.MaxSnapshots < 0 {
		return ErrFRNegativeMaxSnapshots
	}

	return nil
}

func (o FlightRecorderOptions) snapshotDir() string {
	if o.SnapshotDir != "" {
		return o.SnapshotDir
	}

	return os.TempDir()
}

// FlightRecorder wraps github.com/larsartmann/go-flightrecorder.Recorder with
// daemon-serving concerns: a rate-limited trace HTTP handler, Prometheus
// metrics, one-shot slow-request auto-trigger with structured logging, and
// nil-safe lifecycle methods.
//
// The underlying Recorder handles: lifecycle (Start/Stop/Enabled),
// once-semantics, gzip compression, snapshot retention/cleanup, and the
// MetricsHook/LoggerHook integration points. This wrapper adds: the
// slow-request auto-trigger pattern (goroutine-spawned, one-shot per server
// lifetime), the rate-limited download endpoint, and Prometheus registration.
type FlightRecorder struct {
	recorder *flightrecorder.Recorder
	logger   *slog.Logger

	// minAge and maxBytes are stored for logging in Start; the values are
	// passed to the library via options but the library does not expose config
	// accessors.
	minAge   time.Duration
	maxBytes uint64

	// slowThreshold is the duration above which MaybeCaptureOnSlowRequest
	// triggers a snapshot. 0 means auto-trigger is disabled.
	slowThreshold time.Duration

	// triggerKind labels slow-request auto-trigger snapshots.
	triggerKind string

	// autoTriggered ensures only one auto-triggered file capture per server
	// lifetime. CompareAndSwap is non-blocking so the request handler is never
	// stalled by file I/O. The library's SnapshotIfAsync provides goroutine
	// management and shutdown safety; this flag provides the one-shot guard.
	autoTriggered atomic.Bool

	// lastTraceNanos tracks the last successful trace-handler snapshot
	// (UnixNano). Used with the handler's rate limit to enforce a minimum
	// interval between trace requests, preventing DoS via repeated captures
	// (each copies up to 1 MiB of trace data).
	lastTraceNanos atomic.Int64
}

// NewFlightRecorder creates a flight recorder from the given options and
// registers its Prometheus metrics on reg. The recorder is created but not
// yet started; call Start to begin collecting trace data.
func NewFlightRecorder(
	opts FlightRecorderOptions,
	logger *slog.Logger,
	reg *prometheus.Registry,
) (*FlightRecorder, error) {
	opts = opts.withDefaults()

	//nolint:exhaustruct // standalone metric; Namespace/Subsystem/Unit unset by convention
	snapshotsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: opts.MetricPrefix + "_flight_recorder_snapshots_total",
		Help: "Total number of flight recorder snapshots captured.",
	}, []string{"type"})
	//nolint:exhaustruct // standalone metric; advanced native-histogram fields are opt-in
	snapshotDuration := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    opts.MetricPrefix + "_flight_recorder_snapshot_duration_seconds",
		Help:    "Time taken to capture a flight recorder snapshot.",
		Buckets: []float64{0.001, 0.01, 0.05, 0.1, 0.5, 1.0},
	})
	reg.MustRegister(snapshotsTotal, snapshotDuration)

	// Map daemon options to library options.
	frOpts := []flightrecorder.Option{
		flightrecorder.WithMinAge(opts.MinAge),
		flightrecorder.WithMaxBytes(opts.MaxBytes),
		flightrecorder.WithLogger(func(format string, args ...any) {
			logger.Info(fmt.Sprintf(format, args...))
		}),
		flightrecorder.WithMetrics(func(event flightrecorder.SnapshotEvent, err error) {
			if err != nil {
				logger.Error("flight recorder snapshot failed",
					"error", err,
					"source", event.Source,
				)

				return
			}

			snapshotsTotal.WithLabelValues(event.Source).Inc()
			snapshotDuration.Observe(event.Duration.Seconds())

			// MaybeCaptureOnSlowRequest fires asynchronously via
			// SnapshotIfAsync; log the path here because the caller returns
			// before the write completes.
			if event.Source == flightrecorder.SnapshotSourceAsync && event.Path != "" {
				logger.Warn("flight recorder auto-snapshot captured",
					"path", event.Path,
					"duration", event.Duration,
					"bytes", event.Bytes,
					"hint", "analyze with: go tool trace "+event.Path,
				)
			}
		}),
	}

	dir := opts.snapshotDir()
	if dir != "" {
		frOpts = append(frOpts, flightrecorder.WithSnapshotDir(dir))
	}

	if opts.CompressSnapshots {
		frOpts = append(frOpts, flightrecorder.WithCompression(gzip.DefaultCompression))
	}

	if opts.MaxSnapshots > 0 {
		frOpts = append(frOpts, flightrecorder.WithMaxSnapshots(opts.MaxSnapshots))
	}

	recorder, err := flightrecorder.New(frOpts...)
	if err != nil {
		return nil, fmt.Errorf("create flight recorder: %w", err)
	}

	return &FlightRecorder{ //nolint:exhaustruct // autoTriggered/lastTraceNanos intentionally zero-valued
		recorder:      recorder,
		logger:        logger,
		minAge:        opts.MinAge,
		maxBytes:      opts.MaxBytes,
		slowThreshold: opts.SlowRequestThreshold,
		triggerKind:   opts.TriggerKind,
	}, nil
}

// Start begins collecting execution trace data. Returns an error if another
// flight recorder is already active in the process.
func (fr *FlightRecorder) Start() error {
	if err := fr.recorder.Start(); err != nil {
		return fmt.Errorf("start flight recorder: %w", err)
	}

	fr.logger.Info("flight recorder config",
		"min_age", fr.minAge,
		"max_bytes", fr.maxBytes,
		"slow_request_threshold", fr.slowThreshold,
	)

	return nil
}

// Snapshot writes the current trace buffer contents to w. Can be called
// multiple times; each call captures the buffer state at that moment.
func (fr *FlightRecorder) Snapshot(ctx context.Context, w io.Writer) error {
	if _, err := fr.recorder.SnapshotToWriter(ctx, w); err != nil {
		return fmt.Errorf("write flight recorder snapshot: %w", err)
	}

	return nil
}

// SnapshotToFile writes the trace buffer to a timestamped file in SnapshotDir
// and returns the file path. When CompressSnapshots is enabled, the file is
// gzip-compressed with a .trace.gz extension (compatible with `go tool trace`
// since Go 1.19+).
func (fr *FlightRecorder) SnapshotToFile(ctx context.Context) (string, error) {
	path, err := fr.recorder.SnapshotToDir(ctx)
	if err != nil {
		return "", fmt.Errorf("write flight recorder snapshot to dir: %w", err)
	}

	return path, nil
}

// MaybeCaptureOnSlowRequest captures a snapshot to file if the given duration
// exceeds the configured SlowRequestThreshold. Only the first call that
// exceeds the threshold triggers a capture; subsequent calls are suppressed
// via an atomic flag. The file write happens in a goroutine so the request
// handler is never blocked by I/O.
func (fr *FlightRecorder) MaybeCaptureOnSlowRequest(ctx context.Context, duration time.Duration) {
	if fr.slowThreshold == 0 || duration <= fr.slowThreshold {
		return
	}

	if !fr.autoTriggered.CompareAndSwap(false, true) {
		return
	}

	// OnAlways because we validated the threshold above. SnapshotIfAsync
	// handles goroutine spawning, WaitGroup tracking, and shutdown safety.
	// The MetricsHook closure (wired in NewFlightRecorder) logs the path
	// after the async write completes.
	fr.recorder.SnapshotIfAsync(
		ctx,
		flightrecorder.TriggerContext{
			Kind:     fr.triggerKind,
			Type:     "slow_request",
			Duration: duration,
			Err:      nil,
		},
		flightrecorder.OnAlways(),
	)
}

// Enabled reports whether the flight recorder is active. Nil-safe.
func (fr *FlightRecorder) Enabled() bool {
	if fr == nil || fr.recorder == nil {
		return false
	}

	return fr.recorder.Enabled()
}

// Stop shuts down the flight recorder and releases its buffer. Safe to call
// on a nil receiver (no-op). The library drains in-flight async captures
// before stopping.
func (fr *FlightRecorder) Stop() {
	if fr == nil || fr.recorder == nil {
		return
	}

	fr.recorder.Stop()
}

// TraceHandler returns an HTTP handler that captures an execution-trace
// snapshot and streams it as a downloadable .trace file. Returns 404 when the
// flight recorder is not enabled.
//
// The snapshot is buffered in memory before writing to the ResponseWriter so
// that capture errors are reported as HTTP 500 (before headers are sent) and
// the client always receives a complete trace with a Content-Length header.
// The buffer is at most MaxBytes (default 1 MiB).
//
// rateLimit enforces a minimum interval between requests (each Snapshot()
// copies up to 1 MiB of trace data and acquires a mutex; uncontrolled requests
// can degrade daemon responsiveness). 0 disables rate limiting. Rate-limit
// state is shared across all handlers derived from this FlightRecorder.
//
// The output is consumable by `go tool trace <file>`.
func (fr *FlightRecorder) TraceHandler(rateLimit time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !fr.Enabled() {
			WriteErrorf(w, http.StatusNotFound, "flight recorder not enabled")

			return
		}

		// Rate-limit trace snapshots atomically: each Snapshot() copies up to
		// 1 MiB of trace data and acquires a mutex. A CompareAndSwap loop
		// prevents the TOCTOU race where two concurrent requests both pass
		// the elapsed check before either stores its timestamp.
		if rateLimit > 0 {
			now := time.Now().UnixNano()

			for {
				last := fr.lastTraceNanos.Load()
				elapsed := time.Duration(now - last)

				if elapsed < rateLimit {
					retry := (rateLimit - elapsed).Round(time.Second)
					w.Header().Set("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
					WriteErrorf(w, http.StatusTooManyRequests,
						"trace endpoint rate limited; retry in %s", retry)

					return
				}

				if fr.lastTraceNanos.CompareAndSwap(last, now) {
					break
				}
			}
		}

		var buf bytes.Buffer

		if err := fr.Snapshot(r.Context(), &buf); err != nil {
			fr.logger.Error("flight recorder snapshot failed", "error", err)
			WriteErrorf(w, http.StatusInternalServerError, "trace capture failed")

			return
		}

		filename := fmt.Sprintf("snapshot-%d.trace", time.Now().UnixNano())

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))

		if _, err := w.Write(buf.Bytes()); err != nil {
			fr.logger.Error("flight recorder snapshot write failed", "error", err)
		}
	}
}
