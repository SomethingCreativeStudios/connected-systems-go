package repository

import (
	"strings"

	"gorm.io/gorm"
)

// applyKeywordFilter keeps the rows where any of the columns contains any of
// the keywords, ignoring case: the q parameter matches if one keyword does.
func applyKeywordFilter(query *gorm.DB, terms []string, columns ...string) *gorm.DB {
	if len(terms) == 0 || len(columns) == 0 {
		return query
	}
	clauses := make([]string, 0, len(terms)*len(columns))
	args := make([]any, 0, len(terms)*len(columns))
	for _, term := range terms {
		like := "%" + term + "%"
		for _, column := range columns {
			clauses = append(clauses, column+" ILIKE ?")
			args = append(args, like)
		}
	}
	return query.Where("("+strings.Join(clauses, " OR ")+")", args...)
}
