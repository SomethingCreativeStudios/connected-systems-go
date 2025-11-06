package domains

import (
	"github.com/google/uuid"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

type Collection struct {
	ID          string                `json:"id" gorm:"primaryKey"`
	Title       string                `json:"title,omitempty"`
	Description string                `json:"description,omitempty"`
	Links       common_shared.Links   `json:"links" gorm:"type:json"`
	Extent      *common_shared.Extent `json:"extent,omitempty" gorm:"type:json"`
	ItemType    string                `json:"itemType,omitempty" gorm:"default:feature"`
	CRS         []string              `json:"crs,omitempty" gorm:"type:json"`
}

type CollectionGeoJSONFeature struct {
	Collection
}

func NewCollection(id, title, description string, links []common_shared.Link, extent *common_shared.Extent, itemType string, crs []string) *Collection {
	if id == "" {
		id = uuid.New().String()
	}
	if itemType == "" {
		itemType = "feature"
	}
	if crs == nil {
		crs = []string{"http://www.opengis.net/def/crs/OGC/1.3/CRS84"}
	}
	return &Collection{
		ID:          id,
		Title:       title,
		Description: description,
		Links:       common_shared.Links(links),
		Extent:      extent,
		ItemType:    itemType,
		CRS:         crs,
	}
}
