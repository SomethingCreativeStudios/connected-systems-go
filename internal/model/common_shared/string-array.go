package common_shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// StringArray allows []string to be stored as jsonb in Postgres while keeping []string in Go.
type StringArray []string

// Value marshals the slice to JSON for database storage.
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	b, err := json.Marshal([]string(s))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Scan unmarshals JSON/byte/string from the DB into the slice.
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("cannot scan type %T into StringArray", value)
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*s = out
	return nil
}

// GORM type declarations so GORM uses jsonb for this custom type.
func (StringArray) GormDataType() string {
	return "jsonb"
}

func (StringArray) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	return "jsonb"
}
