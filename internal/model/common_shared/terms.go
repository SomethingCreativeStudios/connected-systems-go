package common_shared

import (
	"database/sql/driver"
	"encoding/json"
)

type Term struct {
	Definition string `json:"definition"`
	Label      string `json:"label"`
	CodeSpace  string `json:"codeSpace"`
	Value      string `json:"value"`
}

// Links is a collection of Link objects
type Terms []Term

// Value implements driver.Valuer for JSONB storage
func (l Terms) Value() (driver.Value, error) {
	return json.Marshal(l)
}

// Scan implements sql.Scanner for JSONB retrieval
func (l *Terms) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, l)
}
