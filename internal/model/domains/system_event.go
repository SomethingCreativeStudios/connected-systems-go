package domains

import (
	"encoding/json"
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// SystemEvent represents an event attached to a specific system.
type SystemEvent struct {
	Base

	SystemID string `gorm:"type:varchar(255);index;not null" json:"-"`

	Definition    string                          `gorm:"type:text" json:"definition,omitempty"`
	Label         string                          `gorm:"type:varchar(255);not null" json:"label"`
	Description   string                          `gorm:"type:text" json:"description,omitempty"`
	Identifiers   common_shared.Terms             `gorm:"type:jsonb" json:"identifiers,omitempty"`
	Classifiers   common_shared.Terms             `gorm:"type:jsonb" json:"classifiers,omitempty"`
	Contacts      common_shared.ContactWrappers   `gorm:"type:jsonb" json:"contacts,omitempty"`
	Documentation common_shared.Documents         `gorm:"type:jsonb" json:"documentation,omitempty"`
	Time          common_shared.HistoryTime       `gorm:"type:jsonb" json:"time"`
	Properties    common_shared.ComponentWrappers `gorm:"type:jsonb" json:"properties,omitempty"`
	Configuration json.RawMessage                 `gorm:"type:jsonb" json:"configuration,omitempty"`
	Links         common_shared.Links             `gorm:"type:jsonb" json:"links,omitempty"`

	// Normalized range columns used for datetime filtering.
	TimeStart *time.Time `gorm:"index" json:"-"`
	TimeEnd   *time.Time `gorm:"index" json:"-"`
}

func (SystemEvent) TableName() string {
	return "system_events"
}
