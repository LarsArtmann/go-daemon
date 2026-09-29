package daemon

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-codec"
)

func TestResolveRequestCodec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		want        codec.Encoding
	}{
		{"JSON explicit", "application/json", codec.EncodingJSON},
		{"JSON with charset", "application/json; charset=utf-8", codec.EncodingJSON},
		{"CBOR", "application/cbor", codec.EncodingCBOR},
		{"CBOR with param", "application/cbor; foo=bar", codec.EncodingCBOR},
		{"CBOR structured suffix", "application/vnd.acme+cbor", codec.EncodingCBOR},
		{"uppercase CBOR", "APPLICATION/CBOR; charset=utf-8", codec.EncodingCBOR},
		{"CBOR-seq is not plain CBOR", "application/cbor-seq", codec.EncodingJSON},
		{"near-miss type is not CBOR", "application/cborx", codec.EncodingJSON},
		{"JSON structured suffix", "application/vnd.acme+json", codec.EncodingJSON},
		{"uppercase JSON", "APPLICATION/JSON", codec.EncodingJSON},
		{"empty defaults to JSON", "", codec.EncodingJSON},
		{"unknown defaults to JSON", "application/xml", codec.EncodingJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/discover", nil)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			got := ResolveRequestCodec(req)
			if got.Encoding() != tt.want {
				t.Fatalf("got %q, want %q", got.Encoding(), tt.want)
			}
		})
	}
}

func TestResolveResponseCodec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		accept   string
		reqCodec codec.Codec
		want     codec.Encoding
	}{
		{"Accept CBOR", "application/cbor", codec.JSONCodec{}, codec.EncodingCBOR},
		{"Accept JSON", "application/json", codec.CBORCodec{}, codec.EncodingJSON},
		{"no Accept, JSON req → JSON", "", codec.JSONCodec{}, codec.EncodingJSON},
		{"no Accept, CBOR req → CBOR", "", codec.CBORCodec{}, codec.EncodingCBOR},
		{"wildcard, CBOR req → CBOR", "*/*", codec.CBORCodec{}, codec.EncodingCBOR},
		{"wildcard, JSON req → JSON", "*/*", codec.JSONCodec{}, codec.EncodingJSON},
		{"q prefers CBOR", "application/json;q=0.5, application/cbor;q=0.9", codec.JSONCodec{}, codec.EncodingCBOR},
		{"q prefers JSON", "application/cbor;q=0.1, application/json;q=0.9", codec.CBORCodec{}, codec.EncodingJSON},
		{"tie prefers CBOR", "application/json, application/cbor", codec.JSONCodec{}, codec.EncodingCBOR},
		{"suffix selects CBOR", "application/vnd.acme+cbor", codec.JSONCodec{}, codec.EncodingCBOR},
		{"suffix selects JSON", "application/vnd.acme+json", codec.CBORCodec{}, codec.EncodingJSON},
		{"JSON patch suffix selects JSON", "application/json-patch+json", codec.CBORCodec{}, codec.EncodingJSON},
		{"q=0 refuses JSON, JSON req substituted", "application/json;q=0", codec.JSONCodec{}, codec.EncodingCBOR},
		{"q=0 refuses JSON, CBOR req echoed", "application/json;q=0", codec.CBORCodec{}, codec.EncodingCBOR},
		{"q=0 refuses CBOR, CBOR req substituted", "application/cbor;q=0", codec.CBORCodec{}, codec.EncodingJSON},
		{
			"q=0 both refused echoes request",
			"application/cbor;q=0, application/json;q=0",
			codec.JSONCodec{},
			codec.EncodingJSON,
		},
		{"unparseable q counts as 1", "application/cbor;q=abc", codec.JSONCodec{}, codec.EncodingCBOR},
		{
			"browser-style header echoes request",
			"text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8",
			codec.JSONCodec{},
			codec.EncodingJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/discover", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}

			got := ResolveResponseCodec(req, tt.reqCodec)
			if got.Encoding() != tt.want {
				t.Fatalf("got %q, want %q", got.Encoding(), tt.want)
			}
		})
	}
}

type codecTestPayload struct {
	Projects []codecTestProject `json:"projects"`
}

type codecTestProject struct {
	Path    string `json:"path"`
	DirName string `json:"dir_name"`
}

func TestWriteEncoded_JSONAndCBOR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		codec       codec.Codec
		wantContent string
	}{
		{"JSON", codec.JSONCodec{}, "application/json"},
		{"CBOR", codec.CBORCodec{}, "application/cbor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			payload := codecTestPayload{Projects: []codecTestProject{{Path: "/test", DirName: "test"}}}

			WriteEncoded(w, http.StatusOK, tt.codec, payload)

			if w.Code != http.StatusOK {
				t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
			}

			if ct := w.Header().Get("Content-Type"); ct != tt.wantContent {
				t.Fatalf("content-type: got %q, want %q", ct, tt.wantContent)
			}

			var decoded codecTestPayload

			if err := tt.codec.Decode(w.Body.Bytes(), &decoded); err != nil {
				t.Fatalf("decode: %v", err)
			}

			if len(decoded.Projects) != 1 || decoded.Projects[0].Path != "/test" {
				t.Fatalf("unexpected payload: %+v", decoded)
			}
		})
	}
}

func TestCBOR_RoundTrip(t *testing.T) {
	t.Parallel()

	payload := codecTestPayload{Projects: []codecTestProject{
		{Path: "/home/user/projects", DirName: "projects"},
		{Path: "/home/user/work", DirName: "work"},
	}}

	data, err := codec.CBORCodec{}.Encode(payload)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	var decoded codecTestPayload
	if err := (codec.CBORCodec{}).Decode(data, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(decoded.Projects) != 2 || decoded.Projects[0].Path != "/home/user/projects" {
		t.Fatalf("round-trip mismatch: %+v", decoded)
	}
}

func TestWriteErrorf(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()

	WriteErrorf(w, http.StatusBadRequest, "bad thing: %d", 42)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	if body := w.Body.String(); body != `{"error":"bad thing: 42"}`+"\n" && body != `{"error":"bad thing: 42"}` {
		t.Fatalf("unexpected error body: %q", body)
	}
}

func TestContentTypeFor(t *testing.T) {
	t.Parallel()

	if got := ContentTypeFor(codec.JSONCodec{}); got != "application/json" {
		t.Fatalf("json content type: got %q", got)
	}

	if got := ContentTypeFor(codec.CBORCodec{}); got != "application/cbor" {
		t.Fatalf("cbor content type: got %q", got)
	}
}

func TestCodecForContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
		want   codec.Encoding
	}{
		{"plain CBOR", "application/cbor", codec.EncodingCBOR},
		{"CBOR with param", "application/cbor; charset=utf-8", codec.EncodingCBOR},
		{"plain JSON", "application/json", codec.EncodingJSON},
		{"empty defaults to JSON", "", codec.EncodingJSON},
		{"unknown defaults to JSON", "application/xml", codec.EncodingJSON},
		{"CBOR structured suffix", "application/vnd.acme+cbor", codec.EncodingCBOR},
		{"uppercase", "Application/JSON", codec.EncodingJSON},
		{"near-miss type falls back", "application/cborx", codec.EncodingJSON},
	}

	for _, tt := range tests {
		if got := CodecForContentType(tt.header); got.Encoding() != tt.want {
			t.Fatalf("CodecForContentType(%q): got %q, want %q", tt.header, got.Encoding(), tt.want)
		}
	}
}
