package geojson_formatters

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// SystemGeoJSONFormatter handles serialization and deserialization of System objects in GeoJSON format
type SystemGeoJSONFormatter struct {
	formaters.Formatter[domains.SystemGeoJSONFeature, *domains.System]
	links *formaters.LinkResolver
}

// NewSystemGeoJSONFormatter constructs a formatter with required repository readers
func NewSystemGeoJSONFormatter(links *formaters.LinkResolver) *SystemGeoJSONFormatter {
	return &SystemGeoJSONFormatter{links: links}
}

func (f *SystemGeoJSONFormatter) ContentType() string {
	return mediatype.GeoJSON
}

// --- Serialization ---

func (f *SystemGeoJSONFormatter) Serialize(ctx context.Context, system *domains.System) (domains.SystemGeoJSONFeature, error) {
	return formaters.SerializeOne(ctx, system, f.SerializeAll)
}

func (f *SystemGeoJSONFormatter) SerializeAll(ctx context.Context, systems []*domains.System) ([]domains.SystemGeoJSONFeature, error) {
	if len(systems) == 0 {
		return []domains.SystemGeoJSONFeature{}, nil
	}

	cache, err := f.links.Lookup(ctx, formaters.SystemReferences(systems))
	if err != nil {
		return nil, err
	}

	// Build a kindMap for systemKind@link (same data, different access pattern)
	kindMap := cache.Procedures

	var features []domains.SystemGeoJSONFeature
	for _, system := range systems {
		// Build systemKind@link — prefer the full procedure record (title/uid),
		// fall back to a bare href from SystemKindID, then to TypeOf as a last resort.
		var kindLink *common_shared.Link
		if system.TypeOfID != nil && strings.TrimSpace(*system.TypeOfID) != "" {
			id := strings.TrimSpace(*system.TypeOfID)
			if proc, ok := kindMap[id]; ok {
				kindLink = &common_shared.Link{
					Href:  f.links.Href("/procedures/" + proc.ID),
					Rel:   common_shared.OGCRel("systemKind"),
					Type:  mediatype.GeoJSON,
					Title: proc.Name,
					UID:   (*string)(&proc.UniqueIdentifier),
				}
			} else {
				kindLink = &common_shared.Link{
					Href: f.links.Href("/procedures/" + id),
					Rel:  common_shared.OGCRel("systemKind"),
					Type: mediatype.SensorML,
				}
			}
		} else if system.TypeOf != nil && system.TypeOf.Href != "" {
			// TypeOf was stored before SystemKindID was introduced; surface it as systemKind@link
			kindLink = &common_shared.Link{
				Href:  f.links.Href(system.TypeOf.Href),
				Rel:   common_shared.OGCRel("systemKind"),
				Type:  mediatype.SensorML,
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
				ValidTime:            common_shared.NonEmptyTimeRange(system.ValidTime),
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
			Links: f.links.SystemLinks(system, cache, mediatype.GeoJSON),
		}
		features = append(features, feature)
	}

	return features, nil
}

// --- Deserialization ---

func (f *SystemGeoJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.System, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	geoJSON, err := common_shared.DecodeWithFieldErrors[struct {
		Type       string                          `json:"type"`
		ID         string                          `json:"id,omitempty"`
		Properties domains.SystemGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.GoGeom           `json:"geometry,omitempty"`
		Links      common_shared.Links             `json:"links,omitempty"`
	}](body)
	if err != nil {
		return nil, err
	}

	// ogc-rel links are server-generated, so they are ignored on write; the
	// parent system is only ever set through /systems/{id}/subsystems.
	system := &domains.System{
		Links: common_shared.StripAssociationLinks(geoJSON.Links),
	}

	// Assign geometry
	if geoJSON.Geometry != nil {
		system.Geometry = geoJSON.Geometry
	}

	// Required root fields per OGC CS Part 1 system GeoJSON schema
	if geoJSON.Properties.UID == "" {
		return nil, fmt.Errorf("properties.uid is required")
	}
	if geoJSON.Properties.Name == "" {
		return nil, fmt.Errorf("properties.name is required")
	}
	if geoJSON.Properties.FeatureType == "" {
		return nil, fmt.Errorf("properties.featureType is required")
	}

	// Extract properties
	system.UniqueIdentifier = domains.UniqueID(geoJSON.Properties.UID)
	system.Name = geoJSON.Properties.Name
	system.Description = geoJSON.Properties.Description
	system.SystemType = geoJSON.Properties.FeatureType
	system.AssetType = geoJSON.Properties.AssetType
	system.SMLType = geoJSON.Properties.SMLType
	system.ValidTime = common_shared.NonEmptyTimeRange(geoJSON.Properties.ValidTime)
	if geoJSON.Properties.SystemKind != nil {
		system.TypeOfID = geoJSON.Properties.SystemKind.GetId("procedures")
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

	return system, nil
}
