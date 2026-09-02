package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
	"unicode"
)

const (
	sosaSensor   = "http://www.w3.org/ns/sosa/Sensor"
	sosaActuator = "http://www.w3.org/ns/sosa/Actuator"
	sosaSampler  = "http://www.w3.org/ns/sosa/Sampler"
	sosaPlatform = "http://www.w3.org/ns/sosa/Platform"
	sosaSystem   = "http://www.w3.org/ns/sosa/System"
	ssnSystem    = "http://www.w3.org/ns/ssn/System"
)

type systemRole struct {
	Key                  string
	SystemType           string
	ProcedureType        string
	ProcedureProcessType string
	AssetType            string
	Controller           bool
}

var systemRoles = []systemRole{
	{Key: "sensor", SystemType: sosaSensor, ProcedureType: sosaSensor, ProcedureProcessType: "PhysicalComponent", AssetType: "Equipment"},
	{Key: "actuator", SystemType: sosaActuator, ProcedureType: sosaActuator, ProcedureProcessType: "PhysicalComponent", AssetType: "Equipment", Controller: true},
	{Key: "sampler", SystemType: sosaSampler, ProcedureType: sosaSampler, ProcedureProcessType: "PhysicalComponent", AssetType: "Equipment"},
	{Key: "platform", SystemType: sosaPlatform, ProcedureType: sosaPlatform, ProcedureProcessType: "PhysicalSystem", AssetType: "Equipment", Controller: true},
	{Key: "system", SystemType: ssnSystem, ProcedureType: sosaSystem, ProcedureProcessType: "PhysicalSystem", AssetType: "Equipment", Controller: true},
}

type Seeder struct {
	cfg       Config
	client    *APIClient
	rng       *rand.Rand
	report    Report
	namespace string
	sequence  int
}

type seededFeature struct {
	CollectionID string
	ID           string
}

type seededSystem struct {
	ID          string
	UID         string
	Name        string
	Role        systemRole
	ProcedureID string
	Geometry    map[string]any
	ParentID    string
}

type seededDatastream struct {
	ID                string
	Schema            StreamSchema
	SamplingFeatureID string
	ProcedureID       string
}

type seededControlStream struct {
	ID     string
	Schema StreamSchema
}

type seededDeployment struct {
	ID        string
	SystemIDs []string
}

func NewSeeder(cfg Config, client *APIClient, rng *rand.Rand) *Seeder {
	runID := cfg.EffectiveRunID(time.Now())
	return &Seeder{
		cfg:       cfg,
		client:    client,
		rng:       rng,
		report:    NewReport("seed", cfg.Namespace),
		namespace: slugify(cfg.Namespace) + "-" + runID,
	}
}

func (s *Seeder) Run(ctx context.Context) (Report, error) {
	if err := s.seedProperties(ctx); err != nil {
		return s.report, err
	}
	features, err := s.seedCollectionsAndFeatures(ctx)
	if err != nil {
		return s.report, err
	}
	procedures, err := s.seedProcedurePools(ctx)
	if err != nil {
		return s.report, err
	}
	roots, systems, err := s.seedSystems(ctx, procedures)
	if err != nil {
		return s.report, err
	}
	if err := s.seedHistoryRevisions(ctx, roots); err != nil {
		return s.report, err
	}
	samplingFeatures, err := s.seedSamplingFeatures(ctx, systems, features)
	if err != nil {
		return s.report, err
	}
	if _, err := s.seedDatastreamsAndObservations(ctx, systems, samplingFeatures); err != nil {
		return s.report, err
	}
	if err := s.seedControlStreamsAndCommands(ctx, systems); err != nil {
		return s.report, err
	}
	if err := s.seedDeployments(ctx, systems); err != nil {
		return s.report, err
	}
	if err := s.seedSystemEvents(ctx, systems); err != nil {
		return s.report, err
	}
	return s.report, nil
}

