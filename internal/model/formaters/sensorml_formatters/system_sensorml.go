package sensorml_formatters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

const SensorMLContentType = "application/sml+json"

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
	repos *repository.Repositories
}

// NewSystemSensorMLFormatter constructs a formatter with required repository readers
func NewSystemSensorMLFormatter(repos *repository.Repositories) *SystemSensorMLFormatter {
	return &SystemSensorMLFormatter{repos: repos}
}

func (f *SystemSensorMLFormatter) ContentType() string {
	return SensorMLContentType
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
	features, err := f.SerializeAll(ctx, []*domains.System{system})
	if err != nil {
		return domains.SystemSensorMLFeature{}, err
	}
	return features[0], nil
}

func (f *SystemSensorMLFormatter) SerializeAll(ctx context.Context, systems []*domains.System) ([]domains.SystemSensorMLFeature, error) {
	if len(systems) == 0 {
		return []domains.SystemSensorMLFeature{}, nil
	}

	// Collect IDs for batch loading
	kindIDs := make([]string, 0, len(systems))
	parentIDs := make([]string, 0, len(systems))
	for _, s := range systems {
		if s.TypeOfID != nil && *s.TypeOfID != "" {
			kindIDs = append(kindIDs, *s.TypeOfID)
		}
		if s.ParentSystemID != nil && strings.TrimSpace(*s.ParentSystemID) != "" {
			parentIDs = append(parentIDs, strings.TrimSpace(*s.ParentSystemID))
		}
	}

	// Build resource cache for enriching association links
	cache := formaters.NewResourceCache()
	if len(kindIDs) > 0 && f.repos != nil {
		_ = cache.FetchProcedures(ctx, f.repos.Procedure, kindIDs)
	}
	if len(parentIDs) > 0 && f.repos != nil {
		_ = cache.FetchParentSystems(ctx, f.repos.System, parentIDs)
	}

	var features []domains.SystemSensorMLFeature
	for _, system := range systems {

		// attachedTo is server-generated from ParentSystemID
		var attachedTo *common_shared.Link
		if system.ParentSystemID != nil && strings.TrimSpace(*system.ParentSystemID) != "" {
			parentID := strings.TrimSpace(*system.ParentSystemID)
			attachedTo = &common_shared.Link{
				Href: formaters.ToFunctionalAssociationHref("/systems/" + parentID),
				Rel:  common_shared.OGCRel("attachedTo"),
				Type: formaters.GeoJSONContentType,
			}
			// Enrich attachedTo from cache
			if parent, ok := cache.Systems[parentID]; ok {
				attachedTo.Title = parent.Name
				uid := string(parent.UniqueIdentifier)
				if uid != "" {
					attachedTo.UID = &uid
				}
			}
		}

		// typeOf is the SensorML equivalent of systemKind@link in GeoJSON
		// Build from SystemKindID if TypeOf is not explicitly set
		typeOf := system.TypeOf
		if typeOf == nil && system.TypeOfID != nil && strings.TrimSpace(*system.TypeOfID) != "" {
			kindID := strings.TrimSpace(*system.TypeOfID)
			typeOf = &common_shared.Link{
				Href: formaters.ToFunctionalAssociationHref("/procedures/" + kindID),
				Rel:  common_shared.OGCRel("systemKind"),
				Type: formaters.SensorMLContentType,
			}
			// Enrich typeOf from cache
			if proc, ok := cache.Procedures[kindID]; ok {
				typeOf.Title = proc.Name
				uid := string(proc.UniqueIdentifier)
				if uid != "" {
					typeOf.UID = &uid
				}
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
			Links:                formaters.AppendSensorMLSystemAssociationLinks(system, cache),
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

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	sml, err := common_shared.DecodeWithFieldErrors[domains.SystemSensorMLFeature](body)
	if err != nil {
		return nil, err
	}

	system := &domains.System{
		Links: common_shared.StripAssociationLinks(sml.Links),
	}

	if v, ok := raw["label"].(string); ok && v != "" {
		system.Name = v
	} else if v, ok := raw["name"].(string); ok && v != "" {
		system.Name = v
	}

	if v, ok := raw["description"].(string); ok {
		system.Description = v
	}

	if v, ok := raw["uniqueId"].(string); ok && v != "" {
		system.UniqueIdentifier = domains.UniqueID(v)
	} else if v, ok := raw["uid"].(string); ok && v != "" {
		system.UniqueIdentifier = domains.UniqueID(v)
	}

	if sml.Definition != "" {
		system.SystemType = sml.Definition
	} else if v, ok := raw["definition"].(string); ok {
		system.SystemType = v
	}

	// type is the SML process type, not the semantic definition
	if v, ok := raw["type"].(string); ok && v != "" {
		system.SMLType = &v
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
	if system.ValidTime == nil {
		if vt, ok := raw["validTime"]; ok {
			tr := common_shared.ParseTimeRange(vt)
			system.ValidTime = common_shared.NonEmptyTimeRange(&tr)
		}
	}

	var geomObj interface{}
	if p, ok := raw["position"]; ok {
		geomObj = p
	} else if g, ok := raw["geometry"]; ok {
		geomObj = g
	}
	if geomObj != nil {
		if gb, err := json.Marshal(geomObj); err == nil {
			var gg common_shared.GoGeom
			if err := json.Unmarshal(gb, &gg); err == nil {
				system.Geometry = &gg
			}
		}
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
	// Derive ParentSystemID from ogc-rel:parentSystem link; fall back to attachedTo for incoming SensorML
	formaters.ApplyGeoJSONSystemAssociationLinks(system, sml.Links)
	if system.ParentSystemID == nil && sml.AttachedTo != nil {
		system.ParentSystemID = sml.AttachedTo.GetId("systems")
	}
	system.Contacts = sml.Contacts
	system.Documentation = sml.Documentation
	system.History = sml.History

	return system, nil
}
