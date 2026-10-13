package repository

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrate brings the database schema up to date. Each migration runs once
// and is recorded in goose_db_version, and servers starting together wait on
// a Postgres advisory lock instead of migrating at the same time.
//
// Version 1 builds the schema from the GORM models (see baselineSchema). It
// is idempotent, so a database created before versioned migrations is just
// recorded as being at version 1. Every later change is a numbered SQL file
// in migrations/, written with IF [NOT] EXISTS so that it also applies to a
// fresh database, whose version 1 already reflects the newer models.
func Migrate(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	files, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return err
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return err
	}
	baseline := goose.NewGoMigration(1, &goose.GoFunc{
		// Creating the hypertable cannot run inside a transaction.
		RunDB: func(ctx context.Context, _ *sql.DB) error {
			return baselineSchema(db.WithContext(ctx))
		},
	}, nil)
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, files,
		goose.WithSessionLocker(locker),
		goose.WithGoMigrations(baseline, goose.NewGoMigration(3, &goose.GoFunc{RunDB: func(ctx context.Context, _ *sql.DB) error {
			return repairStreamMetadata(db.WithContext(ctx))
		}}, nil)),
	)
	if err != nil {
		return err
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	return nil
}

// Repair legacy query metadata with the same derivation used for all writes.
func repairStreamMetadata(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var datastreams []domains.Datastream
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).FindInBatches(&datastreams, 200, func(_ *gorm.DB, _ int) error {
			for _, stream := range datastreams {
				if stream.Schema != nil && len(stream.Schemas) == 0 {
					stream.Schemas = stream.Schemas.Upsert(*stream.Schema)
				}
				if stream.Schema == nil && len(stream.Schemas) > 0 {
					stream.Schema = stream.Schemas.ForFormat("application/json")
					if stream.Schema == nil {
						stream.Schema = &stream.Schemas[0]
					}
				}
				stream.RecomputeSchemaMetadata()
				if err := tx.Model(&stream).Select("schema", "schemas", "observed_properties", "result_type", "formats").Updates(&stream).Error; err != nil {
					return err
				}
			}
			return nil
		}).Error; err != nil {
			return err
		}
		var controls []domains.ControlStream
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).FindInBatches(&controls, 200, func(_ *gorm.DB, _ int) error {
			for _, stream := range controls {
				if stream.Schema != nil && len(stream.Schemas) == 0 {
					stream.Schemas = stream.Schemas.Upsert(*stream.Schema)
				}
				if stream.Schema == nil && len(stream.Schemas) > 0 {
					stream.Schema = stream.Schemas.ForFormat("application/json")
					if stream.Schema == nil {
						stream.Schema = &stream.Schemas[0]
					}
				}
				stream.RecomputeSchemaMetadata()
				if err := tx.Model(&stream).Select("schema", "schemas", "controlled_properties", "formats").Updates(&stream).Error; err != nil {
					return err
				}
			}
			return nil
		}).Error; err != nil {
			return err
		}
		return nil
	})
}
