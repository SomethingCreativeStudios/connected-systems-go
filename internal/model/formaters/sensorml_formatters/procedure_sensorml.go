package sensorml_formatters

import (
	"context"
	"fmt"
	"encoding/json"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// ProcedureSensorMLFormatter handles serialization and deserialization of Procedure objects in SensorML format
type ProcedureSensorMLFormatter struct {
	formaters.Formatter[domains.ProcedureSensorMLFeature, *domains.Procedure]
	repos *repository.Repositories
}

// NewProcedureSensorMLFormatter constructs a formatter with required repository readers
func NewProcedureSensorMLFormatter(repos *repository.Repositories) *ProcedureSensorMLFormatter {
	return &ProcedureSensorMLFormatter{repos: repos}
}

func (f *ProcedureSensorMLFormatter) ContentType() string {
	return SensorMLContentType
}

// --- Serialization ---

func (f *ProcedureSensorMLFormatter) Serialize(ctx context.Context, procedure *domains.Procedure) (domains.ProcedureSensorMLFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.Procedure{procedure})
	if err != nil {
		return domains.ProcedureSensorMLFeature{}, err
	}
	return features[0], nil
}

func (f *ProcedureSensorMLFormatter) SerializeAll(ctx context.Context, procedures []*domains.Procedure) ([]domains.ProcedureSensorMLFeature, error) {
	if len(procedures) == 0 {
		return []domains.ProcedureSensorMLFeature{}, nil
	}

	// Collect linked resource IDs for batch enrichment
	typeOfIDs := make([]string, 0, len(procedures))
	attachedToIDs := make([]string, 0, len(procedures))
	for _, p := range procedures {
		if p == nil {
			continue
		}
		if p.TypeOf != nil && p.TypeOf.Href != "" {
			id := p.TypeOf.GetId("procedures")
			if id != nil && *id != "" {
				typeOfIDs = append(typeOfIDs, *id)
			}
		}
		if p.AttachedTo != nil && p.AttachedTo.Href != "" {
			id := p.AttachedTo.GetId("systems")
			if id != nil && *id != "" {
				attachedToIDs = append(attachedToIDs, *id)
			}
		}
	}

	// Build resource cache for enriching inline links
	cache := formaters.NewResourceCache()
	if f.repos != nil {
		if len(typeOfIDs) > 0 {
			_ = cache.FetchProcedures(ctx, f.repos.Procedure, typeOfIDs)
		}
		if len(attachedToIDs) > 0 {
			_ = cache.FetchParentSystems(ctx, f.repos.System, attachedToIDs)
		}
	}

	var features []domains.ProcedureSensorMLFeature

	// typed process representations
	type CommonProcess struct {
		ID                   string
		Type                 string
		Label                string
		Description          string
		UniqueID             string
		Definition           string
		Lang                 *string
		Keywords             []string
		Identifiers          common_shared.Terms
		Classifiers          common_shared.Terms
		SecurityConstraints  common_shared.SecurityConstraints
		LegalConstraints     common_shared.LegalConstraints
		Characteristics      []common_shared.CharacteristicGroup
		Capabilities         []common_shared.CapabilityGroup
		Contacts             []common_shared.ContactWrapper
		Documentation        common_shared.Documents
		History              common_shared.History
		TypeOf               *common_shared.Link
		Configuration        json.RawMessage
		FeaturesOfInterest   common_shared.Links
		Inputs               common_shared.IOList
		Outputs              common_shared.IOList
		Parameters           common_shared.IOList
		Modes                json.RawMessage
		Method               common_shared.Method
		AttachedTo           *common_shared.Link
		LocalReferenceFrames []common_shared.SpatialFrame
		LocalTimeFrames      []common_shared.TemporalFrame
		ValidTime            *common_shared.TimeRange
		Links                common_shared.Links
	}

	type SimpleProcess struct{ CommonProcess }
	type PhysicalComponent struct{ CommonProcess }
	type AggregateProcess struct {
		CommonProcess
		Components  json.RawMessage
		Connections json.RawMessage
	}
	type PhysicalSystem struct {
		CommonProcess
		Components  json.RawMessage
		Connections json.RawMessage
	}

	// helper to map a CommonProcess (or specialized) back to ProcedureSensorMLFeature
	toFeature := func(c CommonProcess) domains.ProcedureSensorMLFeature {
		return domains.ProcedureSensorMLFeature{
			ID:                   c.ID,
			Type:                 c.Type,
			Label:                c.Label,
			Description:          c.Description,
			UniqueID:             c.UniqueID,
			Definition:           c.Definition,
			Lang:                 c.Lang,
			Keywords:             c.Keywords,
			Identifiers:          c.Identifiers,
			Classifiers:          c.Classifiers,
			SecurityConstraints:  c.SecurityConstraints,
			LegalConstraints:     c.LegalConstraints,
			Characteristics:      c.Characteristics,
			Capabilities:         c.Capabilities,
			Contacts:             c.Contacts,
			Documentation:        c.Documentation,
			History:              c.History,
			TypeOf:               c.TypeOf,
			Configuration:        c.Configuration,
			FeaturesOfInterest:   c.FeaturesOfInterest,
			Inputs:               c.Inputs,
			Outputs:              c.Outputs,
			Parameters:           c.Parameters,
			Modes:                c.Modes,
			Method:               c.Method,
			AttachedTo:           c.AttachedTo,
			LocalReferenceFrames: c.LocalReferenceFrames,
			LocalTimeFrames:      c.LocalTimeFrames,
			ValidTime:            common_shared.NonEmptyTimeRange(c.ValidTime),
			Links:                c.Links,
		}
	}

	for _, procedure := range procedures {
		base := CommonProcess{
			ID:                   procedure.ID,
			Type:                 procedure.ProcessType,
			Label:                procedure.Name,
			Description:          procedure.Description,
			UniqueID:             string(procedure.UniqueIdentifier),
			Definition:           procedure.ProcedureType,
			Lang:                 procedure.Lang,
			Keywords:             procedure.Keywords,
			Identifiers:          procedure.Identifiers,
			Classifiers:          procedure.Classifiers,
			SecurityConstraints:  procedure.SecurityConstraints,
			LegalConstraints:     procedure.LegalConstraints,
			Characteristics:      procedure.Characteristics,
			Capabilities:         procedure.Capabilities,
			Contacts:             procedure.Contacts,
			Documentation:        procedure.Documentation,
			History:              procedure.History,
			TypeOf:               absolutizeLink(procedure.TypeOf),
			Configuration:        procedure.Configuration,
			FeaturesOfInterest:   absolutizeLinksValue(procedure.FeaturesOfInterest),
			Inputs:               procedure.Inputs,
			Outputs:              procedure.Outputs,
			Parameters:           procedure.Parameters,
			Modes:                procedure.Modes,
			Method:               procedure.Method,
			AttachedTo:           absolutizeLink(procedure.AttachedTo),
			LocalReferenceFrames: procedure.LocalReferenceFrames,
			LocalTimeFrames:      procedure.LocalTimeFrames,
			ValidTime:            common_shared.NonEmptyTimeRange(procedure.ValidTime),
			Links:                formaters.AppendProcedureAssociationLinks(procedure),
		}

		// Override typeOf Type: absolutizeLink defaults to GeoJSON, but typeOf is a procedure → SensorML
		if base.TypeOf != nil && base.TypeOf.Type == formaters.GeoJSONContentType {
			base.TypeOf.Type = formaters.SensorMLContentType
		}

		// Enrich typeOf from cache
		if base.TypeOf != nil {
			typeOfID := base.TypeOf.GetId("procedures")
			if typeOfID != nil {
				if proc, ok := cache.Procedures[*typeOfID]; ok {
					base.TypeOf.Title = proc.Name
					uid := string(proc.UniqueIdentifier)
					if uid != "" {
						base.TypeOf.UID = &uid
					}
				}
			}
		}

		// Enrich attachedTo from cache
		if base.AttachedTo != nil {
			attachedID := base.AttachedTo.GetId("systems")
			if attachedID != nil {
				if sys, ok := cache.Systems[*attachedID]; ok {
					base.AttachedTo.Title = sys.Name
					uid := string(sys.UniqueIdentifier)
					if uid != "" {
						base.AttachedTo.UID = &uid
					}
				}
			}
		}

		switch procedure.ProcessType {
		case "AggregateProcess":
			ap := AggregateProcess{CommonProcess: base}
			ap.Components = procedure.Components
			ap.Connections = procedure.Connections
			feat := toFeature(ap.CommonProcess)
			feat.Components = ap.Components
			feat.Connections = ap.Connections
			features = append(features, feat)
		case "PhysicalSystem":
			ps := PhysicalSystem{CommonProcess: base}
			ps.Components = procedure.Components
			ps.Connections = procedure.Connections
			feat := toFeature(ps.CommonProcess)
			feat.Components = ps.Components
			feat.Connections = ps.Connections
			features = append(features, feat)
		case "PhysicalComponent":
			pc := PhysicalComponent{CommonProcess: base}
			features = append(features, toFeature(pc.CommonProcess))
		case "SimpleProcess":
			sp := SimpleProcess{CommonProcess: base}
			features = append(features, toFeature(sp.CommonProcess))
		default:
			// unknown type: emit base/common fields only
			features = append(features, toFeature(base))
		}
	}
	return features, nil
}

