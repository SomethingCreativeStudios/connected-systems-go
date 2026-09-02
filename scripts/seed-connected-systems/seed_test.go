package main

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/resourcevalidation"
)

func validConfig(mode string) Config {
	seed := int64(11)
	return Config{
		Endpoint:   "http://example.test/api",
		Mode:       mode,
		Namespace:  "unit-test",
		RandomSeed: &seed,
		HTTP:       HTTPConfig{Timeout: Duration(time.Second)},
		Seed: SeedConfig{
			Properties:                       1,
			Collections:                      1,
			FeaturesPerCollection:            IntRange{Min: 1, Max: 1},
			Systems:                          1,
			SystemTypeWeights:                map[string]int{"actuator": 1},
			ProcedureVariantsPerSystemType:   IntRange{Min: 1, Max: 1},
			SubsystemsPerSystem:              IntRange{},
			SamplingFeaturesPerSystem:        IntRange{},
			DatastreamsPerSystem:             IntRange{Min: 1, Max: 1},
			InitialObservationsPerDatastream: IntRange{Min: 1, Max: 1},
			ControlStreamsPerSystem:          IntRange{Min: 1, Max: 1},
			CommandsPerControlStream:         IntRange{Min: 1, Max: 1},
			StatusReportsPerCommand:          IntRange{Min: 1, Max: 1},
			CommandResultsPerCompleted:       IntRange{Min: 1, Max: 1},
			CompletedCommandPercent:          100,
			Deployments:                      1,
			SystemsPerDeployment:             IntRange{Min: 1, Max: 1},
			SubdeploymentsPerDeployment:      IntRange{},
			SystemEventsPerSystem:            IntRange{Min: 1, Max: 1},
			HistoryUpdatesPerSystem:          IntRange{Min: 1, Max: 1},
		},
		Observe: ObserveConfig{
			Frequency:             Duration(time.Millisecond),
			BatchSize:             IntRange{Min: 1, Max: 1},
			StreamRefreshInterval: Duration(time.Second),
		},
	}
}

func TestConfigValidationRejectsImpossibleTopology(t *testing.T) {
	cfg := validConfig("seed")
	cfg.Seed.SystemTypeWeights = map[string]int{"sensor": 1}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "control streams") {
		t.Fatalf("expected controller topology error, got %v", err)
	}

	cfg = validConfig("seed")
	cfg.Seed.SamplingFeaturesPerSystem = IntRange{Min: 1, Max: 1}
	cfg.Seed.Collections = 0
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "sampling features") {
		t.Fatalf("expected sampling feature topology error, got %v", err)
	}
}

func TestSeedRunIDSeparatesAdditiveRuns(t *testing.T) {
	cfg := validConfig("seed")
	cfg.RunID = "fixture-a"
	first := NewSeeder(cfg, nil, randForTest())
	if !strings.Contains(first.uid("property"), "unit-test-fixture-a") {
		t.Fatalf("fixed run ID was not included in resource UID: %s", first.uid("property"))
	}

	cfg.RunID = "fixture-b"
	second := NewSeeder(cfg, nil, randForTest())
	if first.namespace == second.namespace {
		t.Fatalf("different run IDs produced the same namespace: %q", first.namespace)
	}

	cfg.RunID = ""
	if got := cfg.EffectiveRunID(time.Date(2026, time.August, 28, 2, 30, 0, 123, time.UTC)); got != "20260828t023000000000123z" {
		t.Fatalf("unexpected automatic run ID: %q", got)
	}
}

func TestExampleConfigLoads(t *testing.T) {
	cfg, err := LoadConfig("config.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != "seed" || cfg.Seed.DatastreamsPerSystem.Min != 1 {
		t.Fatalf("unexpected example config: %+v", cfg)
	}
}

func TestGeneratedDatastreamValuesMatchServerValidator(t *testing.T) {
	rng := rand.New(rand.NewPCG(4, 5))
	for _, source := range datastreamProfiles() {
		encodedSchema, err := json.Marshal(source)
		if err != nil {
			t.Fatal(err)
		}
		var schema domains.DatastreamSchema
		if err := json.Unmarshal(encodedSchema, &schema); err != nil {
			t.Fatalf("decode schema %s: %v", source.ObsFormat, err)
		}
		result, err := resultForSchema(rng, source)
		if err != nil {
			t.Fatalf("generate %s result: %v", source.ObsFormat, err)
		}
		encodedResult, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		obs := &domains.Observation{Result: encodedResult}
		datastream := &domains.Datastream{Schema: &schema}
		if err := resourcevalidation.ValidateObservationAgainstDatastreamSchema(obs, datastream, "application/json"); err != nil {
			t.Fatalf("%s result does not match server validation: %v; result=%s", source.ObsFormat, err, encodedResult)
		}
	}
}

