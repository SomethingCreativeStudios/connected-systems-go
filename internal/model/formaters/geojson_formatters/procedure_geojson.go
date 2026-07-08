package geojson_formatters

import (
	"context"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// ProcedureGeoJSONFormatter handles serialization and deserialization of Procedure objects in GeoJSON format
type ProcedureGeoJSONFormatter struct {
	formaters.Formatter[domains.ProcedureGeoJSONFeature, *domains.Procedure]
	repos *repository.Repositories
}

// NewProcedureGeoJSONFormatter constructs a formatter with required repository readers
func NewProcedureGeoJSONFormatter(repos *repository.Repositories) *ProcedureGeoJSONFormatter {
	return &ProcedureGeoJSONFormatter{repos: repos}
}

func (f *ProcedureGeoJSONFormatter) ContentType() string {
	return GeoJSONContentType
}

// --- Serialization ---

func (f *ProcedureGeoJSONFormatter) Serialize(ctx context.Context, procedure *domains.Procedure) (domains.ProcedureGeoJSONFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.Procedure{procedure})
	if err != nil {
		return domains.ProcedureGeoJSONFeature{}, err
	}
	return features[0], nil
}

func (f *ProcedureGeoJSONFormatter) SerializeAll(ctx context.Context, procedures []*domains.Procedure) ([]domains.ProcedureGeoJSONFeature, error) {
	if len(procedures) == 0 {
		return []domains.ProcedureGeoJSONFeature{}, nil
	}

	var features []domains.ProcedureGeoJSONFeature
	for _, procedure := range procedures {
		feature := domains.ProcedureGeoJSONFeature{
			Type:     "Feature",
			ID:       procedure.ID,
			Geometry: nil, // Procedure GeoJson do not have geometry
			Properties: domains.ProcedureGeoJSONProperties{
				UID:         procedure.UniqueIdentifier,
				Name:        procedure.Name,
				Description: procedure.Description,
				FeatureType: procedure.ProcedureType,
				ValidTime:   common_shared.NonEmptyTimeRange(procedure.ValidTime),
			},
			Links: formaters.AppendProcedureAssociationLinks(procedure),
		}
		features = append(features, feature)
	}

	return features, nil
}

// --- Deserialization ---

func (f *ProcedureGeoJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Procedure, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	geoJSON, err := common_shared.DecodeWithFieldErrors[struct {
		Type       string                             `json:"type"`
		ID         string                             `json:"id,omitempty"`
		Properties domains.ProcedureGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.GoGeom              `json:"geometry,omitempty"`
		Links      common_shared.Links                `json:"links,omitempty"`
	}](body)
	if err != nil {
		return nil, err
	}

	procedure := &domains.Procedure{
		Links: common_shared.StripAssociationLinks(geoJSON.Links),
	}

	// Extract properties
	procedure.UniqueIdentifier = domains.UniqueID(geoJSON.Properties.UID)
	procedure.Name = geoJSON.Properties.Name
	procedure.Description = geoJSON.Properties.Description
	procedure.ProcedureType = geoJSON.Properties.FeatureType
	procedure.ValidTime = common_shared.NonEmptyTimeRange(geoJSON.Properties.ValidTime)

	return procedure, nil
}
