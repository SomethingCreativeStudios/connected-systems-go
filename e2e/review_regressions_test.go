package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/stretchr/testify/require"
)

func requestJSONExpect(t *testing.T, method, path string, body any, accept string, status int) *http.Response {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(method, testServer.URL+path, bytes.NewReader(raw))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	if resp.StatusCode != status {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("%s %s: expected %d, got %d: %s", method, path, status, resp.StatusCode, raw)
	}
	return resp
}
func putJSONExpect(t *testing.T, path string, body any, status int) *http.Response {
	return requestJSONExpect(t, http.MethodPut, path, body, "", status)
}
func collectionIDs(t *testing.T, path, accept string) []string {
	t.Helper()
	body := getJSONResource(t, path, accept)
	items, ok := body["items"].([]any)
	if !ok {
		items, ok = body["features"].([]any)
	}
	require.True(t, ok, "%v", body)
	ids := []string{}
	for _, item := range items {
		ids = append(ids, item.(map[string]any)["id"].(string))
	}
	return ids
}
func followAssociation(t *testing.T, body map[string]any, rel string) map[string]any {
	t.Helper()
	for _, raw := range body["links"].([]any) {
		link := raw.(map[string]any)
		if link["rel"] == "ogc-rel:"+rel {
			href, err := url.Parse(link["href"].(string))
			require.NoError(t, err)
			return getJSONResource(t, href.RequestURI(), link["type"].(string))
		}
	}
	t.Fatalf("missing %s link in %v", rel, body)
	return nil
}
func TestReview_CollectionNegotiationBeforeInsertion(t *testing.T) {
	cleanupDB(t)
	payload := map[string]any{"id": "review-collection", "title": "Review", "queryables": map[string]any{"properties": map[string]any{}, "additionalProperties": false}}
	resp := requestJSONExpect(t, http.MethodPost, "/collections", payload, "application/xml", http.StatusNotAcceptable)
	resp.Body.Close()
	var count int64
	require.NoError(t, testDB.Table("collections").Where("id = ?", "review-collection").Count(&count).Error)
	require.Zero(t, count)
	resp = requestJSONExpect(t, http.MethodPost, "/collections", payload, "application/json", http.StatusCreated)
	require.Equal(t, "application/geo+json", resp.Header.Get("Content-Type"))
	require.NotEmpty(t, resp.Header.Get("Location"))
	resp.Body.Close()
}
func TestReview_StreamRoundTripAndSchemaFreeze(t *testing.T) {
	cleanupDB(t)
	system := createSystemViaAPI(t, "/systems", baseSystemPayload("Review schemas"))
	ds := createDatastreamViaAPI(t, "/systems/"+system+"/datastreams", baseDatastreamPayload())
	cs := createControlStreamViaAPI(t, system, baseControlStreamPayload())
	createObservationViaAPI(t, ds, map[string]any{"resultTime": "2026-07-01T00:00:00Z", "result": map[string]any{"temperature": 21, "humidity": 40}})
	createCommandViaAPI(t, cs, baseCommandPayload())
	for _, path := range []string{"/datastreams/" + ds, "/controlstreams/" + cs} {
		before := getJSONResource(t, path, "application/json")
		schema := getJSONResource(t, path+"/schema", "application/json")
		resp := putJSONExpect(t, path, before, http.StatusNoContent)
		resp.Body.Close()
		require.Equal(t, before, getJSONResource(t, path, "application/json"))
		require.Equal(t, schema, getJSONResource(t, path+"/schema", "application/json"))
		resp = putJSONExpect(t, path+"/schema", schema, http.StatusNoContent)
		resp.Body.Close()
		field := "resultSchema"
		if path[1] == 'c' {
			field = "parametersSchema"
		}
		component := schema[field].(map[string]any)
		component["label"] = "Changed"
		resp = putJSONExpect(t, path+"/schema", schema, http.StatusConflict)
		resp.Body.Close()
		newFormat := map[string]any{}
		for key, value := range schema {
			newFormat[key] = value
		}
		formatField := "obsFormat"
		if path[1] == 'c' {
			formatField = "commandFormat"
		}
		newFormat[formatField] = "application/swe+json"
		resp = putJSONExpect(t, path+"/schema", newFormat, http.StatusConflict)
		resp.Body.Close()
		before["schema"] = schema
		resp = putJSONExpect(t, path, before, http.StatusConflict)
		resp.Body.Close()
		delete(before, "schema")
		before["schema"] = nil
		resp = putJSONExpect(t, path, before, http.StatusBadRequest)
		resp.Body.Close()
	}
	require.Equal(t, []string{ds}, collectionIDs(t, "/datastreams?observedProperty="+url.QueryEscape("http://sensorml.com/ont/swe/property/Temperature"), "application/json"))
	require.Equal(t, []string{cs}, collectionIDs(t, "/controlstreams?controlledProperty="+url.QueryEscape("http://sensorml.com/ont/swe/property/Temperature"), "application/json"))
	resp := requestJSONExpect(t, http.MethodGet, "/datastreams/"+ds, nil, "application/json;q=0, */*;q=1", http.StatusNotAcceptable)
	resp.Body.Close()
}
func TestReview_CommandOwnedFieldsAndValidation(t *testing.T) {
	cleanupDB(t)
	system := createSystemViaAPI(t, "/systems", baseSystemPayload("Review command"))
	cs := createControlStreamViaAPI(t, system, baseControlStreamPayload())
	resp := requestJSONExpect(t, http.MethodPost, "/controlstreams/"+cs+"/commands", map[string]any{"parameters": map[string]any{"setPoint": "wrong"}}, "", http.StatusBadRequest)
	resp.Body.Close()
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	cmd := &domains.Command{ControlStreamID: cs, Sender: "original sender", IssueTime: &now, Parameters: []byte(`{"setPoint":21}`)}
	require.NoError(t, testRepos.Command.Create(t.Context(), cmd))
	body := baseCommandPayload()
	body["sender"] = "spoofed"
	body["issueTime"] = "1999-01-01T00:00:00Z"
	resp = putJSONExpect(t, "/commands/"+cmd.ID, body, http.StatusNoContent)
	resp.Body.Close()
	stored, err := testRepos.Command.GetByID(t.Context(), cmd.ID)
	require.NoError(t, err)
	require.Equal(t, "original sender", stored.Sender)
	require.True(t, now.Equal(*stored.IssueTime))
	status := createCommandStatusReportViaAPI(t, cmd.ID, map[string]any{"id": "client-status", "statusCode": "ACCEPTED", "reportTime": "2026-07-02T00:00:00Z"})
	require.NotEqual(t, "client-status", status)
	before := getJSONResource(t, "/commands/"+cmd.ID+"/status/"+status, "application/json")
	before["reportTime"] = "1999-01-01T00:00:00Z"
	resp = putJSONExpect(t, "/commands/"+cmd.ID+"/status/"+status, before, http.StatusNoContent)
	resp.Body.Close()
	after := getJSONResource(t, "/commands/"+cmd.ID+"/status/"+status, "application/json")
	reported, err := time.Parse(time.RFC3339, after["reportTime"].(string))
	require.NoError(t, err)
	require.True(t, time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC).Equal(reported))
}
func TestReview_RecursiveAssociationsFollowLinks(t *testing.T) {
	cleanupDB(t)
	root := createSystemViaAPI(t, "/systems", baseSystemPayload("Root"))
	child := createSystemViaAPI(t, "/systems/"+root+"/subsystems", baseSystemPayload("Child"))
	leaf := createSystemViaAPI(t, "/systems/"+child+"/subsystems", baseSystemPayload("Leaf"))
	ds := createDatastreamViaAPI(t, "/systems/"+leaf+"/datastreams", baseDatastreamPayload())
	cs := createControlStreamViaAPI(t, leaf, baseControlStreamPayload())
	sf := createSamplingFeatureViaAPI(t, leaf, baseSamplingFeaturePayload("Sample"))
	for _, format := range []string{"application/geo+json", "application/sml+json"} {
		system := getJSONResource(t, "/systems/"+root, format)
		for _, rel := range []string{"datastreams", "controlstreams", "samplingFeatures"} {
			followAssociation(t, system, rel)
		}
		require.ElementsMatch(t, []string{ds}, collectionIDs(t, "/systems/"+root+"/datastreams", "application/json"))
		require.ElementsMatch(t, []string{cs}, collectionIDs(t, "/systems/"+root+"/controlstreams", "application/json"))
		require.ElementsMatch(t, []string{sf}, collectionIDs(t, "/systems/"+root+"/samplingFeatures", format))
	}
	second := createDatastreamViaAPI(t, "/systems/"+leaf+"/datastreams", baseDatastreamPayload())
	page := getJSONResource(t, "/systems/"+root+"/datastreams?limit=1", "application/json")
	require.Len(t, page["items"], 1)
	next := ""
	for _, raw := range page["links"].([]any) {
		link := raw.(map[string]any)
		if link["rel"] == "next" {
			u, err := url.Parse(link["href"].(string))
			require.NoError(t, err)
			next = u.RequestURI()
		}
	}
	require.NotEmpty(t, next)
	nextPage := getJSONResource(t, next, "application/json")
	require.Len(t, nextPage["items"], 1)
	require.NotEqual(t, page["items"].([]any)[0].(map[string]any)["id"], nextPage["items"].([]any)[0].(map[string]any)["id"])

	payload := baseDeploymentPayload("Parent deployment", root)
	delete(payload["properties"].(map[string]any), "deployedSystems@link")
	parent := createDeploymentViaAPI(t, "/deployments", payload)
	sub := createDeploymentViaAPI(t, "/deployments/"+parent+"/subdeployments", baseDeploymentPayload("Child deployment", root))
	for _, format := range []string{"application/geo+json", "application/sml+json"} {
		deployment := getJSONResource(t, "/deployments/"+parent, format)
		for _, rel := range []string{"datastreams", "controlstreams", "samplingFeatures"} {
			followAssociation(t, deployment, rel)
		}
		if format == "application/geo+json" {
			require.Len(t, deployment["properties"].(map[string]any)["deployedSystems@link"], 1)
			raw, err := json.Marshal(deployment)
			require.NoError(t, err)
			req, err := http.NewRequest(http.MethodPut, testServer.URL+"/deployments/"+parent, bytes.NewReader(raw))
			require.NoError(t, err)
			req.Header.Set("Content-Type", format)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode)
			resp.Body.Close()
		} else {
			require.Len(t, deployment["deployedSystems"], 1)
		}
	}
	stored, err := testRepos.Deployment.GetByID(t.Context(), parent)
	require.NoError(t, err)
	require.Empty(t, stored.DeployedSystems)
	stored, err = testRepos.Deployment.GetByID(t.Context(), sub)
	require.NoError(t, err)
	require.Len(t, stored.DeployedSystems, 1)
	require.ElementsMatch(t, []string{ds, second}, collectionIDs(t, "/deployments/"+parent+"/datastreams", "application/json"))
	require.Equal(t, []string{cs}, collectionIDs(t, "/deployments/"+parent+"/controlstreams", "application/json"))
	require.Equal(t, []string{sf}, collectionIDs(t, "/deployments/"+parent+"/samplingFeatures", "application/geo+json"))
}

