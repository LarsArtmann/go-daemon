package daemon

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/larsartmann/go-codec"
)

const (
	contentTypeJSON = "application/json"
	contentTypeCBOR = "application/cbor"
)

// NegotiationOption customizes the codecs that negotiation resolves to. The
// defaults are codec.JSONCodec and codec.CBORCodec; pass options to
// ResolveRequestCodec, ResolveResponseCodec, or CodecForContentType to
// substitute codecs with different semantics (e.g. JSONv2Codec for
// encoding/json/v2 marshal options) without touching the negotiation
// contract itself.
type NegotiationOption func(*negotiationCodecs)

// negotiationCodecs holds the codecs negotiation resolves to, one per
// supported encoding.
type negotiationCodecs struct {
	json codec.Codec
	cbor codec.Codec
}

// resolveCodecs applies opts to the built-in defaults.
func resolveCodecs(opts []NegotiationOption) negotiationCodecs {
	codecs := negotiationCodecs{json: codec.JSONCodec{}, cbor: codec.CBORCodec{}}

	for _, opt := range opts {
		if opt != nil {
			opt(&codecs)
		}
	}

	return codecs
}

// WithJSONCodec substitutes the codec negotiation resolves to for JSON in
// both directions. c must report codec.EncodingJSON so content-type
// handling stays coherent; a nil codec keeps the default.
func WithJSONCodec(c codec.Codec) NegotiationOption {
	return func(codecs *negotiationCodecs) {
		if c != nil {
			codecs.json = c
		}
	}
}

// WithCBORCodec substitutes the codec negotiation resolves to for CBOR in
// both directions. c must report codec.EncodingCBOR so content-type
// handling stays coherent; a nil codec keeps the default.
func WithCBORCodec(c codec.Codec) NegotiationOption {
	return func(codecs *negotiationCodecs) {
		if c != nil {
			codecs.cbor = c
		}
	}
}

// ResolveRequestCodec determines the codec for decoding the request body
// based on the Content-Type header. Defaults to JSON for backward
// compatibility (existing clients send no Content-Type or application/json).
// Structured-suffix media types count as their base type (RFC 6839), so
// "application/vnd.acme+cbor" decodes as CBOR. Pass NegotiationOption values
// to substitute the resolved codecs (WithJSONCodec, WithCBORCodec).
func ResolveRequestCodec(r *http.Request, opts ...NegotiationOption) codec.Codec {
	return CodecForContentType(r.Header.Get("Content-Type"), opts...)
}

// ResolveResponseCodec determines the codec for encoding the response body
// from the Accept header (RFC 9110 §12.5.1), falling back to the request
// codec so the response matches the request encoding when no preference is
// expressed: absent header, wildcards like "*/*", or only unrelated types.
//
// With both encodings mentioned, the higher quality value wins and a tie
// prefers CBOR. Quality 0 is an explicit refusal: the refused encoding is
// never returned — when the request codec is the refused one, the other
// supported encoding is served instead. An Accept header that refuses both
// encodings is unsatisfiable; this mechanism cannot answer 406, so the
// request codec stands.
//
// Only the media type is negotiated. Accept-Charset and Accept-Encoding
// are deliberately out of scope: JSON is UTF-8 and CBOR is a binary
// format, so charset parameters carry no additional meaning here, and
// content codings (gzip, zstd) belong to the HTTP transport layer, not
// this package. Responses written via WriteEncoded carry "Vary: Accept".
//
// Pass NegotiationOption values to substitute the codecs resolved for each
// encoding (WithJSONCodec, WithCBORCodec); negotiation ranks media types the
// same way regardless of which codec instance each encoding maps to.
func ResolveResponseCodec(r *http.Request, reqCodec codec.Codec, opts ...NegotiationOption) codec.Codec {
	codecs := resolveCodecs(opts)
	accept := r.Header.Get("Accept")
	cborQ, cborWanted := acceptQuality(accept, codec.EncodingCBOR)
	jsonQ, jsonWanted := acceptQuality(accept, codec.EncodingJSON)

	switch {
	case cborWanted && jsonWanted:
		switch {
		case jsonQ > cborQ:
			return codecs.json
		case cborQ > jsonQ:
			return codecs.cbor
		case cborQ > 0:
			return codecs.cbor
		default:
			return reqCodec
		}
	case cborWanted:
		if cborQ > 0 {
			return codecs.cbor
		}

		return substituteForRefused(codecs, codec.EncodingCBOR, reqCodec)
	case jsonWanted:
		if jsonQ > 0 {
			return codecs.json
		}

		return substituteForRefused(codecs, codec.EncodingJSON, reqCodec)
	default:
		return reqCodec
	}
}

// substituteForRefused resolves the response codec after an explicit
// quality-0 refusal: when the request codec is the refused encoding, the
// other supported encoding is served so a refused encoding is never
// returned; otherwise the request codec stands.
func substituteForRefused(codecs negotiationCodecs, refusedEnc codec.Encoding, reqCodec codec.Codec) codec.Codec {
	if reqCodec.Encoding() != refusedEnc {
		return reqCodec
	}

	if refusedEnc == codec.EncodingCBOR {
		return codecs.json
	}

	return codecs.cbor
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
// list, defaulting to 1 (RFC 9110 §12.5.1) and clamping to [0, 1].
// Parameter names are case-insensitive (RFC 9110 §5.6.2.1), so "Q=" matches
// "q=". Values that fail to parse — including NaN, which orders against
// nothing — are skipped, so a parseable q elsewhere in the entry still wins
// and entries without any valid q default to 1.
func qualityParam(params string) float64 {
	for param := range strings.SplitSeq(params, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(param), "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(name), "q") {
			continue
		}

		if q, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && !math.IsNaN(q) {
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
// unknown or missing content types. Pass NegotiationOption values to
// substitute the resolved codecs (WithJSONCodec, WithCBORCodec).
func CodecForContentType(headerVal string, opts ...NegotiationOption) codec.Codec {
	codecs := resolveCodecs(opts)
	if matchesEncoding(bareMediaType(headerVal), codec.EncodingCBOR) {
		return codecs.cbor
	}

	return codecs.json
}
