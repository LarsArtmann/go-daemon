package daemon

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

const (
	// DefaultMaxSSEBytes bounds a single SSE event payload. The default
	// bufio.Scanner buffer (64KB) would silently truncate large events
	// (e.g. discovery results containing many enriched projects).
	DefaultMaxSSEBytes = 1 << 20 // 1 MiB

	// sseScannerInitCap is the initial buffer capacity for the SSE scanner,
	// avoiding repeated reallocations as it grows toward maxEventBytes.
	sseScannerInitCap = 64 * 1024
)

// ParseSSEData reads an SSE stream from r, parses events per the WHATWG
// Server-Sent Events spec, and emits each event's joined data payload via
// emit. Payload decoding (JSON, CBOR, ...) is the caller's concern — this
// function owns framing only. Returns the scanner error (if any) when the
// stream ends.
//
// maxEventBytes <= 0 falls back to DefaultMaxSSEBytes.
//
// Spec compliance:
//   - Blank lines dispatch the accumulated event data.
//   - Multiple consecutive "data:" lines are joined with "\n" to form the
//     payload (required by the spec).
//   - Lines starting with ':' are comment frames (heartbeats) — ignored.
//   - Other field types (event:, id:, retry:) are ignored.
//   - Exactly one leading space after "data:" is stripped per spec.
func ParseSSEData(r io.Reader, maxEventBytes int, emit func(data string)) error {
	if maxEventBytes <= 0 {
		maxEventBytes = DefaultMaxSSEBytes
	}

	scanner := bufio.NewScanner(r)
	// The initial buffer capacity must not exceed the limit, or bufio would
	// silently allow tokens up to the larger initial capacity.
	scanner.Buffer(make([]byte, 0, min(sseScannerInitCap, maxEventBytes)), maxEventBytes)

	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" { // Event boundary.
			if len(dataLines) > 0 {
				emit(strings.Join(dataLines, "\n"))
				dataLines = dataLines[:0]
			}

			continue
		}

		// Comment lines (heartbeats) and non-data fields are ignored.
		if strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
			continue
		}

		// Strip "data:" prefix and one optional leading space per spec.
		value := line[len("data:"):]
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}

		dataLines = append(dataLines, value)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("parse sse stream: %w", err)
	}

	return nil
}
