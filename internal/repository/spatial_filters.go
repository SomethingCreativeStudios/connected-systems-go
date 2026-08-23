package repository

import (
	"fmt"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"gorm.io/gorm"
)

// applySpatialIntersectionFilters applies the Part 1 bbox and geom filters to
// a trusted, repository-owned geometry column. Resource geometries and query
// geometries use CRS84/EPSG:4326.
func applySpatialIntersectionFilters(query *gorm.DB, geometryColumn string, bbox *common_shared.BoundingBox, geom string) *gorm.DB {
	if bbox != nil {
		if bbox.MinX <= bbox.MaxX {
			query = query.Where(
				fmt.Sprintf("ST_Intersects(%s, ST_MakeEnvelope(?, ?, ?, ?, 4326))", geometryColumn),
				bbox.MinX, bbox.MinY, bbox.MaxX, bbox.MaxY,
			)
		} else {
			// A CRS84 bbox whose west edge is east of its east edge crosses the
			// antimeridian. Split it into the two valid PostGIS envelopes.
			query = query.Where(
				fmt.Sprintf("(ST_Intersects(%[1]s, ST_MakeEnvelope(?, ?, 180, ?, 4326)) OR ST_Intersects(%[1]s, ST_MakeEnvelope(-180, ?, ?, ?, 4326)))", geometryColumn),
				bbox.MinX, bbox.MinY, bbox.MaxY,
				bbox.MinY, bbox.MaxX, bbox.MaxY,
			)
		}
	}

	if geom != "" {
		query = query.Where(
			fmt.Sprintf("ST_Intersects(%s, ST_GeomFromText(?, 4326))", geometryColumn),
			geom,
		)
	}

	return query
}
