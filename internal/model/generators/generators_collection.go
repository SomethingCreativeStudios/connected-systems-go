package generators

import (
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/google/uuid"
)

// FakeCollection returns a populated Collection
func FakeCollection() domains.Collection {
	id := uuid.New().String()
	title := f.Lorem().Sentence(2)
	return domains.Collection{
		ID:          id,
		Title:       title,
		Description: f.Lorem().Sentence(2),
		Links:       FakeLinks(),
		Extent: &common_shared.Extent{
			Spatial: &common_shared.BoundingBox{MinX: -180, MinY: -90, MaxX: 180, MaxY: 90},
		},
		ItemType: "feature",
		CRS:      []string{"http://www.opengis.net/def/crs/OGC/1.3/CRS84"},
	}
}
