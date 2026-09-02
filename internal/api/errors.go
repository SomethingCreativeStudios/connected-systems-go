package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/contractvalidation"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// ValidationErrorResponse is returned for request-body validation failures.
// Error remains available for existing clients; Details lets callers locate
// every actionable field without parsing a human-oriented sentence.
type ValidationErrorResponse struct {
	Error   string                         `json:"error"`
	Details []contractvalidation.Violation `json:"details,omitempty"`
}

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
	writeValidationError(w, r, err)
}

// writeValidationError writes a consistent 400 body for both schema-backed
// and endpoint-specific request validation. Existing clients can keep reading
// error; new clients can render the structured field details.
func writeValidationError(w http.ResponseWriter, r *http.Request, err error) {
	render.Status(r, http.StatusBadRequest)
	var validationErr *contractvalidation.Error
	if errors.As(err, &validationErr) {
		render.JSON(w, r, ValidationErrorResponse{
			Error:   validationErr.Error(),
			Details: validationErr.Details,
		})
		return
	}
	message := sanitizeJSONError(err)
	render.JSON(w, r, ValidationErrorResponse{
		Error:   message,
		Details: []contractvalidation.Violation{violationForDecodeError(err, message)},
	})
}

func violationForDecodeError(err error, message string) contractvalidation.Violation {
	path := "$"
	var unknownField *common_shared.UnknownFieldError
	if errors.As(err, &unknownField) {
		path = unknownField.Field
		if unknownField.Path != "" {
			path = unknownField.Path + "." + unknownField.Field
		}
		return contractvalidation.Violation{Path: path, Message: message}
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) && typeErr.Field != "" {
		path = typeErr.Field
	}
	if path == "$" {
		path = pathFromValidationMessage(message)
	}
	return contractvalidation.Violation{Path: path, Message: message}
}

func pathFromValidationMessage(message string) string {
	for _, marker := range []string{":", " is ", " must ", " required"} {
		index := strings.Index(message, marker)
		if index <= 0 {
			continue
		}
		candidate := message[:index]
		if strings.ContainsAny(candidate, " \t") {
			continue
		}
		return candidate
	}
	return "$"
}

// decodeStrictJSONRequest reads a write payload once and rejects unknown
// fields before endpoint-specific validation runs.
func decodeStrictJSONRequest[T any](r *http.Request) (T, error) {
	var zero T
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return zero, err
	}
	return decodeStrictJSON[T](body)
}

func decodeStrictJSON[T any](body []byte) (T, error) {
	return common_shared.DecodeWithFieldErrors[T](body)
}

// validateRawRequestBody applies the request-validation registry before an
// endpoint-specific decoder consumes the body. It returns a replayable body
// for handlers that do not use a multi-format formatter collection.
func validateRawRequestBody(r *http.Request, validator *contractvalidation.Validator, resource contractvalidation.Resource) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if err := validator.Validate(resource, r.Header.Get("Content-Type"), body); err != nil {
		return nil, err
	}
	return bytes.Clone(body), nil
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
