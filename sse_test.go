package daemon

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSSEData_SingleLineData(t *testing.T) {
	t.Parallel()

	var events []string

	err := ParseSSEData(
		strings.NewReader("data: hello\n\n"),
		0,
		func(data string) { events = append(events, data) },
	)
	if err != nil {
		t.Fatalf("ParseSSEData: %v", err)
	}

	if len(events) != 1 || events[0] != "hello" {
		t.Fatalf("events: got %#v, want [hello]", events)
	}
}

func TestParseSSEData_MultiLineData(t *testing.T) {
	t.Parallel()

	var events []string

	err := ParseSSEData(
		strings.NewReader("data: line1\ndata: line2\n\n"),
		0,
		func(data string) { events = append(events, data) },
	)
	if err != nil {
		t.Fatalf("ParseSSEData: %v", err)
	}

	if len(events) != 1 || events[0] != "line1\nline2" {
		t.Fatalf("multi-line data must join with \\n: got %#v", events)
	}
}

func TestParseSSEData_HeartbeatComment(t *testing.T) {
	t.Parallel()

	var events []string

	err := ParseSSEData(
		strings.NewReader(": heartbeat\ndata: hello\n\n"),
		0,
		func(data string) { events = append(events, data) },
	)
	if err != nil {
		t.Fatalf("ParseSSEData: %v", err)
	}

	if len(events) != 1 || events[0] != "hello" {
		t.Fatalf("comment frames must be ignored: got %#v", events)
	}
}

func TestParseSSEData_NoSpaceAfterColon(t *testing.T) {
	t.Parallel()

	var events []string

	err := ParseSSEData(
		strings.NewReader("data:nospace\n\n"),
		0,
		func(data string) { events = append(events, data) },
	)
	if err != nil {
		t.Fatalf("ParseSSEData: %v", err)
	}

	if len(events) != 1 || events[0] != "nospace" {
		t.Fatalf("exactly one leading space is stripped: got %#v", events)
	}
}

func TestParseSSEData_MultipleEvents(t *testing.T) {
	t.Parallel()

	var events []string

	err := ParseSSEData(
		strings.NewReader("data: first\n\ndata: second\n\n"),
		0,
		func(data string) { events = append(events, data) },
	)
	if err != nil {
		t.Fatalf("ParseSSEData: %v", err)
	}

	if len(events) != 2 || events[0] != "first" || events[1] != "second" {
		t.Fatalf("events: got %#v, want [first second]", events)
	}
}

func TestParseSSEData_TwoLeadingSpaces(t *testing.T) {
	t.Parallel()

	var events []string

	err := ParseSSEData(
		strings.NewReader("data:  two\n\n"),
		0,
		func(data string) { events = append(events, data) },
	)
	if err != nil {
		t.Fatalf("ParseSSEData: %v", err)
	}

	if len(events) != 1 || events[0] != " two" {
		t.Fatalf("only one leading space is stripped per spec: got %#v", events)
	}
}

func TestParseSSEData_NonDataFieldsIgnored(t *testing.T) {
	t.Parallel()

	var events []string

	err := ParseSSEData(
		strings.NewReader("event: update\nid: 7\nretry: 1000\ndata: payload\n\n"),
		0,
		func(data string) { events = append(events, data) },
	)
	if err != nil {
		t.Fatalf("ParseSSEData: %v", err)
	}

	if len(events) != 1 || events[0] != "payload" {
		t.Fatalf("event/id/retry fields must be ignored: got %#v", events)
	}
}

func TestParseSSEData_StreamEndsMidEvent(t *testing.T) {
	t.Parallel()

	// No trailing blank line: the accumulated data is NOT dispatched.
	var events []string

	err := ParseSSEData(
		strings.NewReader("data: hello\n"),
		0,
		func(data string) { events = append(events, data) },
	)
	if err != nil {
		t.Fatalf("ParseSSEData: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("undispatched data must not be emitted at stream end: got %#v", events)
	}
}

func TestParseSSEData_EmptyDataValue(t *testing.T) {
	t.Parallel()

	var events []string

	err := ParseSSEData(
		strings.NewReader("data:\n\n"),
		0,
		func(data string) { events = append(events, data) },
	)
	if err != nil {
		t.Fatalf("ParseSSEData: %v", err)
	}

	if len(events) != 1 || events[0] != "" {
		t.Fatalf("empty data value dispatches an empty payload: got %#v", events)
	}
}

func TestParseSSEData_LargeEvent(t *testing.T) {
	t.Parallel()

	big := strings.Repeat("x", 512*1024)

	var events []string

	err := ParseSSEData(
		strings.NewReader("data: "+big+"\n\n"),
		0,
		func(data string) { events = append(events, data) },
	)
	if err != nil {
		t.Fatalf("ParseSSEData: %v", err)
	}

	if len(events) != 1 || len(events[0]) != len(big) {
		t.Fatalf("large event truncated: got %d bytes, want %d", len(events[0]), len(big))
	}
}

func TestParseSSEData_MaxEventBytesExceeded(t *testing.T) {
	t.Parallel()

	big := strings.Repeat("x", 1024)

	err := ParseSSEData(strings.NewReader("data: "+big+"\n\n"), 64, func(string) {})
	if err == nil {
		t.Fatal("expected scanner error when event exceeds maxEventBytes")
	}
}

// errBoom is the sentinel error errReader fails with.
var errBoom = errors.New("boom")

// errReader fails on first read.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errBoom }

func TestParseSSEData_ScannerErrorWrapped(t *testing.T) {
	t.Parallel()

	err := ParseSSEData(errReader{}, 0, func(string) {})

	if err == nil || !strings.Contains(err.Error(), "parse sse stream") {
		t.Fatalf("scanner error must be wrapped: got %v", err)
	}
}

func FuzzParseSSEData(f *testing.F) {
	f.Add("data: hello\n\n")
	f.Add(": comment\ndata: a\ndata: b\n\n")
	f.Add("data:\n\n")
	f.Add("event: x\nid: 1\ndata: y\n\n")

	f.Fuzz(func(t *testing.T, input string) {
		var count int

		err := ParseSSEData(strings.NewReader(input), 0, func(string) { count++ })
		if err != nil {
			return // scanner errors (oversized events) are fine
		}

		if count < 0 {
			t.Fatalf("negative event count: %d", count)
		}
	})
}
