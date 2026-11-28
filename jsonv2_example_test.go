package daemon

import (
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
)

// exampleTimeout is the payload of the codec examples: one duration and one
// slice, the two shapes where json/v2 defaults and encoding/json (v1) differ.
type exampleTimeout struct {
	Timeout time.Duration `json:"timeout"`
	Scopes  []string      `json:"scopes"`
}

// ExampleJSONv2Codec shows the two differences a consumer meets first: the
// zero value has no time.Duration representation (Encode returns an error),
// while the v1 interop options restore the nanosecond integer and the null
// for a nil slice.
func ExampleJSONv2Codec() {
	in := exampleTimeout{Timeout: 1500 * time.Millisecond}

	_, err := JSONv2Codec{}.Encode(in)
	fmt.Println("default codec rejects the duration:", err != nil)

	v1Shapes := JSONv2Codec{MarshalOptions: []json.Options{
		jsonv1.FormatDurationAsNano(true),
		json.FormatNilSliceAsNull(true),
	}}

	data, _ := v1Shapes.Encode(in)
	fmt.Println("v1 shapes:", string(data))

	// Output:
	// default codec rejects the duration: true
	// v1 shapes: {"timeout":1500000000,"scopes":null}
}

// ExampleWriteJSONWithCodec pins v1 wire shapes on one response while the
// rest of the daemon keeps the default codec: pass the codec the handler
// negotiated instead of the codec-less WriteJSON.
func ExampleWriteJSONWithCodec() {
	v1Shapes := JSONv2Codec{MarshalOptions: []json.Options{
		jsonv1.FormatDurationAsNano(true),
		json.FormatNilSliceAsNull(true),
	}}

	rec := httptest.NewRecorder()
	WriteJSONWithCodec(rec, http.StatusOK, v1Shapes, exampleTimeout{Timeout: time.Second})

	fmt.Println(rec.Code)
	fmt.Println(rec.Header().Get("Content-Type"))
	fmt.Println(rec.Body.String())

	// Output:
	// 200
	// application/json
	// {"timeout":1000000000,"scopes":null}
}

// ExampleWriteErrorfWithCodec keeps the error envelope in the response
// stream's encoding; passing nil selects the default JSON codec.
func ExampleWriteErrorfWithCodec() {
	rec := httptest.NewRecorder()
	WriteErrorfWithCodec(rec, http.StatusBadRequest, JSONv2Codec{}, "unknown scope %q", "admin")

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())

	// Output:
	// 400
	// {"error":"unknown scope \"admin\""}
}
