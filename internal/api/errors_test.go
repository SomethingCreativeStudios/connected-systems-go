package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/stretchr/testify/require"
)

func TestWriteValidationErrorUsesStrictDecoderFieldPath(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/systems", nil)

	writeValidationError(recorder, request, &common_shared.UnknownFieldError{
		Field: "definitions",
		Path:  "observedProperties[0]",
	})

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var response ValidationErrorResponse
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	require.Equal(t, "observedProperties[0].definitions", response.Details[0].Path)
	require.Equal(t, "unknown field 'definitions' in observedProperties[0]", response.Details[0].Message)
}

func TestWriteValidationErrorUsesSemanticFieldPath(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/systems", nil)

	writeValidationError(recorder, request, &fieldTestError{message: "percentCompletion must be between 0 and 100"})

	var response ValidationErrorResponse
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	require.Equal(t, "percentCompletion", response.Details[0].Path)
}

type fieldTestError struct{ message string }

func (e *fieldTestError) Error() string { return e.message }

// Request bodies are read whole before decoding, so an oversized one is cut
// off and answered with 413 rather than buffered.
func TestLimitRequestBodyRejectsOversizedBodies(t *testing.T) {
	handler := limitRequestBody(8)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			writeDeserializeError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	small := httptest.NewRecorder()
	handler.ServeHTTP(small, httptest.NewRequest(http.MethodPost, "/systems", strings.NewReader("{}")))
	require.Equal(t, http.StatusNoContent, small.Code)

	large := httptest.NewRecorder()
	handler.ServeHTTP(large, httptest.NewRequest(http.MethodPost, "/systems", strings.NewReader(`{"name":"too long"}`)))
	require.Equal(t, http.StatusRequestEntityTooLarge, large.Code)
}
