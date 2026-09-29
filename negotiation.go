package daemon

import (
	"net/http"
	"strconv"
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
// Structured-suffix media types count as their base type (RFC 6839), so
// "application/vnd.acme+cbor" decodes as CBOR.
func ResolveRequestCodec(r *http.Request) codec.Codec {
	return CodecForContentType(r.Header.Get("Content-Type"))
}

// ResolveResponseCodec determines the codec for encoding the response body
// from the Accept header (RFC 9110 §12.5.1), falling back to the request
// codec so the response matches the request encoding when no preference is
// expressed: absent header, wildcards like "*/*", or only unrelated types.
//
// With both encodings mentioned, the higher quality value wins and a tie
// prefers CBOR. Quality 0 is an explicit refusal: the refused encoding is
// never returned — when the request codec is the refused one, the other
// supported encoding is served instead.
func ResolveResponseCodec(r *http.Request, reqCodec codec.Codec) codec.Codec {
	accept := r.Header.Get("Accept")
	cborQ, cborWanted := acceptQuality(accept, codec.EncodingCBOR)
	jsonQ, jsonWanted := acceptQuality(accept, codec.EncodingJSON)

	switch {
	case cborWanted && jsonWanted:
		switch {
		case jsonQ > cborQ:
			return codec.JSONCodec{}
		case cborQ > jsonQ:
			return codec.CBORCodec{}
		case cborQ > 0:
			return codec.CBORCodec{}
		default:
			return reqCodec
		}
	case cborWanted:
		if cborQ > 0 {
			return codec.CBORCodec{}
		}

		return substituteForRefused(codec.CBORCodec{}, reqCodec)
	case jsonWanted:
		if jsonQ > 0 {
			return codec.JSONCodec{}
		}

		return substituteForRefused(codec.JSONCodec{}, reqCodec)
	default:
		return reqCodec
	}
}

// substituteForRefused resolves the response codec after an explicit
// quality-0 refusal: when the request codec is the refused encoding, the
// other supported encoding is served so a refused encoding is never
// returned; otherwise the request codec stands.
func substituteForRefused(refused, reqCodec codec.Codec) codec.Codec {
	if reqCodec.Encoding() == refused.Encoding() {
		if refused.Encoding() == codec.EncodingCBOR {
			return codec.JSONCodec{}
		}

		return codec.CBORCodec{}
	}

	return reqCodec
}

// acceptQuality reports the highest quality value the Accept header assigns
// to any media type representing enc (bare type or structured suffix), and
// whether the header mentions the encoding at all. Wildcards are ignored:
// they express no preference between JSON and CBOR, so callers fall back to
// the request codec when only wildcards match.
func acceptQuality(accept string, enc codec.Encoding) (float64, bool) {
	var (
		quality   float64
		mentioned bool
	)

	for entry := range strings.SplitSeq(accept, ",") {
		mediaType, params, _ := strings.Cut(entry, ";")
		if !matchesEncoding(bareMediaType(mediaType), enc) {
			continue
		}

		if q := qualityParam(params); !mentioned || q > quality {
			quality, mentioned = q, true
		}
	}

	return quality, mentioned
}

// matchesEncoding reports whether the bare media type represents enc: its
// canonical type or a structured-suffix variant (RFC 6839), e.g.
// "application/vnd.acme+cbor" for CBOR or "application/json-patch+json" for
// JSON.
func matchesEncoding(mediaType string, enc codec.Encoding) bool {
	base, suffix := contentTypeJSON, "+json"
	if enc == codec.EncodingCBOR {
		base, suffix = contentTypeCBOR, "+cbor"
	}

	return mediaType == base || strings.HasSuffix(mediaType, suffix)
}

// bareMediaType extracts the lowercased media type from a header value,
// stripping parameters: "application/cbor; charset=utf-8" →
// "application/cbor".
func bareMediaType(headerVal string) string {
	mediaType, _, _ := strings.Cut(headerVal, ";")

	return strings.ToLower(strings.TrimSpace(mediaType))
}

// qualityParam extracts the quality value from a media type's parameter
// list, defaulting to 1 (RFC 9110 §12.5.1) and clamping to [0, 1];
// unparseable values count as 1.
func qualityParam(params string) float64 {
	for param := range strings.SplitSeq(params, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(param), "=")
		if !ok || strings.TrimSpace(name) != "q" {
			continue
		}

		if q, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
			return min(max(q, 0), 1)
		}
	}

	return 1
}

// ContentTypeFor returns the HTTP Content-Type header value for a codec.
func ContentTypeFor(c codec.Codec) string {
	if c.Encoding() == codec.EncodingCBOR {
		return contentTypeCBOR
	}

	return contentTypeJSON
}

// CodecForContentType resolves a codec from a Content-Type header value.
// Structured-suffix media types count as their base type (RFC 6839), e.g.
// "application/vnd.acme+json" resolves as JSON. Falls back to JSON for
// unknown or missing content types.
func CodecForContentType(headerVal string) codec.Codec {
	if matchesEncoding(bareMediaType(headerVal), codec.EncodingCBOR) {
		return codec.CBORCodec{}
	}

	return codec.JSONCodec{}
}
