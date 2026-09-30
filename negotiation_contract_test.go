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
func TestResolveResponseCodec_NeverServesRefusedEncoding(t *testing.T) {
	t.Parallel()

	refusals := []struct {
		name        string
		accept      string
		refusesCBOR bool
		refusesJSON bool
	}{
		{"no refusals", "", false, false},
		{"refuses JSON", "application/json;q=0", false, true},
		{"refuses CBOR", "application/cbor;q=0", true, false},
		{"refuses both", "application/cbor;q=0, application/json;q=0", true, true},
	}

	reqCodecs := []struct {
		name  string
		codec codec.Codec
	}{
		{"JSON request", codec.JSONCodec{}},
		{"CBOR request", codec.CBORCodec{}},
	}

	for _, refusal := range refusals {
		for _, rc := range reqCodecs {
			t.Run(fmt.Sprintf("%s / %s", refusal.name, rc.name), func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/discover", nil)
				if refusal.accept != "" {
					req.Header.Set("Accept", refusal.accept)
				}

				got := ResolveResponseCodec(req, rc.codec)

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
