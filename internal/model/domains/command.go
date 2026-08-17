package domains

import (
	"encoding/json"
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// CommandStatus represents the lifecycle status of a command.
type CommandStatus string

const (
	CommandStatusPending   CommandStatus = "PENDING"
	CommandStatusAccepted  CommandStatus = "ACCEPTED"
	CommandStatusRejected  CommandStatus = "REJECTED"
	CommandStatusScheduled CommandStatus = "SCHEDULED"
	CommandStatusUpdated   CommandStatus = "UPDATED"
	CommandStatusCanceled  CommandStatus = "CANCELED"
	CommandStatusExecuting CommandStatus = "EXECUTING"
	CommandStatusFailed    CommandStatus = "FAILED"
	CommandStatusCompleted CommandStatus = "COMPLETED"
)

// Command represents one command sent through a control stream.
type Command struct {
	Base

	ControlStreamID   string              `gorm:"type:varchar(255);index;not null" json:"controlstream@id"`
	SamplingFeatureID *string             `gorm:"type:varchar(255);index" json:"samplingFeature@id,omitempty"`
	ProcedureLink     *common_shared.Link `gorm:"type:jsonb" json:"procedure@link,omitempty"`

	// issueTime: required in responses; set by server on creation if omitted.
	// executionTime: read-only, populated from command status reports.
	IssueTime     *time.Time               `gorm:"index" json:"issueTime"`
	ExecutionTime *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:execution_time_" json:"executionTime,omitempty"`

	Sender        string        `gorm:"type:varchar(255)" json:"sender,omitempty"`
	CurrentStatus CommandStatus `gorm:"type:varchar(64);default:'PENDING'" json:"currentStatus,omitempty"`

	Parameters json.RawMessage `gorm:"type:jsonb" json:"parameters,omitempty"`
}

func (Command) TableName() string {
	return "commands"
}

// CommandStatusReport represents a status report for a command.
type CommandStatusReport struct {
	Base

	CommandID string `gorm:"type:varchar(255);index;not null" json:"command@id"`

	ReportTime        time.Time                `gorm:"index;not null" json:"reportTime"`
	StatusCode        CommandStatus            `gorm:"type:varchar(64);index;not null" json:"statusCode"`
	PercentCompletion *float64                 `json:"percentCompletion,omitempty"`
	ExecutionTime     *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:execution_time_" json:"executionTime,omitempty"`
	Message           string                   `gorm:"type:text" json:"message,omitempty"`
	Results           json.RawMessage          `gorm:"type:jsonb" json:"results,omitempty"`
}

func (CommandStatusReport) TableName() string {
	return "command_status_reports"
}

// CommandResult represents a result resource produced by a command.
type CommandResult struct {
	Base

	CommandID string `gorm:"type:varchar(255);index;not null" json:"command@id"`

	Data               json.RawMessage `gorm:"type:jsonb" json:"data,omitempty"`
	ObservationLink    json.RawMessage `gorm:"type:jsonb" json:"observation@link,omitempty"`
	ObservationSetLink json.RawMessage `gorm:"type:jsonb" json:"observationSet@link,omitempty"`
	DatastreamLink     json.RawMessage `gorm:"type:jsonb" json:"datastream@link,omitempty"`
	ExternalLink       json.RawMessage `gorm:"type:jsonb" json:"external@link,omitempty"`
}

func (CommandResult) TableName() string {
	return "command_results"
}
