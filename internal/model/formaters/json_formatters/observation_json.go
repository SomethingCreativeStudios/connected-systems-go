package json_formatters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// ObservationJSONFeature is the application/json representation of an
// observation. Its fields are listed explicitly so the API shape does not
// follow the database model.
type ObservationJSONFeature struct {
	ID                string                   `json:"id"`
	DatastreamID      string                   `json:"datastream@id"`
	SamplingFeatureID *string                  `json:"samplingFeature@id,omitempty"`
	ProcedureLink     *common_shared.Link      `json:"procedure@link,omitempty"`
	PhenomenonTime    *time.Time               `json:"phenomenonTime,omitempty"`
	ResultTime        time.Time                `json:"resultTime"`
	Parameters        common_shared.Properties `json:"parameters,omitempty"`
	Result            json.RawMessage          `json:"result,omitempty"`
	ResultLink        *common_shared.Link      `json:"result@link,omitempty"`
}

// toDomain keeps the fields a client may write; id and datastream@id are
// assigned by the server.
func (w ObservationJSONFeature) toDomain() *domains.Observation {
	return &domains.Observation{
		SamplingFeatureID: w.SamplingFeatureID,
		ProcedureLink:     w.ProcedureLink,
		PhenomenonTime:    w.PhenomenonTime,
		ResultTime:        w.ResultTime,
		Parameters:        w.Parameters,
		Result:            w.Result,
		ResultLink:        w.ResultLink,
	}
}

// ObservationJSONFormatter handles observation JSON serialization/deserialization.
type ObservationJSONFormatter struct {
	links *formaters.LinkResolver
}

var _ formaters.Formatter[ObservationJSONFeature, *domains.Observation] = (*ObservationJSONFormatter)(nil)

func NewObservationJSONFormatter(links *formaters.LinkResolver) *ObservationJSONFormatter {
	return &ObservationJSONFormatter{links: links}
}

func (f *ObservationJSONFormatter) ContentType() string {
	return mediatype.JSON
}

func (f *ObservationJSONFormatter) Serialize(ctx context.Context, obs *domains.Observation) (ObservationJSONFeature, error) {
	return formaters.SerializeOne(ctx, obs, f.SerializeAll)
}

func (f *ObservationJSONFormatter) SerializeAll(ctx context.Context, observations []*domains.Observation) ([]ObservationJSONFeature, error) {
	observations = slices.DeleteFunc(slices.Clone(observations), func(obs *domains.Observation) bool { return obs == nil })

	var procedureIDs []string
	for _, obs := range observations {
		if obs.ProcedureLink != nil {
			procedureIDs = appendID(procedureIDs, obs.ProcedureLink.GetId("procedures"))
		}
	}
	cache, err := f.links.Lookup(ctx, formaters.References{Procedures: procedureIDs})
	if err != nil {
		return nil, err
	}

	items := make([]ObservationJSONFeature, 0, len(observations))
	for _, obs := range observations {
		procedure := inlineLink(f.links, obs.ProcedureLink, mediatype.SensorML)
		describeProcedure(procedure, cache)
		items = append(items, ObservationJSONFeature{
			ID:                obs.ID,
			DatastreamID:      obs.DatastreamID,
			SamplingFeatureID: obs.SamplingFeatureID,
			ProcedureLink:     procedure,
			PhenomenonTime:    obs.PhenomenonTime,
			ResultTime:        obs.ResultTime,
			Parameters:        obs.Parameters,
			Result:            obs.Result,
			ResultLink:        inlineLink(f.links, obs.ResultLink, ""),
		})
	}
	return items, nil
}

func (f *ObservationJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Observation, error) {
	wire, err := decodeObservation(reader)
	if err != nil {
		return nil, err
	}
	return wire.toDomain(), nil
}

// DeserializeResourceData decodes an observation published as a complete
// Pub/Sub Resource Data Message, which also carries the id and datastream@id
// that a request body cannot set.
func (f *ObservationJSONFormatter) DeserializeResourceData(ctx context.Context, reader io.Reader) (*domains.Observation, error) {
	wire, err := decodeObservation(reader)
	if err != nil {
		return nil, err
	}
	observation := wire.toDomain()
	observation.ID = wire.ID
	observation.DatastreamID = wire.DatastreamID
	return observation, nil
}

func decodeObservation(reader io.Reader) (*ObservationJSONFeature, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// First, validate required fields at the raw JSON level to provide
	// clear error messages consistent with the old decodeObservationPayload.
	var rawMap map[string]any
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return nil, err
	}

	// Validate resultTime is present and is a string
	rtRaw, hasResultTime := rawMap["resultTime"]
	if !hasResultTime {
		return nil, fmt.Errorf("resultTime is required")
	}
	rtStr, ok := rtRaw.(string)
	if !ok {
		return nil, fmt.Errorf("resultTime must be an ISO 8601 string (e.g. \"2026-01-01T00:00:00Z\")")
	}
	if rtStr == "" {
		return nil, fmt.Errorf("resultTime must be a non-empty ISO 8601 string (e.g. \"2026-01-01T00:00:00Z\")")
	}

	// Validate phenomenonTime is a string if present
	if ptRaw, exists := rawMap["phenomenonTime"]; exists {
		ptStr, ok := ptRaw.(string)
		if !ok {
			return nil, fmt.Errorf("phenomenonTime must be an ISO 8601 string (e.g. \"2026-01-01T00:00:00Z\")")
		}
		if ptStr == "" {
			return nil, fmt.Errorf("phenomenonTime must be a non-empty ISO 8601 string (e.g. \"2026-01-01T00:00:00Z\")")
		}
	}

	// Validate either result or result@link is present
	_, hasResult := rawMap["result"]
	_, hasResultLink := rawMap["result@link"]
	if !hasResult && !hasResultLink {
		return nil, errors.New("either result or result@link is required")
	}

	wire, err := common_shared.DecodeWithFieldErrors[ObservationJSONFeature](raw)
	if err != nil {
		return nil, err
	}
	return &wire, nil
}