func TestResourceID(t *testing.T) {
	id, err := resourceID("https://example.test/api/datastreams/a%20stream?x=1", "datastreams")
	if err != nil || id != "a stream" {
		t.Fatalf("expected decoded stream ID, got %q, %v", id, err)
	}
}

func TestCreateErrorIncludesServerBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"schema is required"}`))
	}))
	defer server.Close()
	cfg := validConfig("seed")
	cfg.Endpoint = server.URL
	client, err := NewAPIClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.PostJSON(context.Background(), "/systems/example/datastreams", map[string]any{"name": "bad"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = expectCreated(client, "/systems/example/datastreams", "datastreams", response)
	if err == nil || !strings.Contains(err.Error(), "HTTP 400") || !strings.Contains(err.Error(), "schema is required") {
		t.Fatalf("expected actionable POST error, got %v", err)
	}
}

func TestSystemRolesUseMatchingDatasheetTypes(t *testing.T) {
	expected := map[string]struct {
		systemType           string
		procedureType        string
		procedureProcessType string
		assetType            string
	}{
		"sensor":   {sosaSensor, sosaSensor, "PhysicalComponent", "Equipment"},
		"actuator": {sosaActuator, sosaActuator, "PhysicalComponent", "Equipment"},
		"sampler":  {sosaSampler, sosaSampler, "PhysicalComponent", "Equipment"},
		"platform": {sosaPlatform, sosaPlatform, "PhysicalSystem", "Equipment"},
		"system":   {ssnSystem, sosaSystem, "PhysicalSystem", "Equipment"},
	}
	for _, role := range systemRoles {
		want := expected[role.Key]
		if role.SystemType != want.systemType || role.ProcedureType != want.procedureType || role.ProcedureProcessType != want.procedureProcessType || role.AssetType != want.assetType {
			t.Fatalf("role %s does not use a matching procedure type: %+v", role.Key, role)
		}
	}
}

func TestObserverPaginatesAndSendsRandomBatch(t *testing.T) {
	var mutex sync.Mutex
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test-Seed") != "yes" {
			t.Errorf("missing configured request header on %s", r.URL)
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/conformance":
			_, _ = w.Write([]byte(`{"conformsTo":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/datastreams" && r.URL.Query().Get("cursor") == "":
			_, _ = w.Write([]byte(`{"items":[{"id":"first","formats":["application/json"]}],"links":[{"rel":"next","href":"/datastreams?cursor=next"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/datastreams" && r.URL.Query().Get("cursor") == "next":
			_, _ = w.Write([]byte(`{"items":[{"id":"second","formats":["application/json"]}],"links":[]}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/datastreams/") && strings.HasSuffix(r.URL.Path, "/schema"):
			_, _ = w.Write([]byte(`{"obsFormat":"application/json","resultSchema":{"type":"Quantity","label":"Temperature","definition":"https://example.test/temperature","uom":{"code":"Cel"}}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/observations"):
			mutex.Lock()
			posts++
			mutex.Unlock()
			w.Header().Set("Location", "http://example.test/observations/"+strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/datastreams/"), "/observations"))
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := validConfig("observe")
	cfg.Endpoint = server.URL
	cfg.HTTP.Headers = map[string]string{"X-Test-Seed": "yes"}
	cfg.Observe.Frequency = Duration(5 * time.Millisecond)
	cfg.Observe.BatchSize = IntRange{Min: 2, Max: 2}
	cfg.Observe.StreamRefreshInterval = Duration(time.Hour)
	client, err := NewAPIClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Preflight(context.Background()); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Report, 1)
	go func() {
		report, runErr := NewObserver(cfg, client, rand.New(rand.NewPCG(7, 8))).Run(ctx)
		if runErr != nil {
			t.Errorf("observer run: %v", runErr)
		}
		done <- report
	}()

	deadline := time.After(time.Second)
	for {
		mutex.Lock()
		seen := posts
		mutex.Unlock()
		if seen >= 2 {
			cancel()
			break
		}
		select {
		case <-deadline:
			cancel()
			t.Fatal("observer did not post a full batch")
		case <-time.After(time.Millisecond):
		}
	}
	report := <-done
	if report.Sent < 2 {
		t.Fatalf("expected at least 2 sent observations, got %+v", report)
	}
}
