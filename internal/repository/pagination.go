package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// CursorOrder describes the deterministic database tuple used by a resource
// collection. TimeColumns are decoded from the canonical cursor timestamp.
type CursorOrder struct {
	Columns     []string
	Descending  bool
	TimeColumns map[int]bool
}

// idOrder is the ascending ID tuple of a resource table. Columns are table
// qualified so association filters that join other tables (which have their
// own id column) cannot make the ORDER BY or the cursor predicate ambiguous.
func idOrder(table string) CursorOrder {
	return CursorOrder{Columns: []string{table + ".id"}}
}

func cursorTimeValue(value *time.Time) string {
	if value == nil {
		return queryparams.TimeCursorValue(time.Time{})
	}
	return queryparams.TimeCursorValue(*value)
}

func cursorValues(cursor *queryparams.Cursor, order CursorOrder) ([]any, error) {
	values := make([]any, len(cursor.Values))
	for i, value := range cursor.Values {
		if order.TimeColumns[i] {
			parsed, err := time.Parse(time.RFC3339Nano, value)
			if err != nil {
				return nil, queryparams.InvalidCursorError()
			}
			values[i] = parsed
			continue
		}
		values[i] = value
	}
	return values, nil
}

func cursorPredicate(columns []string, values []any, greater bool) (string, []any) {
	terms := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns)*(len(columns)+1)/2)
	operator := "<"
	if greater {
		operator = ">"
	}
	for i := range columns {
		parts := make([]string, 0, i+1)
		for j := range i {
			parts = append(parts, columns[j]+" = ?")
			args = append(args, values[j])
		}
		parts = append(parts, columns[i]+" "+operator+" ?")
		args = append(args, values[i])
		terms = append(terms, "("+strings.Join(parts, " AND ")+")")
	}
	return "(" + strings.Join(terms, " OR ") + ")", args
}

func orderClause(order CursorOrder, reverse bool) string {
	direction := "ASC"
	if order.Descending {
		direction = "DESC"
	}
	if reverse {
		if direction == "ASC" {
			direction = "DESC"
		} else {
			direction = "ASC"
		}
	}
	columns := make([]string, len(order.Columns))
	for i, column := range order.Columns {
		columns[i] = fmt.Sprintf("%s %s", column, direction)
	}
	return strings.Join(columns, ", ")
}

// ApplyCursorPagination applies a keyset predicate and fetches one extra row
// so FinalizeCursorPage can determine the presence of a neighbouring page.
func ApplyCursorPagination(query *gorm.DB, params *queryparams.QueryParams, order CursorOrder) (*gorm.DB, error) {
	reverse := params.Cursor != nil && params.Cursor.IsBefore()
	if params.Cursor != nil {
		values, err := cursorValues(params.Cursor, order)
		if err != nil {
			return nil, err
		}
		// Normal traversal continues after the tuple. Reverse traversal fetches
		// rows before the first tuple using the inverse comparison/order.
		greater := !order.Descending
		if reverse {
			greater = !greater
		}
		predicate, args := cursorPredicate(order.Columns, values, greater)
		query = query.Where(predicate, args...)
	}

	query = query.Order(orderClause(order, reverse))
	if params.Limit > 0 {
		query = query.Limit(params.Limit + 1)
	}
	return query, nil
}

// FinalizeCursorPage trims an extra row, sets adjacent-page flags, and
// restores normal ordering after a reverse traversal.
func FinalizeCursorPage[T any](items []T, params *queryparams.QueryParams) []T {
	if params.Limit <= 0 {
		params.Page = queryparams.PageInfo{}
		return items
	}

	hasExtra := len(items) > params.Limit
	if hasExtra {
		items = items[:params.Limit]
	}
	if params.Cursor != nil && params.Cursor.IsBefore() {
		for left, right := 0, len(items)-1; left < right; left, right = left+1, right-1 {
			items[left], items[right] = items[right], items[left]
		}
		params.Page = queryparams.PageInfo{HasPrev: hasExtra, HasNext: true}
		return items
	}
	params.Page = queryparams.PageInfo{HasNext: hasExtra, HasPrev: params.Cursor != nil}
	return items
}

// listPage counts the rows matching query, fetches one page of them in the
// cursor order and records the page bounds used for next/prev links.
func listPage[T any](query *gorm.DB, params *queryparams.QueryParams, order CursorOrder, anchor func(T) []string) ([]T, int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query, err := ApplyCursorPagination(query, params, order)
	if err != nil {
		return nil, 0, err
	}
	var items []T
	if err := query.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	items = FinalizeCursorPage(items, params)
	params.Anchors = queryparams.CursorAnchorsFor(items, anchor)
	return items, total, nil
}

// idAnchor is the cursor anchor of resources paged in ID order.
func idAnchor[T interface{ BaseFields() *domains.Base }](item T) []string {
	return []string{item.BaseFields().ID}
}
