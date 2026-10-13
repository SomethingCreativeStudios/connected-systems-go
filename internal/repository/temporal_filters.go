package repository

import (
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"gorm.io/gorm"
)

// applyValidTimeFilter keeps the rows whose validTime intersects tr. A
// missing bound is open-ended, so a resource without any validTime matches
// every datetime filter, as Connected Systems requires.
func applyValidTimeFilter(query *gorm.DB, table string, tr *common_shared.TimeRange) *gorm.DB {
	if tr == nil || tr.Latest {
		return query
	}
	start, end := table+".valid_time_start", table+".valid_time_end"
	if tr.End != nil {
		query = query.Where("("+start+" IS NULL OR "+start+" <= ?)", tr.End)
	}
	if tr.Start != nil {
		query = query.Where("("+end+" IS NULL OR "+end+" >= ?)", tr.Start)
	}
	return query
}

// applyRangeFilter keeps the rows whose [start, end] range overlaps tr. An
// open (NULL) end counts as ongoing; a NULL start never matches a bounded end.
func applyRangeFilter(query *gorm.DB, startColumn, endColumn string, tr *common_shared.TimeRange) *gorm.DB {
	if tr == nil || tr.Latest {
		return query
	}
	if tr.End != nil {
		query = query.Where(startColumn+" <= ?", tr.End)
	}
	if tr.Start != nil {
		query = query.Where("("+endColumn+" IS NULL OR "+endColumn+" >= ?)", tr.Start)
	}
	return query
}

// applyInstantFilter keeps the rows whose single timestamp column falls
// within tr.
func applyInstantFilter(query *gorm.DB, column string, tr *common_shared.TimeRange) *gorm.DB {
	if tr == nil || tr.Latest {
		return query
	}
	if tr.Start != nil {
		query = query.Where(column+" >= ?", tr.Start)
	}
	if tr.End != nil {
		query = query.Where(column+" <= ?", tr.End)
	}
	return query
}
