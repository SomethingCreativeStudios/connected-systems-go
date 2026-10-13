package repository

import (
	"regexp"
	"strconv"

	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// applyCQLFilter appends the precompiled CQL2 predicate to an existing GORM
// statement. Its SQL fragment is produced exclusively by internal/queryables;
// repository code never interpolates client-supplied filter text.
func applyCQLFilter(query *gorm.DB, params *queryparams.QueryParams) *gorm.DB {
	if params == nil || params.CQL.Text == "" {
		return query
	}
	sql, args := gormPlaceholders(params.CQL.Text, params.CQL.Args)
	return query.Where(sql, args...)
}

var cqlPlaceholder = regexp.MustCompile(`\$([1-9][0-9]*)`)

// gocql2 deliberately emits PostgreSQL $n placeholders. GORM's expression
// builder owns the final driver placeholders, so convert each reference to a
// GORM bind marker while preserving argument ordering (including any repeated
// $n reference).
func gormPlaceholders(sql string, values []any) (string, []any) {
	args := make([]any, 0, len(values))
	sql = cqlPlaceholder.ReplaceAllStringFunc(sql, func(match string) string {
		index, err := strconv.Atoi(match[1:])
		if err != nil || index < 1 || index > len(values) {
			return match
		}
		args = append(args, values[index-1])
		return "?"
	})
	return sql, args
}
