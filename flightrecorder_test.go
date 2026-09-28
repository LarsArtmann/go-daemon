package daemon

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// flightRecorderMu serializes tests that hold an active flight recorder: Go's
// runtime/trace allows only ONE active recorder per process, so concurrent
// newTestFlightRecorder calls fail with "another flight recorder is already
// active". Held from creation until the test's cleanup (after Stop).
var flightRecorderMu sync.Mutex

// newTestFlightRecorder creates a started flight recorder for testing.
// The caller is responsible for calling Stop (t.Cleanup is recommended).
func newTestFlightRecorder(t *testing.T, opts FlightRecorderOptions) *FlightRecorder {
	t.Helper()

	flightRecorderMu.Lock()

	// Registered before the Stop cleanup so LIFO runs it after Stop releases
	// the process-global recorder slot.
	t.Cleanup(flightRecorderMu.Unlock)

	if err := opts.Validate(); err != nil {
		t.Fatalf("validate options: %v", err)
	}

	fr, err := NewFlightRecorder(
		opts,
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		prometheus.NewRegistry(),
	)
	if err != nil {
		t.Fatalf("create flight recorder: %v", err)
	}

	if err := fr.Start(); err != nil {
		t.Fatalf("start flight recorder: %v", err)
	}

	t.Cleanup(fr.Stop)

	return fr
}

// generateTraceEvents runs a brief CPU-bound goroutine to produce execution
// trace events in the flight recorder's ring buffer, replacing arbitrary
// time.Sleep calls that relied on wall-clock time for trace accumulation.
func generateTraceEvents() {
	done := make(chan struct{})

	go func() {
		defer close(done)

		var sum int

		for i := range 100_000 {
			sum += i * i
		}

		_ = sum
	}()

	<-done
}

