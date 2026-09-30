package daemon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

// ExampleResolveResponseCodec shows the negotiation round trip a handler
// performs: resolve the request codec from Content-Type, choose the response
// codec from Accept, then write with WriteEncoded — which sets Content-Type
// and emits "Vary: Accept" so caches key on the client's stated preference.
// A structured-suffix Content-Type (RFC 6839) counts as its base encoding,
// so "+cbor" requests decode as CBOR while the response honors the client's
// explicit JSON preference.
func ExampleResolveResponseCodec() {
	type inventory struct {
		Projects []string `json:"projects"`
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		reqCodec := ResolveRequestCodec(r)
		respCodec := ResolveResponseCodec(r, reqCodec)
		WriteEncoded(w, http.StatusOK, respCodec, inventory{Projects: []string{"go-daemon", "go-sse"}})
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/inventory", nil)
	req.Header.Set("Content-Type", "application/vnd.example+cbor")
	req.Header.Set("Accept", "application/json")

	rec := httptest.NewRecorder()
	handler(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Header().Get("Content-Type"))
	fmt.Println(rec.Header().Get("Vary"))
	fmt.Println(strings.TrimSpace(rec.Body.String()))

	// Output:
	// 200
	// application/json
	// Accept
	// {"projects":["go-daemon","go-sse"]}
}
