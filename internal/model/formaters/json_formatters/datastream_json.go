package json_formatters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
)

const JSONContentType = "application/json"

// DatastreamJSONFormatter handles datastream JSON serialization/deserialization.
type DatastreamJSONFormatter struct {
	formaters.Formatter[domains.Datastream, *domains.Datastream]
}

func NewDatastreamJSONFormatter() *DatastreamJSONFormatter {
	return &DatastreamJSONFormatter{}
}

func (f *DatastreamJSONFormatter) ContentType() string {
	return JSONContentType
}

func (f *DatastreamJSONFormatter) Serialize(ctx context.Context, datastream *domains.Datastream) (domains.Datastream, error) {
	if datastream == nil {
		return domains.Datastream{}, fmt.Errorf("datastream cannot be nil")
	}
	out := *datastream
	out.Links = appendDatastreamAssociationLinks(datastream)
	return out, nil
}

func (f *DatastreamJSONFormatter) SerializeAll(ctx context.Context, datastreams []*domains.Datastream) ([]domains.Datastream, error) {
	if len(datastreams) == 0 {
		return []domains.Datastream{}, nil
	}

	items := make([]domains.Datastream, 0, len(datastreams))
	for _, ds := range datastreams {
		if ds == nil {
			continue
		}
		out := *ds
		out.Links = appendDatastreamAssociationLinks(ds)
		items = append(items, out)
	}
	return items, nil
}

func (f *DatastreamJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Datastream, error) {
	var datastream domains.Datastream
	if err := json.NewDecoder(reader).Decode(&datastream); err != nil {
		return nil, err
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
