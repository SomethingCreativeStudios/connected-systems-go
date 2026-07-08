package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// writeNegotiated encodes a content-negotiated body without clobbering the
// Content-Type header. render.JSON always forces "application/json", which
// overwrites the negotiated Content-Type (e.g. application/geo+json,
// application/sml+json) set by the caller. Callers must set the Content-Type
// header before calling this helper.
func writeNegotiated(w http.ResponseWriter, body any) {
	w.WriteHeader(http.StatusOK)
	//nolint:errcheck
	json.NewEncoder(w).Encode(body)
}

// writeDeserializeError writes a 400 Bad Request response with a cleaned-up
// deserialization error message. The caller should log the raw error before
// calling this helper.
func writeDeserializeError(w http.ResponseWriter, r *http.Request, err error) {
	render.Status(r, http.StatusBadRequest)
	render.JSON(w, r, map[string]string{"error": sanitizeJSONError(err)})
}

// sanitizeJSONError cleans up raw Go JSON decoding errors into human-readable
// messages suitable for API error responses.
func sanitizeJSONError(err error) string {
	msg := err.Error()

	// common_shared.UnknownFieldError — strict-mode unknown field
	var ufErr *common_shared.UnknownFieldError
	if errors.As(err, &ufErr) {
		if ufErr.Path == "" {
			return fmt.Sprintf("unknown field '%s'", ufErr.Field)
		}
		return fmt.Sprintf("unknown field '%s' in %s", ufErr.Field, ufErr.Path)
	}

	// json.UnmarshalTypeError — field-level type mismatch
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return fmt.Sprintf("field '%s': expected %s, got %s",
			typeErr.Field, typeErr.Type.Kind().String(), typeErr.Value)
	}

	// json.SyntaxError — malformed JSON
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Sprintf("invalid JSON at offset %d: %s", syntaxErr.Offset, msg)
	}

	// Strip "json: " prefix from encoding/json error wrappers
	msg = strings.TrimPrefix(msg, "json: ")

	// Strip "parsing time " prefix from time.ParseError messages
	msg = strings.TrimPrefix(msg, "parsing time ")

	return msg
}
