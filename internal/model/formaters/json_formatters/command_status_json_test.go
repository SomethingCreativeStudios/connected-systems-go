package json_formatters

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

func TestDecodeCommandStatusReportHTTPPayload(t *testing.T) {
	status, err := DecodeCommandStatusReport(strings.NewReader(`{
		"statusCode":"EXECUTING",
		"percentCompletion":25,
		"executionTime":["2026-08-22T12:00:00Z"],
		"results":[]
	}`), false)

	require.NoError(t, err)
	require.Equal(t, domains.CommandStatusExecuting, status.StatusCode)
	require.NotNil(t, status.PercentCompletion)
	require.NotNil(t, status.ExecutionTime)
}

func TestDecodeCommandStatusReportCompleteRequiresResponseFields(t *testing.T) {
	_, err := DecodeCommandStatusReport(strings.NewReader(`{"statusCode":"ACCEPTED"}`), true)
	require.EqualError(t, err, "reportTime is required for a complete Resource Data Message")
}

func TestDecodeCommandStatusReportRejectsLegacyStatusUpdate(t *testing.T) {
	_, err := DecodeCommandStatusReport(strings.NewReader(`{"currentStatus":"COMPLETED"}`), true)
	require.EqualError(t, err, "statusCode is required")
}

func TestDecodeCommandStatusReportRejectsInvalidFields(t *testing.T) {
	_, err := DecodeCommandStatusReport(strings.NewReader(`{"statusCode":"BOGUS"}`), false)
	require.EqualError(t, err, "statusCode is invalid")

	_, err = DecodeCommandStatusReport(strings.NewReader(`{"statusCode":"EXECUTING","percentCompletion":101}`), false)
	require.EqualError(t, err, "percentCompletion must be between 0 and 100")

	_, err = DecodeCommandStatusReport(strings.NewReader(`{"statusCode":"EXECUTING","message":""}`), false)
	require.EqualError(t, err, "message must be a non-empty string")

	_, err = DecodeCommandStatusReport(strings.NewReader(`{"statusCode":"EXECUTING","results":null}`), false)
	require.EqualError(t, err, "results must be an array")
}
