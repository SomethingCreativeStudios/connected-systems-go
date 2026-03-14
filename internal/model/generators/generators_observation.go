package generators

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

func FakeObservationForDatastream(ds domains.Datastream) domains.Observation {
	result, params := fakeObservationResultAndParametersForSchema(ds.Schema)
	now := time.Now().UTC().Truncate(time.Second)

	obs := domains.Observation{
		Base:         domains.Base{ID: uuid.NewString()},
		DatastreamID: ds.ID,
		ResultTime:   now,
		Parameters:   params,
		Result:       result,
	}

	phen := now.Add(-5 * time.Second)
	obs.PhenomenonTime = &phen

	if ds.SamplingFeatureID != nil {
		obs.SamplingFeatureID = ds.SamplingFeatureID
	} else if ds.SamplingFeatureLink != nil {
		if id := ds.SamplingFeatureLink.GetId("samplingFeatures"); id != nil {
			obs.SamplingFeatureID = id
		}
	}

	if ds.ProcedureLink != nil {
		obs.ProcedureLink = ds.ProcedureLink
	}

	return obs
}

func FakeObservationWithResultLink(ds domains.Datastream) domains.Observation {
	now := time.Now().UTC().Truncate(time.Second)
	linkID := uuid.NewString()
	obs := domains.Observation{
		Base:         domains.Base{ID: uuid.NewString()},
		DatastreamID: ds.ID,
		ResultTime:   now,
		Parameters: common_shared.Properties{
			"source": "external",
		},
		ResultLink: &common_shared.Link{
			Href:  "https://example.org/results/" + linkID,
			Title: "External Result",
			Type:  "application/json",
			Rel:   "alternate",
		},
	}
	phen := now.Add(-2 * time.Second)
	obs.PhenomenonTime = &phen
	return obs
}

func FakeObservationListForDatastream(ds domains.Datastream, n int) []domains.Observation {
	if n <= 0 {
		return []domains.Observation{}
	}

	out := make([]domains.Observation, 0, n)
	for i := 0; i < n; i++ {
		if i%5 == 4 {
			out = append(out, FakeObservationWithResultLink(ds))
			continue
		}
		out = append(out, FakeObservationForDatastream(ds))
	}
	return out
}

func fakeObservationResultAndParametersForSchema(schema *domains.DatastreamSchema) (json.RawMessage, common_shared.Properties) {
	params := common_shared.Properties{
		"qc":      "good",
		"station": f.Lorem().Word(),
	}

	if schema == nil {
		return mustMarshalResult(42.1), params
	}

	switch schema.ObsFormat {
	case "application/x-protobuf":
		// Stored as JSON object payload but shaped to the protobuf schema fields.
		return mustMarshalResult(map[string]interface{}{
			"temperature": 21.5,
			"humidity":    57.0,
			"status":      "ok",
		}), params

	case "application/json", "application/swe+json", "application/swe+csv":
		if schema.ResultSchema != nil && schema.ResultSchema.Type == "DataRecord" {
			return mustMarshalResult(map[string]interface{}{
				"temperature": 20.7,
				"humidity":    54.2,
				"status":      "ok",
			}), params
		}
		if schema.RecordSchema != nil && schema.RecordSchema.Type == "DataRecord" {
			return mustMarshalResult(map[string]interface{}{
				"x": 102.3,
				"y": 27.1,
			}), params
		}
		return mustMarshalResult(22.4), params

	default:
		return mustMarshalResult(map[string]interface{}{
			"value":  99.9,
			"format": schema.ObsFormat,
		}), params
	}
}

func mustMarshalResult(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("null")
	}
	return b
}
