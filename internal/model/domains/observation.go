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
	// TimescaleDB requires every primary or unique key on a hypertable to
	// include its partitioning column. Base.ID supplies the first key column;
	// resultTime is the time partition and second key column.
	ResultTime time.Time `gorm:"primaryKey;not null" json:"resultTime"`

	Parameters common_shared.Properties `gorm:"type:jsonb" json:"parameters,omitempty"`
	Result     json.RawMessage          `gorm:"type:jsonb" json:"result,omitempty"`
	ResultLink *common_shared.Link      `gorm:"type:jsonb" json:"result@link,omitempty"`
}

func (Observation) TableName() string {
	return "observations"
}
