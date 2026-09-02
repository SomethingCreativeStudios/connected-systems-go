package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the complete input contract for the Connected Systems seeder.
// Counts at the seed root are exact; nested resources use IntRange.
type Config struct {
	Endpoint   string        `yaml:"endpoint"`
	Mode       string        `yaml:"mode"`
	Namespace  string        `yaml:"namespace"`
	RunID      string        `yaml:"run_id"`
	RandomSeed *int64        `yaml:"random_seed"`
	HTTP       HTTPConfig    `yaml:"http"`
	Seed       SeedConfig    `yaml:"seed"`
	Observe    ObserveConfig `yaml:"observe"`
}

// EffectiveRunID returns the discriminator used in generated resource IDs.
// A namespace is an ownership tag, whereas a run ID makes consecutive seed
// executions additive even when they use the same deterministic random seed.
func (c Config) EffectiveRunID(now time.Time) string {
	if runID := slugify(c.RunID); runID != "" {
		return runID
	}
	now = now.UTC()
	return fmt.Sprintf("%s%09dz", now.Format("20060102t150405"), now.Nanosecond())
}

type HTTPConfig struct {
	Timeout Duration          `yaml:"timeout"`
	Headers map[string]string `yaml:"headers"`
}

// Duration accepts the familiar Go duration syntax in YAML, such as "10s".
type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
		return fmt.Errorf("duration must be a string such as 10s")
	}
	parsed, err := time.ParseDuration(value.Value)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) Std() time.Duration { return time.Duration(d) }

type IntRange struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
}

func (r IntRange) Validate(name string) error {
	if r.Min < 0 || r.Max < 0 {
		return fmt.Errorf("%s values must be non-negative", name)
	}
	if r.Min > r.Max {
		return fmt.Errorf("%s min must be less than or equal to max", name)
	}
	return nil
}

type SeedConfig struct {
	Properties                       int            `yaml:"properties"`
	Collections                      int            `yaml:"collections"`
	FeaturesPerCollection            IntRange       `yaml:"features_per_collection"`
	Systems                          int            `yaml:"systems"`
	SystemTypeWeights                map[string]int `yaml:"system_type_weights"`
	ProcedureVariantsPerSystemType   IntRange       `yaml:"procedure_variants_per_system_type"`
	SubsystemsPerSystem              IntRange       `yaml:"subsystems_per_system"`
	SamplingFeaturesPerSystem        IntRange       `yaml:"sampling_features_per_system"`
	DatastreamsPerSystem             IntRange       `yaml:"datastreams_per_system"`
	InitialObservationsPerDatastream IntRange       `yaml:"initial_observations_per_datastream"`
	ControlStreamsPerSystem          IntRange       `yaml:"control_streams_per_system"`
	CommandsPerControlStream         IntRange       `yaml:"commands_per_control_stream"`
	StatusReportsPerCommand          IntRange       `yaml:"status_reports_per_command"`
	CommandResultsPerCompleted       IntRange       `yaml:"command_results_per_completed"`
	CompletedCommandPercent          int            `yaml:"completed_command_percent"`
	Deployments                      int            `yaml:"deployments"`
	SubdeploymentsPerDeployment      IntRange       `yaml:"subdeployments_per_deployment"`
	SystemsPerDeployment             IntRange       `yaml:"systems_per_deployment"`
	SystemEventsPerSystem            IntRange       `yaml:"system_events_per_system"`
	HistoryUpdatesPerSystem          IntRange       `yaml:"history_updates_per_system"`
}

type ObserveConfig struct {
	Frequency             Duration `yaml:"frequency"`
	BatchSize             IntRange `yaml:"batch_size"`
	StreamRefreshInterval Duration `yaml:"stream_refresh_interval"`
}

func LoadConfig(path string) (Config, error) {
	var cfg Config
	bytes, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	endpoint, err := url.Parse(c.Endpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return fmt.Errorf("endpoint must be an absolute http or https URL")
	}
	if strings.TrimSpace(c.Namespace) == "" {
		return fmt.Errorf("namespace is required")
	}
	if c.Mode != "seed" && c.Mode != "observe" {
		return fmt.Errorf("mode must be seed or observe")
	}
	if c.HTTP.Timeout.Std() <= 0 {
		return fmt.Errorf("http.timeout must be positive")
	}

	if c.Mode == "observe" {
		if c.Observe.Frequency.Std() <= 0 {
			return fmt.Errorf("observe.frequency must be positive")
		}
		if c.Observe.StreamRefreshInterval.Std() <= 0 {
			return fmt.Errorf("observe.stream_refresh_interval must be positive")
		}
		if err := c.Observe.BatchSize.Validate("observe.batch_size"); err != nil {
			return err
		}
		if c.Observe.BatchSize.Min == 0 {
			return fmt.Errorf("observe.batch_size min must be positive")
		}
		return nil
	}

	return c.Seed.Validate()
}

