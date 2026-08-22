package queryparams

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

const (
	unsupportedOffsetError = "offset is no longer supported; use cursor"
	invalidCursorError     = "invalid cursor"
)

// CursorKind identifies the stable ordering used by a collection endpoint.
// It is part of the opaque server-generated token, not a client input.
type CursorKind string

const (
	CursorKindIDAsc     CursorKind = "id-asc"
	CursorKindTimeDesc  CursorKind = "time-desc"
	CursorKindEventDesc CursorKind = "event-desc"
)

type cursorDirection string

const (
	cursorAfter  cursorDirection = "after"
	cursorBefore cursorDirection = "before"
)

// Cursor is the decoded server-owned continuation token.
type Cursor struct {
	Version   int             `json:"v"`
	Scope     string          `json:"s"`
	Kind      CursorKind      `json:"k"`
	Direction cursorDirection `json:"d"`
	Values    []string        `json:"p"`
}

func (c *Cursor) IsBefore() bool {
	return c != nil && c.Direction == cursorBefore
}

// PageInfo is set by repositories after fetching limit + 1 rows.
type PageInfo struct {
	HasNext bool
	HasPrev bool
}

// CursorAnchors hold the stable sort tuples for the returned page bounds.
type CursorAnchors struct {
	First []string
	Last  []string
}

func CursorAnchorsFor[T any](items []T, keys func(T) []string) CursorAnchors {
	if len(items) == 0 {
		return CursorAnchors{}
	}
	return CursorAnchors{First: keys(items[0]), Last: keys(items[len(items)-1])}
}

func TimeCursorValue(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func buildURLWithQuery(baseURL string, params url.Values) string {
	encoded := params.Encode()
	if encoded == "" {
		return baseURL
	}
	return baseURL + "?" + encoded
}

func cloneURLValues(params url.Values) url.Values {
	cloned := url.Values{}
	for key, values := range params {
		copied := make([]string, len(values))
		copy(copied, values)
		cloned[key] = copied
	}
	return cloned
}

func cursorScope(path string, params url.Values) string {
	filtered := cloneURLValues(params)
	filtered.Del("cursor")
	filtered.Del("offset")
	filtered.Del("limit")
	sum := sha256.Sum256([]byte(path + "?" + filtered.Encode()))
	return hex.EncodeToString(sum[:])
}

func cursorValueCount(kind CursorKind) int {
	switch kind {
	case CursorKindIDAsc:
		return 1
	case CursorKindTimeDesc:
		return 2
	case CursorKindEventDesc:
		return 3
	default:
		return 0
	}
}

func decodeCursor(value, scope string, kind CursorKind) (*Cursor, error) {
	encoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, errors.New(invalidCursorError)
	}

	var cursor Cursor
	if err := json.Unmarshal(encoded, &cursor); err != nil || cursor.Version != 1 ||
		cursor.Scope != scope || cursor.Kind != kind ||
		(cursor.Direction != cursorAfter && cursor.Direction != cursorBefore) ||
		len(cursor.Values) != cursorValueCount(kind) {
		return nil, errors.New(invalidCursorError)
	}
	for _, value := range cursor.Values {
		if value == "" {
			return nil, errors.New(invalidCursorError)
		}
	}
	for _, index := range cursorTimeIndexes(kind) {
		if _, err := time.Parse(time.RFC3339Nano, cursor.Values[index]); err != nil {
			return nil, errors.New(invalidCursorError)
		}
	}
	return &cursor, nil
}

func cursorTimeIndexes(kind CursorKind) []int {
	switch kind {
	case CursorKindTimeDesc:
		return []int{0}
	case CursorKindEventDesc:
		return []int{0, 1}
	default:
		return nil
	}
}

func InvalidCursorError() error {
	return errors.New(invalidCursorError)
}

func encodeCursor(scope string, kind CursorKind, direction cursorDirection, values []string) string {
	encoded, _ := json.Marshal(Cursor{
		Version:   1,
		Scope:     scope,
		Kind:      kind,
		Direction: direction,
		Values:    values,
	})
	return base64.RawURLEncoding.EncodeToString(encoded)
}

type QueryParams struct {
	IDs []string
	Q   []string // Full-text search

	Limit   int
	Cursor  *Cursor
	Page    PageInfo
	Kind    CursorKind
	Path    string
	Anchors CursorAnchors
}

// BuildFromRequest parses shared collection query parameters and validates a
// cursor against the route and active filters.
func (QueryParams) BuildFromRequest(r *http.Request, defaultLimit int, kind CursorKind) (*QueryParams, error) {
	if r.URL.Query().Has("offset") {
		return nil, errors.New(unsupportedOffsetError)
	}

	params := &QueryParams{Limit: defaultLimit, Kind: kind, Path: r.URL.Path}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			params.Limit = val
		}
	}
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		decoded, err := decodeCursor(cursor, cursorScope(r.URL.Path, r.URL.Query()), kind)
		if err != nil {
			return nil, err
		}
		params.Cursor = decoded
	}
	if ids := r.URL.Query().Get("id"); ids != "" {
		params.IDs = strings.Split(ids, ",")
	}
	if queries := r.URL.Query().Get("q"); queries != "" {
		params.Q = strings.Split(queries, ",")
	}
	return params, nil
}

// BuildPaginationLinks preserves request filters while replacing the opaque
// cursor for adjacent pages. It never creates offset links.
func (qp *QueryParams) BuildPaginationLinks(baseURL string, params url.Values) common_shared.Links {
	links := common_shared.Links{{Href: buildURLWithQuery(baseURL, params), Rel: "self"}}
	if len(qp.Anchors.First) == 0 || len(qp.Anchors.Last) == 0 {
		return links
	}

	scope := cursorScope(qp.Path, params)
	if qp.Page.HasNext {
		next := cloneURLValues(params)
		next.Set("cursor", encodeCursor(scope, qp.Kind, cursorAfter, qp.Anchors.Last))
		links = append(links, common_shared.Link{Rel: "next", Href: buildURLWithQuery(baseURL, next)})
	}
	if qp.Page.HasPrev {
		prev := cloneURLValues(params)
		prev.Set("cursor", encodeCursor(scope, qp.Kind, cursorBefore, qp.Anchors.First))
		links = append(links, common_shared.Link{Rel: "prev", Href: buildURLWithQuery(baseURL, prev)})
	}
	return links
}
