package repository_shared

import (
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"gorm.io/gorm"
)

// Repositories holds all repository instances
type Repositories struct {
	System          *repository.SystemRepository
	Deployment      *repository.DeploymentRepository
	Procedure       *repository.ProcedureRepository
	SamplingFeature *repository.SamplingFeatureRepository
	Property        *repository.PropertyRepository
}

// NewRepositories creates new repository instances
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		System:          repository.NewSystemRepository(db),
		Deployment:      repository.NewDeploymentRepository(db),
		Procedure:       repository.NewProcedureRepository(db),
		SamplingFeature: repository.NewSamplingFeatureRepository(db),
		Property:        repository.NewPropertyRepository(db),
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
	)
}
