package formaters

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
)

type Serializer[Output any, Input any] interface {
	Serialize(ctx context.Context, item Input) (Output, error)
	SerializeAll(ctx context.Context, items []Input) ([]Output, error)
}

// Deserializer converts wire format data into domain objects
type Deserializer[Output any] interface {
	// Deserialize reads from a reader and produces the domain object
	Deserialize(ctx context.Context, reader io.Reader) (Output, error)
	// ContentType returns the content type this deserializer handles
	ContentType() string
}

// Formatter combines serialization and deserialization for a specific format
// Output is the wire format type (e.g., SystemGeoJSONFeature)
// Domain is the internal domain type (e.g., *System)
type Formatter[Output any, Domain any] interface {
	// Serialize converts domain object to wire format
	Serialize(ctx context.Context, item Domain) (Output, error)
	// SerializeAll converts multiple domain objects to wire format
	SerializeAll(ctx context.Context, items []Domain) ([]Output, error)
	// Deserialize converts wire format to domain object
	Deserialize(ctx context.Context, reader io.Reader) (Domain, error)
	// ContentType returns the content type this formatter handles
	ContentType() string
}

// SerializeOne formats a single resource exactly as it would appear in a
// collection, so a resource and its collection entry never drift apart.
func SerializeOne[Output any, T any](ctx context.Context, item *T, serializeAll func(context.Context, []*T) ([]Output, error)) (Output, error) {
	var zero Output
	if item == nil {
		return zero, errors.New("cannot serialize a nil resource")
	}
	items, err := serializeAll(ctx, []*T{item})
	if err != nil {
		return zero, err
	}
	if len(items) != 1 {
		return zero, fmt.Errorf("serializing one resource produced %d representations", len(items))
	}
	return items[0], nil
}

// AnyFormatter is a type-erased formatter interface
// This allows different formatters to return different output types while
// being stored in the same collection
type AnyFormatter[Domain any] interface {
	// SerializeAny converts domain object to wire format (type-erased)
	SerializeAny(ctx context.Context, item Domain) (any, error)
	// SerializeAllAny converts multiple domain objects to wire format (type-erased)
	SerializeAllAny(ctx context.Context, items []Domain) ([]any, error)
	// Deserialize converts wire format to domain object
	Deserialize(ctx context.Context, reader io.Reader) (Domain, error)
	// ContentType returns the content type this formatter handles
	ContentType() string
}

// FormatterAdapter wraps a typed Formatter to implement AnyFormatter
type FormatterAdapter[Output any, Domain any] struct {
	formatter Formatter[Output, Domain]
}

// Adapt erases a typed formatter's wire type so that formatters of one domain
// type with different wire types can share a collection.
func Adapt[Output any, Domain any](formatter Formatter[Output, Domain]) AnyFormatter[Domain] {
	return &FormatterAdapter[Output, Domain]{formatter: formatter}
}

func (a *FormatterAdapter[Output, Domain]) SerializeAny(ctx context.Context, item Domain) (any, error) {
	return a.formatter.Serialize(ctx, item)
}

func (a *FormatterAdapter[Output, Domain]) SerializeAllAny(ctx context.Context, items []Domain) ([]any, error) {
	results, err := a.formatter.SerializeAll(ctx, items)
	if err != nil {
		return nil, err
	}
	anyResults := make([]any, len(results))
	for i, r := range results {
		anyResults[i] = r
	}
	return anyResults, nil
}

func (a *FormatterAdapter[Output, Domain]) Deserialize(ctx context.Context, reader io.Reader) (Domain, error) {
	return a.formatter.Deserialize(ctx, reader)
}

func (a *FormatterAdapter[Output, Domain]) ContentType() string {
	return a.formatter.ContentType()
}

// MultiFormatFormatterCollection holds formatters that may return different output types
// for the same domain type. Supports both serialization and deserialization.
type MultiFormatFormatterCollection[Domain any] struct {
	formatters       map[string]AnyFormatter[Domain]
	defaultFormatter AnyFormatter[Domain]
	validateBody     func(contentType string, body []byte) error
}

// NewFormatterCollection registers each formatter under its own content type.
// The first one is the default representation, used when a request names no
// supported media type.
func NewFormatterCollection[Domain any](defaultFormatter AnyFormatter[Domain], others ...AnyFormatter[Domain]) *MultiFormatFormatterCollection[Domain] {
	collection := &MultiFormatFormatterCollection[Domain]{
		formatters:       make(map[string]AnyFormatter[Domain], 1+len(others)),
		defaultFormatter: defaultFormatter,
	}
	for _, formatter := range append([]AnyFormatter[Domain]{defaultFormatter}, others...) {
		collection.formatters[formatter.ContentType()] = formatter
	}
	return collection
}

// SetRequestValidator attaches a raw-body validation hook. The hook runs
// before deserialization and receives a replayable byte slice, allowing API
// handlers to apply contract validation without consuming the request body.
func (m *MultiFormatFormatterCollection[Domain]) SetRequestValidator(validator func(contentType string, body []byte) error) *MultiFormatFormatterCollection[Domain] {
	m.validateBody = validator
	return m
}

// GetFormatter returns the formatter for the given content type, or the
// default one when the content type is not supported.
func (m *MultiFormatFormatterCollection[Domain]) GetFormatter(contentType string) AnyFormatter[Domain] {
	if formatter, exists := m.formatters[mediatype.Normalize(contentType)]; exists {
		return formatter
	}
	return m.defaultFormatter
}

