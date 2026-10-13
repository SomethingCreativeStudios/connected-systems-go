package repository

import (
	"fmt"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// replaceByID overwrites every column of an existing row with replacement,
// which must already carry the row's ID. Unlike gorm's Save it never inserts:
// a missing row is ErrNotFound. created_at is carried over from the stored
// row, and keep (when non-nil) copies any other server-owned fields, such as
// parent IDs, from the locked existing row onto the replacement.
func replaceByID[T any, P interface {
	*T
	BaseFields() *domains.Base
}](tx *gorm.DB, replacement P, keep func(existing P)) error {
	return replaceByIDChecked(tx, replacement, func(existing P) error {
		if keep != nil {
			keep(existing)
		}
		return nil
	})
}

// replaceByIDChecked validates a replacement while the existing row is locked.
func replaceByIDChecked[T any, P interface {
	*T
	BaseFields() *domains.Base
}](tx *gorm.DB, replacement P, keep func(existing P) error) error {
	id := replacement.BaseFields().ID
	if id == "" {
		return ErrNotFound
	}

	existing := P(new(T))
	result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Limit(1).Find(existing)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	replacement.BaseFields().CreatedAt = existing.BaseFields().CreatedAt
	if keep != nil {
		if err := keep(existing); err != nil {
			return err
		}
	}

	// Select("*") writes zero values too, which is what a full replacement
	// means. Associations are synced explicitly by the callers.
	return tx.Model(replacement).Select("*").Omit("id", "created_at", clause.Associations).Updates(replacement).Error
}

// hasRows reports whether any row of model (a domain model or a table name)
// matches the condition. It uses
// LIMIT 1 instead of COUNT so the database can stop at the first match.
func hasRows(tx *gorm.DB, model any, query string, args ...any) (bool, error) {
	scope := tx.Model(model)
	if table, ok := model.(string); ok {
		scope = tx.Table(table)
	}
	var found int
	err := scope.Select("1").Where(query, args...).Limit(1).Scan(&found).Error
	return found == 1, err
}

// requireParent fails with ErrParentNotFound unless a row of model has the
// given id. There are no foreign keys on the parent columns, so without it a
// nested create under a missing parent would store an orphan.
func requireParent(tx *gorm.DB, model any, kind, id string) error {
	exists, err := hasRows(tx, model, "id = ?", id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s %q", ErrParentNotFound, kind, id)
	}
	return nil
}
