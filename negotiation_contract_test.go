package daemon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-codec"
)

// TestResolveResponseCodec_GoldenMatrix is the single reviewable table for the
// response-encoding contract: every behavior the doc comment on
// ResolveResponseCodec promises has a row here. Change the contract, change
// this table in the same commit.
func TestResolveResponseCodec_GoldenMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		accept   string
		reqCodec codec.Codec
		want     codec.Encoding
	}{
		// No preference: request codec echoes.
		{"absent header echoes JSON request", "", codec.JSONCodec{}, codec.EncodingJSON},
		{"absent header echoes CBOR request", "", codec.CBORCodec{}, codec.EncodingCBOR},
		{"*/* echoes JSON request", "*/*", codec.JSONCodec{}, codec.EncodingJSON},
		{"*/* echoes CBOR request", "*/*", codec.CBORCodec{}, codec.EncodingCBOR},
		{"application/* echoes JSON request", "application/*", codec.JSONCodec{}, codec.EncodingJSON},
		{"application/* echoes CBOR request", "application/*", codec.CBORCodec{}, codec.EncodingCBOR},
		{"unrelated type wildcard echoes request", "text/*", codec.CBORCodec{}, codec.EncodingCBOR},
		{"only unrelated types echo request", "text/plain, image/png", codec.CBORCodec{}, codec.EncodingCBOR},

		// Mentions and q-value ranking; tie prefers CBOR.
		{"bare mentions tie to CBOR", "application/json, application/cbor", codec.JSONCodec{}, codec.EncodingCBOR},
		{
			"higher q wins for JSON",
			"application/cbor;q=0.1, application/json;q=0.9",
			codec.CBORCodec{},
			codec.EncodingJSON,
		},
		{
			"higher q wins for CBOR",
			"application/json;q=0.5, application/cbor;q=0.9",
			codec.JSONCodec{},
			codec.EncodingCBOR,
		},
		{"structured suffix mentions CBOR", "application/vnd.acme+cbor", codec.JSONCodec{}, codec.EncodingCBOR},
		{"structured suffix mentions JSON", "application/vnd.acme+json", codec.CBORCodec{}, codec.EncodingJSON},
		{
			"suffix q ranks against bare type",
			"application/vnd.acme+cbor;q=0.2, application/json;q=0.9",
			codec.CBORCodec{},
			codec.EncodingJSON,
		},

		// Duplicate media-range entries: the highest assigned quality wins.
		{
			"duplicate q takes the max (low first)",
			"application/json;q=0.5, application/json;q=0.9, application/cbor;q=0.7",
			codec.CBORCodec{},
			codec.EncodingJSON,
		},
		{
			"duplicate q takes the max (high first)",
			"application/json;q=0.9, application/json;q=0.5, application/cbor;q=0.7",
			codec.CBORCodec{},
			codec.EncodingJSON,
		},
		{
			"duplicate q: higher entry rescues an earlier refusal",
			"application/cbor;q=0, application/cbor;q=0.8, application/json;q=0.5",
			codec.JSONCodec{},
			codec.EncodingCBOR,
		},

		// Malformed quality params default to 1.
		{"empty q= counts as 1", "application/cbor;q=, application/json;q=0.9", codec.JSONCodec{}, codec.EncodingCBOR},
		{
			"bare ;q param counts as 1",
			"application/cbor;q, application/json;q=0.9",
			codec.JSONCodec{},
			codec.EncodingCBOR,
		},
		{
			"unparseable q counts as 1",
			"application/cbor;q=abc, application/json;q=0.9",
			codec.JSONCodec{},
			codec.EncodingCBOR,
		},
		{"q above 1 clamps to 1", "application/cbor;q=1.7, application/json", codec.JSONCodec{}, codec.EncodingCBOR},
		{
			"q below 0 clamps to 0",
			"application/cbor;q=-2, application/json;q=0.4",
			codec.JSONCodec{},
			codec.EncodingJSON,
		},

		// Case-insensitive parameter name (RFC 9110 §5.6.2.1).
		{"uppercase Q= parses as q", "application/cbor;Q=0.2, application/json", codec.CBORCodec{}, codec.EncodingJSON},
		{"uppercase Q=0 refuses CBOR", "application/cbor;Q=0", codec.CBORCodec{}, codec.EncodingJSON},
		{
			"mixed case q= parses",
			"application/json;Q=0.9, application/cbor;q=0.5",
			codec.CBORCodec{},
			codec.EncodingJSON,
		},

		// Explicit refusals: a q=0 encoding is never served unless the
		// client refuses everything speakable.
		{"q=0 refuses JSON, JSON request substituted", "application/json;q=0", codec.JSONCodec{}, codec.EncodingCBOR},
		{"q=0 refuses JSON, CBOR request stands", "application/json;q=0", codec.CBORCodec{}, codec.EncodingCBOR},
		{"q=0 refuses CBOR, CBOR request substituted", "application/cbor;q=0", codec.CBORCodec{}, codec.EncodingJSON},
		{"q=0 refuses CBOR, JSON request stands", "application/cbor;q=0", codec.JSONCodec{}, codec.EncodingJSON},
		{
			"refusal still ranks against the other type",
			"application/cbor;q=0, application/json;q=0.3",
			codec.CBORCodec{},
			codec.EncodingJSON,
		},
		{
			"both refused echoes JSON request (unsatisfiable)",
			"application/cbor;q=0, application/json;q=0",
			codec.JSONCodec{},
			codec.EncodingJSON,
		},
		{
			"both refused echoes CBOR request (unsatisfiable)",
			"application/cbor;q=0, application/json;q=0",
			codec.CBORCodec{},
			codec.EncodingCBOR,
		},

		// Real-world headers.
		{
			"browser header echoes JSON request",
			"text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8",
			codec.JSONCodec{},
			codec.EncodingJSON,
		},
		{
			"browser header echoes CBOR request",
			"text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8",
			codec.CBORCodec{},
			codec.EncodingCBOR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/discover", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}

			if got := ResolveResponseCodec(req, tt.reqCodec); got.Encoding() != tt.want {
				t.Fatalf("ResolveResponseCodec(%q, %s): got %q, want %q",
					tt.accept, tt.reqCodec.Encoding(), got.Encoding(), tt.want)
			}
		})
	}
}

