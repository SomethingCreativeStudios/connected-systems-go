package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/contractvalidation"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/go-chi/render"
	"go.uber.org/zap"
)

// ValidationErrorResponse is returned for request-body validation failures.
// Error remains available for existing clients; Details lets callers locate
// every actionable field without parsing a human-oriented sentence.
type ValidationErrorResponse struct {
	Error   string                         `json:"error"`
	Details []contractvalidation.Violation `json:"details,omitempty"`
}

// writeNegotiated encodes a content-negotiated body without clobbering the
// Content-Type header. render.JSON always forces mediatype.JSON, which
// overwrites the negotiated Content-Type (e.g. application/geo+json,
// application/sml+json) set by the caller. Callers must set the Content-Type
// header before calling this helper.
func writeNegotiated(w http.ResponseWriter, body any) {
	// Encode before writing the status so a failure is a 500, not a
	// truncated 200, and keep hrefs readable ("&" instead of "\u0026").
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(body); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		//nolint:errcheck
		w.Write([]byte(`{"error":"Failed to serialize response"}` + "\n"))
		return
	}
	w.WriteHeader(http.StatusOK)
	//nolint:errcheck
	w.Write(buf.Bytes())
}

// writeError writes the API's error body with the given status.
func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	render.Status(r, status)
	render.JSON(w, r, map[string]string{"error": message})
}

// writeRepositoryError answers a failed repository call for the named
// resource: 404 when it (or the parent it is created under) does not exist,
// 409 when it still has children or collides with an existing resource, and
// 500 otherwise. Only the unexpected failures are logged.
func writeRepositoryError(w http.ResponseWriter, r *http.Request, logger *zap.Logger, err error, resource, failure string) {
	var validationErr *repository.ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeValidationError(w, r, validationErr.Err)
	case errors.Is(err, repository.ErrParentNotFound):
		writeError(w, r, http.StatusNotFound, err.Error())
	case repository.IsNotFound(err):
		writeError(w, r, http.StatusNotFound, resource+" not found")
	case errors.Is(err, repository.ErrSchemaInUse):
		writeError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, repository.ErrHasChildren):
		writeError(w, r, http.StatusConflict, resource+" has dependent records; use ?cascade=true to delete")
	case repository.IsUniqueViolation(err):
		writeError(w, r, http.StatusConflict, "A resource with the same unique identifier already exists")
	default:
		logger.Error(failure, zap.String("path", r.URL.Path), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, failure)
	}
}

// writeSerializeError answers a response representation that could not be
// built. A partial or empty body would read as a valid, empty result.
func writeSerializeError(w http.ResponseWriter, r *http.Request, logger *zap.Logger, err error) {
	logger.Error("Failed to serialize response", zap.String("path", r.URL.Path), zap.Error(err))
	writeError(w, r, http.StatusInternalServerError, "Failed to serialize response")
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
	if tooLarge, ok := errors.AsType[*http.MaxBytesError](err); ok {
		writeError(w, r, http.StatusRequestEntityTooLarge, fmt.Sprintf("request body exceeds %d bytes", tooLarge.Limit))
		return
	}
	render.Status(r, http.StatusBadRequest)
	if validationErr, ok := errors.AsType[*contractvalidation.Error](err); ok {
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
	if unknownField, ok := errors.AsType[*common_shared.UnknownFieldError](err); ok {
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
	if ufErr, ok := errors.AsType[*common_shared.UnknownFieldError](err); ok {
		if ufErr.Path == "" {
			return fmt.Sprintf("unknown field '%s'", ufErr.Field)
		}
		return fmt.Sprintf("unknown field '%s' in %s", ufErr.Field, ufErr.Path)
	}

	// json.UnmarshalTypeError — field-level type mismatch
	if typeErr, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		return fmt.Sprintf("field '%s': expected %s, got %s",
			typeErr.Field, typeErr.Type.Kind().String(), typeErr.Value)
	}

	// json.SyntaxError — malformed JSON
	if syntaxErr, ok := errors.AsType[*json.SyntaxError](err); ok {
		return fmt.Sprintf("invalid JSON at offset %d: %s", syntaxErr.Offset, msg)
	}

	// Strip "json: " prefix from encoding/json error wrappers
	msg = strings.TrimPrefix(msg, "json: ")

	// Strip "parsing time " prefix from time.ParseError messages
	msg = strings.TrimPrefix(msg, "parsing time ")

	return msg
}

// defaultMaxBodyBytes bounds request bodies when api.max_body_bytes is unset.
const defaultMaxBodyBytes int64 = 10 << 20

// limitRequestBody caps every request body. Handlers read bodies whole into
// memory before decoding them, so without a cap one request could exhaust it.
func limitRequestBody(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
