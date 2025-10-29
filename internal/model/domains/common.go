package domains

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base model with common fields
type Base struct {
	ID        string    `gorm:"primaryKey;type:varchar(255)" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`

	// DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type CommonSSN struct {
	// Core properties (from SOSA/SSN)
	UniqueIdentifier UniqueID `gorm:"type:varchar(255);uniqueIndex" json:"uid"`
	Name             string   `gorm:"type:varchar(255);not null" json:"name"`
	Description      string   `gorm:"type:text" json:"description,omitempty"`
}

// BeforeCreate hook to auto-generate UUID if ID is empty
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

// UniqueID represents a globally unique identifier (URI)
type UniqueID string
