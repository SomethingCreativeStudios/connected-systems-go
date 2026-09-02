package formaters

import (
	"bytes"
	"context"
	"io"
	"mime"
	"net/url"
	"strings"

	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
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
	formatter   Formatter[Output, Domain]
	contentType string
}

// NewFormatterAdapter creates a new adapter that wraps a typed formatter
func NewFormatterAdapter[Output any, Domain any](formatter Formatter[Output, Domain], contentType string) *FormatterAdapter[Output, Domain] {
	return &FormatterAdapter[Output, Domain]{
		formatter:   formatter,
		contentType: contentType,
	}
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
	return a.contentType
}

// MultiFormatFormatterCollection holds formatters that may return different output types
// for the same domain type. Supports both serialization and deserialization.
type MultiFormatFormatterCollection[Domain any] struct {
	formatters     map[string]AnyFormatter[Domain]
	defaultKey     string
	defaultContent string
	validateBody   func(contentType string, body []byte) error
}

// NewMultiFormatFormatterCollection creates a new multi-format formatter collection
func NewMultiFormatFormatterCollection[Domain any](defaultContentType string) *MultiFormatFormatterCollection[Domain] {
	return &MultiFormatFormatterCollection[Domain]{
		formatters:     make(map[string]AnyFormatter[Domain]),
		defaultKey:     "default",
		defaultContent: defaultContentType,
	}
}

// Register adds a formatter for a specific content type
func (m *MultiFormatFormatterCollection[Domain]) Register(contentType string, formatter AnyFormatter[Domain]) *MultiFormatFormatterCollection[Domain] {
	m.formatters[contentType] = formatter
	return m
}

// RegisterDefault sets the default formatter (used when content type doesn't match)
func (m *MultiFormatFormatterCollection[Domain]) RegisterDefault(formatter AnyFormatter[Domain]) *MultiFormatFormatterCollection[Domain] {
	m.formatters[m.defaultKey] = formatter
	return m
}

// SetRequestValidator attaches a raw-body validation hook. The hook runs
// before deserialization and receives a replayable byte slice, allowing API
// handlers to apply contract validation without consuming the request body.
func (m *MultiFormatFormatterCollection[Domain]) SetRequestValidator(validator func(contentType string, body []byte) error) *MultiFormatFormatterCollection[Domain] {
	m.validateBody = validator
	return m
}

// RegisterTyped is a convenience method to register a typed formatter
func RegisterFormatterTyped[Output any, Domain any](m *MultiFormatFormatterCollection[Domain], contentType string, formatter Formatter[Output, Domain]) *MultiFormatFormatterCollection[Domain] {
	adapter := NewFormatterAdapter(formatter, contentType)
	return m.Register(contentType, adapter)
}

// RegisterTypedDefault is a convenience method to register a typed formatter as default
func RegisterFormatterTypedDefault[Output any, Domain any](m *MultiFormatFormatterCollection[Domain], formatter Formatter[Output, Domain], contentType string) *MultiFormatFormatterCollection[Domain] {
	adapter := NewFormatterAdapter(formatter, contentType)
	return m.RegisterDefault(adapter)
}

// GetFormatter returns the formatter for the given content type
func (m *MultiFormatFormatterCollection[Domain]) GetFormatter(contentType string) AnyFormatter[Domain] {
	contentType = normalizeContentType(contentType)
	if formatter, exists := m.formatters[contentType]; exists {
		return formatter
	}
	return m.formatters[m.defaultKey]
}

// GetResponseContentType returns the content type that will be produced for the given accept header
func (m *MultiFormatFormatterCollection[Domain]) GetResponseContentType(acceptHeader string) string {
	if formatter := m.GetFormatter(acceptHeader); formatter != nil {
		return formatter.ContentType()
	}
	return m.defaultContent
}

// --- Serialization methods ---

// Serialize serializes a single item using the appropriate formatter
func (m *MultiFormatFormatterCollection[Domain]) Serialize(contentType string, item Domain) (any, error) {
	formatter := m.GetFormatter(contentType)
	return formatter.SerializeAny(context.Background(), item)
}

// SerializeAll serializes multiple items using the appropriate formatter
func (m *MultiFormatFormatterCollection[Domain]) SerializeAll(contentType string, items []Domain) ([]any, error) {
	formatter := m.GetFormatter(contentType)
	return formatter.SerializeAllAny(context.Background(), items)
}

// --- Deserialization methods ---

// Deserialize deserializes from a reader using the appropriate formatter
func (m *MultiFormatFormatterCollection[Domain]) Deserialize(contentType string, reader io.Reader) (Domain, error) {
	var zero Domain
	body, err := io.ReadAll(reader)
	if err != nil {
		return zero, err
	}
	formatter := m.GetFormatter(contentType)
	validationContentType := normalizeContentType(contentType)
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
	return formatter.Deserialize(context.Background(), bytes.NewReader(body))
}

func normalizeContentType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && mediaType != "" {
		return strings.ToLower(mediaType)
	}
	return strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
}

// --- Collection building ---

// BuildCollection builds a collection response using the multi-format formatter.
// The envelope shape follows the resolved formatter's content type: GeoJSON
// formats produce a "FeatureCollection" with "features", while SensorML
// (application/sml+json) produces a plain "items" collection per the OGC spec.
func (m *MultiFormatFormatterCollection[Domain]) BuildCollection(
	contentType string,
	items []Domain,
	basePath string,
	total int,
	requestParams url.Values,
	queryParams queryparams.QueryParams,
) any {
	formatter := m.GetFormatter(contentType)

	elements, err := formatter.SerializeAllAny(context.Background(), items)
	if err != nil {
		elements = []any{}
	}

	totalInt := int(total)
	links := queryParams.BuildPaginationLinks(basePath, requestParams)

	if formatter.ContentType() == "application/sml+json" {
		return AnyItemCollection{
			Items: elements,
			Links: links,
		}
	}

	return AnyFeatureCollection{
		Type:           "FeatureCollection",
		Features:       elements,
		NumberMatched:  &totalInt,
		NumberReturned: len(items),
		Links:          links,
	}
}
