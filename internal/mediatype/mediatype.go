// Package mediatype names the representations the API serves and accepts.
package mediatype

import (
	"cmp"
	"mime"
	"slices"
	"strconv"
	"strings"
)

const (
	// GeoJSON is the Part 1 feature representation.
	GeoJSON = "application/geo+json"
	// SensorML is the Part 1 SensorML JSON representation.
	SensorML = "application/sml+json"
	// JSON is the Part 2 (dynamic data) representation.
	JSON = "application/json"
)

// Normalize reduces a Content-Type value to its lower-case media type,
// dropping parameters such as charset.
func Normalize(value string) string {
	mediaType, _, err := mime.ParseMediaType(value)
	if err == nil && mediaType != "" {
		return strings.ToLower(mediaType)
	}
	before, _, _ := strings.Cut(value, ";")
	return strings.ToLower(strings.TrimSpace(before))
}

// Range is one media range of an Accept header with its quality value.
type Range struct {
	Type string
	Q    float64
}

// IsFallback reports whether the range accepts the default representation:
// a wildcard, or plain JSON, which every representation here is.
func (r Range) IsFallback() bool {
	return r.Type == "*/*" || r.Type == "application/*" || r.Type == JSON
}

// ParseAccept returns the media ranges of an Accept header, most preferred
// first; ranges of equal quality keep their order. Zero-quality ranges are
// retained because a specific exclusion takes precedence over a wildcard.
// A missing or malformed q counts as 1.
func ParseAccept(header string) []Range {
	var ranges []Range
	for part := range strings.SplitSeq(header, ",") {
		mediaType, params, _ := strings.Cut(part, ";")
		mediaType = strings.ToLower(strings.TrimSpace(mediaType))
		if mediaType == "" {
			continue
		}
		q := 1.0
		for param := range strings.SplitSeq(params, ";") {
			name, value, _ := strings.Cut(param, "=")
			if strings.EqualFold(strings.TrimSpace(name), "q") {
				if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && parsed >= 0 && parsed <= 1 {
					q = parsed
				}
			}
		}
		ranges = append(ranges, Range{Type: mediaType, Q: q})
	}
	slices.SortStableFunc(ranges, func(a, b Range) int { return cmp.Compare(b.Q, a.Q) })
	return ranges
}

// formatAliases are the short names the f query parameter accepts.
var formatAliases = map[string]string{
	"json":     JSON,
	"geojson":  GeoJSON,
	"sml":      SensorML,
	"smljson":  SensorML,
	"sensorml": SensorML,
}

// FromFormatParam resolves the f query parameter, a short name such as
// "geojson" or a full media type, to a media type.
func FromFormatParam(f string) string {
	f = strings.ToLower(strings.TrimSpace(f))
	if mediaType, ok := formatAliases[f]; ok {
		return mediaType
	}
	return Normalize(f)
}
