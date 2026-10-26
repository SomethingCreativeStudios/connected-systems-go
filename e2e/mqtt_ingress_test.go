package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/api"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/generators"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/pubsub"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mqttRecord struct {
	topic string
	body  []byte
	media string
}
type mqttCapture struct {
	sync.Mutex
	messages []mqttRecord
}

func (*mqttCapture) IsConnected() bool { return true }
func (c *mqttCapture) Publish(topic string, b []byte, media string) {
	c.Lock()
	defer c.Unlock()
	c.messages = append(c.messages, mqttRecord{topic, bytes.Clone(b), media})
}
func (c *mqttCapture) count(topic string) int {
	c.Lock()
	defer c.Unlock()
	n := 0
	for _, m := range c.messages {
		if m.topic == topic {
			n++
		}
	}
	return n
}
func mqttConfig() *config.Config {
	return &config.Config{API: config.APIConfig{BaseURL: testServer.URL, Title: "MQTT Test", Version: "1.0"}, MQTT: config.MQTTConfig{Enabled: true, NodeID: "node1", MaxMessageBytes: 10 << 20}, PubSub: config.PubSubConfig{ResourceData: config.PubSubFeatureConfig{Enabled: true}, ResourceEvents: config.PubSubFeatureConfig{Enabled: true}, BatchResourceEvents: config.BatchResourceEventsConfig{Enabled: true, Window: time.Minute}}}
}
func ingressRequest(handler pubsub.Ingress, cfg *config.Config, topic, key, username string, body []byte) *httptest.ResponseRecorder {
	media := "application/json"
	if strings.HasSuffix(topic, "systems:data") {
		media = "application/geo+json"
	}
	principal := "anonymous"
	if username != "" {
		principal = "user:" + username
	}
	var operationKey *string
	if key != "" {
		operationKey = &key
	}
	result, err := handler.Create(context.Background(), pubsub.Publication{Principal: principal, Topic: topic, IdempotencyKey: operationKey, ProtocolVersion: 5, QoS: 1, ContentType: media, Payload: body})
	w := httptest.NewRecorder()
	if err != nil {
		status := 500
		var failure *pubsub.IngressError
		var validation *repository.ValidationError
		switch {
		case errors.As(err, &failure):
			status = failure.Status
		case errors.As(err, &validation):
			status = 400
		case repository.IsUniqueViolation(err):
			status = 409
		case repository.IsNotFound(err):
			status = 404
		}
		http.Error(w, err.Error(), status)
		return w
	}
	// The broker publishes the change after distributing the canonical data.
	if result.Publish != nil {
		result.Publish()
	}
	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("Location", result.ResourceURL)
	if result.Replayed {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(201)
	}
	_, _ = w.Write(result.Payload)
	return w
}
func TestMQTTIngressConcurrentIdempotencyAndTombstones(t *testing.T) {
	cleanupDB(t)
	cfg := mqttConfig()
	capture := &mqttCapture{}
	p := pubsub.NewPublisher(cfg.API.BaseURL, "node1", cfg.PubSub, capture, zap.NewNop())
	defer p.Close()
	handler := api.NewMQTTIngress(cfg, testRepos, p, zap.NewNop())
	body, _ := json.Marshal(baseSystemPayload("retry system"))
	const workers = 16
	results := make(chan *httptest.ResponseRecorder, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Go(func() { results <- ingressRequest(handler, cfg, "node1/systems:data", "same-key", "alice", body) })
	}
	wg.Wait()
	close(results)
	var original []byte
	location := ""
	fresh := 0
	for r := range results {
		require.Contains(t, []int{200, 201}, r.Code, r.Body.String())
		if r.Code == 201 {
			fresh++
		}
		if original == nil {
			original = bytes.Clone(r.Body.Bytes())
			location = r.Header().Get("Location")
		}
		require.Equal(t, original, r.Body.Bytes())
	}
	require.Equal(t, 1, fresh)
	for table, want := range map[string]int64{"systems": 1, "system_history_revisions": 1, "mqtt_create_operations": 1} {
		var n int64
		require.NoError(t, testDB.Table(table).Count(&n).Error)
		require.Equal(t, want, n, table)
	}
	require.Equal(t, 1, capture.count("node1/systems:data/sml-json"))
	require.Equal(t, 1, capture.count("node1/systems/"+strings.TrimPrefix(location, cfg.API.BaseURL+"/systems/")))
	// A new handler and new repository handles model process restart; client IDs are not part of scope.
	restarted := api.NewMQTTIngress(cfg, repository.NewRepositories(testDB), p, zap.NewNop())
	retry := ingressRequest(restarted, cfg, "node1/systems:data", "same-key", "alice", body)
	require.Equal(t, 200, retry.Code)
	require.Equal(t, original, retry.Body.Bytes())
	conflict := ingressRequest(handler, cfg, "node1/systems:data", "same-key", "alice", []byte(`{"type":"Feature","properties":{}}`))
	require.Equal(t, 409, conflict.Code)
	// Another principal does not reuse Alice's result; normal uniqueness still applies.
	other := ingressRequest(handler, cfg, "node1/systems:data", "same-key", "bob", body)
	require.Equal(t, 409, other.Code, other.Body.String())
	id := strings.TrimPrefix(location, cfg.API.BaseURL+"/systems/")
	require.NoError(t, testRepos.System.Delete(t.Context(), id, true))
	deleted := ingressRequest(handler, cfg, "node1/systems:data", "same-key", "alice", body)
	require.Equal(t, 410, deleted.Code, deleted.Body.String())
	var n int64
	testDB.Table("systems").Count(&n)
	require.Zero(t, n)
	testDB.Table("mqtt_create_operations").Count(&n)
	require.EqualValues(t, 1, n)
}
func TestMQTTIngressRollbackAndDistinctOperations(t *testing.T) {
	cleanupDB(t)
	cfg := mqttConfig()
	handler := api.NewMQTTIngress(cfg, testRepos, nil, zap.NewNop())
	system := createSystemViaAPI(t, "/systems", baseSystemPayload("parent"))
	topic := "node1/systems/" + system + "/datastreams:data"
	body, _ := json.Marshal(baseDatastreamPayload())
	require.NoError(t, testDB.Exec(`CREATE FUNCTION mqtt_test_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected before commit'; END $$; CREATE TRIGGER mqtt_test_fail BEFORE INSERT ON datastreams FOR EACH ROW EXECUTE FUNCTION mqtt_test_fail()`).Error)
	r := ingressRequest(handler, cfg, topic, "recover", "alice", body)
	require.GreaterOrEqual(t, r.Code, 400)
	var n int64
	testDB.Table("mqtt_create_operations").Count(&n)
	require.Zero(t, n)
	testDB.Table("datastreams").Count(&n)
	require.Zero(t, n)
	require.NoError(t, testDB.Exec(`DROP TRIGGER mqtt_test_fail ON datastreams; DROP FUNCTION mqtt_test_fail()`).Error)
	r = ingressRequest(handler, cfg, topic, "recover", "alice", body)
	require.Equal(t, 201, r.Code, r.Body.String())
	for _, x := range []struct{ key, user string }{{"different", "alice"}, {"recover", "bob"}, {"", "alice"}, {"", "alice"}} {
		r = ingressRequest(handler, cfg, topic, x.key, x.user, body)
		require.Equal(t, 201, r.Code, r.Body.String())
	}
	testDB.Table("datastreams").Count(&n)
	require.EqualValues(t, 5, n)
	testDB.Table("mqtt_create_operations").Count(&n)
	require.EqualValues(t, 3, n)
}
func TestMQTTIngressEveryResourceAndHTTPPublication(t *testing.T) {
	cleanupDB(t)
	cfg := mqttConfig()
	capture := &mqttCapture{}
	p := pubsub.NewPublisher(cfg.API.BaseURL, "node1", cfg.PubSub, capture, zap.NewNop())
	defer p.Close()
	handler := api.NewMQTTIngress(cfg, testRepos, p, zap.NewNop())
	public := api.NewRouter(cfg, zap.NewNop(), testRepos, p)
	system := createSystemViaAPI(t, "/systems", baseSystemPayload("root"))
	sub := createSystemViaAPI(t, "/systems/"+system+"/subsystems", baseSystemPayload("child"))
	prefix := "node1/systems/" + system + "/subsystems/" + sub
	create := func(topic, collection string, payload any) string {
		t.Helper()
		b, _ := json.Marshal(payload)
		r := ingressRequest(handler, cfg, topic, "key-"+collection, "alice", b)
		require.Equal(t, 201, r.Code, r.Body.String())
		var value map[string]any
		require.NoError(t, json.Unmarshal(r.Body.Bytes(), &value))
		require.NotEmpty(t, value["id"])
		req := httptest.NewRequest("GET", r.Header().Get("Location"), nil)
		w := httptest.NewRecorder()
		public.ServeHTTP(w, req)
		require.Equal(t, 200, w.Code, w.Body.String())
		retry := ingressRequest(handler, cfg, topic, "key-"+collection, "alice", b)
		require.Equal(t, 200, retry.Code, retry.Body.String())
		require.Equal(t, r.Body.Bytes(), retry.Body.Bytes())
		return value["id"].(string)
	}
	ds := create(prefix+"/datastreams:data", "datastream", baseDatastreamPayload())
	cs := create(prefix+"/controlstreams:data", "controlstream", baseControlStreamPayload())
	create(prefix+"/datastreams/"+ds+"/observations:data", "observation", map[string]any{"resultTime": "2026-10-14T12:00:00Z", "result": map[string]any{"temperature": 22, "humidity": 50}})
	cmd := create(prefix+"/controlstreams/"+cs+"/commands:data", "command", baseCommandPayload())
	create(prefix+"/controlstreams/"+cs+"/commands/"+cmd+"/status:data", "commandstatus", map[string]any{"statusCode": "ACCEPTED", "reportTime": "2026-10-14T12:00:00Z"})
	create(prefix+"/controlstreams/"+cs+"/commands/"+cmd+"/result:data", "commandresult", map[string]any{"data": map[string]any{"value": 1}})
	create(prefix+"/events:data", "systemevent", baseSystemEventPayload("test"))
	// Every HTTP create uses the same operation and publishes its canonical Resource Data.
	cases := []struct {
		path, topic string
		payload     any
	}{
		{"/systems", "node1/systems:data", baseSystemPayload("HTTP root")},
		{"/systems/" + sub + "/subsystems", prefix + "/subsystems:data", baseSystemPayload("HTTP grandchild")},
		{"/systems/" + sub + "/datastreams", prefix + "/datastreams:data", baseDatastreamPayload()},
		{"/systems/" + sub + "/controlstreams", prefix + "/controlstreams:data", baseControlStreamPayload()},
		{"/datastreams/" + ds + "/observations", prefix + "/datastreams/" + ds + "/observations:data", map[string]any{"resultTime": "2026-10-14T12:01:00Z", "result": map[string]any{"temperature": 23, "humidity": 52}}},
		{"/controlstreams/" + cs + "/commands", prefix + "/controlstreams/" + cs + "/commands:data", baseCommandPayload()},
		{"/commands/" + cmd + "/status", prefix + "/controlstreams/" + cs + "/commands/" + cmd + "/status:data", map[string]any{"statusCode": "COMPLETED", "reportTime": "2026-10-14T12:01:00Z"}},
		{"/commands/" + cmd + "/result", prefix + "/controlstreams/" + cs + "/commands/" + cmd + "/result:data", map[string]any{"data": 1}},
		{"/systems/" + sub + "/events", prefix + "/events:data", baseSystemEventPayload("HTTP event")},
	}
	for _, c := range cases {
		b, _ := json.Marshal(c.payload)
		req := httptest.NewRequest("POST", c.path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		before := capture.count(c.topic)
		public.ServeHTTP(w, req)
		require.Equal(t, 201, w.Code, w.Body.String())
		require.Equal(t, before+1, capture.count(c.topic), c.topic)
	}
	// Deletion uses captured ownership and canonical HTTP parent identity.
	req := httptest.NewRequest("DELETE", "/systems/"+sub+"?cascade=true", nil)
	w := httptest.NewRecorder()
	public.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)
	require.Equal(t, 1, capture.count(prefix))
}
func TestMQTTIngressSecurityAndLimits(t *testing.T) {
	cleanupDB(t)
	cfg := mqttConfig()
	cfg.MQTT.MaxMessageBytes = 32
	h := api.NewMQTTIngress(cfg, testRepos, nil, zap.NewNop())
	_, err := h.Create(t.Context(), pubsub.Publication{Topic: "node1/systems:data"})
	require.Error(t, err)
	var r *httptest.ResponseRecorder
	r = ingressRequest(h, cfg, "node1/systems:data", "key", "alice", make([]byte, 33))
	require.Equal(t, 413, r.Code)
	r = ingressRequest(h, cfg, "node1/observations:data", "key", "alice", []byte(`{}`))
	require.Equal(t, 400, r.Code)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = testRepos.Transaction(ctx, func(tx *repository.Repositories) error { return tx.System.Create(ctx, &domains.System{}) })
	require.Error(t, err)
}

