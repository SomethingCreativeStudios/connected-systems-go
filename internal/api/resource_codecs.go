package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/pubsub"
)

// resourceCodec encodes a committed resource exactly as its HTTP GET handler
// does, so MQTT Resource Data and HTTP responses cannot drift apart.
type resourceCodec interface {
	// ContentTypes lists the representations the codec produces, default first.
	ContentTypes() []string
	Encode(ctx context.Context, links *formaters.LinkResolver, resource any, contentType string) ([]byte, error)
}

// formatterCodec encodes with the formatter collection HTTP GET serves.
type formatterCodec[T any] func(*formaters.LinkResolver) *formaters.MultiFormatFormatterCollection[T]

func (c formatterCodec[T]) ContentTypes() []string { return c(nil).ContentTypes() }

func (c formatterCodec[T]) Encode(ctx context.Context, links *formaters.LinkResolver, resource any, contentType string) ([]byte, error) {
	item, ok := resource.(T)
	if !ok {
		return nil, fmt.Errorf("cannot encode %T as %s", resource, contentType)
	}
	return encodeResource(ctx, c(links), contentType, item)
}

// jsonCodec encodes a resource served only as JSON. shape, when set, prepares
// the value the way the HTTP GET handler does.
type jsonCodec[T any] struct{ shape func(T) T }

func (jsonCodec[T]) ContentTypes() []string { return []string{mediatype.JSON} }

func (c jsonCodec[T]) Encode(_ context.Context, _ *formaters.LinkResolver, resource any, contentType string) ([]byte, error) {
	item, ok := resource.(T)
	if !ok || contentType != mediatype.JSON {
		return nil, fmt.Errorf("cannot encode %T as %s", resource, contentType)
	}
	if c.shape != nil {
		item = c.shape(item)
	}
	return json.Marshal(item)
}

// representer encodes Resource Data for MQTT with each type's codec, in the
// media types the capability catalog advertises.
func representer(links *formaters.LinkResolver) pubsub.Representer {
	return func(ctx context.Context, kind string, resource any) ([]pubsub.Representation, error) {
		c, ok := creatorsByKind[kind]
		if !ok {
			return nil, nil
		}
		media := c.mediaTypes()
		out := make([]pubsub.Representation, 0, len(media))
		for _, contentType := range media {
			payload, err := c.Codec.Encode(ctx, links, resource, contentType)
			if err != nil {
				return nil, err
			}
			out = append(out, pubsub.Representation{MediaType: contentType, Payload: payload})
		}
		return out, nil
	}
}
