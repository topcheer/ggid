package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// jsonUnmarshalBody decodes a JSON request body into the given interface.
func jsonUnmarshalBody(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// suppress unused import
var _ = fmt.Sprintf
