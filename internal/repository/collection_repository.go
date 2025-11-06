package repository

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"gorm.io/gorm"
)

type CollectionRepository struct {
	DB *gorm.DB
}

func NewCollectionRepository(db *gorm.DB) *CollectionRepository {
	return &CollectionRepository{DB: db}
}

func (r *CollectionRepository) ListCollections(ctx context.Context) ([]*domains.Collection, error) {
	var collections []*domains.Collection
	if err := r.DB.WithContext(ctx).Find(&collections).Error; err != nil {
		return nil, err
	}
	return collections, nil
}

func (r *CollectionRepository) GetCollectionByID(ctx context.Context, id string) (*domains.Collection, error) {
	var collection domains.Collection
	if err := r.DB.WithContext(ctx).First(&collection, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &collection, nil
}

func (r *CollectionRepository) CreateCollection(ctx context.Context, collection *domains.Collection) error {
	return r.DB.WithContext(ctx).Create(collection).Error
}

func (r *CollectionRepository) UpdateCollection(ctx context.Context, collection *domains.Collection) error {
	return r.DB.WithContext(ctx).Save(collection).Error
}

func (r *CollectionRepository) DeleteCollection(ctx context.Context, id string) error {
	return r.DB.WithContext(ctx).Delete(&domains.Collection{}, "id = ?", id).Error
}