// --- Deserialization ---

func (f *ProcedureSensorMLFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Procedure, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	sensorML, err := common_shared.DecodeWithFieldErrors[domains.ProcedureSensorMLFeature](body)
	if err != nil {
		return nil, err
	}

	procedure := &domains.Procedure{
		Links: common_shared.StripAssociationLinks(sensorML.Links),
	}

	procedure.UniqueIdentifier = domains.UniqueID(sensorML.UniqueID)

	if sensorML.Label != "" {
		procedure.Name = sensorML.Label
	} else if v, ok := raw["name"].(string); ok {
		procedure.Name = v
	}

	procedure.Description = sensorML.Description

	if sensorML.Definition != "" {
		procedure.ProcedureType = sensorML.Definition
	}
	if sensorML.Type != "" {
		procedure.ProcessType = sensorML.Type
	}

	// Required root fields per OGC CS Part 1 procedure SensorML schema
	if procedure.ProcessType == "" {
		return nil, fmt.Errorf("type is required")
	}
	if procedure.UniqueIdentifier == "" {
		return nil, fmt.Errorf("uniqueId is required")
	}
	if procedure.Name == "" {
		return nil, fmt.Errorf("label is required")
	}
	if procedure.ProcedureType == "" {
		return nil, fmt.Errorf("definition is required")
	}

	procedure.Lang = sensorML.Lang
	procedure.Keywords = sensorML.Keywords
	procedure.Identifiers = sensorML.Identifiers
	procedure.Classifiers = sensorML.Classifiers
	procedure.SecurityConstraints = sensorML.SecurityConstraints
	procedure.LegalConstraints = sensorML.LegalConstraints
	procedure.Characteristics = sensorML.Characteristics
	procedure.Capabilities = sensorML.Capabilities
	procedure.Contacts = sensorML.Contacts
	procedure.Documentation = sensorML.Documentation
	procedure.History = sensorML.History
	procedure.TypeOf = sensorML.TypeOf
	procedure.Configuration = sensorML.Configuration
	procedure.FeaturesOfInterest = sensorML.FeaturesOfInterest
	procedure.Inputs = sensorML.Inputs
	procedure.Outputs = sensorML.Outputs
	procedure.Parameters = sensorML.Parameters
	procedure.Modes = sensorML.Modes
	procedure.Method = sensorML.Method
	procedure.Components = sensorML.Components
	procedure.Connections = sensorML.Connections
	procedure.AttachedTo = sensorML.AttachedTo
	procedure.LocalReferenceFrames = sensorML.LocalReferenceFrames
	procedure.LocalTimeFrames = sensorML.LocalTimeFrames
	procedure.ValidTime = common_shared.NonEmptyTimeRange(sensorML.ValidTime)

	// Handle validTime from raw if not in structured form
	if procedure.ValidTime == nil {
		if vt, ok := raw["validTime"]; ok {
			tr := common_shared.ParseTimeRange(vt)
			procedure.ValidTime = common_shared.NonEmptyTimeRange(&tr)
		}
	}

	return procedure, nil
}
