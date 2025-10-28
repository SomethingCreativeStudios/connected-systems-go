package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base model with common fields
type Base struct {
	ID        string    `gorm:"primaryKey;type:varchar(255)" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
	// DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate hook to auto-generate UUID if ID is empty
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

// UniqueID represents a globally unique identifier (URI)
type UniqueID string

// TimeRange represents a time period with start and end
type TimeRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// Value implements driver.Valuer for JSONB storage
func (tr TimeRange) Value() (driver.Value, error) {
	return json.Marshal(tr)
}

// Scan implements sql.Scanner for JSONB retrieval
func (tr *TimeRange) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, tr)
}

// Link represents a web link (RFC 8288)
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

// Collection represents a paginated collection response
type Collection struct {
	Links          Links       `json:"links"`
	NumberMatched  *int        `json:"numberMatched,omitempty"`
	NumberReturned int         `json:"numberReturned"`
	Features       interface{} `json:"features,omitempty"`
	Items          interface{} `json:"items,omitempty"`
}
