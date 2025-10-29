package common_shared

import (
	"database/sql/driver"
	"encoding/json"
)

// Point represents a geographic point (lon, lat)
type Point struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

// Value implements driver.Valuer for PostGIS
func (p Point) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan implements sql.Scanner for PostGIS
func (p *Point) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, p)
}
