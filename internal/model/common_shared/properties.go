package common_shared

import (
	"database/sql/driver"
	"encoding/json"
)

// Properties represents a flexible key-value map for additional properties
type Properties map[string]interface{}

// Value implements driver.Valuer for JSONB storage
func (p Properties) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	return json.Marshal(p)
}

// Scan implements sql.Scanner for JSONB retrieval
func (p *Properties) Scan(value interface{}) error {
	if value == nil {
		*p = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, p)
}
