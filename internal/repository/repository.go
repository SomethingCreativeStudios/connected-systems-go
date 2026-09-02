package repository

import (
	"fmt"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"gorm.io/gorm"
)

const observationHypertableChunkInterval = "7 days"

// Repositories holds all repository instances
type Repositories struct {
	System          *SystemRepository
	Deployment      *DeploymentRepository
	Procedure       *ProcedureRepository
	SamplingFeature *SamplingFeatureRepository
	Property        *PropertyRepository
	Feature         *FeatureRepository
	Datastream      *DatastreamRepository
	Observation     *ObservationRepository
	Collection      *CollectionRepository
	ControlStream   *ControlStreamRepository
	Command         *CommandRepository
	SystemEvent     *SystemEventRepository
	SystemHistory   *SystemHistoryRepository
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
		Datastream:      NewDatastreamRepository(db),
		Observation:     NewObservationRepository(db),
		Collection:      NewCollectionRepository(db),
		ControlStream:   NewControlStreamRepository(db),
		Command:         NewCommandRepository(db),
		SystemEvent:     NewSystemEventRepository(db),
		SystemHistory:   NewSystemHistoryRepository(db),
	}
}

// AutoMigrate runs database migrations for all models
func AutoMigrate(db *gorm.DB) error {
	if err := ensureDatabaseExtensions(db); err != nil {
		return err
	}
	if err := migrateLegacyArrayColumnsToJSONB(db); err != nil {
		return err
	}

	if err := db.AutoMigrate(
		&domains.System{},
		&domains.Deployment{},
		&domains.Procedure{},
		&domains.SamplingFeature{},
		&domains.Property{},
		&domains.Feature{},
		&domains.Datastream{},
		&domains.Observation{},
		&domains.Collection{},
		&domains.DeploymentClosure{},
		&domains.ControlStream{},
		&domains.Command{},
		&domains.CommandStatusReport{},
		&domains.CommandResult{},
		&domains.SystemEvent{},
		&domains.SystemHistoryRevision{},
	); err != nil {
		return err
	}
	if err := ensureObservationHypertable(db); err != nil {
		return err
	}
	if err := ensureSpatialGeometryStorage(db); err != nil {
		return err
	}

	// Ensure generic closure support for deployments (creates triggers/functions)
	if err := EnsureClosureSupport(db, "deployments", "id", "parent_deployment_id", "deployment_closures"); err != nil {
		return err
	}
	if err := ensureCursorIndexes(db); err != nil {
		return err
	}

	return nil
}

// ensureDatabaseExtensions makes the application's spatial and time-series
// dependencies explicit instead of relying on a particular container image to
// preload them. Both statements are idempotent.
func ensureDatabaseExtensions(db *gorm.DB) error {
	for _, extension := range []string{"timescaledb", "postgis"} {
		if err := db.Exec("CREATE EXTENSION IF NOT EXISTS " + extension).Error; err != nil {
			return fmt.Errorf("enable %s extension: %w", extension, err)
		}
	}
	return nil
}

// ensureObservationHypertable partitions observations by result time. A fresh
// database is required when upgrading the former id-only primary key because
// TimescaleDB requires the partition key in every primary or unique key.
func ensureObservationHypertable(db *gorm.DB) error {
	statement := fmt.Sprintf(`SELECT create_hypertable(
		'observations',
		by_range('result_time', INTERVAL '%s'),
		if_not_exists => TRUE
	)`, observationHypertableChunkInterval)
	if err := db.Exec(statement).Error; err != nil {
		if strings.Contains(err.Error(), "cannot create a unique index without") ||
			strings.Contains(err.Error(), "primary key") {
			return fmt.Errorf("convert observations to a TimescaleDB hypertable: %w; recreate the fresh development database because the legacy primary key must include result_time", err)
		}
		return fmt.Errorf("convert observations to a TimescaleDB hypertable: %w", err)
	}
	return nil
}

// ensureSpatialGeometryStorage upgrades legacy GeoJSON rows that were stored
// with SRID 0 and adds indexes used by bbox/geom intersection queries. GeoJSON
// coordinates are CRS84, which PostGIS represents with SRID 4326.
func ensureSpatialGeometryStorage(db *gorm.DB) error {
	tables := []string{"systems", "sampling_features", "deployments"}
	for _, table := range tables {
		statement := fmt.Sprintf(
			`UPDATE "%s"
			 SET geometry = ST_SetSRID(geometry, 4326)
			 WHERE geometry IS NOT NULL AND ST_SRID(geometry) = 0`,
			table,
		)
		if err := db.Exec(statement).Error; err != nil {
			return err
		}

		indexStatement := fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS "idx_%s_geometry_gist" ON "%s" USING GIST (geometry)`,
			table,
			table,
		)
		if err := db.Exec(indexStatement).Error; err != nil {
			return err
		}
	}
	return nil
}

// ensureCursorIndexes keeps keyset pagination efficient for the time-sorted
// nested collections. PostgreSQL accepts these idempotent statements on both
// fresh schemas and existing deployments.
func ensureCursorIndexes(db *gorm.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_observations_datastream_result_cursor ON observations (datastream_id, result_time DESC, id DESC)",
		"CREATE INDEX IF NOT EXISTS idx_commands_controlstream_issue_cursor ON commands (control_stream_id, issue_time DESC, id DESC)",
		"CREATE INDEX IF NOT EXISTS idx_command_status_reports_command_report_cursor ON command_status_reports (command_id, report_time DESC, id DESC)",
		"CREATE INDEX IF NOT EXISTS idx_command_results_command_created_cursor ON command_results (command_id, created_at DESC, id DESC)",
		"CREATE INDEX IF NOT EXISTS idx_system_events_system_time_cursor ON system_events (system_id, time_start DESC, created_at DESC, id DESC)",
		"CREATE INDEX IF NOT EXISTS idx_system_history_revisions_system_created_cursor ON system_history_revisions (system_id, created_at DESC, id DESC)",
	}
	for _, statement := range indexes {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func migrateLegacyArrayColumnsToJSONB(db *gorm.DB) error {
	columns := []struct {
		tableName  string
		columnName string
	}{
		{tableName: "sampling_features", columnName: "sample_of"},
		{tableName: "deployments", columnName: "system_ids"},
	}

	for _, c := range columns {
		shouldConvert, err := isLegacyStringArrayColumn(db, c.tableName, c.columnName)
		if err != nil {
			return err
		}
		if !shouldConvert {
			continue
		}

		statement := fmt.Sprintf(
			`ALTER TABLE "%s" ALTER COLUMN "%s" TYPE jsonb USING to_jsonb("%s")`,
			c.tableName,
			c.columnName,
			c.columnName,
		)
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}

	return nil
}

func isLegacyStringArrayColumn(db *gorm.DB, tableName, columnName string) (bool, error) {
	var result struct {
		DataType string `gorm:"column:data_type"`
		UDTName  string `gorm:"column:udt_name"`
	}

	queryResult := db.Raw(
		`SELECT data_type, udt_name
		 FROM information_schema.columns
		 WHERE table_schema = current_schema()
		   AND table_name = ?
		   AND column_name = ?`,
		tableName,
		columnName,
	).Scan(&result)
	if queryResult.Error != nil {
		return false, queryResult.Error
	}
	if queryResult.RowsAffected == 0 {
		return false, nil
	}

	if result.DataType != "ARRAY" {
		return false, nil
	}

	switch result.UDTName {
	case "_varchar", "_text", "_bpchar":
		return true, nil
	default:
		return false, nil
	}
}