// TestResolveResponseCodec_NeverServesRefusedEncoding pins the refusal
// property across the full q=0 x request-codec matrix: whenever the Accept
// header refuses (q=0) exactly one supported encoding, the resolved response
// codec is never that encoding. When the client refuses every supported
// encoding the Accept is unsatisfiable; this mechanism cannot answer 406, so
// the request codec stands (also pinned in the golden matrix above).
// acceptRefusalCase describes one Accept header in the refusal matrix.
type acceptRefusalCase struct {
	name        string
	accept      string
	refusesCBOR bool
	refusesJSON bool
}

// requestCodecCase pairs a request codec with a label for the matrix.
type requestCodecCase struct {
	name  string
	codec codec.Codec
}

func requestWithAccept(accept string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/discover", nil)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	return req
}

// assertNeverRefused checks the refusal property for one matrix cell: when
// exactly one supported encoding is refused it must not be served; when both
// are refused the Accept is unsatisfiable and the request codec stands.
func assertNeverRefused(t *testing.T, refusal acceptRefusalCase, rc requestCodecCase) {
	t.Helper()

	got := ResolveResponseCodec(requestWithAccept(refusal.accept), rc.codec)

	switch enc := got.Encoding(); enc {
	case codec.EncodingJSON, codec.EncodingCBOR:
	default:
		t.Fatalf("unsupported encoding %q returned", enc)
	}

	if refusal.refusesJSON && refusal.refusesCBOR {
		if got.Encoding() != rc.codec.Encoding() {
			t.Fatalf("unsatisfiable Accept must echo the request codec: got %q, want %q",
				got.Encoding(), rc.codec.Encoding())
		}

		return
	}

	if refusal.refusesJSON && got.Encoding() == codec.EncodingJSON {
		t.Fatalf("served refused encoding JSON (Accept %q, request %s)",
			refusal.accept, rc.codec.Encoding())
	}

	if refusal.refusesCBOR && got.Encoding() == codec.EncodingCBOR {
		t.Fatalf("served refused encoding CBOR (Accept %q, request %s)",
			refusal.accept, rc.codec.Encoding())
	}
}

func TestResolveResponseCodec_NeverServesRefusedEncoding(t *testing.T) {
	t.Parallel()

	refusals := []acceptRefusalCase{
		{name: "no refusals", accept: ""},
		{name: "refuses JSON", accept: "application/json;q=0", refusesJSON: true},
		{name: "refuses CBOR", accept: "application/cbor;q=0", refusesCBOR: true},
		{
			name:        "refuses both",
			accept:      "application/cbor;q=0, application/json;q=0",
			refusesCBOR: true,
			refusesJSON: true,
		},
	}

	reqCodecs := []requestCodecCase{
		{name: "JSON request", codec: codec.JSONCodec{}},
		{name: "CBOR request", codec: codec.CBORCodec{}},
	}

	for _, refusal := range refusals {
		for _, rc := range reqCodecs {
			t.Run(fmt.Sprintf("%s / %s", refusal.name, rc.name), func(t *testing.T) {
				t.Parallel()

				assertNeverRefused(t, refusal, rc)
			})
		}
	}
}

