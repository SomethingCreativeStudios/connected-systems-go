package json_formatters

import (
	"context"
	"fmt"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// ControlStreamJSONFeature is the wire-format representation of a control stream.
// It adds system@link (derived from SystemID) to the domain model.
type ControlStreamJSONFeature struct {
	domains.ControlStream
	SystemLink *common_shared.Link `json:"system@link,omitempty"`
}

// ControlStreamJSONFormatter handles control stream JSON serialization/deserialization.
type ControlStreamJSONFormatter struct {
	formaters.Formatter[ControlStreamJSONFeature, *domains.ControlStream]
	repos *repository.Repositories
}

func NewControlStreamJSONFormatter(repos *repository.Repositories) *ControlStreamJSONFormatter {
	return &ControlStreamJSONFormatter{repos: repos}
}

func (f *ControlStreamJSONFormatter) ContentType() string {
	return JSONContentType
}

func (f *ControlStreamJSONFormatter) Serialize(ctx context.Context, cs *domains.ControlStream) (ControlStreamJSONFeature, error) {
	if cs == nil {
		return ControlStreamJSONFeature{}, fmt.Errorf("control stream cannot be nil")
	}
	out := ControlStreamJSONFeature{ControlStream: sanitizeControlStreamTimeRanges(*cs)}
	// schema is writeOnly per spec: retrievable only via the schema endpoint.
	out.Schema = nil
	out.Links = appendControlStreamAssociationLinks(cs)
	if cs.SystemID != nil {
		out.SystemLink = &common_shared.Link{
			Href: formaters.ToFunctionalAssociationHref("/systems/" + *cs.SystemID),
			Type: formaters.GeoJSONContentType,
		}
	}
	if out.ProcedureLink != nil && out.ProcedureLink.Href != "" {
		out.ProcedureLink.Href = formaters.ToFunctionalAssociationHref(out.ProcedureLink.Href)
		if out.ProcedureLink.Type == "" {
			out.ProcedureLink.Type = formaters.SensorMLContentType
		}
	}
	if out.DeploymentLink != nil && out.DeploymentLink.Href != "" {
		out.DeploymentLink.Href = formaters.ToFunctionalAssociationHref(out.DeploymentLink.Href)
		if out.DeploymentLink.Type == "" {
			out.DeploymentLink.Type = formaters.GeoJSONContentType
		}
	}
	if out.FeatureOfInterest != nil && out.FeatureOfInterest.Href != "" {
		out.FeatureOfInterest.Href = formaters.ToFunctionalAssociationHref(out.FeatureOfInterest.Href)
		if out.FeatureOfInterest.Type == "" {
			out.FeatureOfInterest.Type = formaters.GeoJSONContentType
		}
	}
	if out.SamplingFeatureLink != nil && out.SamplingFeatureLink.Href != "" {
		out.SamplingFeatureLink.Href = formaters.ToFunctionalAssociationHref(out.SamplingFeatureLink.Href)
		if out.SamplingFeatureLink.Type == "" {
			out.SamplingFeatureLink.Type = formaters.GeoJSONContentType
		}
	}
	return out, nil
}

func (f *ControlStreamJSONFormatter) SerializeAll(ctx context.Context, controlStreams []*domains.ControlStream) ([]ControlStreamJSONFeature, error) {
	if len(controlStreams) == 0 {
		return []ControlStreamJSONFeature{}, nil
	}

	// Collect linked resource IDs for batch enrichment
	systemIDs := make([]string, 0, len(controlStreams))
	procedureIDs := make([]string, 0, len(controlStreams))
	deploymentIDs := make([]string, 0, len(controlStreams))
	sfIDs := make([]string, 0, len(controlStreams))
	for _, cs := range controlStreams {
		if cs == nil {
			continue
		}
		if cs.SystemID != nil && *cs.SystemID != "" {
			systemIDs = append(systemIDs, *cs.SystemID)
		}
		if cs.ProcedureID != nil && *cs.ProcedureID != "" {
			procedureIDs = append(procedureIDs, *cs.ProcedureID)
		}
		if cs.DeploymentID != nil && *cs.DeploymentID != "" {
			deploymentIDs = append(deploymentIDs, *cs.DeploymentID)
		}
		if cs.SamplingFeatureID != nil && *cs.SamplingFeatureID != "" {
			sfIDs = append(sfIDs, *cs.SamplingFeatureID)
		}
	}

	// Build resource cache for enriching inline links
	cache := formaters.NewResourceCache()
	if f.repos != nil {
		if len(systemIDs) > 0 {
			_ = cache.FetchParentSystems(ctx, f.repos.System, systemIDs)
		}
		if len(procedureIDs) > 0 {
			_ = cache.FetchProcedures(ctx, f.repos.Procedure, procedureIDs)
		}
		if len(deploymentIDs) > 0 {
			_ = cache.FetchDeployments(ctx, f.repos.Deployment, deploymentIDs)
		}
		if len(sfIDs) > 0 {
			_ = cache.FetchSamplingFeatures(ctx, f.repos.SamplingFeature, sfIDs)
		}
	}

	items := make([]ControlStreamJSONFeature, 0, len(controlStreams))
	for _, cs := range controlStreams {
		if cs == nil {
			continue
		}
		out := ControlStreamJSONFeature{ControlStream: sanitizeControlStreamTimeRanges(*cs)}
		// schema is writeOnly per spec: retrievable only via the schema endpoint.
		out.Schema = nil
		out.Links = appendControlStreamAssociationLinks(cs)
		if cs.SystemID != nil {
			out.SystemLink = &common_shared.Link{
				Href: formaters.ToFunctionalAssociationHref("/systems/" + *cs.SystemID),
				Type: formaters.GeoJSONContentType,
			}
			// Enrich system@link from cache
			if sys, ok := cache.Systems[*cs.SystemID]; ok {
				out.SystemLink.Title = sys.Name
				uid := string(sys.UniqueIdentifier)
				if uid != "" {
					out.SystemLink.UID = &uid
				}
			}
		}
		if out.ProcedureLink != nil && out.ProcedureLink.Href != "" {
			out.ProcedureLink.Href = formaters.ToFunctionalAssociationHref(out.ProcedureLink.Href)
			if out.ProcedureLink.Type == "" {
				out.ProcedureLink.Type = formaters.SensorMLContentType
			}
			// Enrich procedure@link from cache
			if cs.ProcedureID != nil {
				if proc, ok := cache.Procedures[*cs.ProcedureID]; ok {
					out.ProcedureLink.Title = proc.Name
					uid := string(proc.UniqueIdentifier)
					if uid != "" {
						out.ProcedureLink.UID = &uid
					}
				}
			}
		}
		if out.DeploymentLink != nil && out.DeploymentLink.Href != "" {
			out.DeploymentLink.Href = formaters.ToFunctionalAssociationHref(out.DeploymentLink.Href)
			if out.DeploymentLink.Type == "" {
				out.DeploymentLink.Type = formaters.GeoJSONContentType
			}
			// Enrich deployment@link from cache
			if cs.DeploymentID != nil {
				if dep, ok := cache.Deployments[*cs.DeploymentID]; ok {
					out.DeploymentLink.Title = dep.Name
					uid := string(dep.UniqueIdentifier)
					if uid != "" {
						out.DeploymentLink.UID = &uid
					}
				}
			}
		}
		if out.FeatureOfInterest != nil && out.FeatureOfInterest.Href != "" {
			out.FeatureOfInterest.Href = formaters.ToFunctionalAssociationHref(out.FeatureOfInterest.Href)
			if out.FeatureOfInterest.Type == "" {
				out.FeatureOfInterest.Type = formaters.GeoJSONContentType
			}
		}
		if out.SamplingFeatureLink != nil && out.SamplingFeatureLink.Href != "" {
			out.SamplingFeatureLink.Href = formaters.ToFunctionalAssociationHref(out.SamplingFeatureLink.Href)
			if out.SamplingFeatureLink.Type == "" {
				out.SamplingFeatureLink.Type = formaters.GeoJSONContentType
			}
			// Enrich samplingFeature@link from cache
			if cs.SamplingFeatureID != nil {
				if sf, ok := cache.SamplingFeatures[*cs.SamplingFeatureID]; ok {
					out.SamplingFeatureLink.Title = sf.Name
					uid := string(sf.UniqueIdentifier)
					if uid != "" {
						out.SamplingFeatureLink.UID = &uid
					}
				}
			}
		}
		items = append(items, out)
	}
	return items, nil
}

func (f *ControlStreamJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.ControlStream, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	wire, err := common_shared.DecodeWithFieldErrors[struct {
		domains.ControlStream
		SystemLink *common_shared.Link `json:"system@link,omitempty"`
	}](raw)
	if err != nil {
		return nil, err
	}
	cs := wire.ControlStream
	// Required root fields per OGC CS Part 2 controlStream schema
	if cs.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	cs = sanitizeControlStreamTimeRanges(cs)
	if wire.SystemLink != nil {
		cs.SystemID = wire.SystemLink.GetId("systems")
	}
	return &cs, nil
}

func sanitizeControlStreamTimeRanges(cs domains.ControlStream) domains.ControlStream {
	cs.ValidTime = common_shared.NonEmptyTimeRange(cs.ValidTime)
	cs.IssueTime = common_shared.NonEmptyTimeRange(cs.IssueTime)
	cs.ExecutionTime = common_shared.NonEmptyTimeRange(cs.ExecutionTime)
	cs.Formats = deriveControlStreamFormats(&cs)
	// live and async are required booleans defaulting to false.
	if cs.Live == nil {
		cs.Live = new(bool)
	}
	if cs.Async == nil {
		cs.Async = new(bool)
	}
	return cs
}

// deriveControlStreamFormats computes the read-only, required "formats" field
// from the command formats of all registered schemas.
func deriveControlStreamFormats(cs *domains.ControlStream) common_shared.StringArray {
	var formats common_shared.StringArray
	seen := map[string]bool{}
	add := func(format string) {
		key := domains.NormalizeMediaType(format)
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		formats = append(formats, format)
	}
	for i := range cs.Schemas {
		add(cs.Schemas[i].CommandFormat)
	}
	// Legacy rows predate the schema registry.
	if cs.Schema != nil {
		add(cs.Schema.CommandFormat)
	}
	if len(formats) == 0 {
		formats = common_shared.StringArray{JSONContentType}
	}
	return formats
}

func appendControlStreamAssociationLinks(cs *domains.ControlStream) common_shared.Links {
	links := append(common_shared.Links{}, cs.Links...)

	if cs.ID == "" {
		return links
	}

	commandLink := common_shared.Link{
		Rel:  common_shared.OGCRel("commands"),
		Href: formaters.ToFunctionalAssociationHref("/controlstreams/" + cs.ID + "/commands"),
	}

	for _, link := range links {
		if common_shared.RelEquals(link.Rel, commandLink.Rel) && link.Href == commandLink.Href {
			return links
		}
	}

	return append(links, commandLink)
}
