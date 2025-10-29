package common_shared

import (
	"database/sql/driver"
	"encoding/json"
)

// Geometry can be any GeoJSON geometry type
type Geometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

// Value implements driver.Valuer for PostGIS
func (g Geometry) Value() (driver.Value, error) {
	return json.Marshal(g)
}

// Scan implements sql.Scanner for PostGIS
func (g *Geometry) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, g)
}