// TestAcceptQualityDuplicateMax pins the unexported ranking helper directly:
// among duplicate mentions the highest quality wins, and a mention without a
// q parameter counts as 1.
func TestAcceptQualityDuplicateMax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		accept string
		enc    codec.Encoding
		want   float64
	}{
		{"single mention defaults to 1", "application/cbor", codec.EncodingCBOR, 1},
		{"duplicate low then high", "application/cbor;q=0.2, application/cbor;q=0.8", codec.EncodingCBOR, 0.8},
		{"duplicate high then low", "application/cbor;q=0.8, application/cbor;q=0.2", codec.EncodingCBOR, 0.8},
		{"duplicate rescue of refusal", "application/cbor;q=0, application/cbor;q=0.9", codec.EncodingCBOR, 0.9},
		{"q-less duplicate rescues refusal", "application/cbor;q=0, application/cbor", codec.EncodingCBOR, 1},
		{"other encodings do not pollute", "application/json;q=0.9, application/cbor;q=0.3", codec.EncodingCBOR, 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q, mentioned := acceptQuality(tt.accept, tt.enc)
			if !mentioned {
				t.Fatalf("acceptQuality(%q, %s): not mentioned", tt.accept, tt.enc)
			}

			if q != tt.want {
				t.Fatalf("acceptQuality(%q, %s): got %v, want %v", tt.accept, tt.enc, q, tt.want)
			}
		})
	}

	if _, mentioned := acceptQuality(
		strings.Repeat("application/json;q=0.5, ", 100)+"text/plain",
		codec.EncodingCBOR,
	); mentioned {
		t.Fatal("CBOR must not be mentioned by an unrelated header")
	}
}

// taggedTestCodec is a comparable codec stub carrying a tag, so injection
// tests can assert identity: which codec instance negotiation resolved to.
type taggedTestCodec struct {
	tag string
	enc codec.Encoding
}

func (c taggedTestCodec) Encoding() codec.Encoding { return c.enc }

func (taggedTestCodec) Encode(_ any) ([]byte, error) { return nil, nil }

func (taggedTestCodec) Decode(_ []byte, _ any) error { return nil }

var _ codec.Codec = taggedTestCodec{}

// TestResolveResponseCodec_InjectedCodecs pins the codec injectability the
// doc comment on ResolveResponseCodec promises: NegotiationOption values
// substitute the codecs resolved for each encoding without changing the
// ranking contract, and absent options keep the built-in defaults.
func TestResolveResponseCodec_InjectedCodecs(t *testing.T) {
	t.Parallel()

	customJSON := taggedTestCodec{tag: "json-custom", enc: codec.EncodingJSON}
	customCBOR := taggedTestCodec{tag: "cbor-custom", enc: codec.EncodingCBOR}
	bothOpts := []NegotiationOption{WithJSONCodec(customJSON), WithCBORCodec(customCBOR)}

	tests := []struct {
		name     string
		accept   string
		reqCodec codec.Codec
		opts     []NegotiationOption
		want     codec.Codec
	}{
		{
			name:     "q prefers JSON returns the injected JSON codec",
			accept:   "application/json",
			reqCodec: codec.CBORCodec{},
			opts:     bothOpts,
			want:     customJSON,
		},
		{
			name:     "q prefers CBOR returns the injected CBOR codec",
			accept:   "application/cbor",
			reqCodec: codec.JSONCodec{},
			opts:     bothOpts,
			want:     customCBOR,
		},
		{
			name:     "tie prefers the injected CBOR codec",
			accept:   "application/json, application/cbor",
			reqCodec: codec.JSONCodec{},
			opts:     bothOpts,
			want:     customCBOR,
		},
		{
			name:     "q=0 refusal substitutes the other injected codec",
			accept:   "application/json;q=0",
			reqCodec: customJSON,
			opts:     bothOpts,
			want:     customCBOR,
		},
		{
			name:     "wildcard echoes the request codec, not the injected JSON default",
			accept:   "*/*",
			reqCodec: taggedTestCodec{tag: "request", enc: codec.EncodingJSON},
			opts:     []NegotiationOption{WithJSONCodec(customJSON)},
			want:     taggedTestCodec{tag: "request", enc: codec.EncodingJSON},
		},
		{
			name:     "absent header echoes the request codec, not the injected JSON default",
			accept:   "",
			reqCodec: taggedTestCodec{tag: "request", enc: codec.EncodingJSON},
			opts:     []NegotiationOption{WithJSONCodec(customJSON)},
			want:     taggedTestCodec{tag: "request", enc: codec.EncodingJSON},
		},
		{
			name:     "no options keep the built-in JSON default",
			accept:   "application/json",
			reqCodec: codec.CBORCodec{},
			want:     codec.JSONCodec{},
		},
		{
			name:     "nil codec option keeps the built-in JSON default",
			accept:   "application/json",
			reqCodec: codec.CBORCodec{},
			opts:     []NegotiationOption{WithJSONCodec(nil)},
			want:     codec.JSONCodec{},
		},
		{
			name:     "nil option value is skipped",
			accept:   "application/cbor",
			reqCodec: codec.JSONCodec{},
			opts:     []NegotiationOption{nil},
			want:     codec.CBORCodec{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ResolveResponseCodec(requestWithAccept(tt.accept), tt.reqCodec, tt.opts...)
			if got != tt.want {
				t.Fatalf("ResolveResponseCodec(%q, %s): got %v, want %v",
					tt.accept, tt.reqCodec.Encoding(), got, tt.want)
			}
		})
	}
}