func (s *Seeder) seedProperties(ctx context.Context) error {
	definitions := []struct {
		Label      string
		Base       string
		ObjectType string
	}{
		{"Air Temperature", "https://qudt.org/vocab/quantitykind/Temperature", "http://dbpedia.org/resource/Atmosphere"},
		{"Relative Humidity", "https://qudt.org/vocab/quantitykind/RelativeHumidity", "http://dbpedia.org/resource/Atmosphere"},
		{"Atmospheric Pressure", "https://qudt.org/vocab/quantitykind/Pressure", "http://dbpedia.org/resource/Atmosphere"},
		{"Water Conductivity", "https://qudt.org/vocab/quantitykind/ElectricalConductivity", "http://dbpedia.org/resource/Water"},
		{"Valve Position", "https://example.org/def/property/valve-position", "http://dbpedia.org/resource/Valve"},
	}
	for i := 0; i < s.cfg.Seed.Properties; i++ {
		definition := definitions[i%len(definitions)]
		payload := map[string]any{
			"uniqueId":     s.uid("property"),
			"label":        s.name(definition.Label),
			"description":  "Synthetic " + strings.ToLower(definition.Label) + " property",
			"baseProperty": definition.Base,
			"objectType":   definition.ObjectType,
		}
		response, err := s.client.Post(ctx, "/properties", "application/sml+json", "application/sml+json", payload)
		if err != nil {
			return fmt.Errorf("create property %d: %w", i+1, err)
		}
		if _, err := expectCreated(s.client, "/properties", "properties", response); err != nil {
			return err
		}
		s.report.AddCreated("properties")
	}
	return nil
}

func (s *Seeder) seedCollectionsAndFeatures(ctx context.Context) ([]seededFeature, error) {
	features := make([]seededFeature, 0)
	for i := 0; i < s.cfg.Seed.Collections; i++ {
		collectionID := s.next("sites")
		payload := map[string]any{
			"id":          collectionID,
			"title":       s.name("Synthetic observation sites"),
			"description": "Generic features created by the Connected Systems seeder",
		}
		response, err := s.client.PostJSON(ctx, "/collections", payload)
		if err != nil {
			return nil, fmt.Errorf("create collection %s: %w", collectionID, err)
		}
		if response.StatusCode != http.StatusCreated {
			return nil, s.client.statusError(http.MethodPost, "/collections", response)
		}
		createdID, err := collectionIDFromResponse(response)
		if err != nil {
			return nil, fmt.Errorf("create collection %s: %w", collectionID, err)
		}
		s.report.AddCreated("collections")
		for count := s.draw(s.cfg.Seed.FeaturesPerCollection); count > 0; count-- {
			featurePayload := map[string]any{
				"type": "Feature",
				"properties": map[string]any{
					"uid":         s.uid("generic-feature"),
					"name":        s.name("Monitoring site"),
					"description": "Synthetic generic feature used as a sampled feature target",
				},
				"geometry": s.point(),
			}
			path := "/collections/" + createdID + "/items"
			featureResponse, err := s.client.Post(ctx, path, "application/geo+json", "application/geo+json", featurePayload)
			if err != nil {
				return nil, fmt.Errorf("create generic feature in collection %s: %w", createdID, err)
			}
			featureID, err := expectCreated(s.client, path, "items", featureResponse)
			if err != nil {
				return nil, err
			}
			features = append(features, seededFeature{CollectionID: createdID, ID: featureID})
			s.report.AddCreated("features")
		}
	}
	return features, nil
}

func collectionIDFromResponse(response APIResponse) (string, error) {
	return collectionID(response)
}

func (s *Seeder) seedProcedurePools(ctx context.Context) (map[string][]string, error) {
	pools := make(map[string][]string)
	for _, role := range systemRoles {
		if s.cfg.Seed.SystemTypeWeights[role.Key] <= 0 {
			continue
		}
		for count := s.draw(s.cfg.Seed.ProcedureVariantsPerSystemType); count > 0; count-- {
			payload := map[string]any{
				"type":        role.ProcedureProcessType,
				"label":       s.name(strings.Title(role.Key) + " datasheet"),
				"description": "Synthetic type definition for seeded " + role.Key + " systems",
				"uniqueId":    s.uid("procedure-" + role.Key),
				"definition":  role.ProcedureType,
			}
			response, err := s.client.Post(ctx, "/procedures", "application/sml+json", "application/sml+json", payload)
			if err != nil {
				return nil, fmt.Errorf("create %s procedure: %w", role.Key, err)
			}
			id, err := expectCreated(s.client, "/procedures", "procedures", response)
			if err != nil {
				return nil, err
			}
			pools[role.Key] = append(pools[role.Key], id)
			s.report.AddCreated("procedures")
		}
	}
	return pools, nil
}

