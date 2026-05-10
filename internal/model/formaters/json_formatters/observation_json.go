package json_formatters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
)

// ObservationJSONFeature is the wire-format representation of an observation.
type ObservationJSONFeature struct {
	domains.Observation
}

// ObservationJSONFormatter handles observation JSON serialization/deserialization.
type ObservationJSONFormatter struct {
	formaters.Formatter[ObservationJSONFeature, *domains.Observation]
}

func NewObservationJSONFormatter() *ObservationJSONFormatter {
	return &ObservationJSONFormatter{}
}

func (f *ObservationJSONFormatter) ContentType() string {
	return JSONContentType
}

func (f *ObservationJSONFormatter) Serialize(ctx context.Context, obs *domains.Observation) (ObservationJSONFeature, error) {
	if obs == nil {
		return ObservationJSONFeature{}, fmt.Errorf("observation cannot be nil")
	}
	out := ObservationJSONFeature{Observation: *obs}
	if out.ProcedureLink != nil && out.ProcedureLink.Href != "" {
		out.ProcedureLink.Href = formaters.ToFunctionalAssociationHref(out.ProcedureLink.Href)
	}
	if out.ResultLink != nil && out.ResultLink.Href != "" {
		out.ResultLink.Href = formaters.ToFunctionalAssociationHref(out.ResultLink.Href)
	}
	return out, nil
}

func (f *ObservationJSONFormatter) SerializeAll(ctx context.Context, observations []*domains.Observation) ([]ObservationJSONFeature, error) {
	if len(observations) == 0 {
		return []ObservationJSONFeature{}, nil
	}

	items := make([]ObservationJSONFeature, 0, len(observations))
	for _, obs := range observations {
		if obs == nil {
			continue
		}
		out := ObservationJSONFeature{Observation: *obs}
		if out.ProcedureLink != nil && out.ProcedureLink.Href != "" {
			out.ProcedureLink.Href = formaters.ToFunctionalAssociationHref(out.ProcedureLink.Href)
		}
		if out.ResultLink != nil && out.ResultLink.Href != "" {
			out.ResultLink.Href = formaters.ToFunctionalAssociationHref(out.ResultLink.Href)
		}
		items = append(items, out)
	}
	return items, nil
}

func (f *ObservationJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Observation, error) {
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
	if _, ok := rtRaw.(string); !ok {
		return nil, fmt.Errorf("resultTime must be an ISO 8601 string (e.g. \"2026-01-01T00:00:00Z\")")
	}

	// Validate phenomenonTime is a string if present
	if ptRaw, exists := rawMap["phenomenonTime"]; exists {
		if _, ok := ptRaw.(string); !ok {
			return nil, fmt.Errorf("phenomenonTime must be an ISO 8601 string (e.g. \"2026-01-01T00:00:00Z\")")
		}
	}

	// Validate either result or result@link is present
	_, hasResult := rawMap["result"]
	_, hasResultLink := rawMap["result@link"]
	if !hasResult && !hasResultLink {
		return nil, fmt.Errorf("Either result or result@link is required")
	}

	wire, err := common_shared.DecodeWithFieldErrors[domains.Observation](raw)
	if err != nil {
		return nil, err
	}
	return &wire, nil
}