func (c SeedConfig) Validate() error {
	for name, value := range map[string]int{
		"seed.properties":  c.Properties,
		"seed.collections": c.Collections,
		"seed.systems":     c.Systems,
		"seed.deployments": c.Deployments,
	} {
		if value < 0 {
			return fmt.Errorf("%s must be non-negative", name)
		}
	}
	for name, value := range map[string]IntRange{
		"seed.features_per_collection":             c.FeaturesPerCollection,
		"seed.procedure_variants_per_system_type":  c.ProcedureVariantsPerSystemType,
		"seed.subsystems_per_system":               c.SubsystemsPerSystem,
		"seed.sampling_features_per_system":        c.SamplingFeaturesPerSystem,
		"seed.datastreams_per_system":              c.DatastreamsPerSystem,
		"seed.initial_observations_per_datastream": c.InitialObservationsPerDatastream,
		"seed.control_streams_per_system":          c.ControlStreamsPerSystem,
		"seed.commands_per_control_stream":         c.CommandsPerControlStream,
		"seed.status_reports_per_command":          c.StatusReportsPerCommand,
		"seed.command_results_per_completed":       c.CommandResultsPerCompleted,
		"seed.subdeployments_per_deployment":       c.SubdeploymentsPerDeployment,
		"seed.systems_per_deployment":              c.SystemsPerDeployment,
		"seed.system_events_per_system":            c.SystemEventsPerSystem,
		"seed.history_updates_per_system":          c.HistoryUpdatesPerSystem,
	} {
		if err := value.Validate(name); err != nil {
			return err
		}
	}
	if c.StatusReportsPerCommand.Max > 3 {
		return fmt.Errorf("seed.status_reports_per_command max must be at most 3 to preserve command lifecycle order")
	}
	if c.CommandResultsPerCompleted.Max > 1 {
		return fmt.Errorf("seed.command_results_per_completed max must be at most 1; a completed command has one result")
	}
	if c.CompletedCommandPercent < 0 || c.CompletedCommandPercent > 100 {
		return fmt.Errorf("seed.completed_command_percent must be between 0 and 100")
	}

	if c.Systems == 0 {
		for name, requested := range map[string]bool{
			"subsystems":        c.SubsystemsPerSystem.Max > 0,
			"sampling features": c.SamplingFeaturesPerSystem.Max > 0,
			"datastreams":       c.DatastreamsPerSystem.Max > 0,
			"control streams":   c.ControlStreamsPerSystem.Max > 0,
			"deployments":       c.Deployments > 0,
			"system events":     c.SystemEventsPerSystem.Max > 0,
		} {
			if requested {
				return fmt.Errorf("seed.systems must be positive when %s are requested", name)
			}
		}
	}
	if c.SamplingFeaturesPerSystem.Max > 0 && (c.Collections == 0 || c.FeaturesPerCollection.Max == 0) {
		return fmt.Errorf("sampling features require at least one collection with generic features")
	}
	if c.SubdeploymentsPerDeployment.Max > 0 && c.Deployments == 0 {
		return fmt.Errorf("subdeployments require seed.deployments to be positive")
	}
	if c.Deployments > 0 && c.SystemsPerDeployment.Min == 0 {
		return fmt.Errorf("seed.systems_per_deployment.min must be positive when deployments are requested")
	}
	if c.DatastreamsPerSystem.Max == 0 && c.InitialObservationsPerDatastream.Max > 0 {
		return fmt.Errorf("initial observations require datastreams")
	}
	if c.ControlStreamsPerSystem.Max == 0 && c.CommandsPerControlStream.Max > 0 {
		return fmt.Errorf("commands require control streams")
	}

	validTypes := map[string]bool{"sensor": true, "actuator": true, "sampler": true, "platform": true, "system": true}
	totalWeight := 0
	controllerWeight := 0
	for role, weight := range c.SystemTypeWeights {
		if !validTypes[role] {
			return fmt.Errorf("unknown system type weight %q", role)
		}
		if weight < 0 {
			return fmt.Errorf("system type weight %q must be non-negative", role)
		}
		totalWeight += weight
		if role == "actuator" || role == "platform" || role == "system" {
			controllerWeight += weight
		}
	}
	if c.Systems > 0 && totalWeight == 0 {
		return fmt.Errorf("seed.system_type_weights must contain a positive weight when systems are requested")
	}
	if c.ControlStreamsPerSystem.Max > 0 && controllerWeight == 0 {
		return fmt.Errorf("control streams require a positive actuator, platform, or system type weight")
	}
	if c.Systems > 0 && c.ProcedureVariantsPerSystemType.Min == 0 {
		return fmt.Errorf("seed.procedure_variants_per_system_type.min must be positive when systems are requested")
	}
	return nil
}
