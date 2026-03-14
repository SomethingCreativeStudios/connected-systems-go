package json_formatters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

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
	return *datastream, nil
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
		items = append(items, *ds)
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
