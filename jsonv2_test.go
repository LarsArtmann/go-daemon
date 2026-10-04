package daemon

import (
	"bytes"
	"context"
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// jsonv2EncodePayload exercises the v2-only omitzero tag and the nil-slice
// divergence between the v1 and v2 defaults.
type jsonv2EncodePayload struct {
	Name  string   `json:"name,omitzero"`
	Items []string `json:"items"`
}

type jsonv2DurationPayload struct {
	Timeout time.Duration `json:"timeout"`
}

func TestJSONv2Codec_Encode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		codec   JSONv2Codec
		in      any
		want    string
		wantErr bool
	}{
		{
			name:  "v2 defaults: omitzero drops zero field, nil slice is empty array",
			codec: JSONv2Codec{},
			in:    jsonv2EncodePayload{},
			want:  `{"items":[]}`,
		},
		{
			name:  "FormatNilSliceAsNull restores v1 nil handling",
			codec: JSONv2Codec{MarshalOptions: []json.Options{json.FormatNilSliceAsNull(true)}},
			in:    jsonv2EncodePayload{},
			want:  `{"items":null}`,
		},
		{
			name:  "non-zero fields encode normally",
			codec: JSONv2Codec{},
			in:    jsonv2EncodePayload{Name: "daemon", Items: []string{"a"}},
			want:  `{"name":"daemon","items":["a"]}`,
		},
		{
			name:  "deterministic sorts map keys",
			codec: JSONv2Codec{MarshalOptions: []json.Options{json.Deterministic(true)}},
			in:    map[string]int{"c": 3, "a": 1, "b": 2},
			want:  `{"a":1,"b":2,"c":3}`,
		},
		{
			name:  "duration encodes as integer nanos with the v1 interop option",
			codec: JSONv2Codec{MarshalOptions: []json.Options{jsonv1.FormatDurationAsNano(true)}},
			in:    jsonv2DurationPayload{Timeout: 1500 * time.Millisecond},
			want:  `{"timeout":1500000000}`,
		},
		{
			name:    "duration without the option is a semantic error",
			codec:   JSONv2Codec{},
			in:      jsonv2DurationPayload{Timeout: time.Second},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.codec.Encode(tt.in)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Encode(%v): expected error, got %q", tt.in, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("Encode(%v): %v", tt.in, err)
			}

			if string(got) != tt.want {
				t.Fatalf("Encode(%v): got %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestJSONv2Codec_EncodeToBuffer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		codec JSONv2Codec
		in    any
	}{
		{"defaults", JSONv2Codec{}, jsonv2EncodePayload{Name: "daemon", Items: []string{"a"}}},
		{
			"deterministic map",
			JSONv2Codec{MarshalOptions: []json.Options{json.Deterministic(true)}},
			map[string]int{"c": 3, "a": 1, "b": 2},
		},
		{
			"duration option",
			JSONv2Codec{MarshalOptions: []json.Options{jsonv1.FormatDurationAsNano(true)}},
			jsonv2DurationPayload{Timeout: time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want, err := tt.codec.Encode(tt.in)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}

			var buf bytes.Buffer

			if err := tt.codec.EncodeToBuffer(tt.in, &buf); err != nil {
				t.Fatalf("EncodeToBuffer: %v", err)
			}

			if buf.String() != string(want) {
				t.Fatalf("EncodeToBuffer: got %q, want %q", buf.String(), want)
			}
		})
	}
}

func TestJSONv2Codec_Decode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		codec    JSONv2Codec
		in       string
		wantName string
	}{
		{"round trip", JSONv2Codec{}, `{"name":"daemon"}`, "daemon"},
		{"case-sensitive by default", JSONv2Codec{}, `{"Name":"daemon"}`, ""},
		{
			"case-insensitive with option",
			JSONv2Codec{UnmarshalOptions: []json.Options{json.MatchCaseInsensitiveNames(true)}},
			`{"Name":"daemon"}`,
			"daemon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got struct {
				Name string `json:"name"`
			}

			if err := tt.codec.Decode([]byte(tt.in), &got); err != nil {
				t.Fatalf("Decode(%q): %v", tt.in, err)
			}

			if got.Name != tt.wantName {
				t.Fatalf("Decode(%q): got %q, want %q", tt.in, got.Name, tt.wantName)
			}
		})
	}
}

func TestJSONv2Codec_DurationRoundTrip(t *testing.T) {
	t.Parallel()

	opts := []json.Options{jsonv1.FormatDurationAsNano(true)}
	c := JSONv2Codec{MarshalOptions: opts, UnmarshalOptions: opts}

	data, err := c.Encode(jsonv2DurationPayload{Timeout: 1500 * time.Millisecond})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	var got jsonv2DurationPayload

	if err := c.Decode(data, &got); err != nil {
		t.Fatalf("Decode(%q): %v", data, err)
	}

	if got.Timeout != 1500*time.Millisecond {
		t.Fatalf("round trip: got %v, want %v", got.Timeout, 1500*time.Millisecond)
	}
}

// TestJSONv2Codec_NegotiatedWrite wires JSONv2Codec through the full
// negotiated write path (the adoption pattern the codec exists for) and
// asserts a byte shape the v1-backed default cannot produce: a nil slice
// marshaling as [] instead of null.
func TestJSONv2Codec_NegotiatedWrite(t *testing.T) {
	t.Parallel()

	opts := []NegotiationOption{WithJSONCodec(JSONv2Codec{})}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/status", nil)
	req.Header.Set("Accept", "application/json")

	respCodec := ResolveResponseCodec(req, ResolveRequestCodec(req, opts...), opts...)

	if _, ok := respCodec.(JSONv2Codec); !ok {
		t.Fatalf("resolved codec is %T, want JSONv2Codec", respCodec)
	}

	w := httptest.NewRecorder()

	WriteEncoded(w, http.StatusOK, respCodec, jsonv2EncodePayload{})

	const wantBody = `{"items":[]}`

	if got := w.Body.String(); got != wantBody {
		t.Fatalf("body: got %q, want %q", got, wantBody)
	}

	if ct := w.Header().Get("Content-Type"); ct != contentTypeJSON {
		t.Fatalf("content-type: got %q, want %q", ct, contentTypeJSON)
	}

	if vary := w.Header().Get("Vary"); vary != "Accept" {
		t.Fatalf("vary: got %q, want %q", vary, "Accept")
	}
}
