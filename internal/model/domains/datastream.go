package domains

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// Datastream represents an OGC Connected Systems datastream resource.
// The model follows the bundled JSON schema shape while using existing
// project conventions for common fields, links, and JSONB persistence.
type Datastream struct {
	Base
	CommonSSN

	ValidTime *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:valid_time_" json:"validTime,omitempty"`

	// Read-only list of advertised observation/result encodings.
	Formats common_shared.StringArray `gorm:"type:jsonb" json:"formats,omitempty"`

	// Resource links expected by the schema.
	SystemLink          *common_shared.Link `gorm:"type:jsonb" json:"system@link,omitempty"`
	OutputName          string              `gorm:"type:varchar(255)" json:"outputName,omitempty"`
	ProcedureLink       *common_shared.Link `gorm:"type:jsonb" json:"procedure@link,omitempty"`
	DeploymentLink      *common_shared.Link `gorm:"type:jsonb" json:"deployment@link,omitempty"`
	FeatureOfInterest   *common_shared.Link `gorm:"type:jsonb" json:"featureOfInterest@link,omitempty"`
	SamplingFeatureLink *common_shared.Link `gorm:"type:jsonb" json:"samplingFeature@link,omitempty"`

	ObservedProperties *DatastreamObservedProperties `gorm:"type:jsonb" json:"observedProperties,omitempty"`

	PhenomenonTime         *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:phenomenon_time_" json:"phenomenonTime,omitempty"`
	PhenomenonTimeInterval *string                  `gorm:"type:varchar(64)" json:"phenomenonTimeInterval,omitempty"`
	ResultTime             *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:result_time_" json:"resultTime,omitempty"`
	ResultTimeInterval     *string                  `gorm:"type:varchar(64)" json:"resultTimeInterval,omitempty"`

	Type       string  `gorm:"type:varchar(32)" json:"type,omitempty"`
	ResultType *string `gorm:"type:varchar(32)" json:"resultType,omitempty"`
	Live       *bool   `gorm:"type:boolean" json:"live,omitempty"`

	// Schema is intentionally flexible because it supports multiple
	// encoding families (JSON, SWE, protobuf, vendor-specific).
	Schema *DatastreamSchema `gorm:"type:jsonb" json:"schema,omitempty"`

	// Additional links (schema field "links").
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Optional normalized IDs retained server-side for easier joins/filtering.
	SystemID            *string `gorm:"type:varchar(255);index" json:"-"`
	ProcedureID         *string `gorm:"type:varchar(255);index" json:"-"`
	DeploymentID        *string `gorm:"type:varchar(255);index" json:"-"`
	FeatureOfInterestID *string `gorm:"type:varchar(255);index" json:"-"`
	SamplingFeatureID   *string `gorm:"type:varchar(255);index" json:"-"`
}

// TableName specifies the table name.
func (Datastream) TableName() string {
	return "datastreams"
}

// DatastreamType constants.
const (
	DatastreamTypeStatus      = "status"
	DatastreamTypeObservation = "observation"
)

// DatastreamResultType constants.
const (
	DatastreamResultTypeMeasure  = "measure"
	DatastreamResultTypeVector   = "vector"
	DatastreamResultTypeRecord   = "record"
	DatastreamResultTypeCoverage = "coverage"
	DatastreamResultTypeComplex  = "complex"
)

