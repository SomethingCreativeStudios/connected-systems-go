package queryables

import (
	"fmt"
	"strings"

	"github.com/cwygoda/gocql2/api"
	cql2sql "github.com/cwygoda/gocql2/sql"
)

type intervalColumns struct{ start, end string }

// intervalDialect supplements gocql2's PostGIS dialect with the two-column
// temporal intervals used by Connected Systems. gocql2 intentionally does not
// impose a storage shape for intervals, so the adaptation stays here rather
// than leaking database columns into the dependency or handlers.
type intervalDialect struct {
	cql2sql.Dialect
	intervals map[string]intervalColumns
}

func (d intervalDialect) RenderTemporalPredicate(ctx cql2sql.RenderContext, expression *api.TemporalPredicateExpression) (cql2sql.Fragment, bool, error) {
	if !d.hasInterval(expression.Left) && !d.hasInterval(expression.Right) {
		return d.Dialect.RenderTemporalPredicate(ctx, expression)
	}
	start := func(node api.Node) (cql2sql.Fragment, error) { return d.endpoint(ctx, node, true) }
	end := func(node api.Node) (cql2sql.Fragment, error) { return d.endpoint(ctx, node, false) }
	both := func(node api.Node) (cql2sql.TemporalOperand, error) {
		s, err := start(node)
		if err != nil {
			return cql2sql.TemporalOperand{}, err
		}
		e, err := end(node)
		if err != nil {
			return cql2sql.TemporalOperand{}, err
		}
		return cql2sql.TemporalOperand{Start: s, End: e}, nil
	}
	cmp := func(a cql2sql.Fragment, op string, b cql2sql.Fragment) string {
		return fmt.Sprintf("(%s %s %s)", a.Text, op, b.Text)
	}
	and := func(parts ...string) cql2sql.Fragment {
		return cql2sql.Fragment{Text: "(" + strings.Join(parts, " AND ") + ")"}
	}
	or := func(parts ...string) cql2sql.Fragment {
		return cql2sql.Fragment{Text: "(" + strings.Join(parts, " OR ") + ")"}
	}

	switch expression.Op {
	case api.TemporalOpAfter:
		left, err := start(expression.Left)
		if err != nil {
			return cql2sql.Fragment{}, true, err
		}
		right, err := end(expression.Right)
		if err != nil {
			return cql2sql.Fragment{}, true, err
		}
		return cql2sql.Fragment{Text: cmp(left, ">", right)}, true, nil
	case api.TemporalOpBefore:
		left, err := end(expression.Left)
		if err != nil {
			return cql2sql.Fragment{}, true, err
		}
		right, err := start(expression.Right)
		if err != nil {
			return cql2sql.Fragment{}, true, err
		}
		return cql2sql.Fragment{Text: cmp(left, "<", right)}, true, nil
	case api.TemporalOpMeets, api.TemporalOpMetBy:
		var left, right cql2sql.Fragment
		var err error
		if expression.Op == api.TemporalOpMeets {
			left, err = end(expression.Left)
			if err == nil {
				right, err = start(expression.Right)
			}
		} else {
			left, err = start(expression.Left)
			if err == nil {
				right, err = end(expression.Right)
			}
		}
		if err != nil {
			return cql2sql.Fragment{}, true, err
		}
		return cql2sql.Fragment{Text: cmp(left, "=", right)}, true, nil
	}
	left, err := both(expression.Left)
	if err != nil {
		return cql2sql.Fragment{}, true, err
	}
	right, err := both(expression.Right)
	if err != nil {
		return cql2sql.Fragment{}, true, err
	}
	disjoint := or(cmp(left.End, "<", right.Start), cmp(left.Start, ">", right.End)).Text
	switch expression.Op {
	case api.TemporalOpDisjoint:
		return cql2sql.Fragment{Text: disjoint}, true, nil
	case api.TemporalOpEquals:
		return and(cmp(left.Start, "=", right.Start), cmp(left.End, "=", right.End)), true, nil
	case api.TemporalOpIntersects:
		return cql2sql.Fragment{Text: "(NOT " + disjoint + ")"}, true, nil
	case api.TemporalOpContains:
		return and(cmp(left.Start, "<=", right.Start), cmp(left.End, ">=", right.End)), true, nil
	case api.TemporalOpDuring:
		return and(cmp(left.Start, ">=", right.Start), cmp(left.End, "<=", right.End)), true, nil
	case api.TemporalOpFinishedBy:
		return and(cmp(left.End, "=", right.End), cmp(left.Start, "<", right.Start)), true, nil
	case api.TemporalOpFinishes:
		return and(cmp(left.End, "=", right.End), cmp(left.Start, ">", right.Start)), true, nil
	case api.TemporalOpOverlappedBy:
		return and(cmp(right.Start, "<", left.Start), cmp(right.End, ">", left.Start), cmp(right.End, "<", left.End)), true, nil
	case api.TemporalOpOverlaps:
		return and(cmp(left.Start, "<", right.Start), cmp(left.End, ">", right.Start), cmp(left.End, "<", right.End)), true, nil
	case api.TemporalOpStartedBy:
		return and(cmp(left.Start, "=", right.Start), cmp(left.End, ">", right.End)), true, nil
	case api.TemporalOpStarts:
		return and(cmp(left.Start, "=", right.Start), cmp(left.End, "<", right.End)), true, nil
	default:
		return cql2sql.Fragment{}, false, nil
	}
}

func (d intervalDialect) hasInterval(node api.Node) bool {
	ref, ok := node.(*api.PropertyRef)
	if !ok {
		return false
	}
	_, ok = d.intervals[ref.Name]
	return ok
}

func (d intervalDialect) endpoint(ctx cql2sql.RenderContext, node api.Node, isStart bool) (cql2sql.Fragment, error) {
	if ref, ok := node.(*api.PropertyRef); ok {
		if columns, ok := d.intervals[ref.Name]; ok {
			column, infinity := columns.end, "'infinity'::timestamptz"
			if isStart {
				column, infinity = columns.start, "'-infinity'::timestamptz"
			}
			return cql2sql.Fragment{Text: "COALESCE(" + column + ", " + infinity + ")"}, nil
		}
	}
	if isStart {
		return ctx.RenderTemporalStart(node)
	}
	return ctx.RenderTemporalEnd(node)
}
