package daemon

import (
	"testing"

	"github.com/larsartmann/go-codec"
)

// The helpers under test are unexported negotiation internals; the exported
// surface is covered by TestResolveRequestCodec / TestResolveResponseCodec /
// TestCodecForContentType and the contract matrix. These tables pin the
// helpers' own edges so failures localize instead of surfacing transitively.
func TestBareMediaType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		headerVal string
		want      string
	}{
		{"plain type", "application/json", "application/json"},
		{"strips one parameter", "application/cbor; charset=utf-8", "application/cbor"},
		{"strips many parameters", "application/cbor; a=1; b=2", "application/cbor"},
		{"strips malformed q", "application/json;q=not-a-number", "application/json"},
		{"leading whitespace", "  application/json", "application/json"},
		{"trailing whitespace", "application/json\t", "application/json"},
		{"whitespace before semicolon", "application/json ; q=0.5", "application/json"},
		{"lowercased", "APPLICATION/JSON", "application/json"},
		{"mixed case with params", "Application/CBOR; Q=0.5", "application/cbor"},
		{"empty", "", ""},
		{"only parameters", ";q=0.5", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := bareMediaType(tt.headerVal); got != tt.want {
				t.Fatalf("bareMediaType(%q): got %q, want %q", tt.headerVal, got, tt.want)
			}
		})
	}
}

func TestQualityParam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params string
		want   float64
	}{
		{"no params default 1", "", 1},
		{"q at head", "q=0.5", 0.5},
		{"q after other params", "charset=utf-8; q=0.5", 0.5},
		{"spaces around name and value", " q = 0.5 ", 0.5},
		{"uppercase Q name", "Q=0.5", 0.5},
		{"uppercase Q after other params", "charset=utf-8; Q=0.3", 0.3},
		{"first q wins within one entry", "Q=0.5; q=0.2", 0.5},
		{"q=0 refusal", "q=0", 0},
		{"three decimals", "q=0.001", 0.001},
		{"clamps above 1", "q=7", 1},
		{"clamps below 0", "q=-0.5", 0},
		{"empty value defaults 1", "q=", 1},
		{"bare param defaults 1", "q", 1},
		{"unparseable defaults 1", "q=abc", 1},
		{"non-q params ignored", "charset=utf-8; level=1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := qualityParam(tt.params); got != tt.want {
				t.Fatalf("qualityParam(%q): got %v, want %v", tt.params, got, tt.want)
			}
		})
	}
}

// matchesEncoding receives bareMediaType output: a lowercased,
// parameter-stripped media type.
func TestMatchesEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mediaType string
		enc       codec.Encoding
		want      bool
	}{
		{"exact json", "application/json", codec.EncodingJSON, true},
		{"json structured suffix", "application/vnd.acme+json", codec.EncodingJSON, true},
		{"json-patch suffix is json", "application/json-patch+json", codec.EncodingJSON, true},
		{"json not cbor", "application/json", codec.EncodingCBOR, false},
		{"exact cbor", "application/cbor", codec.EncodingCBOR, true},
		{"cbor structured suffix", "application/vnd.acme+cbor", codec.EncodingCBOR, true},
		{"cbor-seq is not plain cbor", "application/cbor-seq", codec.EncodingCBOR, false},
		{"cborx near miss", "application/cborx", codec.EncodingCBOR, false},
		{"json suffix is not cbor", "application/thing+json", codec.EncodingCBOR, false},
		{"full wildcard is not a mention", "*/*", codec.EncodingJSON, false},
		{"type wildcard is not a mention", "application/*", codec.EncodingCBOR, false},
		{"empty is not a mention", "", codec.EncodingJSON, false},
		{"unrelated type", "text/plain", codec.EncodingJSON, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := matchesEncoding(tt.mediaType, tt.enc); got != tt.want {
				t.Fatalf("matchesEncoding(%q, %s): got %v, want %v", tt.mediaType, tt.enc, got, tt.want)
			}
		})
	}
}

func TestAcceptQuality(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		accept   string
		enc      codec.Encoding
		wantQ    float64
		wantSeen bool
	}{
		{"absent when only other type", "application/json", codec.EncodingCBOR, 0, false},
		{"wildcards are not mentions", "*/*, application/*", codec.EncodingJSON, 0, false},
		{"bare mention defaults 1", "application/cbor", codec.EncodingCBOR, 1, true},
		{"mention via suffix", "application/vnd.acme+cbor;q=0.4", codec.EncodingCBOR, 0.4, true},
		{
			"max across bare and suffix shapes",
			"application/cbor;q=0.1, application/vnd.x+cbor;q=0.6",
			codec.EncodingCBOR,
			0.6,
			true,
		},
		{"uppercase type still matches", "APPLICATION/CBOR;Q=0.5", codec.EncodingCBOR, 0.5, true},
		{"refusal is mentioned at 0", "application/cbor;q=0", codec.EncodingCBOR, 0, true},
		{"empty header mentions nothing", "", codec.EncodingJSON, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q, mentioned := acceptQuality(tt.accept, tt.enc)
			if mentioned != tt.wantSeen {
				t.Fatalf("acceptQuality(%q, %s): mentioned %v, want %v", tt.accept, tt.enc, mentioned, tt.wantSeen)
			}

			if q != tt.wantQ {
				t.Fatalf("acceptQuality(%q, %s): got %v, want %v", tt.accept, tt.enc, q, tt.wantQ)
			}
		})
	}
}