func (s *Seeder) seedSystems(ctx context.Context, procedures map[string][]string) ([]seededSystem, []seededSystem, error) {
	roots := make([]seededSystem, 0, s.cfg.Seed.Systems)
	systems := make([]seededSystem, 0, s.cfg.Seed.Systems)
	for i := 0; i < s.cfg.Seed.Systems; i++ {
		role := s.chooseRole()
		procedureID := s.chooseString(procedures[role.Key])
		system, err := s.createSystem(ctx, "/systems", role, procedureID, "")
		if err != nil {
			return nil, nil, err
		}
		roots = append(roots, system)
		systems = append(systems, system)
		for count := s.draw(s.cfg.Seed.SubsystemsPerSystem); count > 0; count-- {
			childRole := s.chooseRole()
			childProcedureID := s.chooseString(procedures[childRole.Key])
			child, err := s.createSystem(ctx, "/systems/"+system.ID+"/subsystems", childRole, childProcedureID, system.ID)
			if err != nil {
				return nil, nil, err
			}
			systems = append(systems, child)
		}
	}
	return roots, systems, nil
}

func (s *Seeder) createSystem(ctx context.Context, resourcePath string, role systemRole, procedureID, parentID string) (seededSystem, error) {
	system := seededSystem{
		UID:         s.uid("system-" + role.Key),
		Name:        s.name(strings.Title(role.Key) + " system"),
		Role:        role,
		ProcedureID: procedureID,
		Geometry:    s.point(),
		ParentID:    parentID,
	}
	response, err := s.client.Post(ctx, resourcePath, "application/geo+json", "application/geo+json", s.systemPayload(system, ""))
	if err != nil {
		return seededSystem{}, fmt.Errorf("create %s system: %w", role.Key, err)
	}
	id, err := expectCreated(s.client, resourcePath, "systems", response)
	if err != nil {
		return seededSystem{}, err
	}
	system.ID = id
	s.report.AddCreated("systems")
	if parentID != "" {
		s.report.AddCreated("subsystems")
	}
	return system, nil
}

func (s *Seeder) seedHistoryRevisions(ctx context.Context, roots []seededSystem) error {
	for _, system := range roots {
		for revision := 1; revision <= s.draw(s.cfg.Seed.HistoryUpdatesPerSystem); revision++ {
			payload := s.systemPayload(system, fmt.Sprintf("configuration revision %d", revision))
			response, err := s.client.Put(ctx, "/systems/"+system.ID, "application/geo+json", payload)
			if err != nil {
				return fmt.Errorf("update system %s for history revision: %w", system.ID, err)
			}
			if response.StatusCode != http.StatusNoContent {
				return s.client.statusError(http.MethodPut, "/systems/"+system.ID, response)
			}
			s.report.AddCreated("system_history_revisions")
		}
	}
	return nil
}

func (s *Seeder) systemPayload(system seededSystem, revision string) map[string]any {
	description := "Synthetic " + system.Role.Key + " system"
	if revision != "" {
		description += "; " + revision
	}
	return map[string]any{
		"type": "Feature",
		"properties": map[string]any{
			"uid":         system.UID,
			"name":        system.Name,
			"description": description,
			"featureType": system.Role.SystemType,
			"assetType":   system.Role.AssetType,
			"systemKind@link": map[string]any{
				"href": s.client.URL("/procedures/" + system.ProcedureID),
				"rel":  "systemKind",
			},
		},
		"geometry": system.Geometry,
	}
}

func (s *Seeder) seedSamplingFeatures(ctx context.Context, systems []seededSystem, features []seededFeature) (map[string][]string, error) {
	bySystem := make(map[string][]string)
	for _, system := range systems {
		for count := s.draw(s.cfg.Seed.SamplingFeaturesPerSystem); count > 0; count-- {
			feature := features[s.rng.IntN(len(features))]
			payload := map[string]any{
				"type": "Feature",
				"properties": map[string]any{
					"uid":         s.uid("sampling-feature"),
					"name":        s.name("Sampling feature"),
					"description": "Synthetic sampling location for " + system.Name,
					"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
					"sampledFeature@link": map[string]any{
						"href": s.client.URL("/collections/" + feature.CollectionID + "/items/" + feature.ID),
					},
				},
				"geometry": s.point(),
			}
			resourcePath := "/systems/" + system.ID + "/samplingFeatures"
			response, err := s.client.Post(ctx, resourcePath, "application/geo+json", "application/geo+json", payload)
			if err != nil {
				return nil, fmt.Errorf("create sampling feature for system %s: %w", system.ID, err)
			}
			id, err := expectCreated(s.client, resourcePath, "samplingFeatures", response)
			if err != nil {
				return nil, err
			}
			bySystem[system.ID] = append(bySystem[system.ID], id)
			s.report.AddCreated("sampling_features")
		}
	}
	return bySystem, nil
}

