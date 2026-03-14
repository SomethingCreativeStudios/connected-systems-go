package domains

import (
	"encoding/json"
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// Observation represents one datastream observation (Part 2 dynamic data).
type Observation struct {
	Base

	DatastreamID      string              `gorm:"type:varchar(255);index;not null" json:"datastream@id"`
	SamplingFeatureID *string             `gorm:"type:varchar(255);index" json:"samplingFeature@id,omitempty"`
	ProcedureLink     *common_shared.Link `gorm:"type:jsonb" json:"procedure@link,omitempty"`

	PhenomenonTime *time.Time `json:"phenomenonTime,omitempty"`
	ResultTime     time.Time  `gorm:"index;not null" json:"resultTime"`

	Parameters common_shared.Properties `gorm:"type:jsonb" json:"parameters,omitempty"`
	Result     json.RawMessage          `gorm:"type:jsonb" json:"result,omitempty"`
	ResultLink *common_shared.Link      `gorm:"type:jsonb" json:"result@link,omitempty"`
}

func (Observation) TableName() string {
	return "observations"
}
