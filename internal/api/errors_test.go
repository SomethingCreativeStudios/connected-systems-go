package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
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
