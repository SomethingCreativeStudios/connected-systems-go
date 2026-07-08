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

const JSONContentType = "application/json"

// DatastreamJSONFeature is the wire-format representation of a datastream.
// It adds system@link (derived from SystemID) to the domain model.
type DatastreamJSONFeature struct {
	domains.Datastream
	SystemLink *common_shared.Link `json:"system@link,omitempty"`
}

// DatastreamJSONFormatter handles datastream JSON serialization/deserialization.
type DatastreamJSONFormatter struct {
	formaters.Formatter[DatastreamJSONFeature, *domains.Datastream]
	repos *repository.Repositories
}

func NewDatastreamJSONFormatter(repos *repository.Repositories) *DatastreamJSONFormatter {
	return &DatastreamJSONFormatter{repos: repos}
}

func (f *DatastreamJSONFormatter) ContentType() string {
	return JSONContentType
}

func (f *DatastreamJSONFormatter) Serialize(ctx context.Context, datastream *domains.Datastream) (DatastreamJSONFeature, error) {
	if datastream == nil {
		return DatastreamJSONFeature{}, fmt.Errorf("datastream cannot be nil")
	}
	out := DatastreamJSONFeature{Datastream: sanitizeDatastreamTimeRanges(*datastream)}
	out.Links = appendDatastreamAssociationLinks(datastream)
	if datastream.SystemID != nil {
		out.SystemLink = &common_shared.Link{
			Href: formaters.ToFunctionalAssociationHref("/systems/" + *datastream.SystemID),
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

func (f *DatastreamJSONFormatter) SerializeAll(ctx context.Context, datastreams []*domains.Datastream) ([]DatastreamJSONFeature, error) {
	if len(datastreams) == 0 {
		return []DatastreamJSONFeature{}, nil
	}

	// Collect linked resource IDs for batch enrichment
	systemIDs := make([]string, 0, len(datastreams))
	procedureIDs := make([]string, 0, len(datastreams))
	deploymentIDs := make([]string, 0, len(datastreams))
	sfIDs := make([]string, 0, len(datastreams))
	for _, ds := range datastreams {
		if ds == nil {
			continue
		}
		if ds.SystemID != nil && *ds.SystemID != "" {
			systemIDs = append(systemIDs, *ds.SystemID)
		}
		if ds.ProcedureID != nil && *ds.ProcedureID != "" {
			procedureIDs = append(procedureIDs, *ds.ProcedureID)
		}
		if ds.DeploymentID != nil && *ds.DeploymentID != "" {
			deploymentIDs = append(deploymentIDs, *ds.DeploymentID)
		}
		if ds.SamplingFeatureID != nil && *ds.SamplingFeatureID != "" {
			sfIDs = append(sfIDs, *ds.SamplingFeatureID)
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

	items := make([]DatastreamJSONFeature, 0, len(datastreams))
	for _, ds := range datastreams {
		if ds == nil {
			continue
		}
		out := DatastreamJSONFeature{Datastream: sanitizeDatastreamTimeRanges(*ds)}
		out.Links = appendDatastreamAssociationLinks(ds)
		if ds.SystemID != nil {
			out.SystemLink = &common_shared.Link{
				Href: formaters.ToFunctionalAssociationHref("/systems/" + *ds.SystemID),
				Type: formaters.GeoJSONContentType,
			}
			// Enrich system@link from cache
			if sys, ok := cache.Systems[*ds.SystemID]; ok {
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
			if ds.ProcedureID != nil {
				if proc, ok := cache.Procedures[*ds.ProcedureID]; ok {
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
			if ds.DeploymentID != nil {
				if dep, ok := cache.Deployments[*ds.DeploymentID]; ok {
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
			if ds.SamplingFeatureID != nil {
				if sf, ok := cache.SamplingFeatures[*ds.SamplingFeatureID]; ok {
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

func (f *DatastreamJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Datastream, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	wire, err := common_shared.DecodeWithFieldErrors[struct {
		domains.Datastream
		SystemLink *common_shared.Link `json:"system@link,omitempty"`
	}](raw)
	if err != nil {
		return nil, err
	}
	datastream := wire.Datastream
	datastream = sanitizeDatastreamTimeRanges(datastream)
	if wire.SystemLink != nil {
		datastream.SystemID = wire.SystemLink.GetId("systems")
	}
	return &datastream, nil
}

func sanitizeDatastreamTimeRanges(datastream domains.Datastream) domains.Datastream {
	datastream.ValidTime = common_shared.NonEmptyTimeRange(datastream.ValidTime)
	datastream.PhenomenonTime = common_shared.NonEmptyTimeRange(datastream.PhenomenonTime)
	datastream.ResultTime = common_shared.NonEmptyTimeRange(datastream.ResultTime)
	return datastream
}

func appendDatastreamAssociationLinks(ds *domains.Datastream) common_shared.Links {
	links := append(common_shared.Links{}, ds.Links...)

	if ds.ID == "" {
		return links
	}

	observationLink := common_shared.Link{
		Rel:  common_shared.OGCRel("observations"),
		Href: formaters.ToFunctionalAssociationHref("/datastreams/" + ds.ID + "/observations"),
	}

	systemLink := common_shared.Link{
		Rel:  common_shared.OGCRel("systems"),
		Href: formaters.ToFunctionalAssociationHref("/systems/" + *ds.SystemID),
	}

	links = append(links, systemLink)

	return append(links, observationLink)
}
