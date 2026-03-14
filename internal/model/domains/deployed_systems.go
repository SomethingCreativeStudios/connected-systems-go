package domains

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// DeployedSystemItems provides JSONB support for []DeployedSystemItem
type DeployedSystemItems []DeployedSystemItem

// Value implements driver.Valuer for JSONB storage
func (d DeployedSystemItems) Value() (driver.Value, error) {
	if d == nil {
		return []byte("[]"), nil
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Scan implements sql.Scanner for JSONB retrieval
func (d *DeployedSystemItems) Scan(value interface{}) error {
	if value == nil {
		*d = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan type %T into DeployedSystemItems", value)
	}
	return json.Unmarshal(b, d)
}

// GORM hints
func (DeployedSystemItems) GormDataType() string { return "jsonb" }

func (DeployedSystemItems) GormDBDataType(db *gorm.DB, _ *schema.Field) string { return "jsonb" }

// Value implements driver.Valuer for a single DeployedSystemItem
func (d DeployedSystemItem) Value() (driver.Value, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Scan implements sql.Scanner for a single DeployedSystemItem
func (d *DeployedSystemItem) Scan(value interface{}) error {
	if value == nil {
		*d = DeployedSystemItem{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan type %T into DeployedSystemItem", value)
	}
	return json.Unmarshal(b, d)
}
