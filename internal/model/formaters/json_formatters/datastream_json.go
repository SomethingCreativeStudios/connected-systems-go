package json_formatters

import (
	"context"
	"fmt"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
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
}

func NewDatastreamJSONFormatter() *DatastreamJSONFormatter {
	return &DatastreamJSONFormatter{}
}

func (f *DatastreamJSONFormatter) ContentType() string {
	return JSONContentType
}

func (f *DatastreamJSONFormatter) Serialize(ctx context.Context, datastream *domains.Datastream) (DatastreamJSONFeature, error) {
	if datastream == nil {
		return DatastreamJSONFeature{}, fmt.Errorf("datastream cannot be nil")
	}
	out := DatastreamJSONFeature{Datastream: *datastream}
	out.Links = appendDatastreamAssociationLinks(datastream)
	if datastream.SystemID != nil {
		out.SystemLink = &common_shared.Link{
			Href: formaters.ToFunctionalAssociationHref("/systems/" + *datastream.SystemID),
		}
	}
	if out.ProcedureLink != nil && out.ProcedureLink.Href != "" {
		out.ProcedureLink.Href = formaters.ToFunctionalAssociationHref(out.ProcedureLink.Href)
	}
	if out.DeploymentLink != nil && out.DeploymentLink.Href != "" {
		out.DeploymentLink.Href = formaters.ToFunctionalAssociationHref(out.DeploymentLink.Href)
	}
	if out.FeatureOfInterest != nil && out.FeatureOfInterest.Href != "" {
		out.FeatureOfInterest.Href = formaters.ToFunctionalAssociationHref(out.FeatureOfInterest.Href)
	}
	if out.SamplingFeatureLink != nil && out.SamplingFeatureLink.Href != "" {
		out.SamplingFeatureLink.Href = formaters.ToFunctionalAssociationHref(out.SamplingFeatureLink.Href)
	}
	return out, nil
}

func (f *DatastreamJSONFormatter) SerializeAll(ctx context.Context, datastreams []*domains.Datastream) ([]DatastreamJSONFeature, error) {
	if len(datastreams) == 0 {
		return []DatastreamJSONFeature{}, nil
	}

	items := make([]DatastreamJSONFeature, 0, len(datastreams))
	for _, ds := range datastreams {
		if ds == nil {
			continue
		}
		out := DatastreamJSONFeature{Datastream: *ds}
		out.Links = appendDatastreamAssociationLinks(ds)
		if ds.SystemID != nil {
			out.SystemLink = &common_shared.Link{
				Href: formaters.ToFunctionalAssociationHref("/systems/" + *ds.SystemID),
			}
		}
		if out.ProcedureLink != nil && out.ProcedureLink.Href != "" {
			out.ProcedureLink.Href = formaters.ToFunctionalAssociationHref(out.ProcedureLink.Href)
		}
		if out.DeploymentLink != nil && out.DeploymentLink.Href != "" {
			out.DeploymentLink.Href = formaters.ToFunctionalAssociationHref(out.DeploymentLink.Href)
		}
		if out.FeatureOfInterest != nil && out.FeatureOfInterest.Href != "" {
			out.FeatureOfInterest.Href = formaters.ToFunctionalAssociationHref(out.FeatureOfInterest.Href)
		}
		if out.SamplingFeatureLink != nil && out.SamplingFeatureLink.Href != "" {
			out.SamplingFeatureLink.Href = formaters.ToFunctionalAssociationHref(out.SamplingFeatureLink.Href)
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
	if wire.SystemLink != nil {
		datastream.SystemID = wire.SystemLink.GetId("systems")
	}
	return &datastream, nil
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