func TestFlightRecorderOptions_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts FlightRecorderOptions
		want error
	}{
		{"zero values valid", FlightRecorderOptions{}, nil},
		{"negative MinAge", FlightRecorderOptions{MinAge: -1}, ErrFRNegativeMinAge},
		{
			"negative SlowRequestThreshold",
			FlightRecorderOptions{SlowRequestThreshold: -1},
			ErrFRNegativeSlowRequestThreshold,
		},
		{"negative MaxSnapshots", FlightRecorderOptions{MaxSnapshots: -1}, ErrFRNegativeMaxSnapshots},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.opts.Validate()
			if !errors.Is(err, tt.want) {
				t.Fatalf("Validate(): got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestFlightRecorder_SnapshotToBuffer(t *testing.T) {
	fr := newTestFlightRecorder(t, FlightRecorderOptions{})

	generateTraceEvents()

	var buf bytes.Buffer
	if err := fr.Snapshot(context.Background(), &buf); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("snapshot buffer is empty; expected trace data")
	}
}

func TestFlightRecorder_SnapshotToFile(t *testing.T) {
	dir := t.TempDir()
	fr := newTestFlightRecorder(t, FlightRecorderOptions{SnapshotDir: dir})

	generateTraceEvents()

	path, err := fr.SnapshotToFile(context.Background())
	if err != nil {
		t.Fatalf("SnapshotToFile: %v", err)
	}

	if !strings.HasPrefix(path, dir) {
		t.Fatalf("snapshot path %q not in configured dir %q", path, dir)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat snapshot file: %v", err)
	}

	if info.Size() == 0 {
		t.Fatal("snapshot file is empty")
	}

	if !strings.HasSuffix(path, ".trace") {
		t.Errorf("snapshot file should have .trace extension, got %q", path)
	}
}

func TestFlightRecorder_SlowRequestTrigger(t *testing.T) {
	dir := t.TempDir()
	fr := newTestFlightRecorder(t, FlightRecorderOptions{
		SlowRequestThreshold: 1 * time.Nanosecond, // any duration triggers
		SnapshotDir:          dir,
	})

	generateTraceEvents()

	fr.MaybeCaptureOnSlowRequest(context.Background(), 100*time.Millisecond)

	// The snapshot write happens in a goroutine; poll for a non-empty file.
	deadline := time.Now().Add(2 * time.Second)

	found := false

	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("ReadDir: %v", err)
		}

		if len(entries) > 0 {
			if len(entries) > 1 {
				t.Fatalf("expected exactly 1 auto-triggered file, got %d", len(entries))
			}

			info, err := entries[0].Info()
			if err != nil {
				t.Fatalf("stat snapshot file: %v", err)
			}

			// The file may be created but not yet written to (race between
			// os.Create and WriteTo in the goroutine). Keep polling until
			// it has content rather than failing on an empty file.
			if info.Size() > 0 {
				found = true

				break
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	if !found {
		t.Fatal("no snapshot file appeared within timeout")
	}
}

// TestFlightRecorder_SlowRequestTrigger_CancelledContext is a regression test
// for the production bug where the capture goroutine inherited the caller's
// request context: net/http cancels that context the moment the handler
// returns, so the async capture raced handler teardown and was silently
// dropped whenever the goroutine lost (observed as load-dependent flakiness in
// project-discovery-daemon). MaybeCaptureOnSlowRequest must detach the context
// from cancellation so the capture survives the caller returning.
func TestFlightRecorder_SlowRequestTrigger_CancelledContext(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fr := newTestFlightRecorder(t, FlightRecorderOptions{
		SlowRequestThreshold: 1 * time.Nanosecond,
		SnapshotDir:          dir,
	})

	generateTraceEvents()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fr.MaybeCaptureOnSlowRequest(ctx, 100*time.Millisecond)

	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("ReadDir: %v", err)
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.Size() > 0 {
				return
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("capture was dropped: no snapshot file appeared despite cancelled caller context")
}

func TestFlightRecorder_SlowRequestTrigger_OneShot(t *testing.T) {
	dir := t.TempDir()
	fr := newTestFlightRecorder(t, FlightRecorderOptions{
		SlowRequestThreshold: 1 * time.Nanosecond,
		SnapshotDir:          dir,
	})

	generateTraceEvents()

	// Trigger multiple times — only the first should produce a file.
	fr.MaybeCaptureOnSlowRequest(context.Background(), 100*time.Millisecond)
	fr.MaybeCaptureOnSlowRequest(context.Background(), 200*time.Millisecond)
	fr.MaybeCaptureOnSlowRequest(context.Background(), 300*time.Millisecond)

	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("ReadDir: %v", err)
		}

		if len(entries) > 0 {
			if len(entries) > 1 {
				t.Fatalf("expected at most 1 file (one-shot), got %d", len(entries))
			}

			return
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func TestFlightRecorder_BelowThreshold_NoCapture(t *testing.T) {
	dir := t.TempDir()
	fr := newTestFlightRecorder(t, FlightRecorderOptions{
		SlowRequestThreshold: 1 * time.Hour,
		SnapshotDir:          dir,
	})

	fr.MaybeCaptureOnSlowRequest(context.Background(), 1*time.Millisecond)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("expected 0 files (below threshold), got %d", len(entries))
	}
}

func TestFlightRecorder_DisabledThreshold(t *testing.T) {
	dir := t.TempDir()
	fr := newTestFlightRecorder(t, FlightRecorderOptions{
		SlowRequestThreshold: 0, // disabled
		SnapshotDir:          dir,
	})

	fr.MaybeCaptureOnSlowRequest(context.Background(), 1*time.Hour)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("expected 0 files (threshold disabled), got %d", len(entries))
	}
}

func TestFlightRecorder_Enabled(t *testing.T) {
	var fr *FlightRecorder

	if fr.Enabled() {
		t.Fatal("nil FlightRecorder should not be Enabled")
	}

	fr = newTestFlightRecorder(t, FlightRecorderOptions{})
	if !fr.Enabled() {
		t.Fatal("started FlightRecorder should be Enabled")
	}

	fr.Stop()

	if fr.Enabled() {
		t.Fatal("stopped FlightRecorder should not be Enabled")
	}
}

func TestFlightRecorder_StopNilSafe(t *testing.T) {
	var fr *FlightRecorder
	fr.Stop() // should not panic
}

func TestFlightRecorder_ConcurrentSnapshots(t *testing.T) {
	fr := newTestFlightRecorder(t, FlightRecorderOptions{})

	generateTraceEvents()

	const goroutines = 4

	errs := make(chan error, goroutines)
	results := make(chan int, goroutines)

	for range goroutines {
		go func() {
			var buf bytes.Buffer
			if err := fr.Snapshot(context.Background(), &buf); err != nil {
				errs <- err

				return
			}

			results <- buf.Len()
		}()
	}

	for range goroutines {
		select {
		case err := <-errs:
			t.Fatalf("concurrent Snapshot: %v", err)
		case n := <-results:
			if n == 0 {
				t.Fatal("concurrent snapshot returned no data")
			}
		}
	}
}

func TestFlightRecorder_CompressedSnapshot(t *testing.T) {
	dir := t.TempDir()
	fr := newTestFlightRecorder(t, FlightRecorderOptions{
		SnapshotDir:       dir,
		CompressSnapshots: true,
	})

	generateTraceEvents()

	path, err := fr.SnapshotToFile(context.Background())
	if err != nil {
		t.Fatalf("SnapshotToFile: %v", err)
	}

	if !strings.HasSuffix(path, ".trace.gz") {
		t.Fatalf("compressed snapshot should use .trace.gz extension, got %q", path)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}
	defer func() { _ = file.Close() }()

	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer func() { _ = gz.Close() }()

	data, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("read decompressed snapshot: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("decompressed snapshot is empty")
	}
}

// errWriteFailed is the sentinel error failingWriter fails with.
var errWriteFailed = errors.New("write failed")

// failingWriter fails every write.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWriteFailed }

func TestFlightRecorder_SnapshotFailingWriter(t *testing.T) {
	fr := newTestFlightRecorder(t, FlightRecorderOptions{})

	generateTraceEvents()

	err := fr.Snapshot(context.Background(), failingWriter{})
	if err == nil {
		t.Fatal("expected error from failing writer")
	}

	if !strings.Contains(err.Error(), "write flight recorder snapshot") {
		t.Fatalf("error should be wrapped: got %v", err)
	}
}

func TestFlightRecorder_MetricPrefix(t *testing.T) {
	t.Parallel()

	flightRecorderMu.Lock()

	// Runs after fr.Stop (LIFO) so the process-global recorder slot is free
	// before the next recorder test starts.
	t.Cleanup(flightRecorderMu.Unlock)

	reg := prometheus.NewRegistry()

	fr, err := NewFlightRecorder(
		FlightRecorderOptions{MetricPrefix: "project_discovery"},
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		reg,
	)
	if err != nil {
		t.Fatalf("create flight recorder: %v", err)
	}

	t.Cleanup(fr.Stop)

	if err := fr.Start(); err != nil {
		t.Fatalf("start flight recorder: %v", err)
	}

	generateTraceEvents()

	// A CounterVec registers no metric family until a child is observed;
	// exercise a capture so the snapshots counter exists.
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/debug/trace", nil)
	fr.TraceHandler(0).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("trace handler: got %d", rec.Code)
	}

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	names := map[string]bool{}
	for _, mf := range mfs {
		names[mf.GetName()] = true
	}

	for _, want := range []string{
		"project_discovery_flight_recorder_snapshots_total",
		"project_discovery_flight_recorder_snapshot_duration_seconds",
	} {
		if !names[want] {
			t.Fatalf("expected metric %q, registered: %v", want, names)
		}
	}
}

func TestTraceHandler_NotEnabled(t *testing.T) {
	t.Parallel()

	var fr *FlightRecorder

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/debug/trace", nil)

	fr.TraceHandler(0).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 (flight recorder not enabled), got %d", rec.Code)
	}
}

