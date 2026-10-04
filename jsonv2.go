package daemon

import (
	"bytes"
	"encoding/json/v2"

	"github.com/larsartmann/go-codec"
)

// JSONv2Codec implements codec.Codec over encoding/json/v2 with caller-chosen
// options, so consumers already committed to json/v2 semantics (omitzero
// tags, deterministic map ordering, duration formats) can adopt content
// negotiation without regressing to the v1-backed codec.JSONCodec default.
//
// The zero value marshals and unmarshals with pure json/v2 defaults, which
// differ from encoding/json (v1) in observable ways:
//
//   - time.Duration has no default representation: Encode and Decode return a
//     semantic error unless encoding/json.FormatDurationAsNano(true) is set.
//   - map iteration order is non-deterministic unless
//     json.Deterministic(true) is set.
//   - nil maps and slices marshal as {} and [] instead of null unless
//     json.FormatNilMapAsNull(true) / json.FormatNilSliceAsNull(true) are set.
//   - Decode matches struct field names case-sensitively unless
//     json.MatchCaseInsensitiveNames(true) is set.
//
// Wire it into negotiation with WithJSONCodec(JSONv2Codec{...}). It reports
// codec.EncodingJSON, so ContentTypeFor and WriteEncoded treat it exactly
// like the default JSON codec, including the zero-allocation buffer path.
type JSONv2Codec struct {
	// MarshalOptions configure Encode and EncodeToBuffer, e.g.
	// json.Deterministic(true) or encoding/json.FormatDurationAsNano(true).
	// Nil means json/v2 defaults.
	MarshalOptions []json.Options

	// UnmarshalOptions configure Decode, e.g.
	// json.MatchCaseInsensitiveNames(true). Nil means json/v2 defaults.
	UnmarshalOptions []json.Options
}

var (
	_ codec.Codec         = JSONv2Codec{}
	_ codec.BufferEncoder = JSONv2Codec{}
)

// Encoding identifies JSON as the serialization format.
func (JSONv2Codec) Encoding() codec.Encoding { return codec.EncodingJSON }

// Encode marshals v to JSON bytes with the configured marshal options.
func (c JSONv2Codec) Encode(v any) ([]byte, error) {
	//nolint:wrapcheck // thin adapter: pass json/v2 errors through verbatim
	return json.Marshal(v, c.MarshalOptions...)
}

// Decode unmarshals JSON bytes into v with the configured unmarshal options.
func (c JSONv2Codec) Decode(data []byte, v any) error {
	//nolint:wrapcheck // thin adapter: pass json/v2 errors through verbatim
	return json.Unmarshal(data, v, c.UnmarshalOptions...)
}

// EncodeToBuffer writes the JSON encoding of v directly into buf, avoiding
// the allocation Encode returns. Implements codec.BufferEncoder, so
// WriteEncoded takes the zero-allocation path.
func (c JSONv2Codec) EncodeToBuffer(v any, buf *bytes.Buffer) error {
	//nolint:wrapcheck // thin adapter: pass json/v2 errors through verbatim
	return json.MarshalWrite(buf, v, c.MarshalOptions...)
}
