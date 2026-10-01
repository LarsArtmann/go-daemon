package daemon

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestParseSSEData_Dispatch(t *testing.T) {
	tests := []struct {
		name string
		wire string
		want []string
	}{
		{
			name: "single data line dispatches its payload",
			wire: "data: hello\n\n",
			want: []string{"hello"},
		},
		{
			name: "multi-line data joins with \\n",
			wire: "data: line1\ndata: line2\n\n",
			want: []string{"line1\nline2"},
		},
		{
			name: "comment frames are ignored",
			wire: ": heartbeat\ndata: hello\n\n",
			want: []string{"hello"},
		},
		{
			name: "exactly one leading space is stripped",
			wire: "data:nospace\n\n",
			want: []string{"nospace"},
		},
		{
			name: "multiple events dispatch in order",
			wire: "data: first\n\ndata: second\n\n",
			want: []string{"first", "second"},
		},
		{
			name: "only one leading space is stripped per spec",
			wire: "data:  two\n\n",
			want: []string{" two"},
		},
		{
			name: "event/id/retry fields are ignored",
			wire: "event: update\nid: 7\nretry: 1000\ndata: payload\n\n",
			want: []string{"payload"},
		},
		{
			name: "undispatched data is not emitted at stream end",
			wire: "data: hello\n",
			want: nil,
		},
		{
			name: "empty data value dispatches an empty payload",
			wire: "data:\n\n",
			want: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var events []string

			err := ParseSSEData(
				strings.NewReader(tt.wire),
				0,
				func(data string) { events = append(events, data) },
			)
			if err != nil {
				t.Fatalf("ParseSSEData: %v", err)
			}

			if !slices.Equal(events, tt.want) {
				t.Fatalf("events: got %#v, want %#v", events, tt.want)
			}
		})
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

func TestSSELineReader_LineAtEOF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantFirst  string
		wantSecond string
	}{
		{
			name:       "unterminated final line is returned then clean EOF",
			input:      "data",
			wantFirst:  "data",
			wantSecond: "",
		},
		{
			name:       "empty stream is clean EOF on every call",
			input:      "",
			wantFirst:  "",
			wantSecond: "",
		},
		{
			name:       "terminated line then EOF yields nil",
			input:      "data\n",
			wantFirst:  "data",
			wantSecond: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := newSSELineReader(strings.NewReader(tt.input), 1024)

			first, err := r.next()
			if err != nil {
				t.Fatalf("first next: %v", err)
			}

			if string(first) != tt.wantFirst {
				t.Errorf("first line = %q, want %q", first, tt.wantFirst)
			}

			second, err := r.next()
			if err != nil {
				t.Fatalf("second next: %v", err)
			}

			if string(second) != tt.wantSecond {
				t.Errorf("second line = %q, want %q", second, tt.wantSecond)
			}
		})
	}
}
