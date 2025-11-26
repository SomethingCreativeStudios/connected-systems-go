package serializers

import (
	"context"
	"net/url"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
)

// AnySerializer is a type-erased serializer interface that returns any
// This allows different serializers to return different output types
type AnySerializer[Input any] interface {
	SerializeAny(ctx context.Context, item Input) (any, error)
	SerializeAllAny(ctx context.Context, items []Input) ([]any, error)
	// ContentType returns the content type this serializer produces
	ContentType() string
}

// SerializerAdapter wraps a typed Serializer to implement AnySerializer
type SerializerAdapter[Output any, Input any] struct {
	serializer  Serializer[Output, Input]
	contentType string
}

// NewSerializerAdapter creates a new adapter that wraps a typed serializer
func NewSerializerAdapter[Output any, Input any](serializer Serializer[Output, Input], contentType string) *SerializerAdapter[Output, Input] {
	return &SerializerAdapter[Output, Input]{
		serializer:  serializer,
		contentType: contentType,
	}
}

func (a *SerializerAdapter[Output, Input]) SerializeAny(ctx context.Context, item Input) (any, error) {
	return a.serializer.Serialize(ctx, item)
}

func (a *SerializerAdapter[Output, Input]) SerializeAllAny(ctx context.Context, items []Input) ([]any, error) {
	results, err := a.serializer.SerializeAll(ctx, items)
	if err != nil {
		return nil, err
	}
	// Convert []Output to []any
	anyResults := make([]any, len(results))
	for i, r := range results {
		anyResults[i] = r
	}
	return anyResults, nil
}

func (a *SerializerAdapter[Output, Input]) ContentType() string {
	return a.contentType
}

// MultiFormatSerializerCollection holds serializers that may return different output types
// for the same input type. Use this when you need to support multiple encodings
// (e.g., SensorML and GeoJSON) for the same domain object.
type MultiFormatSerializerCollection[Input any] struct {
	serializers    map[string]AnySerializer[Input]
	defaultKey     string
	defaultContent string
}

// NewMultiFormatSerializerCollection creates a new multi-format serializer collection
func NewMultiFormatSerializerCollection[Input any](defaultContentType string) *MultiFormatSerializerCollection[Input] {
	return &MultiFormatSerializerCollection[Input]{
		serializers:    make(map[string]AnySerializer[Input]),
		defaultKey:     "default",
		defaultContent: defaultContentType,
	}
}

// Register adds a serializer for a specific content type
func (m *MultiFormatSerializerCollection[Input]) Register(contentType string, serializer AnySerializer[Input]) *MultiFormatSerializerCollection[Input] {
	m.serializers[contentType] = serializer
	return m
}

// RegisterDefault sets the default serializer (used when content type doesn't match)
func (m *MultiFormatSerializerCollection[Input]) RegisterDefault(serializer AnySerializer[Input]) *MultiFormatSerializerCollection[Input] {
	m.serializers[m.defaultKey] = serializer
	return m
}

// RegisterTyped is a convenience method to register a typed serializer
func RegisterTyped[Output any, Input any](m *MultiFormatSerializerCollection[Input], contentType string, serializer Serializer[Output, Input]) *MultiFormatSerializerCollection[Input] {
	adapter := NewSerializerAdapter(serializer, contentType)
	return m.Register(contentType, adapter)
}

// RegisterTypedDefault is a convenience method to register a typed serializer as default
func RegisterTypedDefault[Output any, Input any](m *MultiFormatSerializerCollection[Input], serializer Serializer[Output, Input], contentType string) *MultiFormatSerializerCollection[Input] {
	adapter := NewSerializerAdapter(serializer, contentType)
	return m.RegisterDefault(adapter)
}

// GetSerializer returns the serializer for the given content type
func (m *MultiFormatSerializerCollection[Input]) GetSerializer(contentType string) AnySerializer[Input] {
	if serializer, exists := m.serializers[contentType]; exists {
		return serializer
	}
	return m.serializers[m.defaultKey]
}

// GetResponseContentType returns the content type that will be produced for the given accept header
func (m *MultiFormatSerializerCollection[Input]) GetResponseContentType(acceptHeader string) string {
	if serializer := m.GetSerializer(acceptHeader); serializer != nil {
		return serializer.ContentType()
	}
	return m.defaultContent
}

// Serialize serializes a single item using the appropriate serializer
func (m *MultiFormatSerializerCollection[Input]) Serialize(contentType string, item Input) (any, error) {
	serializer := m.GetSerializer(contentType)
	return serializer.SerializeAny(context.Background(), item)
}

// SerializeAll serializes multiple items using the appropriate serializer
func (m *MultiFormatSerializerCollection[Input]) SerializeAll(contentType string, items []Input) ([]any, error) {
	serializer := m.GetSerializer(contentType)
	return serializer.SerializeAllAny(context.Background(), items)
}

// SerializeWithContext serializes a single item using the appropriate serializer with context
func (m *MultiFormatSerializerCollection[Input]) SerializeWithContext(ctx context.Context, contentType string, item Input) (any, error) {
	serializer := m.GetSerializer(contentType)
	return serializer.SerializeAny(ctx, item)
}

// SerializeAllWithContext serializes multiple items using the appropriate serializer with context
func (m *MultiFormatSerializerCollection[Input]) SerializeAllWithContext(ctx context.Context, contentType string, items []Input) ([]any, error) {
	serializer := m.GetSerializer(contentType)
	return serializer.SerializeAllAny(ctx, items)
}

// AnyFeatureCollection represents a feature collection where the features can be any type
// This is used with MultiFormatSerializerCollection where different formats produce different types
type AnyFeatureCollection struct {
	Type           string              `json:"type"`
	Features       []any               `json:"features"`
	NumberMatched  *int                `json:"numberMatched,omitempty"`
	NumberReturned int                 `json:"numberReturned"`
	Links          common_shared.Links `json:"links"`
}

// BuildCollection builds a feature collection using the multi-format serializer
func (m *MultiFormatSerializerCollection[Input]) BuildCollection(
	contentType string,
	items []Input,
	basePath string,
	total int,
	requestParams url.Values,
	queryParams queryparams.QueryParams,
) AnyFeatureCollection {
	features, err := m.SerializeAll(contentType, items)
	if err != nil {
		features = []any{}
	}

	totalInt := int(total)
	return AnyFeatureCollection{
		Type:           "FeatureCollection",
		Features:       features,
		NumberMatched:  &totalInt,
		NumberReturned: len(items),
		Links:          queryParams.BuildPagintationLinks(basePath, requestParams, &totalInt, len(items)),
	}
}
