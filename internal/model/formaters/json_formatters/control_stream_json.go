package json_formatters

import (
	"context"
	"fmt"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
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
}

func NewControlStreamJSONFormatter() *ControlStreamJSONFormatter {
	return &ControlStreamJSONFormatter{}
}

func (f *ControlStreamJSONFormatter) ContentType() string {
	return JSONContentType
}

func (f *ControlStreamJSONFormatter) Serialize(ctx context.Context, cs *domains.ControlStream) (ControlStreamJSONFeature, error) {
	if cs == nil {
		return ControlStreamJSONFeature{}, fmt.Errorf("control stream cannot be nil")
	}
	out := ControlStreamJSONFeature{ControlStream: *cs}
	out.Links = appendControlStreamAssociationLinks(cs)
	if cs.SystemID != nil {
		out.SystemLink = &common_shared.Link{
			Href: formaters.ToFunctionalAssociationHref("/systems/" + *cs.SystemID),
		}
	}
	return out, nil
}

func (f *ControlStreamJSONFormatter) SerializeAll(ctx context.Context, controlStreams []*domains.ControlStream) ([]ControlStreamJSONFeature, error) {
	if len(controlStreams) == 0 {
		return []ControlStreamJSONFeature{}, nil
	}
	items := make([]ControlStreamJSONFeature, 0, len(controlStreams))
	for _, cs := range controlStreams {
		if cs == nil {
			continue
		}
		out := ControlStreamJSONFeature{ControlStream: *cs}
		out.Links = appendControlStreamAssociationLinks(cs)
		if cs.SystemID != nil {
			out.SystemLink = &common_shared.Link{
				Href: formaters.ToFunctionalAssociationHref("/systems/" + *cs.SystemID),
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
	if wire.SystemLink != nil {
		cs.SystemID = wire.SystemLink.GetId("systems")
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
