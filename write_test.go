package daemon

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-codec"
)

var (
	errEncodeBoom   = errors.New("boom")
	errFallbackBoom = errors.New("fallback boom")
)

// stubCodec is a Codec that returns canned results, letting tests inject
// encode failures without depending on a real codec's failure modes.
type stubCodec struct {
	encoding codec.Encoding
	data     []byte
	encErr   error
}

func (c stubCodec) Encoding() codec.Encoding   { return c.encoding }
func (c stubCodec) Encode(any) ([]byte, error) { return c.data, c.encErr }
func (c stubCodec) Decode([]byte, any) error   { return nil }

// stubBufferCodec is a Codec + BufferEncoder whose EncodeToBuffer fails with
// a canned error, isolating the WriteEncoded buffer path from real codecs.
type stubBufferCodec struct {
	bufErr error
}

func (stubBufferCodec) Encoding() codec.Encoding                  { return codec.EncodingJSON }
func (stubBufferCodec) Encode(any) ([]byte, error)                { return nil, nil }
func (stubBufferCodec) Decode([]byte, any) error                  { return nil }
func (c stubBufferCodec) EncodeToBuffer(any, *bytes.Buffer) error { return c.bufErr }

func TestWriteEncoded_BufferEncoderEncodeError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()

	WriteEncoded(rec, http.StatusOK, stubBufferCodec{
		bufErr: errEncodeBoom,
	}, map[string]string{"secret": "payload"})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	body := rec.Body.String()

	wantBody := `{"error":"encode response: boom"}`
	if body != wantBody {
		t.Errorf("body = %q, want %q", body, wantBody)
	}

	if bytes.Contains(rec.Body.Bytes(), []byte("payload")) {
		t.Error("partial encoded payload leaked into the error response")
	}
}

func TestWriteEncoded_FallbackCodec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		codec      stubCodec
		wantStatus int
		wantBody   string
	}{
		{
			name:       "encode success writes data with headers",
			codec:      stubCodec{encoding: codec.EncodingJSON, data: []byte(`{"k":"v"}`)},
			wantStatus: http.StatusOK,
			wantBody:   `{"k":"v"}`,
		},
		{
			name:       "encode error becomes 500 error body",
			codec:      stubCodec{encoding: codec.EncodingJSON, encErr: errFallbackBoom},
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":"encode response: fallback boom"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()

			WriteEncoded(rec, http.StatusOK, tt.codec, map[string]string{"k": "v"})

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if got := rec.Body.String(); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}

			if got := rec.Header().Get("Vary"); got != "Accept" {
				t.Errorf("Vary = %q, want Accept", got)
			}

			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
		})
	}
}