func TestReview_WritableStreamLinksAndDeploymentFOI(t *testing.T) {
	cleanupDB(t)
	procedure := createProcedureViaAPI(t, map[string]any{"type": "SimpleProcess", "uniqueId": "urn:test:review:procedure", "label": "Procedure", "definition": "http://www.w3.org/ns/sosa/Procedure"})
	system := createSystemViaAPI(t, "/systems", baseSystemPayload("Linked system"))
	deployment := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Linked deployment", system))
	collection := &domains.Collection{ID: "review-foi", Title: "FOI", Queryables: common_shared.Properties{"properties": map[string]any{}, "additionalProperties": false}}
	require.NoError(t, testRepos.Collection.CreateCollection(t.Context(), collection))
	feature := &domains.Feature{CollectionID: collection.ID, CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:review:foi", Name: "Lake"}}
	require.NoError(t, testRepos.Feature.Create(t.Context(), feature))
	foi := testServer.URL + "/collections/" + collection.ID + "/items/" + feature.ID
	sfPayload := baseSamplingFeaturePayload("Local sample")
	sfPayload["properties"].(map[string]any)["sampledFeature@link"] = map[string]any{"href": foi}
	sf := createSamplingFeatureViaAPI(t, system, sfPayload)
	for _, kind := range []string{"datastreams", "controlstreams"} {
		payload := baseDatastreamPayload()
		if kind == "controlstreams" {
			payload = baseControlStreamPayload()
		}
		payload["procedure@link"] = map[string]any{"href": "/procedures/" + procedure}
		payload["deployment@link"] = map[string]any{"href": "/deployments/" + deployment}
		payload["samplingFeature@link"] = map[string]any{"href": "/samplingFeatures/" + sf}
		payload["featureOfInterest@link"] = map[string]any{"href": foi}
		id := ""
		if kind == "datastreams" {
			id = createDatastreamViaAPI(t, "/systems/"+system+"/datastreams", payload)
		} else {
			id = createControlStreamViaAPI(t, system, payload)
		}
		path := "/" + kind + "/" + id
		read := getJSONResource(t, path, "application/json")
		require.Equal(t, foi, read["featureOfInterest@link"].(map[string]any)["href"])
		external := "https://remote.example/procedures/" + procedure
		read["procedure@link"] = map[string]any{"href": external}
		resp := putJSONExpect(t, path, read, http.StatusNoContent)
		resp.Body.Close()
		read = getJSONResource(t, path, "application/json")
		require.Equal(t, external, read["procedure@link"].(map[string]any)["href"])
		// A remote ID collision must not acquire the local target's title/UID.
		require.NotContains(t, read["procedure@link"].(map[string]any), "uid")
		read["procedure@link"] = map[string]any{"href": "/procedures/missing"}
		resp = putJSONExpect(t, path, read, http.StatusBadRequest)
		resp.Body.Close()
		read = getJSONResource(t, path, "application/json")
		for _, field := range []string{"procedure@link", "deployment@link", "samplingFeature@link", "featureOfInterest@link"} {
			delete(read, field)
		}
		resp = putJSONExpect(t, path, read, http.StatusNoContent)
		resp.Body.Close()
		read = getJSONResource(t, path, "application/json")
		require.NotContains(t, read, "procedure@link")
		require.NotContains(t, read, "featureOfInterest@link")
	}
	native := baseSamplingFeaturePayload("Native system sample")
	native["properties"].(map[string]any)["sampledFeature@link"] = map[string]any{"href": "/systems/" + system}
	createSamplingFeatureViaAPI(t, system, native)
	native = baseSamplingFeaturePayload("Native sampling feature sample")
	native["properties"].(map[string]any)["sampledFeature@link"] = map[string]any{"href": "/samplingFeatures/" + sf}
	createSamplingFeatureViaAPI(t, system, native)
	for kind, id := range map[string]string{"deployments": deployment, "procedures": procedure} {
		payload := baseSamplingFeaturePayload("Native " + kind + " sample")
		payload["properties"].(map[string]any)["sampledFeature@link"] = map[string]any{"href": "/" + kind + "/" + id}
		createSamplingFeatureViaAPI(t, system, payload)
	}

	remoteFeature := &domains.Feature{CollectionID: collection.ID, CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:remote:collision", Name: "Remote collision"}}
	require.NoError(t, testRepos.Feature.Create(t.Context(), remoteFeature))
	remote := baseSamplingFeaturePayload("Remote sample")
	remote["properties"].(map[string]any)["sampledFeature@link"] = map[string]any{"href": "https://remote.example/collections/" + collection.ID + "/items/" + remoteFeature.ID}
	createSamplingFeatureViaAPI(t, system, remote)

	require.ElementsMatch(t, []string{feature.ID, system, sf, deployment, procedure}, collectionIDs(t, "/deployments/"+deployment+"/featuresOfInterest", "application/geo+json"))
	followAssociation(t, getJSONResource(t, "/deployments/"+deployment, "application/geo+json"), "featuresOfInterest")
}

func TestReview_AttachedToParentAndProcedureLinks(t *testing.T) {
	cleanupDB(t)
	procedure := createProcedureViaAPI(t, map[string]any{"type": "SimpleProcess", "uniqueId": "urn:test:parent:procedure", "label": "Procedure", "definition": "http://www.w3.org/ns/sosa/Procedure"})
	root := createSystemViaAPI(t, "/systems", baseSystemPayload("Parent"))
	payload := baseSystemPayload("Implementer")
	payload["properties"].(map[string]any)["systemKind@link"] = map[string]any{"href": "/procedures/" + procedure}
	child := createSystemViaAPI(t, "/systems/"+root+"/subsystems", payload)
	sml := getJSONResource(t, "/systems/"+child, "application/sml+json")
	require.Equal(t, testServer.URL+"/systems/"+root, sml["attachedTo"].(map[string]any)["href"])
	sml["attachedTo"] = map[string]any{"href": "/systems/unrelated"}
	raw, err := json.Marshal(sml)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPut, testServer.URL+"/systems/"+child, bytes.NewReader(raw))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/sml+json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
	after := getJSONResource(t, "/systems/"+child, "application/sml+json")
	require.Equal(t, testServer.URL+"/systems/"+root, after["attachedTo"].(map[string]any)["href"])
	followAssociation(t, after, "parentSystem")
	for _, format := range []string{"application/geo+json", "application/sml+json"} {
		implementing := followAssociation(t, getJSONResource(t, "/procedures/"+procedure, format), "implementingSystems")
		items := implementing["items"]
		if items == nil {
			items = implementing["features"]
		}
		require.Len(t, items, 1)
	}
}
