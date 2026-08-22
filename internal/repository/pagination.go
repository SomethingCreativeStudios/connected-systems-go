package repository

import (
	"fmt"
	"strings"
	"time"

	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// CursorOrder describes the deterministic database tuple used by a resource
// collection. TimeColumns are decoded from the canonical cursor timestamp.
type CursorOrder struct {
	Columns     []string
	Descending  bool
	TimeColumns map[int]bool
}

var (
	CursorOrderIDAsc    = CursorOrder{Columns: []string{"id"}}
	CursorOrderTimeDesc = CursorOrder{
		Columns:     []string{"result_time", "id"},
		Descending:  true,
		TimeColumns: map[int]bool{0: true},
	}
)

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
		for j := 0; j < i; j++ {
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