func (s *Seeder) seedDatastreamsAndObservations(ctx context.Context, systems []seededSystem, samplingFeatures map[string][]string) ([]seededDatastream, error) {
	streams := make([]seededDatastream, 0)
	for _, system := range systems {
		for index, count := 0, s.draw(s.cfg.Seed.DatastreamsPerSystem); index < count; index++ {
			schema := datastreamProfiles()[index%len(datastreamProfiles())]
			samplingFeatureID := ""
			if candidates := samplingFeatures[system.ID]; len(candidates) > 0 {
				samplingFeatureID = candidates[s.rng.IntN(len(candidates))]
			}
			payload := map[string]any{
				"name":       s.name("Datastream " + profileName(schema)),
				"type":       "observation",
				"outputName": profileName(schema),
				"live":       true,
				"schema":     schema,
				"procedure@link": map[string]any{
					"href": s.client.URL("/procedures/" + system.ProcedureID),
				},
			}
			if samplingFeatureID != "" {
				payload["samplingFeature@link"] = map[string]any{"href": s.client.URL("/samplingFeatures/" + samplingFeatureID)}
			}
			resourcePath := "/systems/" + system.ID + "/datastreams"
			response, err := s.client.PostJSON(ctx, resourcePath, payload)
			if err != nil {
				return nil, fmt.Errorf("create datastream for system %s: %w", system.ID, err)
			}
			id, err := expectCreated(s.client, resourcePath, "datastreams", response)
			if err != nil {
				return nil, err
			}
			stream := seededDatastream{ID: id, Schema: schema, SamplingFeatureID: samplingFeatureID, ProcedureID: system.ProcedureID}
			streams = append(streams, stream)
			s.report.AddCreated("datastreams")
			for observations := s.draw(s.cfg.Seed.InitialObservationsPerDatastream); observations > 0; observations-- {
				if err := s.createObservation(ctx, stream); err != nil {
					return nil, err
				}
			}
		}
	}
	return streams, nil
}

