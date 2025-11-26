package common_shared

import (
	"database/sql/driver"
	"encoding/json"
)

type CodeList struct {
	CodeSpace string `json:"codeSpace"`
	Value     string `json:"value"`
}

type CodeLists []CodeList

// Value implements driver.Valuer for JSONB storage
func (l CodeLists) Value() (driver.Value, error) {
	return json.Marshal(l)
}

// Scan implements sql.Scanner for JSONB retrieval
func (l *CodeLists) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, l)
}