func TestTraceHandler_Enabled(t *testing.T) {
	fr := newTestFlightRecorder(t, FlightRecorderOptions{})

	generateTraceEvents()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/debug/trace", nil)

	fr.TraceHandler(0).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("content-type: got %q", ct)
	}

	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, `filename="snapshot-`) {
		t.Fatalf("content-disposition: got %q", cd)
	}

	if rec.Body.Len() == 0 {
		t.Fatal("trace body is empty")
	}
}

func TestTraceHandler_RateLimited(t *testing.T) {
	fr := newTestFlightRecorder(t, FlightRecorderOptions{})

	generateTraceEvents()

	handler := fr.TraceHandler(150 * time.Millisecond)

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/debug/trace", nil))

	if first.Code != http.StatusOK {
		t.Fatalf("first request: got %d", first.Code)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/debug/trace", nil))

	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request should be rate limited, got %d", second.Code)
	}

	if ra := second.Header().Get("Retry-After"); ra == "" {
		t.Fatal("rate-limited response must set Retry-After")
	}

	// After the window elapses, requests succeed again.
	time.Sleep(160 * time.Millisecond)

	third := httptest.NewRecorder()
	handler.ServeHTTP(third, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/debug/trace", nil))

	if third.Code != http.StatusOK {
		t.Fatalf("third request after window: got %d", third.Code)
	}
}

func TestTraceHandler_ConcurrentRateLimit(t *testing.T) {
	fr := newTestFlightRecorder(t, FlightRecorderOptions{})

	generateTraceEvents()

	handler := fr.TraceHandler(time.Hour) // effectively one allowed per lifetime

	const requests = 8

	codes := make([]int, requests)
	wg := sync.WaitGroup{}

	for i := range requests {
		wg.Go(func() {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(
				rec,
				httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/debug/trace", nil),
			)
			codes[i] = rec.Code
		})
	}

	wg.Wait()

	ok, limited := 0, 0

	for _, code := range codes {
		switch code {
		case http.StatusOK:
			ok++
		case http.StatusTooManyRequests:
			limited++
		default:
			t.Fatalf("unexpected status %d", code)
		}
	}

	if ok != 1 || limited != requests-1 {
		t.Fatalf("exactly 1 request should pass the rate limit: got %d ok, %d limited", ok, limited)
	}
}
