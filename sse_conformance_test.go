package daemon

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-sse/sseparse"
)

// TestParseSSEData_CorpusConformance drives the framing-only parser against
// the WPT/Chromium-derived corpus exported by sseparse as data — the same
// vectors that pin the reference parser — projected onto ParseSSEData's
// surface: which frames dispatch, with which data payload, in order.
//
// The corpus is the oracle: if this parser disagrees with a vector, the
// parser is wrong unless the divergence is justified by the framing-only
// mechanism itself (none is expected — everything the corpus pins about
// dispatch and data payloads is framing).
func TestParseSSEData_CorpusConformance(t *testing.T) {
	t.Parallel()

	for _, vector := range sseparse.MustCorpus(t) {
		vector := vector

		t.Run(vector.Name, func(t *testing.T) {
			t.Parallel()

			var got []string
			err := ParseSSEData(strings.NewReader(vector.Wire), 0, func(data string) {
				got = append(got, data)
			})
			if err != nil {
				t.Fatalf("ParseSSEData: %v", err)
			}

			want := make([]string, 0, len(vector.Events))
			for _, evt := range vector.Events {
				want = append(want, evt.Data)
			}

			if len(got) != len(want) {
				t.Fatalf("dispatch count: got %d (%q), want %d (%q)", len(got), got, len(want), want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("payload %d: got %q, want %q (full: got %q, want %q)", i, got[i], want[i], got, want)
				}
			}
		})
	}
}
