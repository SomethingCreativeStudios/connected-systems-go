package domains

import (
	"encoding/json"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// SystemHistoryRevision stores one immutable historical snapshot of a system.
type SystemHistoryRevision struct {
	Base

	SystemID string `gorm:"type:varchar(255);index;not null" json:"system@id"`

	// Snapshot stores the serialized system payload as of this revision.
	Snapshot json.RawMessage `gorm:"type:jsonb;not null" json:"-"`

	// Keep valid_time indexed for history filtering.
	ValidTime *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:valid_time_" json:"validTime,omitempty"`
}

func (SystemHistoryRevision) TableName() string {
	return "system_history_revisions"

}
