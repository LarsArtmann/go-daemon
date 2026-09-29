package daemon

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

const (
	// DefaultMaxSSEBytes bounds a single SSE line on the wire. Lines longer
	// than the cap fail the parse instead of being silently truncated (the
	// framed event would be corrupt).
	DefaultMaxSSEBytes = 1 << 20 // 1 MiB

	// sseReadChunk is how much is read from the stream per fill; the line
	// buffer starts at the same hint and grows toward the cap.
	sseReadChunk = 32 * 1024
)

// ParseSSEData reads an SSE stream from r, parses events per the WHATWG
// Server-Sent Events spec, and emits each event's joined data payload via
// emit. Payload decoding (JSON, CBOR, ...) is the caller's concern — this
// function owns framing only. Returns an error when a read fails or a line
// exceeds maxEventBytes (wrapping bufio.ErrTooLong, errors.Is-matchable).
//
// maxEventBytes <= 0 falls back to DefaultMaxSSEBytes.
//
// Spec compliance:
//   - CR, LF, and CRLF all terminate a line; CRLF counts as one terminator.
//   - Exactly one leading UTF-8 BOM is stripped from the first line; a
//     mid-stream BOM is ordinary bytes and poisons the field name (spec).
//   - Blank lines dispatch the accumulated event data.
//   - Multiple consecutive "data:" lines are joined with "\n" to form the
//     payload (required by the spec).
//   - A line with no colon is a field whose value is empty, so a bare
//     "data" line contributes an empty payload line.
//   - Lines whose field name is not exactly "data" (comments — name "" —
//     and event/id/retry/unknown fields, case-sensitively) are ignored.
//   - Exactly one leading space after "data:" is stripped per spec.
//   - An incomplete final frame at EOF (no blank line) never dispatches.
func ParseSSEData(r io.Reader, maxEventBytes int, emit func(data string)) error {
	if maxEventBytes <= 0 {
		maxEventBytes = DefaultMaxSSEBytes
	}

	reader := newSSELineReader(r, maxEventBytes)

	var dataLines []string

	for {
		raw, err := reader.next()
		if err != nil {
			return fmt.Errorf("parse sse stream: %w", err)
		}

		if raw == nil {
			break
		}

		line := string(raw)

		if reader.isFirstLine {
			line = strings.TrimPrefix(line, "\ufeff")
			reader.isFirstLine = false
		}

		if line == "" { // Event boundary.
			if len(dataLines) > 0 {
				emit(strings.Join(dataLines, "\n"))
				dataLines = dataLines[:0]
			}

			continue
		}

		name, value, hasColon := strings.Cut(line, ":")

		if !hasColon {
			// A line with no colon is a field whose value is empty.
			value = ""
		} else if strings.HasPrefix(value, " ") {
			value = value[1:] // Exactly one leading space after the colon.
		}

		// Comments (name "") and non-data fields are ignored.
		if name != "data" {
			continue
		}

		dataLines = append(dataLines, value)
	}

	return nil
}

// sseLineReader splits an SSE byte stream into lines per spec § 9.2.5:
// CR, LF, and CRLF terminate a line, and CRLF counts as one terminator
// (a lone CR must not be glued to the following line's first byte).
type sseLineReader struct {
	reader      io.Reader
	limit       int    // maximum line length in bytes
	chunk       []byte // read buffer
	pending     []byte // bytes read but not yet consumed
	line        []byte // the line being assembled (without terminator)
	isFirstLine bool
	eof         bool // the underlying reader has returned io.EOF
}

func newSSELineReader(r io.Reader, limit int) *sseLineReader {
	hint := min(sseReadChunk, limit)

	return &sseLineReader{
		reader:      r,
		limit:       limit,
		chunk:       make([]byte, sseReadChunk),
		pending:     make([]byte, 0, hint),
		line:        make([]byte, 0, hint),
		isFirstLine: true,
	}
}

// next returns the next line without its terminator, or nil at clean EOF.
// A final line with no terminator before EOF is still returned (EOF ends a
// line). Read errors and over-cap lines are returned as errors.
func (s *sseLineReader) next() ([]byte, error) {
	s.line = s.line[:0]

	for {
		done, err := s.consumePending()
		if err != nil {
			return nil, err
		}

		if done {
			return s.line, nil
		}

		if s.eof {
			return s.lineAtEOF(), nil
		}

		if err := s.fill(); err != nil {
			return nil, err
		}
	}
}

// consumePending drains pending bytes into line until a terminator ends the
// line. done reports whether a full line was produced.
func (s *sseLineReader) consumePending() (bool, error) {
	for len(s.pending) > 0 {
		b := s.pending[0]
		s.pending = s.pending[1:]

		if b == '\n' {
			return true, nil
		}

		if b == '\r' {
			if err := s.consumeCR(); err != nil {
				return false, err
			}

			return true, nil
		}

		s.line = append(s.line, b)

		if len(s.line) > s.limit {
			return false, fmt.Errorf("line exceeds %d bytes: %w", s.limit, bufio.ErrTooLong)
		}
	}

	return false, nil
}

// consumeCR finishes a CR-terminated line, swallowing a following LF so
// CRLF counts as one terminator.
func (s *sseLineReader) consumeCR() error {
	if len(s.pending) == 0 && !s.eof {
		if err := s.fill(); err != nil {
			return err
		}
	}

	if len(s.pending) > 0 && s.pending[0] == '\n' {
		s.pending = s.pending[1:] // CRLF is one terminator.
	}

	return nil
}

// lineAtEOF closes out a final line that has no terminator: a non-empty
// remainder is the last line, an empty one means clean EOF (nil).
func (s *sseLineReader) lineAtEOF() []byte {
	if len(s.line) > 0 {
		return s.line
	}

	return nil
}

// fill reads one chunk into pending; io.EOF just marks the end, hard read
// errors are wrapped and returned.
func (s *sseLineReader) fill() error {
	n, err := s.reader.Read(s.chunk)

	if n > 0 {
		s.pending = append(s.pending, s.chunk[:n]...)
	}

	if err == io.EOF {
		s.eof = true
	} else if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	return nil
}
