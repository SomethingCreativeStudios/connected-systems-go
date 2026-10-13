package sensorml_formatters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// getSMLType returns the SensorML process type for a system.
// Defaults to "PhysicalSystem" if not explicitly set.
func getSMLType(system *domains.System) string {
	if system.SMLType != nil && *system.SMLType != "" {
		return *system.SMLType
	}
	return "PhysicalSystem"
}

// SystemSensorMLFormatter handles serialization and deserialization of System objects in SensorML format
type SystemSensorMLFormatter struct {
	formaters.Formatter[domains.SystemSensorMLFeature, *domains.System]
	links *formaters.LinkResolver
}

// NewSystemSensorMLFormatter constructs a formatter with required repository readers
func NewSystemSensorMLFormatter(links *formaters.LinkResolver) *SystemSensorMLFormatter {
	return &SystemSensorMLFormatter{links: links}
}

func (f *SystemSensorMLFormatter) ContentType() string {
	return mediatype.SensorML
}

func nonNullRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return nil
	}
	return raw
}

func systemSensorMLPosition(system *domains.System) json.RawMessage {
	if position := nonNullRawMessage(system.Position); position != nil {
		return position
	}
	if system.Geometry == nil {
		return nil
	}

	position, err := json.Marshal(system.Geometry)
	if err != nil {
		return nil
	}
	return nonNullRawMessage(position)
}

// --- Serialization ---

func (f *SystemSensorMLFormatter) Serialize(ctx context.Context, system *domains.System) (domains.SystemSensorMLFeature, error) {
	return formaters.SerializeOne(ctx, system, f.SerializeAll)
}

func (f *SystemSensorMLFormatter) SerializeAll(ctx context.Context, systems []*domains.System) ([]domains.SystemSensorMLFeature, error) {
	if len(systems) == 0 {
		return []domains.SystemSensorMLFeature{}, nil
	}

	cache, err := f.links.Lookup(ctx, formaters.SystemReferences(systems))
	if err != nil {
		return nil, err
	}

	var features []domains.SystemSensorMLFeature
	for _, system := range systems {

		// attachedTo is server-generated from ParentSystemID
		var attachedTo *common_shared.Link
		if system.ParentSystemID != nil && strings.TrimSpace(*system.ParentSystemID) != "" {
			parentID := strings.TrimSpace(*system.ParentSystemID)
			attachedTo = &common_shared.Link{
				Href: f.links.Href("/systems/" + parentID),
				Rel:  common_shared.OGCRel("attachedTo"),
				Type: mediatype.GeoJSON,
			}
			// Enrich attachedTo from cache
			if parent, ok := cache.Systems[parentID]; ok {
				attachedTo.Describe(parent.Name, string(parent.UniqueIdentifier))
			}
		}

		// typeOf is the SensorML equivalent of systemKind@link in GeoJSON
		// Build from SystemKindID if TypeOf is not explicitly set
		typeOf := system.TypeOf
		if typeOf == nil && system.TypeOfID != nil && strings.TrimSpace(*system.TypeOfID) != "" {
			kindID := strings.TrimSpace(*system.TypeOfID)
			typeOf = &common_shared.Link{
				Href: f.links.Href("/procedures/" + kindID),
				Rel:  common_shared.OGCRel("systemKind"),
				Type: mediatype.SensorML,
			}
			// Enrich typeOf from cache
			if proc, ok := cache.Procedures[kindID]; ok {
				typeOf.Describe(proc.Name, string(proc.UniqueIdentifier))
			}
		}

		// Build classifiers list, injecting assetType as a cs:AssetType classifier
		classifiers := system.Classifiers
		if system.AssetType != nil && strings.TrimSpace(*system.AssetType) != "" {
			assetClassifier := common_shared.Term{
				Definition: "cs:AssetType",
				Label:      "Asset Type",
				CodeSpace:  "cs",
				Value:      strings.TrimSpace(*system.AssetType),
			}
			classifiers = append(classifiers, assetClassifier)
		}

		feature := domains.SystemSensorMLFeature{
			ID:                   system.ID,
			Type:                 getSMLType(system),
			Label:                system.Name,
			Description:          system.Description,
			UniqueID:             string(system.UniqueIdentifier),
			ValidTime:            common_shared.NonEmptyTimeRange(system.ValidTime),
			Definition:           system.SystemType,
			Lang:                 system.Lang,
			Keywords:             system.Keywords,
			Identifiers:          system.Identifiers,
			Classifiers:          classifiers,
			SecurityConstraints:  system.SecurityConstraints,
			LegalConstraints:     system.LegalConstraints,
			Contacts:             system.Contacts,
			Documentation:        system.Documentation,
			History:              system.History,
			TypeOf:               typeOf,
			Configuration:        system.Configuration,
			FeaturesOfInterest:   system.FeaturesOfInterest,
			Inputs:               system.Inputs,
			Outputs:              system.Outputs,
			Parameters:           system.Parameters,
			Modes:                system.Modes,
			Position:             systemSensorMLPosition(system),
			AttachedTo:           attachedTo,
			LocalReferenceFrames: system.LocalReferenceFrames,
			LocalTimeFrames:      system.LocalTimeFrames,
			Links:                f.links.SystemLinks(system, cache, mediatype.SensorML),
		}
		features = append(features, feature)
	}
	return features, nil
}

