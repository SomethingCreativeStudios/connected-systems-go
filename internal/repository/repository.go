package repository

import (
	"fmt"

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

	// Ensure generic closure support for deployments (creates triggers/functions)
	if err := EnsureClosureSupport(db, "deployments", "id", "parent_deployment_id", "deployment_closures"); err != nil {
		return err
	}
	if err := ensureCursorIndexes(db); err != nil {
		return err
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
