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

// ControlStreamJSONFormatter handles control stream JSON serialization/deserialization.
type ControlStreamJSONFormatter struct {
	formaters.Formatter[domains.ControlStream, *domains.ControlStream]
}

func NewControlStreamJSONFormatter() *ControlStreamJSONFormatter {
	return &ControlStreamJSONFormatter{}
}

func (f *ControlStreamJSONFormatter) ContentType() string {
	return JSONContentType
}

func (f *ControlStreamJSONFormatter) Serialize(ctx context.Context, cs *domains.ControlStream) (domains.ControlStream, error) {
	if cs == nil {
		return domains.ControlStream{}, fmt.Errorf("control stream cannot be nil")
	}
	out := *cs
	out.Links = appendControlStreamAssociationLinks(cs)
	return out, nil
}

func (f *ControlStreamJSONFormatter) SerializeAll(ctx context.Context, controlStreams []*domains.ControlStream) ([]domains.ControlStream, error) {
	if len(controlStreams) == 0 {
		return []domains.ControlStream{}, nil
	}
	items := make([]domains.ControlStream, 0, len(controlStreams))
	for _, cs := range controlStreams {
		if cs == nil {
			continue
		}
		out := *cs
		out.Links = appendControlStreamAssociationLinks(cs)
		items = append(items, out)
	}
	return items, nil
}

func (f *ControlStreamJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.ControlStream, error) {
	var cs domains.ControlStream
	if err := json.NewDecoder(reader).Decode(&cs); err != nil {
		return nil, err
	}
	return &cs, nil
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
