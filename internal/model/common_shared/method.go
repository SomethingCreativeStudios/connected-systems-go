package common_shared

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Method represents the method used in a procedure
type Method struct {
	Algorithm   string `json:"algorithm,omitempty"`
	Description string `json:"description,omitempty"`
}

// Value implementation for Method
func (m Method) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implementation for Method
func (m *Method) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, m)
}
