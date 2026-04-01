package geojson_formatters

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

const GeoJSONContentType = "application/geo+json"

// SystemGeoJSONFormatter handles serialization and deserialization of System objects in GeoJSON format
type SystemGeoJSONFormatter struct {
	formaters.Formatter[domains.SystemGeoJSONFeature, *domains.System]
	repos *repository.Repositories
}

// NewSystemGeoJSONFormatter constructs a formatter with required repository readers
func NewSystemGeoJSONFormatter(repos *repository.Repositories) *SystemGeoJSONFormatter {
	return &SystemGeoJSONFormatter{repos: repos}
}

func (f *SystemGeoJSONFormatter) ContentType() string {
	return GeoJSONContentType
}

// --- Serialization ---

func (f *SystemGeoJSONFormatter) Serialize(ctx context.Context, system *domains.System) (domains.SystemGeoJSONFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.System{system})
	if err != nil {
		return domains.SystemGeoJSONFeature{}, err
	}
	return features[0], nil
}

func (f *SystemGeoJSONFormatter) SerializeAll(ctx context.Context, systems []*domains.System) ([]domains.SystemGeoJSONFeature, error) {
	if len(systems) == 0 {
		return []domains.SystemGeoJSONFeature{}, nil
	}

	// Collect system kind IDs for batch loading
	kindIDs := make([]string, 0, len(systems))
	for _, s := range systems {
		if s.SystemKindID != nil && *s.SystemKindID != "" {
			kindIDs = append(kindIDs, *s.SystemKindID)
		}
	}

	// Batch-fetch procedures (system kinds)
	kindMap := make(map[string]*domains.Procedure)
	if len(kindIDs) > 0 && f.repos != nil {
		procedures, err := f.repos.Procedure.GetByIDs(ctx, kindIDs)
		if err == nil {
			for _, p := range procedures {
				kindMap[p.ID] = p
			}
		}
	}

	var features []domains.SystemGeoJSONFeature
	for _, system := range systems {
		// Build systemKind@link — prefer the full procedure record (title/uid),
		// fall back to a bare href from SystemKindID, then to TypeOf as a last resort.
		var kindLink *common_shared.Link
		if system.SystemKindID != nil && strings.TrimSpace(*system.SystemKindID) != "" {
			id := strings.TrimSpace(*system.SystemKindID)
			if proc, ok := kindMap[id]; ok {
				kindLink = &common_shared.Link{
					Href:  "procedures/" + proc.ID,
					Rel:   common_shared.OGCRel("systemKind"),
					Type:  GeoJSONContentType,
					Title: proc.Name,
					UID:   (*string)(&proc.UniqueIdentifier),
				}
			} else {
				kindLink = &common_shared.Link{
					Href: "procedures/" + id,
					Rel:  common_shared.OGCRel("systemKind"),
				}
			}
		} else if system.TypeOf != nil && system.TypeOf.Href != "" {
			// TypeOf was stored before SystemKindID was introduced; surface it as systemKind@link
			kindLink = &common_shared.Link{
				Href:  system.TypeOf.Href,
				Rel:   common_shared.OGCRel("systemKind"),
				Title: system.TypeOf.Title,
			}
		}

		feature := domains.SystemGeoJSONFeature{
			Type:     "Feature",
			ID:       system.ID,
			Geometry: system.Geometry,
			Properties: domains.SystemGeoJSONProperties{
				UID:                  system.UniqueIdentifier,
				Name:                 system.Name,
				Description:          system.Description,
				FeatureType:          system.SystemType,
				AssetType:            system.AssetType,
				SMLType:              system.SMLType,
				ValidTime:            system.ValidTime,
				SystemKind:           kindLink,
				Lang:                 system.Lang,
				Keywords:             system.Keywords,
				Identifiers:          system.Identifiers,
				Classifiers:          system.Classifiers,
				Contacts:             system.Contacts,
				Documentation:        system.Documentation,
				History:              system.History,
				Configuration:        system.Configuration,
				FeaturesOfInterest:   system.FeaturesOfInterest,
				Inputs:               system.Inputs,
				Outputs:              system.Outputs,
				Parameters:           system.Parameters,
				Modes:                system.Modes,
				LocalReferenceFrames: system.LocalReferenceFrames,
				LocalTimeFrames:      system.LocalTimeFrames,
				Position:             system.Position,
			},
			Links: formaters.AppendGeoJSONSystemAssociationLinks(system),
		}
		features = append(features, feature)
	}

	return features, nil
}

// --- Deserialization ---

func (f *SystemGeoJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.System, error) {
	var geoJSON struct {
		Type       string                          `json:"type"`
		ID         string                          `json:"id,omitempty"`
		Properties domains.SystemGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.GoGeom           `json:"geometry,omitempty"`
		Links      common_shared.Links             `json:"links,omitempty"`
	}

	if err := json.NewDecoder(reader).Decode(&geoJSON); err != nil {
		return nil, err
	}

	associationLinks := formaters.GeoJSONSystemAssociationLinks(common_shared.StripAssociationLinks(geoJSON.Links))

	system := &domains.System{
		Links: common_shared.StripAssociationLinks(geoJSON.Links),
	}

	// Assign geometry
	if geoJSON.Geometry != nil {
		system.Geometry = geoJSON.Geometry
	}

	// Extract properties
	system.UniqueIdentifier = domains.UniqueID(geoJSON.Properties.UID)
	system.Name = geoJSON.Properties.Name
	system.Description = geoJSON.Properties.Description
	system.SystemType = geoJSON.Properties.FeatureType
	system.AssetType = geoJSON.Properties.AssetType
	system.SMLType = geoJSON.Properties.SMLType
	system.ValidTime = geoJSON.Properties.ValidTime
	if geoJSON.Properties.SystemKind != nil {
		system.SystemKindID = geoJSON.Properties.SystemKind.GetId("procedures")
	}

	// Map additional SWE/System fields
	system.Lang = geoJSON.Properties.Lang
	system.Keywords = geoJSON.Properties.Keywords
	system.Identifiers = geoJSON.Properties.Identifiers
	system.Classifiers = geoJSON.Properties.Classifiers
	system.Contacts = geoJSON.Properties.Contacts
	system.Documentation = geoJSON.Properties.Documentation
	system.History = geoJSON.Properties.History
	system.Configuration = geoJSON.Properties.Configuration
	system.FeaturesOfInterest = geoJSON.Properties.FeaturesOfInterest
	system.Inputs = geoJSON.Properties.Inputs
	system.Outputs = geoJSON.Properties.Outputs
	system.Parameters = geoJSON.Properties.Parameters
	system.Modes = geoJSON.Properties.Modes
	system.LocalReferenceFrames = geoJSON.Properties.LocalReferenceFrames
	system.LocalTimeFrames = geoJSON.Properties.LocalTimeFrames
	system.Position = geoJSON.Properties.Position

	formaters.ApplyGeoJSONSystemAssociationLinks(system, associationLinks)

	return system, nil
}