// TestCodecForContentType_InjectedCodecs pins the same injectability for the
// Content-Type driven resolvers (CodecForContentType, ResolveRequestCodec).
func TestCodecForContentType_InjectedCodecs(t *testing.T) {
	t.Parallel()

	customJSON := taggedTestCodec{tag: "json-custom", enc: codec.EncodingJSON}
	customCBOR := taggedTestCodec{tag: "cbor-custom", enc: codec.EncodingCBOR}
	bothOpts := []NegotiationOption{WithJSONCodec(customJSON), WithCBORCodec(customCBOR)}

	tests := []struct {
		name   string
		header string
		opts   []NegotiationOption
		want   codec.Codec
	}{
		{
			name:   "JSON content type resolves the injected JSON codec",
			header: "application/json",
			opts:   bothOpts,
			want:   customJSON,
		},
		{
			name:   "unknown content type resolves the injected JSON fallback",
			header: "application/xml",
			opts:   bothOpts,
			want:   customJSON,
		},
		{
			name:   "CBOR suffix type resolves the injected CBOR codec",
			header: "application/vnd.acme+cbor",
			opts:   bothOpts,
			want:   customCBOR,
		},
		{
			name:   "no options keep the built-in JSON default",
			header: "application/json",
			want:   codec.JSONCodec{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := CodecForContentType(tt.header, tt.opts...); got != tt.want {
				t.Fatalf("CodecForContentType(%q): got %v, want %v", tt.header, got, tt.want)
			}

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/discover", nil)
			req.Header.Set("Content-Type", tt.header)

			if got := ResolveRequestCodec(req, tt.opts...); got != tt.want {
				t.Fatalf("ResolveRequestCodec(%q): got %v, want %v", tt.header, got, tt.want)
			}
		})
	}
}

// FuzzResolveResponseCodec guards the hand-rolled Accept parser — plain
// SplitSeq/Cut/ParseFloat over untrusted header bytes — against panics and
// contract violations on arbitrary input. Invariants per input, for both
// possible request codecs: the result is always a supported encoding, and an
// encoding explicitly refused (mentioned at q=0, with the other encoding not
// refused) is never served. The unsatisfiable both-refused case is excluded
// here and pinned explicitly in the golden matrix.
func FuzzResolveResponseCodec(f *testing.F) {
	f.Add("application/json, application/cbor;q=0.9")
	f.Add("text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8")
	f.Add("*/*")
	f.Add("application/*")
	f.Add("application/cbor;q=0")
	f.Add("application/json;q=0, application/cbor;q=0")
	f.Add("application/vnd.acme+cbor;Q=0.7")
	f.Add("application/json;q=NaN")
	f.Add("application/json;q=-Inf")
	f.Add("application/cbor;q=")
	f.Add(",,,;;;")
	f.Add("application/json;  ;q=0.3;;")
	f.Add(strings.Repeat("application/json;q=0.5, ", 100) + "application/cbor;q=0.8")

	f.Fuzz(func(t *testing.T, accept string) {
		for _, reqCodec := range []codec.Codec{codec.JSONCodec{}, codec.CBORCodec{}} {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/discover", nil)
			req.Header.Set("Accept", accept)

			got := ResolveResponseCodec(req, reqCodec)

			switch enc := got.Encoding(); enc {
			case codec.EncodingJSON, codec.EncodingCBOR:
			default:
				t.Fatalf("Accept %q: unsupported encoding %q returned", accept, enc)
			}

			cborQ, cborMentioned := acceptQuality(accept, codec.EncodingCBOR)
			jsonQ, jsonMentioned := acceptQuality(accept, codec.EncodingJSON)
			cborRefused := cborMentioned && cborQ == 0
			jsonRefused := jsonMentioned && jsonQ == 0

			if cborRefused && jsonRefused {
				continue
			}

			if jsonRefused && got.Encoding() == codec.EncodingJSON {
				t.Fatalf("Accept %q: served refused encoding JSON (request %s)", accept, reqCodec.Encoding())
			}

			if cborRefused && got.Encoding() == codec.EncodingCBOR {
				t.Fatalf("Accept %q: served refused encoding CBOR (request %s)", accept, reqCodec.Encoding())
			}
		}
	})
}
