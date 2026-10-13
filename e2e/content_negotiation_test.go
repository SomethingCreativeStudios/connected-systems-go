package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The response representation follows the f parameter, or else the Accept
// header with its quality values; a request for a representation the
// resource does not have is 406.
func TestContentNegotiation(t *testing.T) {
	cleanupDB(t)
	systemID := createSystemForControlStreamTest(t)

	get := func(path, accept string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, testServer.URL+path, nil)
		require.NoError(t, err)
		if accept != "" {
			req.Header.Set("Accept", accept)
		}
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { resp.Body.Close() })
		return resp
	}

	cases := []struct {
		name, path, accept string
		status             int
		contentType        string
	}{
		{"default", "/systems/" + systemID, "", http.StatusOK, "application/geo+json"},
		{"f parameter", "/systems/" + systemID + "?f=smljson", "application/geo+json", http.StatusOK, "application/sml+json"},
		{"quality values", "/systems/" + systemID, "application/geo+json;q=0.5, application/sml+json", http.StatusOK, "application/sml+json"},
		{"plain json", "/systems", "application/json", http.StatusOK, "application/geo+json"},
		{"browser accept", "/systems", "text/html,application/xhtml+xml,*/*;q=0.8", http.StatusOK, "application/geo+json"},
		{"unsupported", "/systems/" + systemID, "application/xml", http.StatusNotAcceptable, "application/json"},
		{"geojson on part 2", "/datastreams", "application/geo+json", http.StatusNotAcceptable, "application/json"},
		{"unknown f", "/datastreams?f=xml", "", http.StatusNotAcceptable, "application/json"},
		{"part 2 json", "/datastreams", "application/json", http.StatusOK, "application/json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := get(tc.path, tc.accept)
			assert.Equal(t, tc.status, resp.StatusCode)
			assert.Contains(t, resp.Header.Get("Content-Type"), tc.contentType)
			assert.Contains(t, resp.Header.Values("Vary"), "Accept")
		})
	}
}
