package repository

import (
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"gorm.io/gorm"
)

// Repositories holds all repository instances
type Repositories struct {
	System          *SystemRepository
	Deployment      *DeploymentRepository
	Procedure       *ProcedureRepository
	SamplingFeature *SamplingFeatureRepository
	Property        *PropertyRepository
	Feature         *FeatureRepository
	Collection      *CollectionRepository
}

// NewRepositories creates new repository instances
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		System:          NewSystemRepository(db),
		Deployment:      NewDeploymentRepository(db),
		Procedure:       NewProcedureRepository(db),
		SamplingFeature: NewSamplingFeatureRepository(db),
		Property:        NewPropertyRepository(db),
		Feature:         NewFeatureRepository(db),
		Collection:      NewCollectionRepository(db),
	}
}

// AutoMigrate runs database migrations for all models
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domains.System{},
		&domains.Deployment{},
		&domains.Procedure{},
		&domains.SamplingFeature{},
		&domains.Property{},
		&domains.Feature{},
		&domains.Collection{},
	)
}
