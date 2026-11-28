package daemon

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"

	"github.com/larsartmann/go-codec"
)

// bufPool reuses bytes.Buffers across response writes to avoid allocating a
// full-size []byte for every HTTP response. Shared by JSON and CBOR paths.
// After warmup, the pool's buffer grows to the largest response size and is
// reused indefinitely, reducing GC pressure on the daemon's hot path.
var bufPool = sync.Pool{ //nolint:gochecknoglobals // package-level sync.Pool is the idiomatic shared-buffer pattern
	New: func() any { return new(bytes.Buffer) },
}

// WriteEncoded serializes v with the given codec and writes it to the response.
// Uses the codec's BufferEncoder interface when available for zero-allocation
// encoding directly into a pooled buffer.
//
// Every response carries "Vary: Accept": response encodings on this daemon
// are commonly chosen from the request's Accept header (ResolveResponseCodec),
// so a cache must not reuse a stored response for requests expressing a
// different Accept preference. The header is added rather than set, so
// handlers can declare additional variance (e.g. "Vary: Authorization").
// A nil contentCodec falls back to the default JSON codec, the same rule
// WriteJSONWithCodec and WriteErrorfWithCodec apply.
func WriteEncoded(w http.ResponseWriter, status int, contentCodec codec.Codec, v any) {
	if contentCodec == nil {
		contentCodec = codec.JSONCodec{}
	}

	buf := bufPool.Get().(*bytes.Buffer) //nolint:forcetypeassert // Pool always returns *bytes.Buffer

	buf.Reset()
	defer bufPool.Put(buf)

	var err error
	if be, ok := contentCodec.(codec.BufferEncoder); ok {
		err = be.EncodeToBuffer(v, buf)
	} else {
		var data []byte

		data, err = contentCodec.Encode(v)
		if err == nil {
			buf.Write(data)
		}
	}

	if err != nil {
		WriteErrorf(w, http.StatusInternalServerError, "encode response: %v", err)

		return
	}

	w.Header().Add("Vary", "Accept")
	w.Header().Set("Content-Type", ContentTypeFor(contentCodec))
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// WriteJSON writes a JSON response with the default v1-backed codec.
// Backward-compat wrapper around WriteJSONWithCodec.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	WriteJSONWithCodec(w, status, codec.JSONCodec{}, v)
}

// WriteJSONWithCodec writes a JSON response encoded with jsonCodec, the
// codec-aware counterpart of WriteJSON. Pass the codec the handler negotiated
// for this response (e.g. a JSONv2Codec resolved through WithJSONCodec) so the
// whole response stream keeps one JSON flavour instead of falling back to the
// v1 defaults. jsonCodec must report codec.EncodingJSON; nil keeps the default
// codec.
func WriteJSONWithCodec(w http.ResponseWriter, status int, jsonCodec codec.Codec, v any) {
	if jsonCodec == nil {
		jsonCodec = codec.JSONCodec{}
	}

	WriteEncoded(w, status, jsonCodec, v)
}

// WriteErrorf writes a JSON error response with the given status and
// formatted message: {"error": "<message>"}. Backward-compat wrapper around
// WriteErrorfWithCodec.
func WriteErrorf(w http.ResponseWriter, status int, format string, args ...any) {
	WriteErrorfWithCodec(w, status, codec.JSONCodec{}, format, args...)
}

// WriteErrorfWithCodec writes the {"error": "<message>"} envelope encoded with
// contentCodec, the codec-aware counterpart of WriteErrorf. Pass the codec the
// handler negotiated for this response (JSON, CBOR, or an injected
// JSONv2Codec) so an error body never regresses to an encoding the rest of the
// response stream does not use. nil keeps the default JSON codec.
//
//nolint:goprintffuncname // mirrors WriteErrorf; the suffix names the codec, not a format tier
func WriteErrorfWithCodec(w http.ResponseWriter, status int, contentCodec codec.Codec, format string, args ...any) {
	if contentCodec == nil {
		contentCodec = codec.JSONCodec{}
	}

	WriteEncoded(w, status, contentCodec, map[string]string{
		"error": fmt.Sprintf(format, args...),
	})
}
