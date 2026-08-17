package json_formatters

import (
	"context"
	"fmt"
	"io"
	"strings"

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
	// schema is writeOnly per spec: retrievable only via the schema endpoint.
	out.Schema = nil
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
		// schema is writeOnly per spec: retrievable only via the schema endpoint.
		out.Schema = nil
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
	// Required root fields per OGC CS Part 2 dataStream schema
	if datastream.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
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
	datastream.Formats = deriveDatastreamFormats(&datastream)
	datastream.ObservedProperties = deriveDatastreamObservedProperties(&datastream)
	datastream.ResultType = deriveDatastreamResultType(&datastream)
	// live is a required boolean defaulting to false.
	if datastream.Live == nil {
		datastream.Live = new(bool)
	}
	return datastream
}

// definedComponent pairs a schema component carrying a semantic definition
// with its effective name (DatastreamNamedComponent shadows the embedded Name).
type definedComponent struct {
	component *domains.DatastreamDataComponent
	name      string
}

// collectDefinedComponents recursively gathers every data component carrying a
// semantic definition URI from a schema component tree.
func collectDefinedComponents(component *domains.DatastreamDataComponent, name string, out *[]definedComponent) {
	if component == nil {
		return
	}
	if component.Name != "" {
		name = component.Name
	}
	if component.Definition != "" {
		*out = append(*out, definedComponent{component: component, name: name})
	}
	for i := range component.Fields {
		collectDefinedComponents(&component.Fields[i].DatastreamDataComponent, component.Fields[i].Name, out)
	}
	for i := range component.Coordinates {
		collectDefinedComponents(&component.Coordinates[i].DatastreamDataComponent, component.Coordinates[i].Name, out)
	}
	for i := range component.Items {
		collectDefinedComponents(&component.Items[i].DatastreamDataComponent, component.Items[i].Name, out)
	}
	if component.ElementType != nil {
		collectDefinedComponents(&component.ElementType.DatastreamDataComponent, component.ElementType.Name, out)
	}
}

// datastreamResultComponent returns the observation result component of a
// datastream schema (JSON or SWE branch).
func datastreamResultComponent(schema *domains.DatastreamSchema) *domains.DatastreamDataComponent {
	if schema == nil {
		return nil
	}
	if schema.ResultSchema != nil {
		return schema.ResultSchema
	}
	return schema.RecordSchema
}

// deriveDatastreamObservedProperties computes the read-only, required
// "observedProperties" field from the semantic definitions declared in the
// result schemas of all registered schemas. Nil (-> null) when none declared.
func deriveDatastreamObservedProperties(datastream *domains.Datastream) *domains.DatastreamObservedProperties {
	var components []definedComponent
	for i := range datastream.Schemas {
		collectDefinedComponents(datastreamResultComponent(&datastream.Schemas[i]), "", &components)
	}
	// Legacy rows predate the schema registry.
	if len(components) == 0 {
		collectDefinedComponents(datastreamResultComponent(datastream.Schema), "", &components)
	}

	var props domains.DatastreamObservedProperties
	seen := map[string]bool{}
	for _, dc := range components {
		c := dc.component
		if seen[c.Definition] {
			continue
		}
		seen[c.Definition] = true
		label := c.Label
		if label == "" {
			label = dc.name
		}
		props = append(props, domains.DatastreamObservedProperty{
			Definition:  c.Definition,
			Label:       label,
			Description: c.Description,
		})
	}
	if len(props) == 0 {
		return nil
	}
	return &props
}

// deriveDatastreamResultType computes the read-only, required "resultType"
// field from the root component of the observation result schema.
// Nil (-> null) when no schema declares a result component.
func deriveDatastreamResultType(datastream *domains.Datastream) *string {
	component := datastreamResultComponent(datastream.Schema)
	for i := range datastream.Schemas {
		if component != nil {
			break
		}
		component = datastreamResultComponent(&datastream.Schemas[i])
	}
	if component == nil {
		return nil
	}
	var resultType string
	switch strings.ToLower(component.Type) {
	case "quantity", "count":
		resultType = "measure"
	case "vector":
		resultType = "vector"
	case "datarecord":
		resultType = "record"
	case "dataarray", "matrix":
		resultType = "coverage"
	default:
		resultType = "complex"
	}
	return &resultType
}

// deriveDatastreamFormats computes the read-only, required "formats" field
// from the observation formats of all registered schemas.
func deriveDatastreamFormats(datastream *domains.Datastream) common_shared.StringArray {
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
	for i := range datastream.Schemas {
		add(datastream.Schemas[i].ObsFormat)
	}
	// Legacy rows predate the schema registry.
	if datastream.Schema != nil {
		add(datastream.Schema.ObsFormat)
	}
	if len(formats) == 0 {
		formats = common_shared.StringArray{JSONContentType}
	}
	return formats
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
