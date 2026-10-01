package daemon

import (
	"fmt"
	"strings"
)

// ExampleParseSSEData shows the framing-only parse loop: every dispatched
// event's data payload is handed to the emit callback, comments and
// non-data fields are skipped, and payload decoding stays with the caller
// (mechanism here, policy there).
func ExampleParseSSEData() {
	var events []string

	err := ParseSSEData(
		strings.NewReader(": heartbeat\n\nevent: tick\ndata: one\n\ndata: two\ndata: two-b\n\n"),
		0,
		func(data string) { events = append(events, data) },
	)
	fmt.Println(err)
	fmt.Println(events)

	// Output:
	// <nil>
	// [one two
	// two-b]
}
