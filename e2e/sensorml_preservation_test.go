package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func sensorMLJSON(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var object map[string]any
	require.NoError(t, decoder.Decode(&object))
	return object
}

func sensorMLRequest(t *testing.T, method, path string, payload any, status int) (http.Header, map[string]any) {
	t.Helper()
	var reader io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	request, err := http.NewRequest(method, testServer.URL+strings.TrimPrefix(path, testServer.URL), reader)
	require.NoError(t, err)
	request.Header.Set("Content-Type", mediatype.SensorML)
	request.Header.Set("Accept", mediatype.SensorML)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, status, response.StatusCode, "%s %s: %s", method, path, raw)
	if len(raw) == 0 {
		return response.Header, nil
	}
	if status == http.StatusOK {
		require.Equal(t, mediatype.SensorML, response.Header.Get("Content-Type"))
	}
	return response.Header, sensorMLJSON(t, raw)
}

func sensorMLPayload(resource string) map[string]any {
	typ, definition := "PhysicalSystem", "http://www.w3.org/ns/sosa/Sensor"
	if resource == "deployments" {
		typ, definition = "Deployment", "http://www.w3.org/ns/sosa/Deployment"
	}
	return map[string]any{
		"type": typ, "definition": definition,
		"uniqueId": "urn:uuid:" + uuid.NewString(), "label": "SensorML preservation",
		"validTime": []string{"2026-10-01T00:00:00Z", "2026-10-02T00:00:00Z"},
	}
}

// Compare JSON numbers without float64 conversion so a rounded constraint value
// cannot pass a persistence assertion.
func sensorMLValue(t *testing.T, value any) any {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"value": value})
	require.NoError(t, err)
	return sensorMLJSON(t, raw)["value"]
}

func requireSensorMLFields(t *testing.T, expected, actual map[string]any) {
	t.Helper()
	for _, field := range []string{"documents", "characteristics", "capabilities", "history", "inputs", "outputs", "parameters", "position"} {
		if value, exists := expected[field]; exists {
			require.Equal(t, sensorMLValue(t, value), actual[field], "field %s", field)
		}
	}
	require.NotContains(t, actual, "documentation")
	if platform, exists := expected["platform"]; exists {
		want := sensorMLValue(t, platform).(map[string]any)
		got := requireObjectMember(t, actual, "platform")
		require.Equal(t, want["name"], got["name"])
		require.Equal(t, want["configuration"], got["configuration"])
		require.Equal(t, want["description"], got["description"])
		require.Equal(t, requireObjectMember(t, want, "system")["href"], requireObjectMember(t, got, "system")["href"])
	}
	if deployed, exists := expected["deployedSystems"]; exists {
		want := sensorMLValue(t, deployed).([]any)
		got, ok := actual["deployedSystems"].([]any)
		require.True(t, ok)
		require.Len(t, got, len(want))
		for i := range want {
			wantItem, gotItem := want[i].(map[string]any), got[i].(map[string]any)
			for _, field := range []string{"name", "description", "configuration"} {
				require.Equal(t, wantItem[field], gotItem[field], "deployedSystems[%d].%s", i, field)
			}
			require.Equal(t, requireObjectMember(t, wantItem, "system")["href"], requireObjectMember(t, gotItem, "system")["href"])
		}
	}
}

func requireStoredSensorMLFields(t *testing.T, resource, id string, expected map[string]any) {
	t.Helper()
	for field, column := range map[string]string{
		"documents": "documentation", "characteristics": "characteristics", "capabilities": "capabilities",
		"history": "history", "inputs": "inputs", "outputs": "outputs", "parameters": "parameters",
		"position": "position", "platform": "platform", "deployedSystems": "deployed_systems",
	} {
		value, exists := expected[field]
		if !exists {
			continue
		}
		var raw []byte
		require.NoError(t, testDB.Table(resource).Select(column).Where("id = ?", id).Row().Scan(&raw))
		stored := sensorMLJSON(t, append(append([]byte(`{"value":`), raw...), '}'))["value"]
		if field == "platform" || field == "deployedSystems" {
			// A GET body adds link titles/types/UIDs that a subsequent PUT stores.
			requireSensorMLFields(t, map[string]any{field: value}, map[string]any{field: stored})
		} else {
			require.Equal(t, sensorMLValue(t, value), stored, "stored %s.%s", resource, column)
		}
	}
}

