package repository

import (
	"github.com/yourusername/connected-systems-go/internal/model"
	"gorm.io/gorm"
)

// Repositories holds all repository instances
type Repositories struct {
	System          *SystemRepository
	Deployment      *DeploymentRepository
	Procedure       *ProcedureRepository
	SamplingFeature *SamplingFeatureRepository
	Property        *PropertyRepository
}

// NewRepositories creates new repository instances
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		System:          NewSystemRepository(db),
		Deployment:      NewDeploymentRepository(db),
		Procedure:       NewProcedureRepository(db),
		SamplingFeature: NewSamplingFeatureRepository(db),
		Property:        NewPropertyRepository(db),
	}
}

// AutoMigrate runs database migrations for all models
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.System{},
		&model.Deployment{},
		&model.Procedure{},
		&model.SamplingFeature{},
		&model.Property{},
	)
}

// TODO: Simplify and centralize QueryParams across repositories
// QueryParams represents common query parameters
type QueryParams struct {
	IDs                []string
	Bbox               *BoundingBox
	Datetime           *TimeFilter
	Limit              int
	Offset             int    // Not part of standard, but useful for pagination (till i do curorsors)
	Q                  string // Full-text search
	Geom               string // WKT geometry
	Parent             []string
	Procedure          []string
	FOI                []string
	ObservedProperty   []string
	ControlledProperty []string
	Recursive          bool
}

type PropertiesQueryParams struct {
	IDs          []string
	Q            string // Full-text search
	BaseProperty []string
	ObjectType   []string
	Limit        int
	Offset       int
}

// BoundingBox represents a spatial bounding box
type BoundingBox struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

// TimeFilter represents a temporal filter
type TimeFilter struct {
	Start *string
	End   *string
}
