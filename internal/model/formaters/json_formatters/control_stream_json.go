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

// ControlStreamJSONFeature is the application/json representation of a
// control stream. Its fields are listed explicitly so the API shape does not
// follow the database model. Requests may carry the read-only fields, so a
// fetched control stream can be sent back unchanged, but only the writable
// ones are kept (see toDomain).
type ControlStreamJSONFeature struct {
	ID string `json:"id"`
	// uid is not part of the control stream schema; it is kept as an
	// optional extension for clients that already send one.
	UniqueIdentifier     domains.UniqueID                           `json:"uid,omitempty"`
	Name                 string                                     `json:"name"`
	Description          string                                     `json:"description,omitempty"`
	ValidTime            *common_shared.TimeRange                   `json:"validTime,omitempty"`
	Formats              common_shared.StringArray                  `json:"formats,omitempty"`
	SystemLink           *common_shared.Link                        `json:"system@link,omitempty"`
	InputName            string                                     `json:"inputName,omitempty"`
	ProcedureLink        *common_shared.Link                        `json:"procedure@link,omitempty"`
	DeploymentLink       *common_shared.Link                        `json:"deployment@link,omitempty"`
	FeatureOfInterest    *common_shared.Link                        `json:"featureOfInterest@link,omitempty"`
	SamplingFeatureLink  *common_shared.Link                        `json:"samplingFeature@link,omitempty"`
	ControlledProperties *domains.ControlStreamControlledProperties `json:"controlledProperties"`
	IssueTime            *common_shared.TimeRange                   `json:"issueTime"`
	ExecutionTime        *common_shared.TimeRange                   `json:"executionTime"`
	Live                 *bool                                      `json:"live"`
	Async                *bool                                      `json:"async"`
	Schema               *domains.ControlStreamSchema               `json:"schema,omitempty"`
	Links                common_shared.Links                        `json:"links,omitempty"`
}

// toDomain keeps the fields a client may write. The server assigns the id
// and system, and derives formats, controlledProperties and the issue and
// execution time extents. ogc-rel links are server-generated.
func (w ControlStreamJSONFeature) toDomain() *domains.ControlStream {
	cs := sanitizeControlStreamTimeRanges(domains.ControlStream{
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: w.UniqueIdentifier,
			Name:             w.Name,
			Description:      w.Description,
		},
		ValidTime:           w.ValidTime,
		InputName:           w.InputName,
		ProcedureLink:       w.ProcedureLink,
		DeploymentLink:      w.DeploymentLink,
		FeatureOfInterest:   w.FeatureOfInterest,
		SamplingFeatureLink: w.SamplingFeatureLink,
		Live:                w.Live,
		Async:               w.Async,
		Schema:              w.Schema,
		Links:               common_shared.StripAssociationLinks(w.Links),
	})
	return &cs
}

// ControlStreamJSONFormatter handles control stream JSON serialization/deserialization.
type ControlStreamJSONFormatter struct {
	links *formaters.LinkResolver
}

var _ formaters.Formatter[ControlStreamJSONFeature, *domains.ControlStream] = (*ControlStreamJSONFormatter)(nil)

func NewControlStreamJSONFormatter(links *formaters.LinkResolver) *ControlStreamJSONFormatter {
	return &ControlStreamJSONFormatter{links: links}
}

func (f *ControlStreamJSONFormatter) ContentType() string {
	return mediatype.JSON
}

func (f *ControlStreamJSONFormatter) Serialize(ctx context.Context, cs *domains.ControlStream) (ControlStreamJSONFeature, error) {
	return formaters.SerializeOne(ctx, cs, f.SerializeAll)
}

func (f *ControlStreamJSONFormatter) SerializeAll(ctx context.Context, controlStreams []*domains.ControlStream) ([]ControlStreamJSONFeature, error) {
	controlStreams = slices.DeleteFunc(slices.Clone(controlStreams), func(cs *domains.ControlStream) bool { return cs == nil })
	refs := make([]streamRefs, len(controlStreams))
	for i, cs := range controlStreams {
		refs[i] = streamRefs{
			SystemID:          cs.SystemID,
			ProcedureID:       cs.ProcedureID,
			DeploymentID:      cs.DeploymentID,
			SamplingFeatureID: cs.SamplingFeatureID,
			Procedure:         cs.ProcedureLink,
			Deployment:        cs.DeploymentLink,
			FeatureOfInterest: cs.FeatureOfInterest,
			SamplingFeature:   cs.SamplingFeatureLink,
		}
	}
	cache, err := f.links.Lookup(ctx, streamReferences(refs))
	if err != nil {
		return nil, err
	}

	items := make([]ControlStreamJSONFeature, 0, len(controlStreams))
	for i, cs := range controlStreams {
		derived := sanitizeControlStreamTimeRanges(*cs)
		links := refs[i].resolve(f.links, cache)
		// schema is writeOnly per spec: retrievable only via the schema endpoint.
		items = append(items, ControlStreamJSONFeature{
			ID:                   cs.ID,
			UniqueIdentifier:     cs.UniqueIdentifier,
			Name:                 cs.Name,
			Description:          cs.Description,
			ValidTime:            derived.ValidTime,
			Formats:              derived.Formats,
			SystemLink:           links.System,
			InputName:            cs.InputName,
			ProcedureLink:        links.Procedure,
			DeploymentLink:       links.Deployment,
			FeatureOfInterest:    links.FeatureOfInterest,
			SamplingFeatureLink:  links.SamplingFeature,
			ControlledProperties: derived.ControlledProperties,
			IssueTime:            derived.IssueTime,
			ExecutionTime:        derived.ExecutionTime,
			Live:                 derived.Live,
			Async:                derived.Async,
			Links:                f.associationLinks(cs),
		})
	}
	return items, nil
}

func (f *ControlStreamJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.ControlStream, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	wire, err := common_shared.DecodeWithFieldErrors[ControlStreamJSONFeature](raw)
	if err != nil {
		return nil, err
	}
	// Required root fields per OGC CS Part 2 controlStream schema
	if wire.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	return wire.toDomain(), nil
}

func sanitizeControlStreamTimeRanges(cs domains.ControlStream) domains.ControlStream {
	cs.ValidTime = common_shared.NonEmptyTimeRange(cs.ValidTime)
	cs.IssueTime = common_shared.NonEmptyTimeRange(cs.IssueTime)
	cs.ExecutionTime = common_shared.NonEmptyTimeRange(cs.ExecutionTime)
	cs.RecomputeSchemaMetadata()
	// live and async are required booleans defaulting to false.
	if cs.Live == nil {
		cs.Live = new(bool)
	}
	if cs.Async == nil {
		cs.Async = new(bool)
	}
	return cs
}

func (f *ControlStreamJSONFormatter) associationLinks(cs *domains.ControlStream) common_shared.Links {
	links := append(common_shared.Links{}, cs.Links...)

	if cs.ID == "" {
		return links
	}

	commandLink := common_shared.Link{
		Rel:  common_shared.OGCRel("commands"),
		Href: f.links.Href("/controlstreams/" + cs.ID + "/commands"),
	}

	for _, link := range links {
		if common_shared.RelEquals(link.Rel, commandLink.Rel) && link.Href == commandLink.Href {
			return links
		}
	}

	return append(links, commandLink)
}