func TestSensorMLPreservationHTTPRoundTrip(t *testing.T) {
	component := json.RawMessage(`{"id":"serial","type":"Text","definition":"urn:test:serial","label":"Serial","description":"Factory serial number","optional":false,"updatable":true,"value":"ABC"}`)
	namedComponent := json.RawMessage(`{"name":"serial","id":"serial","type":"Text","definition":"urn:test:serial","label":"Serial","description":"Factory serial number","optional":true,"updatable":false,"value":"ABC"}`)
	documents := json.RawMessage(`[{"name":"Manual","description":"Operator manual","link":{"href":"https://example.test/manual","type":"application/pdf"}}]`)
	constraints := json.RawMessage(`[
		{"type":"AllowedTokens","ref":"parameters/mode","values":["fast"],"reason":"calibration"},
		{"type":"AllowedTokens","ref":"parameters/token","pattern":"^[a-z]+$","reason":{"enabled":false}},
		{"type":"AllowedValues","ref":"parameters/count","values":[9007199254740993,1.234567890123456789],"reason":"precision"},
		{"type":"AllowedTimes","ref":"parameters/time","values":["2026-10-01T00:00:00Z"],"reason":"schedule"}
	]`)
	for _, resource := range []string{"procedures", "deployments", "systems"} {
		t.Run(resource, func(t *testing.T) {
			cleanupDB(t)
			payload := sensorMLPayload(resource)
			payload["documents"] = documents
			// Embedded SensorML event history is separate from automatic /history revisions.
			payload["history"] = []any{map[string]any{
				"id": "calibration", "label": "Calibration", "time": "2026-10-01T00:00:00Z",
				"properties": []any{component}, "documentation": documents,
			}}
			if resource != "systems" {
				payload["characteristics"] = []any{map[string]any{
					"label": "Technical", "conditions": []any{component}, "characteristics": []any{component},
				}}
				payload["capabilities"] = []any{map[string]any{
					"label": "Operating", "conditions": []any{component}, "capabilities": []any{component},
				}}
			}
			if resource != "deployments" {
				for _, field := range []string{"inputs", "outputs", "parameters"} {
					payload[field] = []any{namedComponent}
				}
			}
			if resource == "systems" {
				payload["position"] = json.RawMessage(`{"type":"Point","coordinates":[1.23456789,2.34567891,3.45678912]}`)
			}
			if resource == "deployments" {
				linkedSystem := createSystemViaAPI(t, "/systems", baseSystemPayload("Deployed system"))
				configuration := map[string]any{"setConstraints": constraints}
				payload["platform"] = map[string]any{
					"name": "platform", "description": "Configured platform",
					"system": map[string]any{"href": testServer.URL + "/systems/" + linkedSystem}, "configuration": configuration,
				}
				payload["deployedSystems"] = []any{map[string]any{
					"name": "sensor", "description": "Configured sensor",
					"system": map[string]any{"href": testServer.URL + "/systems/" + linkedSystem}, "configuration": configuration,
				}}
			}

			headers, _ := sensorMLRequest(t, http.MethodPost, "/"+resource, payload, http.StatusCreated)
			location := headers.Get("Location")
			require.NotEmpty(t, location)
			id := parseID(location, "/"+resource+"/")
			for pass := range 2 {
				_, current := sensorMLRequest(t, http.MethodGet, location, nil, http.StatusOK)
				requireSensorMLFields(t, payload, current)
				requireStoredSensorMLFields(t, resource, id, payload)
				_, collection := sensorMLRequest(t, http.MethodGet, "/"+resource+"?id="+id, nil, http.StatusOK)
				items, ok := collection["items"].([]any)
				require.True(t, ok)
				require.Len(t, items, 1)
				requireSensorMLFields(t, payload, items[0].(map[string]any))
				if pass == 0 {
					// Replace with the complete GET body, including generated links and ID.
					sensorMLRequest(t, http.MethodPut, location, current, http.StatusNoContent)
				}
			}
		})
	}
}

func TestSensorMLDocumentsFromExistingRows(t *testing.T) {
	for _, resource := range []string{"procedures", "deployments"} {
		t.Run(resource, func(t *testing.T) {
			cleanupDB(t)
			documents := common_shared.Documents{{Name: "Stored manual", Link: common_shared.Link{Href: "https://example.test/stored-manual"}}}
			common := domains.CommonSSN{UniqueIdentifier: domains.UniqueID("urn:uuid:" + uuid.NewString()), Name: "Existing stored metadata"}
			var id string
			if resource == "procedures" {
				procedure := &domains.Procedure{CommonSSN: common, ProcessType: "PhysicalComponent", ProcedureType: domains.ProcedureTypeSensor, Documentation: documents}
				require.NoError(t, testRepos.Procedure.Create(t.Context(), procedure))
				id = procedure.ID
			} else {
				deployment := &domains.Deployment{CommonSSN: common, DeploymentType: domains.DeploymentTypeDeployment, Documentation: documents}
				require.NoError(t, testRepos.Deployment.Create(t.Context(), deployment))
				id = deployment.ID
			}
			expected := map[string]any{"documents": documents}
			_, current := sensorMLRequest(t, http.MethodGet, "/"+resource+"/"+id, nil, http.StatusOK)
			requireSensorMLFields(t, expected, current)
			_, collection := sensorMLRequest(t, http.MethodGet, "/"+resource, nil, http.StatusOK)
			items := collection["items"].([]any)
			require.Len(t, items, 1)
			requireSensorMLFields(t, expected, items[0].(map[string]any))
		})
	}
}

func TestSensorMLPreservationRetainsWriteRejections(t *testing.T) {
	cleanupDB(t)
	for _, resource := range []string{"procedures", "deployments"} {
		t.Run(resource+" documentation", func(t *testing.T) {
			payload := sensorMLPayload(resource)
			payload["documentation"] = []any{map[string]any{"link": map[string]any{"href": "https://example.test/manual"}}}
			_, result := sensorMLRequest(t, http.MethodPost, "/"+resource, payload, http.StatusBadRequest)
			require.Contains(t, result["error"], "documentation")
			var count int64
			require.NoError(t, testDB.Table(resource).Count(&count).Error)
			require.Zero(t, count)
		})
	}
	t.Run("procedure position", func(t *testing.T) {
		payload := sensorMLPayload("procedures")
		payload["position"] = map[string]any{"type": "Point", "coordinates": []float64{1, 2, 3}}
		_, result := sensorMLRequest(t, http.MethodPost, "/procedures", payload, http.StatusBadRequest)
		details := result["details"].([]any)
		require.NotEmpty(t, details)
		require.Equal(t, "position", details[0].(map[string]any)["path"])
	})
}
