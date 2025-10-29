package common_shared

import (
	"database/sql/driver"
	"encoding/json"
)

type Link struct {
	Href  string `json:"href"`
	Rel   string `json:"rel,omitempty"`
	Type  string `json:"type,omitempty"`
	Title string `json:"title,omitempty"`
}

// Links is a collection of Link objects
type Links []Link

// Value implements driver.Valuer for JSONB storage
func (l Links) Value() (driver.Value, error) {
	return json.Marshal(l)
}

// Scan implements sql.Scanner for JSONB retrieval
func (l *Links) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, l)
}