func TestMQTTOwnershipEscapingAndDeleteIdentity(t *testing.T) {
	cleanupDB(t)
	ids := []string{"root/%2F", "sub+#é", "%literal"}
	var parent *string
	resolver := pubsub.NewRepositoryChangeResolver(testServer.URL, testRepos, nil)
	for _, id := range ids {
		sys := generators.FakeSystemMinimal()
		sys.ID = id
		sys.Links = nil
		sys.ParentSystemID = parent
		require.NoError(t, testRepos.System.Create(t.Context(), &sys))
		v := id
		parent = &v
	}
	path := "/systems/" + url.PathEscape(ids[2])
	change, err := resolver.Resolve(t.Context(), path, pubsub.OperationDelete)
	require.NoError(t, err)
	expected := "systems/" + pubsub.EncodeID(ids[0]) + "/subsystems/" + pubsub.EncodeID(ids[1]) + "/subsystems/" + pubsub.EncodeID(ids[2])
	require.Equal(t, expected, change.TopicPath)
	require.Equal(t, path, change.SubjectPath)
	alias, e := resolver.Resolve(t.Context(), strings.ReplaceAll(path, "literal", "%6Citeral"), pubsub.OperationDelete)
	require.NoError(t, e)
	require.Equal(t, path, alias.SubjectPath)
	require.Equal(t, "/systems/"+url.PathEscape(ids[1]), change.ParentPath)
	require.NoError(t, testRepos.System.Delete(t.Context(), ids[2], true))
	p := pubsub.NewPublisher(testServer.URL, "node1", mqttConfig().PubSub, &mqttCapture{}, zap.NewNop())
	event, err := p.NewResourceEvent(change)
	require.NoError(t, err)
	require.Equal(t, testServer.URL+path, event.Subject)
	require.Equal(t, testServer.URL+"/systems/"+url.PathEscape(ids[1]), event.ParentID)
	for i, id := range ids {
		d := domains.Deployment{Base: domains.Base{ID: id}, CommonSSN: domains.CommonSSN{Name: "Deployment", UniqueIdentifier: domains.UniqueID("urn:deployment:" + strconv.Itoa(i))}}
		if i > 0 {
			d.ParentDeploymentID = &ids[i-1]
		}
		require.NoError(t, testRepos.Deployment.Create(t.Context(), &d))
	}
	change, err = resolver.Resolve(t.Context(), "/deployments/"+url.PathEscape(ids[2]), pubsub.OperationDelete)
	require.NoError(t, err)
	require.Equal(t, strings.ReplaceAll(expected, "systems/", "deployments/"), change.TopicPath)
}
func TestMQTTKeyedCommandContributesOnceToBatch(t *testing.T) {
	cleanupDB(t)
	cfg := mqttConfig()
	capture := &mqttCapture{}
	p := pubsub.NewPublisher(cfg.API.BaseURL, "node1", cfg.PubSub, capture, zap.NewNop())
	defer p.Close()
	h := api.NewMQTTIngress(cfg, testRepos, p, zap.NewNop())
	sys := createSystemViaAPI(t, "/systems", baseSystemPayload("batch owner"))
	cs := createControlStreamViaAPI(t, sys, baseControlStreamPayload())
	topic := "node1/systems/" + sys + "/controlstreams/" + cs + "/commands:data"
	body, _ := json.Marshal(baseCommandPayload())
	for i := 0; i < 8; i++ {
		r := ingressRequest(h, cfg, topic, "command-operation", "alice", body)
		if i == 0 {
			require.Equal(t, 201, r.Code, r.Body.String())
		} else {
			require.Equal(t, 200, r.Code, r.Body.String())
		}
	}
	p.Close()
	require.Equal(t, 1, capture.count(strings.TrimSuffix(topic, ":data")))
	capture.Lock()
	defer capture.Unlock()
	for _, m := range capture.messages {
		if m.topic == strings.TrimSuffix(topic, ":data") {
			var e pubsub.CloudEvent
			require.NoError(t, json.Unmarshal(m.body, &e))
			require.EqualValues(t, 1, e.Data["count"])
		}
	}
	var n int64
	testDB.Table("commands").Count(&n)
	require.EqualValues(t, 1, n)
}

