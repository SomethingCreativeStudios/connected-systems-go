package common_shared

import (
	"database/sql/driver"
	"encoding/json"
)

// Document represents an external document reference as described in the OpenAPI
// schema: contains a role (semantic URI), name, optional description and a
// required Link object that points to the actual document resource.
type Document struct {
	Role        string `json:"role,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Link        Link   `json:"link"`
}

// Documents is a JSONB-friendly slice of Document objects.
type Documents []Document

// Value implements driver.Valuer for storing Documents as JSONB in the DB.
func (d Documents) Value() (driver.Value, error) {
	return json.Marshal(d)
}

// Scan implements sql.Scanner to load Documents from a JSONB column.
func (d *Documents) Scan(value interface{}) error {
	if value == nil {
		*d = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, d)
}

// FirstLink returns the Link of the first document, or nil if none present.
func (d Documents) FirstLink() *Link {
	if len(d) == 0 {
		return nil
	}
	return &d[0].Link
}
