package daemon

import (
	"net/http"
	"strings"

	"github.com/larsartmann/go-codec"
)

const (
	contentTypeJSON = "application/json"
	contentTypeCBOR = "application/cbor"
)

// ResolveRequestCodec determines the codec for decoding the request body
// based on the Content-Type header. Defaults to JSON for backward
// compatibility (existing clients send no Content-Type or application/json).
func ResolveRequestCodec(r *http.Request) codec.Codec {
	if isCBORContentType(r.Header.Get("Content-Type")) {
		return codec.CBORCodec{}
	}

	return codec.JSONCodec{}
}

// ResolveResponseCodec determines the codec for encoding the response body
// based on the Accept header, falling back to the request codec so the
// response matches the request encoding when no preference is expressed.
func ResolveResponseCodec(r *http.Request, reqCodec codec.Codec) codec.Codec {
	accept := r.Header.Get("Accept")

	switch {
	case isCBORContentType(accept):
		return codec.CBORCodec{}
	case strings.Contains(accept, contentTypeJSON):
		return codec.JSONCodec{}
	default:
		return reqCodec
	}
}

// isCBORContentType reports whether the header value indicates CBOR.
// Handles parameter suffixes like "application/cbor; charset=utf-8".
func isCBORContentType(headerVal string) bool {
	return strings.HasPrefix(strings.TrimSpace(strings.ToLower(headerVal)), contentTypeCBOR)
}

// ContentTypeFor returns the HTTP Content-Type header value for a codec.
func ContentTypeFor(c codec.Codec) string {
	if c.Encoding() == codec.EncodingCBOR {
		return contentTypeCBOR
	}

	return contentTypeJSON
}

// CodecForContentType resolves a codec from a response Content-Type header.
// Falls back to JSON for unknown/missing content types.
func CodecForContentType(headerVal string) codec.Codec {
	if isCBORContentType(headerVal) {
		return codec.CBORCodec{}
	}

	return codec.JSONCodec{}
}