// Negotiate scores each representation using its most specific matching range.
// A specific q=0 exclusion cannot be overridden by a wildcard or JSON alias.
func (m *MultiFormatFormatterCollection[Domain]) Negotiate(accepted []mediatype.Range) (string, bool) {
	if len(accepted) == 0 {
		return m.defaultFormatter.ContentType(), true
	}
	bestType, bestQ, bestSpecificity, bestOrder := "", 0.0, -2, len(accepted)
	for _, contentType := range m.ContentTypes() {
		q, specificity, order := 0.0, -2, len(accepted)
		for i, r := range accepted {
			match := -2
			switch {
			case r.Type == contentType:
				match = 2
			case r.Type == strings.SplitN(contentType, "/", 2)[0]+"/*":
				match = 1
			case r.Type == "*/*":
				match = 0
			case r.Type == mediatype.JSON && contentType == m.defaultFormatter.ContentType():
				// Keep the existing plain-JSON alias below actual media ranges.
				match = -1
			}
			if match > specificity {
				q, specificity, order = r.Q, match, i
			}
		}
		if q > bestQ || (q > 0 && q == bestQ && (specificity > bestSpecificity || (specificity == bestSpecificity && order < bestOrder))) {
			bestType, bestQ, bestSpecificity, bestOrder = contentType, q, specificity, order
		}
	}
	return bestType, bestQ > 0
}

// ContentTypes lists the representations the collection produces, the
// default first.
func (m *MultiFormatFormatterCollection[Domain]) ContentTypes() []string {
	types := []string{m.defaultFormatter.ContentType()}
	for contentType := range m.formatters {
		if contentType != types[0] {
			types = append(types, contentType)
		}
	}
	slices.Sort(types[1:])
	return types
}

// --- Serialization methods ---

// Serialize serializes a single item using the appropriate formatter
func (m *MultiFormatFormatterCollection[Domain]) Serialize(contentType string, item Domain) (any, error) {
	return m.SerializeContext(context.Background(), contentType, item)
}

func (m *MultiFormatFormatterCollection[Domain]) SerializeContext(ctx context.Context, contentType string, item Domain) (any, error) {
	formatter := m.GetFormatter(contentType)
	return formatter.SerializeAny(ctx, item)
}

// SerializeAll serializes multiple items using the appropriate formatter
func (m *MultiFormatFormatterCollection[Domain]) SerializeAll(contentType string, items []Domain) ([]any, error) {
	return m.SerializeAllContext(context.Background(), contentType, items)
}

func (m *MultiFormatFormatterCollection[Domain]) SerializeAllContext(ctx context.Context, contentType string, items []Domain) ([]any, error) {
	formatter := m.GetFormatter(contentType)
	return formatter.SerializeAllAny(ctx, items)
}

// --- Deserialization methods ---

// Deserialize deserializes from a reader using the appropriate formatter
func (m *MultiFormatFormatterCollection[Domain]) Deserialize(contentType string, reader io.Reader) (Domain, error) {
	return m.DeserializeContext(context.Background(), contentType, reader)
}

func (m *MultiFormatFormatterCollection[Domain]) DeserializeContext(ctx context.Context, contentType string, reader io.Reader) (Domain, error) {
	var zero Domain
	body, err := io.ReadAll(reader)
	if err != nil {
		return zero, err
	}
	formatter := m.GetFormatter(contentType)
	validationContentType := mediatype.Normalize(contentType)
	if _, registered := m.formatters[validationContentType]; !registered {
		// The formatter falls back to its default representation for a missing
		// or unsupported Content-Type, so validate that resolved representation
		// too. In particular, a Procedure cannot bypass its SensorML-only write
		// rule by omitting Content-Type.
		validationContentType = formatter.ContentType()
	}
	if m.validateBody != nil {
		if err := m.validateBody(validationContentType, body); err != nil {
			return zero, err
		}
	}
	return formatter.Deserialize(ctx, bytes.NewReader(body))
}

// --- Collection building ---

// BuildCollection builds a collection response using the multi-format formatter.
// The envelope shape follows the resolved formatter's content type: GeoJSON
// produces a "FeatureCollection" with "features"; SensorML and Part 2 JSON
// produce a plain "items" collection per the OGC spec.
func (m *MultiFormatFormatterCollection[Domain]) BuildCollection(
	contentType string,
	items []Domain,
	basePath string,
	total int,
	requestParams url.Values,
	queryParams queryparams.QueryParams,
) (any, error) {
	return m.BuildCollectionContext(context.Background(), contentType, items, basePath, total, requestParams, queryParams)
}

func (m *MultiFormatFormatterCollection[Domain]) BuildCollectionContext(
	ctx context.Context,
	contentType string,
	items []Domain,
	basePath string,
	total int,
	requestParams url.Values,
	queryParams queryparams.QueryParams,
) (any, error) {
	formatter := m.GetFormatter(contentType)

	elements, err := formatter.SerializeAllAny(ctx, items)
	if err != nil {
		return nil, err
	}

	links := queryParams.BuildPaginationLinks(basePath, requestParams)

	// Only GeoJSON has a FeatureCollection; SensorML and Part 2 JSON use the
	// plain items envelope.
	if formatter.ContentType() != mediatype.GeoJSON {
		return AnyItemCollection{
			Items: elements,
			Links: links,
		}, nil
	}

	return AnyFeatureCollection{
		Type:           "FeatureCollection",
		Features:       elements,
		NumberMatched:  &total,
		NumberReturned: len(items),
		Links:          links,
	}, nil
}
