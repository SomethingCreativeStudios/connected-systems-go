package domains

import (
	"database/sql/driver"
	"fmt"
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

// BaseFields exposes the embedded Base so generic repository helpers can read
// the ID and carry lifecycle timestamps across a replacement.
func (b *Base) BaseFields() *Base {
	return b
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

// Value stores an empty uid as NULL. The column has a unique index, and
// resources whose uid is optional (control streams, features) would otherwise
// all collide on the empty string.
func (u UniqueID) Value() (driver.Value, error) {
	if u == "" {
		return nil, nil
	}
	return string(u), nil
}

// Scan reads a uid, treating NULL as empty.
func (u *UniqueID) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*u = ""
	case string:
		*u = UniqueID(v)
	case []byte:
		*u = UniqueID(v)
	default:
		return fmt.Errorf("scan uid: unsupported value type %T", value)
	}
	return nil
}
