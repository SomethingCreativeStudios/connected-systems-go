package domains

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// ControlStream represents an OGC Connected Systems control stream resource.
// A control stream is a channel through which commands are sent to a system.
// It mirrors the Datastream resource but for task/control messages.
type ControlStream struct {
	Base
	CommonSSN

	ValidTime *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:valid_time_" json:"validTime,omitempty"`

	// Read-only list of advertised command encodings.
	Formats common_shared.StringArray `gorm:"type:jsonb" json:"formats,omitempty"`

	// Resource links.
	InputName           string              `gorm:"type:varchar(255)" json:"inputName,omitempty"`
	ProcedureLink       *common_shared.Link `gorm:"type:jsonb" json:"procedure@link,omitempty"`
	DeploymentLink      *common_shared.Link `gorm:"type:jsonb" json:"deployment@link,omitempty"`
	FeatureOfInterest   *common_shared.Link `gorm:"type:jsonb" json:"featureOfInterest@link,omitempty"`
	SamplingFeatureLink *common_shared.Link `gorm:"type:jsonb" json:"samplingFeature@link,omitempty"`

	ControlledProperties *ControlStreamControlledProperties `gorm:"type:jsonb" json:"controlledProperties,omitempty"`

	// Read-only time extents derived from commands; required (nullable) per spec.
	IssueTime     *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:issue_time_" json:"issueTime"`
	ExecutionTime *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:execution_time_" json:"executionTime"`

	// Required booleans defaulting to false.
	Live  *bool `gorm:"type:boolean" json:"live"`
	Async *bool `gorm:"type:boolean" json:"async"`

	// Schema describing command parameters. It holds the current (most
	// recently written) schema.
	Schema *ControlStreamSchema `gorm:"type:jsonb" json:"schema,omitempty"`

	// Schemas is the server-managed registry of all schemas for this control
	// stream, one per command format. Maintained via the schema endpoint;
	// never set by clients directly.
	Schemas ControlStreamSchemas `gorm:"type:jsonb" json:"-"`

	// Additional links.
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Normalized IDs for filtering.
	SystemID            *string `gorm:"type:varchar(255);index" json:"-"`
	ProcedureID         *string `gorm:"type:varchar(255);index" json:"-"`
	DeploymentID        *string `gorm:"type:varchar(255);index" json:"-"`
	FeatureOfInterestID *string `gorm:"type:varchar(255);index" json:"-"`
	SamplingFeatureID   *string `gorm:"type:varchar(255);index" json:"-"`

	Systems []System `gorm:"many2many:system_controlstreams;" json:"-"`
}

// TableName specifies the table name.
func (ControlStream) TableName() string {
	return "control_streams"
}

// ControlStreamControlledProperty represents one member of controlledProperties.
type ControlStreamControlledProperty struct {
	Definition  string `json:"definition,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

// ControlStreamControlledProperties is a slice of controlled properties.
type ControlStreamControlledProperties []ControlStreamControlledProperty

// Value implements driver.Valuer for JSONB storage.
func (p ControlStreamControlledProperties) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan implements sql.Scanner for JSONB retrieval.
func (p *ControlStreamControlledProperties) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan type %T into ControlStreamControlledProperties", value)
	}
	return json.Unmarshal(bytes, p)
}

// ControlStreamSchema captures the command schema similar to DatastreamSchema.
type ControlStreamSchema struct {
	CommandFormat string `json:"commandFormat"`

	// JSON encoding branch
	ParametersSchema        *DatastreamDataComponent `json:"parametersSchema,omitempty"`
	ResultSchema            *DatastreamDataComponent `json:"resultSchema,omitempty"`
	FeasibilityResultSchema *DatastreamDataComponent `json:"feasibilityResultSchema,omitempty"`

	// SWE Common branch
	RecordSchema *DatastreamDataComponent `json:"recordSchema,omitempty"`
	Encoding     *DatastreamEncoding      `json:"encoding,omitempty"`
}

// Value implements driver.Valuer for JSONB storage.
func (s ControlStreamSchema) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements sql.Scanner for JSONB retrieval.
func (s *ControlStreamSchema) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan type %T into ControlStreamSchema", value)
	}
	return json.Unmarshal(bytes, s)
}

// ControlStreamSchemas is the per-format schema registry of a control stream.
// Each entry describes commands in one format (keyed by CommandFormat).
type ControlStreamSchemas []ControlStreamSchema

// Upsert replaces the schema registered for the same command format, or
// appends the schema when its format is new. An empty CommandFormat is
// treated as "application/json".
func (s ControlStreamSchemas) Upsert(schema ControlStreamSchema) ControlStreamSchemas {
	if schema.CommandFormat == "" {
		schema.CommandFormat = "application/json"
	}
	format := NormalizeMediaType(schema.CommandFormat)
	for i := range s {
		if NormalizeMediaType(s[i].CommandFormat) == format {
			s[i] = schema
			return s
		}
	}
	return append(s, schema)
}

// ForFormat returns the schema registered for the given command format.
func (s ControlStreamSchemas) ForFormat(format string) *ControlStreamSchema {
	format = NormalizeMediaType(format)
	if format == "" {
		return nil
	}
	for i := range s {
		if NormalizeMediaType(s[i].CommandFormat) == format {
			return &s[i]
		}
	}
	return nil
}

// Value implements driver.Valuer for JSONB storage.
func (s ControlStreamSchemas) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements sql.Scanner for JSONB retrieval.
func (s *ControlStreamSchemas) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan type %T into ControlStreamSchemas", value)
	}
	return json.Unmarshal(bytes, s)
}
