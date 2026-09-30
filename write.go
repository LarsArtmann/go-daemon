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
func WriteEncoded(w http.ResponseWriter, status int, contentCodec codec.Codec, v any) {
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

// WriteJSON writes a JSON response. Backward-compat wrapper around WriteEncoded.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	WriteEncoded(w, status, codec.JSONCodec{}, v)
}

// WriteErrorf writes a JSON error response with the given status and
// formatted message: {"error": "<message>"}.
func WriteErrorf(w http.ResponseWriter, status int, format string, args ...any) {
	WriteJSON(w, status, map[string]string{
		"error": fmt.Sprintf(format, args...),
	})
}
