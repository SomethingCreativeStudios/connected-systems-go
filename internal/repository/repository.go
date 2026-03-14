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
		&domains.SystemEvent{},
		&domains.SystemHistoryRevision{},
	); err != nil {
		return err
	}

	// Ensure generic closure support for deployments (creates triggers/functions)
	if err := EnsureClosureSupport(db, "deployments", "id", "parent_deployment_id", "deployment_closures"); err != nil {
		return err
	}

	// Ensure delete-reparent trigger for deployments (reparent children to deleted node's parent)
	if err := EnsureDeleteReparentSupport(db, "deployments", "id", "parent_deployment_id"); err != nil {
		return err
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
