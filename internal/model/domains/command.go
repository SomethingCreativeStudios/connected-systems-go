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

	// issueTime: set by server on creation if omitted
	IssueTime     *time.Time               `gorm:"index" json:"issueTime,omitempty"`
	ExecutionTime *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:execution_time_" json:"executionTime,omitempty"`

	Sender        string        `gorm:"type:varchar(255)" json:"sender,omitempty"`
	CurrentStatus CommandStatus `gorm:"type:varchar(64);default:'PENDING'" json:"currentStatus,omitempty"`

	Parameters json.RawMessage `gorm:"type:jsonb" json:"parameters,omitempty"`
}

func (Command) TableName() string {
	return "commands"
}
