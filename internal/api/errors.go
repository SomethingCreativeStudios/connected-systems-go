package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/render"
)

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
