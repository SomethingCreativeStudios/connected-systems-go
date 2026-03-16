package common_shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
)

const OGCRelPrefix = "ogc-rel:"

var associationRels = map[string]struct{}{
	"parentSystem":        {},
	"parentDeployment":    {},
	"subsystems":          {},
	"subdeployments":      {},
	"deployedSystems":     {},
	"samplingFeatures":    {},
	"featuresOfInterest":  {},
	"systemKind":          {},
	"sampleOf":            {},
	"datastreams":         {},
	"controlstreams":      {},
	"observations":        {},
	"commands":            {},
	"procedures":          {},
	"deployments":         {},
	"implementingSystems": {},
	"implementedBy":       {},
	"usedProcedures":      {},
}

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

func (l Links) FilterByRels(rels []string, match bool) Links {
	var filtered Links
	relSet := make(map[string]struct{})
	for _, rel := range rels {
		relSet[CanonicalRel(rel)] = struct{}{}
	}

	for _, link := range l {
		if _, exists := relSet[CanonicalRel(link.Rel)]; exists == match {
			filtered = append(filtered, link)
		}
	}

	return filtered
}

// CanonicalRel normalizes OGC relation aliases (e.g. "ogc-rel:parentSystem" -> "parentSystem").
func CanonicalRel(rel string) string {
	return strings.TrimPrefix(rel, OGCRelPrefix)
}

// OGCRel converts an association rel to the normative OGC form (e.g. "parentSystem" -> "ogc-rel:parentSystem").
func OGCRel(rel string) string {
	if strings.HasPrefix(rel, OGCRelPrefix) {
		return rel
	}
	return OGCRelPrefix + rel
}

// RelEquals compares rel values while treating prefixed and unprefixed OGC rels as equivalent.
func RelEquals(actual, expected string) bool {
	return CanonicalRel(actual) == CanonicalRel(expected)
}

// StripAssociationLinks removes association links while preserving any non-association user links.
func StripAssociationLinks(links Links) Links {
	if len(links) == 0 {
		return nil
	}

	kept := make(Links, 0, len(links))
	for _, link := range links {
		if _, isAssociation := associationRels[CanonicalRel(link.Rel)]; isAssociation {
			continue
		}
		kept = append(kept, link)
	}

	return kept
}
