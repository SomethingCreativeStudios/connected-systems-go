package json_formatters

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// DatastreamJSONFeature is the application/json representation of a
// datastream. Its fields are listed explicitly so the API shape does not
// follow the database model. Requests may carry the read-only fields, so a
// fetched datastream can be sent back unchanged, but only the writable ones
// are kept (see toDomain).
type DatastreamJSONFeature struct {
	ID                     string                                `json:"id"`
	Name                   string                                `json:"name"`
	Description            string                                `json:"description,omitempty"`
	ValidTime              *common_shared.TimeRange              `json:"validTime,omitempty"`
	Formats                common_shared.StringArray             `json:"formats,omitempty"`
	SystemLink             *common_shared.Link                   `json:"system@link,omitempty"`
	OutputName             string                                `json:"outputName,omitempty"`
	ProcedureLink          *common_shared.Link                   `json:"procedure@link,omitempty"`
	DeploymentLink         *common_shared.Link                   `json:"deployment@link,omitempty"`
	FeatureOfInterest      *common_shared.Link                   `json:"featureOfInterest@link,omitempty"`
	SamplingFeatureLink    *common_shared.Link                   `json:"samplingFeature@link,omitempty"`
	ObservedProperties     *domains.DatastreamObservedProperties `json:"observedProperties"`
	PhenomenonTime         *common_shared.TimeRange              `json:"phenomenonTime"`
	PhenomenonTimeInterval *string                               `json:"phenomenonTimeInterval,omitempty"`
	ResultTime             *common_shared.TimeRange              `json:"resultTime"`
	ResultTimeInterval     *string                               `json:"resultTimeInterval,omitempty"`
	Type                   string                                `json:"type,omitempty"`
	ResultType             *string                               `json:"resultType"`
	Live                   *bool                                 `json:"live"`
	Schema                 *domains.DatastreamSchema             `json:"schema,omitempty"`
	Links                  common_shared.Links                   `json:"links,omitempty"`
}

// toDomain keeps the fields a client may write. The server assigns the id
// and system, and derives formats, observedProperties, resultType and the
// phenomenon and result time extents. ogc-rel links are server-generated.
func (w DatastreamJSONFeature) toDomain() *domains.Datastream {
	datastream := sanitizeDatastreamTimeRanges(domains.Datastream{
		Name:                   w.Name,
		Description:            w.Description,
		ValidTime:              w.ValidTime,
		OutputName:             w.OutputName,
		ProcedureLink:          w.ProcedureLink,
		DeploymentLink:         w.DeploymentLink,
		FeatureOfInterest:      w.FeatureOfInterest,
		SamplingFeatureLink:    w.SamplingFeatureLink,
		PhenomenonTimeInterval: w.PhenomenonTimeInterval,
		ResultTimeInterval:     w.ResultTimeInterval,
		Type:                   w.Type,
		Live:                   w.Live,
		Schema:                 w.Schema,
		Links:                  common_shared.StripAssociationLinks(w.Links),
	})
	return &datastream
}

// DatastreamJSONFormatter handles datastream JSON serialization/deserialization.
type DatastreamJSONFormatter struct {
	links *formaters.LinkResolver
}

var _ formaters.Formatter[DatastreamJSONFeature, *domains.Datastream] = (*DatastreamJSONFormatter)(nil)

func NewDatastreamJSONFormatter(links *formaters.LinkResolver) *DatastreamJSONFormatter {
	return &DatastreamJSONFormatter{links: links}
}

func (f *DatastreamJSONFormatter) ContentType() string {
	return mediatype.JSON
}

func (f *DatastreamJSONFormatter) Serialize(ctx context.Context, datastream *domains.Datastream) (DatastreamJSONFeature, error) {
	return formaters.SerializeOne(ctx, datastream, f.SerializeAll)
}

func (f *DatastreamJSONFormatter) SerializeAll(ctx context.Context, datastreams []*domains.Datastream) ([]DatastreamJSONFeature, error) {
	datastreams = slices.DeleteFunc(slices.Clone(datastreams), func(ds *domains.Datastream) bool { return ds == nil })
	refs := make([]streamRefs, len(datastreams))
	for i, ds := range datastreams {
		refs[i] = streamRefs{
			SystemID:          ds.SystemID,
			ProcedureID:       ds.ProcedureID,
			DeploymentID:      ds.DeploymentID,
			SamplingFeatureID: ds.SamplingFeatureID,
			Procedure:         ds.ProcedureLink,
			Deployment:        ds.DeploymentLink,
			FeatureOfInterest: ds.FeatureOfInterest,
			SamplingFeature:   ds.SamplingFeatureLink,
		}
	}
	cache, err := f.links.Lookup(ctx, streamReferences(refs))
	if err != nil {
		return nil, err
	}

	items := make([]DatastreamJSONFeature, 0, len(datastreams))
	for i, ds := range datastreams {
		derived := sanitizeDatastreamTimeRanges(*ds)
		links := refs[i].resolve(f.links, cache)
		// schema is writeOnly per spec: retrievable only via the schema endpoint.
		items = append(items, DatastreamJSONFeature{
			ID:                     ds.ID,
			Name:                   ds.Name,
			Description:            ds.Description,
			ValidTime:              derived.ValidTime,
			Formats:                derived.Formats,
			SystemLink:             links.System,
			OutputName:             ds.OutputName,
			ProcedureLink:          links.Procedure,
			DeploymentLink:         links.Deployment,
			FeatureOfInterest:      links.FeatureOfInterest,
			SamplingFeatureLink:    links.SamplingFeature,
			ObservedProperties:     derived.ObservedProperties,
			PhenomenonTime:         derived.PhenomenonTime,
			PhenomenonTimeInterval: ds.PhenomenonTimeInterval,
			ResultTime:             derived.ResultTime,
			ResultTimeInterval:     ds.ResultTimeInterval,
			Type:                   ds.Type,
			ResultType:             derived.ResultType,
			Live:                   derived.Live,
			Links:                  f.associationLinks(ds),
		})
	}
	return items, nil
}

func (f *DatastreamJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Datastream, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	wire, err := common_shared.DecodeWithFieldErrors[DatastreamJSONFeature](raw)
	if err != nil {
		return nil, err
	}
	// Required root fields per OGC CS Part 2 dataStream schema
	if wire.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return wire.toDomain(), nil
}

func sanitizeDatastreamTimeRanges(datastream domains.Datastream) domains.Datastream {
	datastream.ValidTime = common_shared.NonEmptyTimeRange(datastream.ValidTime)
	datastream.PhenomenonTime = common_shared.NonEmptyTimeRange(datastream.PhenomenonTime)
	datastream.ResultTime = common_shared.NonEmptyTimeRange(datastream.ResultTime)
	datastream.RecomputeSchemaMetadata()
	// live is a required boolean defaulting to false.
	if datastream.Live == nil {
		datastream.Live = new(bool)
	}
	return datastream
}

func (f *DatastreamJSONFormatter) associationLinks(ds *domains.Datastream) common_shared.Links {
	links := append(common_shared.Links{}, ds.Links...)

	if ds.ID == "" {
		return links
	}

	if ds.SystemID != nil {
		links = append(links, common_shared.Link{
			Rel:  common_shared.OGCRel("systems"),
			Href: f.links.Href("/systems/" + *ds.SystemID),
		})
	}
	return append(links, common_shared.Link{
		Rel:  common_shared.OGCRel("observations"),
		Href: f.links.Href("/datastreams/" + ds.ID + "/observations"),
	})
}
