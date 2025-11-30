package common_shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
)

type Link struct {
	Href  string  `json:"href"`
	Rel   string  `json:"rel,omitempty"`
	Type  string  `json:"type,omitempty"`
	Title string  `json:"title,omitempty"`
	UID   *string `json:"uid,omitempty"`
}

// Value implements driver.Valuer for JSONB storage
func (l Link) Value() (driver.Value, error) {
	return json.Marshal(l)
}

// Scan implements sql.Scanner for JSONB retrieval
func (l *Link) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, l)
}

func (l Link) GetId(basePath string) *string {
	// Build a regex that matches the resource and captures the next segment
	// Anchored to the end to avoid grabbing extra segments.
	pattern := fmt.Sprintf(`(?i)/%s/([^/]+)$`, regexp.QuoteMeta(basePath))
	re := regexp.MustCompile(pattern)

	if m := re.FindStringSubmatch(l.Href); len(m) >= 2 {
		return &m[1]
	}

	// Fallback: return the last path segment
	id := path.Base(l.Href)
	return &id
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

func (l Links) GetIds(basePath string) *[]string {
	var ids []string

	for _, link := range l {
		id := link.GetId(basePath)
		if id != nil {
			ids = append(ids, *id)
		}
	}

	return &ids

}