// --- Deserialization ---

func (f *SystemSensorMLFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.System, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	sml, err := common_shared.DecodeWithFieldErrors[domains.SystemSensorMLFeature](body)
	if err != nil {
		return nil, err
	}

	system := &domains.System{
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(sml.UniqueID),
			Name:             sml.Label,
			Description:      sml.Description,
		},
		SystemType: sml.Definition,
		Links:      common_shared.StripAssociationLinks(sml.Links),
	}
	// type is the SML process type, not the semantic definition
	if sml.Type != "" {
		system.SMLType = &sml.Type
	}

	// Required root fields per OGC CS Part 1 system SensorML schema
	if system.SMLType == nil {
		return nil, fmt.Errorf("type is required")
	}
	if system.UniqueIdentifier == "" {
		return nil, fmt.Errorf("uniqueId is required")
	}
	if system.Name == "" {
		return nil, fmt.Errorf("label is required")
	}
	if system.SystemType == "" {
		return nil, fmt.Errorf("definition is required")
	}

	system.ValidTime = common_shared.NonEmptyTimeRange(sml.ValidTime)
	if system.Geometry, err = positionGeometry(sml.Position); err != nil {
		return nil, err
	}

	// typeOf is the SensorML equivalent of systemKind@link — extract the procedure ID
	system.TypeOf = sml.TypeOf
	if sml.TypeOf != nil {
		system.TypeOfID = sml.TypeOf.GetId("procedures")
	}

	// Extract assetType from classifiers (it's stored as cs:AssetType classifier in SensorML)
	var filteredClassifiers common_shared.Terms
	for _, c := range sml.Classifiers {
		if c.Definition == "cs:AssetType" {
			val := c.Value
			system.AssetType = &val
		} else {
			filteredClassifiers = append(filteredClassifiers, c)
		}
	}
	system.Classifiers = filteredClassifiers

	system.Identifiers = sml.Identifiers
	system.SecurityConstraints = sml.SecurityConstraints
	system.LegalConstraints = sml.LegalConstraints
	system.Lang = sml.Lang
	system.Keywords = sml.Keywords

	system.Configuration = sml.Configuration
	if sml.FeaturesOfInterest != nil {
		system.FeaturesOfInterest = sml.FeaturesOfInterest
	}
	system.Inputs = sml.Inputs
	system.Outputs = sml.Outputs
	system.Parameters = sml.Parameters
	system.Modes = sml.Modes
	system.LocalReferenceFrames = sml.LocalReferenceFrames
	system.LocalTimeFrames = sml.LocalTimeFrames
	system.Position = sml.Position
	// ogc-rel links and attachedTo are server-generated, so they are ignored on
	// write; the parent system is only ever set through /systems/{id}/subsystems.
	system.Contacts = sml.Contacts
	system.Documentation = sml.Documentation
	system.History = sml.History

	return system, nil
}

// positionGeometry reads a SensorML position as the system's geometry when it
// is a GeoJSON geometry. Other position encodings (a pose, a SWE vector or
// record, a link) are kept only in the raw position.
func positionGeometry(position json.RawMessage) (*common_shared.GoGeom, error) {
	var probe struct {
		Type string `json:"type"`
	}
	if len(position) == 0 || json.Unmarshal(position, &probe) != nil {
		return nil, nil
	}
	switch probe.Type {
	case "Point", "LineString", "Polygon", "MultiPoint", "MultiLineString", "MultiPolygon", "GeometryCollection":
	default:
		return nil, nil
	}
	var geometry common_shared.GoGeom
	if err := json.Unmarshal(position, &geometry); err != nil {
		return nil, fmt.Errorf("position: %w", err)
	}
	return &geometry, nil
}