// DatastreamObservedProperty represents one member of observedProperties.
type DatastreamObservedProperty struct {
	Definition  string `json:"definition,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

// DatastreamObservedProperties is nullable in the schema; pointer usage in
// Datastream supports omitting/null as needed.
type DatastreamObservedProperties []DatastreamObservedProperty

// DatastreamSchema captures the common schema descriptors while allowing
// unmodeled vendor- or encoding-specific extensions.
type DatastreamSchema struct {
	ObsFormat string `json:"obsFormat"`

	// JSON encoding branch
	ParametersSchema *DatastreamDataComponent `json:"parametersSchema,omitempty"`
	ResultSchema     *DatastreamDataComponent `json:"resultSchema,omitempty"`

	ResultLink *DatastreamResultLink `json:"resultLink,omitempty"`

	// SWE branch
	RecordSchema *DatastreamDataComponent `json:"recordSchema,omitempty"`
	Encoding     *DatastreamEncoding      `json:"encoding,omitempty"`

	// Protobuf branch
	MessageSchema *DatastreamMessageSchema `json:"messageSchema,omitempty"`

	// Other-format branch
	Any common_shared.Properties `json:"any,omitempty"`
}

// DatastreamResultLink describes out-of-band result link media type info.
type DatastreamResultLink struct {
	MediaType string `json:"mediaType"`
}

// Value implements driver.Valuer for JSONB storage.
func (s DatastreamSchema) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements sql.Scanner for JSONB retrieval.
func (s *DatastreamSchema) Scan(value interface{}) error {
	if value == nil {
		*s = DatastreamSchema{}
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan type %T into DatastreamSchema", value)
	}

	return json.Unmarshal(b, s)
}

// DatastreamMessageSchema supports the schema oneOf for protobuf messageSchema:
// either an inline schema string or a link object.
type DatastreamMessageSchema struct {
	Inline *string             `json:"-"`
	Link   *common_shared.Link `json:"-"`
}

func (m DatastreamMessageSchema) MarshalJSON() ([]byte, error) {
	if m.Link != nil {
		return json.Marshal(m.Link)
	}
	if m.Inline != nil {
		return json.Marshal(*m.Inline)
	}
	return []byte("null"), nil
}

func (m *DatastreamMessageSchema) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		m.Inline = nil
		m.Link = nil
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		m.Inline = &asString
		m.Link = nil
		return nil
	}

	var asLink common_shared.Link
	if err := json.Unmarshal(data, &asLink); err == nil {
		m.Link = &asLink
		m.Inline = nil
		return nil
	}

	return fmt.Errorf("unsupported messageSchema payload")
}

// DatastreamEncoding maps encoding definitions used by SWE/JSON branches.
type DatastreamEncoding struct {
	Type string `json:"type,omitempty"`

	// Text/CSV encoding knobs
	CollapseWhiteSpaces *bool  `json:"collapseWhiteSpaces,omitempty"`
	DecimalSeparator    string `json:"decimalSeparator,omitempty"`
	TokenSeparator      string `json:"tokenSeparator,omitempty"`
	BlockSeparator      string `json:"blockSeparator,omitempty"`

	// SWE JSON encoding knobs
	RecordsAsArrays *bool `json:"recordsAsArrays,omitempty"`
	VectorsAsArrays *bool `json:"vectorsAsArrays,omitempty"`

	// Binary encoding knobs
	ByteOrder    string                   `json:"byteOrder,omitempty"`
	ByteEncoding string                   `json:"byteEncoding,omitempty"`
	ByteLength   *int                     `json:"byteLength,omitempty"`
	Members      []DatastreamBinaryMember `json:"members,omitempty"`

	// Custom extension values
	Extensions common_shared.Properties `json:"extensions,omitempty"`
}

// DatastreamBinaryMember captures per-member options in binary encodings.
type DatastreamBinaryMember struct {
	Ref         string                   `json:"ref,omitempty"`
	Compression string                   `json:"compression,omitempty"`
	Encryption  string                   `json:"encryption,omitempty"`
	DataType    string                   `json:"dataType,omitempty"`
	ByteLength  *int                     `json:"byteLength,omitempty"`
	ByteOrder   string                   `json:"byteOrder,omitempty"`
	Extensions  common_shared.Properties `json:"extensions,omitempty"`
}

// DatastreamDataComponent is a strongly-typed recursive representation of SWE
// component trees (scalars, records, vectors, arrays, choices, geometry).
type DatastreamDataComponent struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Definition  string `json:"definition,omitempty"`

	Updatable *bool `json:"updatable,omitempty"`
	Optional  *bool `json:"optional,omitempty"`

	ReferenceFrame string `json:"referenceFrame,omitempty"`
	LocalFrame     string `json:"localFrame,omitempty"`
	AxisID         string `json:"axisID,omitempty"`
	CodeSpace      string `json:"codeSpace,omitempty"`

	UOM *DatastreamUOM `json:"uom,omitempty"`

	Constraint *DatastreamConstraint `json:"constraint,omitempty"`
	NilValues  []DatastreamNilValue  `json:"nilValues,omitempty"`

	// Scalar/inline value payload.
	Value json.RawMessage `json:"value,omitempty"`

	// Record/vector/choice structures.
	Fields      []DatastreamNamedComponent `json:"fields,omitempty"`
	Coordinates []DatastreamNamedComponent `json:"coordinates,omitempty"`
	Items       []DatastreamNamedComponent `json:"items,omitempty"`

	// Array/matrix structures.
	ElementCount *DatastreamElementCount   `json:"elementCount,omitempty"`
	ElementType  *DatastreamNamedComponent `json:"elementType,omitempty"`
	Encoding     *DatastreamEncoding       `json:"encoding,omitempty"`
	Values       json.RawMessage           `json:"values,omitempty"`

	// DataChoice selector.
	ChoiceValue *DatastreamDataComponent `json:"choiceValue,omitempty"`

	// Geometry-specific fields.
	SRS string `json:"srs,omitempty"`

	// Extension values for vendor-specific attributes.
	Extensions common_shared.Properties `json:"extensions,omitempty"`
}

// DatastreamNamedComponent is used where schema members require a "name"
// plus a nested component definition.
type DatastreamNamedComponent struct {
	Name string `json:"name"`
	DatastreamDataComponent
}

// DatastreamElementCount supports integer literal or component-based counters.
type DatastreamElementCount struct {
	Fixed     *int                     `json:"fixed,omitempty"`
	Component *DatastreamDataComponent `json:"component,omitempty"`
}

func (ec DatastreamElementCount) MarshalJSON() ([]byte, error) {
	if ec.Component != nil {
		return json.Marshal(ec.Component)
	}
	if ec.Fixed != nil {
		return json.Marshal(*ec.Fixed)
	}
	return []byte("null"), nil
}

func (ec *DatastreamElementCount) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		ec.Fixed = nil
		ec.Component = nil
		return nil
	}

	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		ec.Fixed = &n
		ec.Component = nil
		return nil
	}

	var c DatastreamDataComponent
	if err := json.Unmarshal(data, &c); err == nil {
		ec.Component = &c
		ec.Fixed = nil
		return nil
	}

	return fmt.Errorf("unsupported elementCount payload")
}

// DatastreamUOM maps SWE unit definitions.
type DatastreamUOM struct {
	Label  string `json:"label,omitempty"`
	Symbol string `json:"symbol,omitempty"`
	Code   string `json:"code,omitempty"`
	Href   string `json:"href,omitempty"`
}

// DatastreamConstraint maps common SWE constraint shapes.
type DatastreamConstraint struct {
	Type string `json:"type,omitempty"`

	Values             json.RawMessage `json:"values,omitempty"`
	Intervals          json.RawMessage `json:"intervals,omitempty"`
	Pattern            string          `json:"pattern,omitempty"`
	SignificantFigures *int            `json:"significantFigures,omitempty"`

	Extensions common_shared.Properties `json:"extensions,omitempty"`
}

// DatastreamNilValue maps reserved nil value entries.
type DatastreamNilValue struct {
	Reason string          `json:"reason"`
	Value  json.RawMessage `json:"value"`
}

// Value implements driver.Valuer for JSONB storage.
func (o DatastreamObservedProperties) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Scan implements sql.Scanner for JSONB retrieval.
func (o *DatastreamObservedProperties) Scan(value interface{}) error {
	if value == nil {
		*o = nil
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan type %T into DatastreamObservedProperties", value)
	}

	return json.Unmarshal(b, o)
}