func (s *Seeder) createObservation(ctx context.Context, stream seededDatastream) error {
	result, err := resultForSchema(s.rng, stream.Schema)
	if err != nil {
		return fmt.Errorf("generate observation for datastream %s: %w", stream.ID, err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	payload := map[string]any{
		"resultTime":     now.Format(time.RFC3339),
		"phenomenonTime": now.Add(-5 * time.Second).Format(time.RFC3339),
		"result":         result,
		"parameters":     map[string]any{"source": "seed-connected-systems", "qc": "good"},
	}
	if stream.SamplingFeatureID != "" {
		payload["samplingFeature@id"] = stream.SamplingFeatureID
	}
	if stream.ProcedureID != "" {
		payload["procedure@link"] = map[string]any{"href": s.client.URL("/procedures/" + stream.ProcedureID)}
	}
	resourcePath := "/datastreams/" + stream.ID + "/observations"
	response, err := s.client.PostJSON(ctx, resourcePath, payload)
	if err != nil {
		return fmt.Errorf("create observation for datastream %s: %w", stream.ID, err)
	}
	if _, err := expectCreated(s.client, resourcePath, "observations", response); err != nil {
		return err
	}
	s.report.AddCreated("observations")
	return nil
}

func (s *Seeder) seedControlStreamsAndCommands(ctx context.Context, systems []seededSystem) error {
	for _, system := range systems {
		if !system.Role.Controller {
			continue
		}
		for index, count := 0, s.draw(s.cfg.Seed.ControlStreamsPerSystem); index < count; index++ {
			schema := controlStreamProfiles()[index%len(controlStreamProfiles())]
			payload := map[string]any{
				"uid":       s.uid("control-stream"),
				"name":      s.name("Control stream " + profileName(schema)),
				"inputName": profileName(schema),
				"live":      true,
				"async":     false,
				"schema":    schema,
				"procedure@link": map[string]any{
					"href": s.client.URL("/procedures/" + system.ProcedureID),
				},
			}
			resourcePath := "/systems/" + system.ID + "/controlstreams"
			response, err := s.client.PostJSON(ctx, resourcePath, payload)
			if err != nil {
				return fmt.Errorf("create control stream for system %s: %w", system.ID, err)
			}
			id, err := expectCreated(s.client, resourcePath, "controlstreams", response)
			if err != nil {
				return err
			}
			controlStream := seededControlStream{ID: id, Schema: schema}
			s.report.AddCreated("control_streams")
			for commands := s.draw(s.cfg.Seed.CommandsPerControlStream); commands > 0; commands-- {
				if err := s.createCommandLifecycle(ctx, controlStream); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Seeder) createCommandLifecycle(ctx context.Context, controlStream seededControlStream) error {
	parameters, err := parametersForSchema(s.rng, controlStream.Schema)
	if err != nil {
		return fmt.Errorf("generate command for control stream %s: %w", controlStream.ID, err)
	}
	resourcePath := "/controlstreams/" + controlStream.ID + "/commands"
	response, err := s.client.PostJSON(ctx, resourcePath, map[string]any{
		"sender":     "seed-connected-systems",
		"parameters": parameters,
	})
	if err != nil {
		return fmt.Errorf("create command for control stream %s: %w", controlStream.ID, err)
	}
	commandID, err := expectCreated(s.client, resourcePath, "commands", response)
	if err != nil {
		return err
	}
	s.report.AddCreated("commands")

	statusCount := s.draw(s.cfg.Seed.StatusReportsPerCommand)
	completed := false
	successful := s.rng.IntN(100) < s.cfg.Seed.CompletedCommandPercent
	statuses := []string{"ACCEPTED", "EXECUTING", "COMPLETED"}
	if !successful {
		statuses = []string{"ACCEPTED", "EXECUTING", "REJECTED"}
	}
	for index := 0; index < statusCount; index++ {
		status := statuses[index]
		if status == "COMPLETED" {
			completed = true
		}
		now := time.Now().UTC().Add(time.Duration(index) * time.Second).Truncate(time.Second)
		statusPath := "/commands/" + commandID + "/status"
		statusResponse, err := s.client.PostJSON(ctx, statusPath, map[string]any{
			"reportTime":        now.Format(time.RFC3339),
			"statusCode":        status,
			"percentCompletion": float64((index + 1) * 100 / len(statuses)),
			"message":           "Synthetic command " + strings.ToLower(status),
		})
		if err != nil {
			return fmt.Errorf("create %s status for command %s: %w", status, commandID, err)
		}
		if _, err := expectCreated(s.client, statusPath, "status", statusResponse); err != nil {
			return err
		}
		s.report.AddCreated("command_status_reports")
	}
	if completed && s.draw(s.cfg.Seed.CommandResultsPerCompleted) == 1 {
		resultPath := "/commands/" + commandID + "/result"
		resultResponse, err := s.client.PostJSON(ctx, resultPath, map[string]any{
			"data": map[string]any{"outcome": "completed", "command": commandID},
		})
		if err != nil {
			return fmt.Errorf("create result for command %s: %w", commandID, err)
		}
		if _, err := expectCreated(s.client, resultPath, "result", resultResponse); err != nil {
			return err
		}
		s.report.AddCreated("command_results")
	}
	return nil
}

func (s *Seeder) seedDeployments(ctx context.Context, systems []seededSystem) error {
	if len(systems) == 0 {
		return nil
	}
	for deploymentIndex := 0; deploymentIndex < s.cfg.Seed.Deployments; deploymentIndex++ {
		systemIDs := s.chooseSystemIDs(s.draw(s.cfg.Seed.SystemsPerDeployment), systems)
		deployment, err := s.createDeployment(ctx, "/deployments", systemIDs, systems)
		if err != nil {
			return err
		}
		for children := s.draw(s.cfg.Seed.SubdeploymentsPerDeployment); children > 0; children-- {
			childCount := s.draw(s.cfg.Seed.SystemsPerDeployment)
			childSystems := chooseSubset(s.rng, deployment.SystemIDs, childCount)
			if len(childSystems) == 0 {
				childSystems = deployment.SystemIDs[:1]
			}
			if _, err := s.createDeployment(ctx, "/deployments/"+deployment.ID+"/subdeployments", childSystems, systems); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Seeder) createDeployment(ctx context.Context, resourcePath string, systemIDs []string, systems []seededSystem) (seededDeployment, error) {
	links := make([]map[string]any, 0, len(systemIDs))
	for _, id := range systemIDs {
		links = append(links, map[string]any{"href": s.client.URL("/systems/" + id)})
	}
	properties := map[string]any{
		"uid":                  s.uid("deployment"),
		"name":                 s.name("Deployment"),
		"description":          "Synthetic deployment containing seeded systems",
		"featureType":          "http://www.w3.org/ns/sosa/Deployment",
		"validTime":            []string{time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339), ".."},
		"deployedSystems@link": links,
	}
	for _, system := range systems {
		if system.Role.Key == "platform" && contains(systemIDs, system.ID) {
			properties["platform@link"] = map[string]any{"href": s.client.URL("/systems/" + system.ID)}
			break
		}
	}
	response, err := s.client.Post(ctx, resourcePath, "application/geo+json", "application/geo+json", map[string]any{
		"type":       "Feature",
		"properties": properties,
		"geometry":   s.point(),
	})
	if err != nil {
		return seededDeployment{}, fmt.Errorf("create deployment: %w", err)
	}
	id, err := expectCreated(s.client, resourcePath, "deployments", response)
	if err != nil {
		return seededDeployment{}, err
	}
	s.report.AddCreated("deployments")
	if strings.Contains(resourcePath, "/subdeployments") {
		s.report.AddCreated("subdeployments")
	}
	return seededDeployment{ID: id, SystemIDs: systemIDs}, nil
}

func (s *Seeder) seedSystemEvents(ctx context.Context, systems []seededSystem) error {
	for _, system := range systems {
		for event := s.draw(s.cfg.Seed.SystemEventsPerSystem); event > 0; event-- {
			resourcePath := "/systems/" + system.ID + "/events"
			response, err := s.client.PostJSON(ctx, resourcePath, map[string]any{
				"definition":  "https://example.org/events/seeded-maintenance",
				"label":       s.name("System inspection"),
				"description": "Synthetic operational event for " + system.Name,
				"time":        time.Now().UTC().Format(time.RFC3339),
			})
			if err != nil {
				return fmt.Errorf("create event for system %s: %w", system.ID, err)
			}
			if _, err := expectCreated(s.client, resourcePath, "events", response); err != nil {
				return err
			}
			s.report.AddCreated("system_events")
		}
	}
	return nil
}

func datastreamProfiles() []StreamSchema {
	return []StreamSchema{
		{
			ObsFormat:    "application/json",
			ResultSchema: &DataComponent{Type: "Quantity", Name: "temperature", Label: "Air temperature", Definition: "https://qudt.org/vocab/quantitykind/Temperature", UOM: uom("Cel"), Constraint: numericConstraint(-30, 50)},
		},
		{
			ObsFormat: "application/json",
			ResultSchema: &DataComponent{Type: "DataRecord", Name: "weather", Fields: []NamedComponent{
				{Name: "temperature", DataComponent: DataComponent{Type: "Quantity", Label: "Air temperature", Definition: "https://qudt.org/vocab/quantitykind/Temperature", UOM: uom("Cel"), Constraint: numericConstraint(-30, 50)}},
				{Name: "humidity", DataComponent: DataComponent{Type: "Quantity", Label: "Relative humidity", Definition: "https://qudt.org/vocab/quantitykind/RelativeHumidity", UOM: uom("%"), Constraint: numericConstraint(0, 100)}},
			}},
		},
		{
			ObsFormat: "application/swe+json",
			RecordSchema: &DataComponent{Type: "DataRecord", Name: "position", Fields: []NamedComponent{
				{Name: "x", DataComponent: DataComponent{Type: "Quantity", Label: "Longitude", Definition: "https://example.org/def/property/x", UOM: uom("deg"), Constraint: numericConstraint(-180, 180)}},
				{Name: "y", DataComponent: DataComponent{Type: "Quantity", Label: "Latitude", Definition: "https://example.org/def/property/y", UOM: uom("deg"), Constraint: numericConstraint(-90, 90)}},
			}},
			Encoding: map[string]any{"type": "JSONEncoding", "recordsAsArrays": false},
		},
		{
			ObsFormat:     "application/x-protobuf",
			MessageSchema: inlineProto(`syntax = "proto3"; message EnvironmentalReading { double temperature = 1; double humidity = 2; string status = 3; }`),
		},
	}
}

func controlStreamProfiles() []StreamSchema {
	return []StreamSchema{
		{
			CommandFormat: "application/json",
			ParametersSchema: &DataComponent{Type: "DataRecord", Name: "setpoint", Fields: []NamedComponent{
				{Name: "setPoint", DataComponent: DataComponent{Type: "Quantity", Label: "Set point", Definition: "https://qudt.org/vocab/quantitykind/Temperature", UOM: uom("Cel"), Constraint: numericConstraint(10, 30)}},
			}},
		},
		{
			CommandFormat: "application/json",
			ParametersSchema: &DataComponent{Type: "DataRecord", Name: "valve", Fields: []NamedComponent{
				{Name: "position", DataComponent: DataComponent{Type: "Quantity", Label: "Valve position", Definition: "https://example.org/def/property/valve-position", UOM: uom("%"), Constraint: numericConstraint(0, 100)}},
				{Name: "enabled", DataComponent: DataComponent{Type: "Boolean", Label: "Enabled", Definition: "https://example.org/def/property/enabled"}},
			}},
		},
	}
}

func numericConstraint(minimum, maximum float64) *DataConstraint {
	return &DataConstraint{Intervals: []byte(fmt.Sprintf("[[%g,%g]]", minimum, maximum))}
}

func uom(code string) *DataUOM { return &DataUOM{Code: code} }

func inlineProto(source string) json.RawMessage {
	encoded, _ := json.Marshal(source)
	return encoded
}

func profileName(schema StreamSchema) string {
	if schema.ResultSchema != nil && schema.ResultSchema.Name != "" {
		return schema.ResultSchema.Name
	}
	if schema.RecordSchema != nil && schema.RecordSchema.Name != "" {
		return schema.RecordSchema.Name
	}
	if schema.ParametersSchema != nil && schema.ParametersSchema.Name != "" {
		return schema.ParametersSchema.Name
	}
	if schema.ObsFormat != "" {
		return schema.ObsFormat
	}
	return schema.CommandFormat
}

func (s *Seeder) draw(value IntRange) int {
	if value.Min == value.Max {
		return value.Min
	}
	return value.Min + s.rng.IntN(value.Max-value.Min+1)
}

func (s *Seeder) chooseRole() systemRole {
	total := 0
	for _, role := range systemRoles {
		total += s.cfg.Seed.SystemTypeWeights[role.Key]
	}
	selection := s.rng.IntN(total)
	for _, role := range systemRoles {
		selection -= s.cfg.Seed.SystemTypeWeights[role.Key]
		if selection < 0 {
			return role
		}
	}
	return systemRoles[0]
}

func (s *Seeder) chooseString(values []string) string {
	return values[s.rng.IntN(len(values))]
}

func (s *Seeder) chooseSystemIDs(count int, systems []seededSystem) []string {
	ids := make([]string, 0, len(systems))
	for _, system := range systems {
		ids = append(ids, system.ID)
	}
	return chooseSubset(s.rng, ids, count)
}

func (s *Seeder) uid(resource string) string {
	return "urn:ogc:seed:" + s.namespace + ":" + s.next(resource)
}

func (s *Seeder) name(prefix string) string {
	return prefix + " " + s.next("name")
}

func (s *Seeder) next(prefix string) string {
	s.sequence++
	return fmt.Sprintf("%s-%s-%04d", s.namespace, slugify(prefix), s.sequence)
}

func (s *Seeder) point() map[string]any {
	return map[string]any{
		"type":        "Point",
		"coordinates": []float64{-117.1625 + (s.rng.Float64()-0.5)/10, 32.7157 + (s.rng.Float64()-0.5)/10},
	}
}

func slugify(input string) string {
	var builder strings.Builder
	lastDash := false
	for _, char := range strings.ToLower(strings.TrimSpace(input)) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			builder.WriteRune(char)
			lastDash = false
		} else if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