func TestMQTTAnonymousPrincipalCannotCollideWithNamedUser(t *testing.T) {
	cleanupDB(t)
	cfg := mqttConfig()
	h := api.NewMQTTIngress(cfg, testRepos, nil, zap.NewNop())
	sys := createSystemViaAPI(t, "/systems", baseSystemPayload("principal owner"))
	topic := "node1/systems/" + sys + "/datastreams:data"
	body, _ := json.Marshal(baseDatastreamPayload())
	first := ingressRequest(h, cfg, topic, "shared-key", "", body)
	second := ingressRequest(h, cfg, topic, "shared-key", "anonymous", body)
	require.Equal(t, 201, first.Code, first.Body.String())
	require.Equal(t, 201, second.Code, second.Body.String())
	require.NotEqual(t, first.Header().Get("Location"), second.Header().Get("Location"))
}

func TestCreateRejectsParentMismatchAndOversizedOutput(t *testing.T) {
	cleanupDB(t)
	cfg := mqttConfig()
	p := pubsub.NewPublisher(cfg.API.BaseURL, "node1", cfg.PubSub, &mqttCapture{}, zap.NewNop())
	defer p.Close()
	post := func(router http.Handler, path, media string, payload any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", path, bytes.NewReader(b))
		req.Header.Set("Content-Type", media)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	public := api.NewRouter(cfg, zap.NewNop(), testRepos, p)
	system := createSystemViaAPI(t, "/systems", baseSystemPayload("owner"))
	ds := createDatastreamViaAPI(t, "/systems/"+system+"/datastreams", baseDatastreamPayload())
	other := createDatastreamViaAPI(t, "/systems/"+system+"/datastreams", baseDatastreamPayload())
	observation := map[string]any{"resultTime": "2026-10-14T12:00:00Z", "result": map[string]any{"temperature": 22, "humidity": 50}, "datastream@id": other}
	// HTTP and MQTT both reject an association ID that names another parent.
	w := post(public, "/datastreams/"+ds+"/observations", "application/json", observation)
	require.Equal(t, 400, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "does not match the parent")
	body, _ := json.Marshal(observation)
	topic := "node1/systems/" + system + "/datastreams/" + ds + "/observations:data"
	r := ingressRequest(api.NewMQTTIngress(cfg, testRepos, p, zap.NewNop()), cfg, topic, "", "alice", body)
	require.Equal(t, 400, r.Code, r.Body.String())
	var n int64
	require.NoError(t, testDB.Table("observations").Count(&n).Error)
	require.Zero(t, n)

	// A resource whose canonical output cannot be published within the MQTT
	// limit is rejected with 413, and its transaction leaves nothing behind.
	payload := baseSystemPayload("too large to publish")
	input, _ := json.Marshal(payload)
	limited := *cfg
	limited.MQTT.MaxMessageBytes = int64(len(input)) + 1
	require.NoError(t, testDB.Table("systems").Count(&n).Error)
	before := n
	w = post(api.NewRouter(&limited, zap.NewNop(), testRepos, p), "/systems", "application/geo+json", payload)
	require.Equal(t, 413, w.Code, w.Body.String())
	r = ingressRequest(api.NewMQTTIngress(&limited, testRepos, p, zap.NewNop()), &limited, "node1/systems:data", "", "alice", input)
	require.Equal(t, 413, r.Code, r.Body.String())
	require.NoError(t, testDB.Table("systems").Count(&n).Error)
	require.Equal(t, before, n)
}

// TestResourceDataMatchesHTTPRepresentation checks that every Resource Data
// message carries the document HTTP GET returns for the same representation.
func TestResourceDataMatchesHTTPRepresentation(t *testing.T) {
	cleanupDB(t)
	cfg := mqttConfig()
	capture := &mqttCapture{}
	p := pubsub.NewPublisher(cfg.API.BaseURL, "node1", cfg.PubSub, capture, zap.NewNop())
	defer p.Close()
	public := api.NewRouter(cfg, zap.NewNop(), testRepos, p)
	create := func(path, media string, payload any, representations ...string) string {
		t.Helper()
		capture.Lock()
		before := len(capture.messages)
		capture.Unlock()
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", path, bytes.NewReader(body))
		req.Header.Set("Content-Type", media)
		w := httptest.NewRecorder()
		public.ServeHTTP(w, req)
		require.Equal(t, 201, w.Code, w.Body.String())
		location := w.Header().Get("Location")
		published := map[string][]byte{}
		capture.Lock()
		for _, m := range capture.messages[before:] {
			if m.media != pubsub.CloudEventsContentType() {
				published[m.media] = m.body
			}
		}
		capture.Unlock()
		require.Len(t, published, len(representations), path)
		for _, representation := range representations {
			req := httptest.NewRequest("GET", location, nil)
			req.Header.Set("Accept", representation)
			w := httptest.NewRecorder()
			public.ServeHTTP(w, req)
			require.Equal(t, 200, w.Code, w.Body.String())
			require.Contains(t, published, representation, path)
			require.JSONEq(t, w.Body.String(), string(published[representation]), "%s as %s", path, representation)
		}
		return location[strings.LastIndex(location, "/")+1:]
	}
	system := create("/systems", "application/geo+json", baseSystemPayload("parity"), "application/geo+json", "application/sml+json")
	create("/systems/"+system+"/subsystems", "application/geo+json", baseSystemPayload("parity child"), "application/geo+json", "application/sml+json")
	ds := create("/systems/"+system+"/datastreams", "application/json", baseDatastreamPayload(), "application/json")
	cs := create("/systems/"+system+"/controlstreams", "application/json", baseControlStreamPayload(), "application/json")
	create("/datastreams/"+ds+"/observations", "application/json", map[string]any{"resultTime": "2026-10-14T12:00:00Z", "result": map[string]any{"temperature": 22, "humidity": 50}}, "application/json")
	cmd := create("/controlstreams/"+cs+"/commands", "application/json", baseCommandPayload(), "application/json")
	create("/commands/"+cmd+"/status", "application/json", map[string]any{"statusCode": "ACCEPTED", "reportTime": "2026-10-14T12:00:00Z"}, "application/json")
	create("/commands/"+cmd+"/result", "application/json", map[string]any{"data": map[string]any{"value": 1}}, "application/json")
	create("/systems/"+system+"/events", "application/json", baseSystemEventPayload("parity"), "application/json")
}
