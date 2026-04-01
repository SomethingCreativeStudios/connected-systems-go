package repository

import (
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// DatastreamRepository handles Datastream data access.
type DatastreamRepository struct {
	db *gorm.DB
}

// NewDatastreamRepository creates a new DatastreamRepository.
func NewDatastreamRepository(db *gorm.DB) *DatastreamRepository {
	return &DatastreamRepository{db: db}
}

// Create creates a new datastream.
func (r *DatastreamRepository) Create(datastream *domains.Datastream) error {
	normalizeDatastreamRefs(datastream)
	r.populateSystemAssociations(datastream)
	return r.db.Create(datastream).Error
}

// GetByID retrieves a datastream by ID.
func (r *DatastreamRepository) GetByID(id string) (*domains.Datastream, error) {
	var datastream domains.Datastream
	err := r.db.Where("id = ?", id).First(&datastream).Error
	if err != nil {
		return nil, err
	}
	return &datastream, nil
}

// List retrieves datastreams with filtering.
func (r *DatastreamRepository) List(params *queryparams.DatastreamsQueryParams, systemID *string) ([]*domains.Datastream, int64, error) {
	var datastreams []*domains.Datastream
	var total int64

	query := r.db.Model(&domains.Datastream{})
	query = r.applyFilters(query, params, systemID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	err := query.Find(&datastreams).Error
	return datastreams, total, err
}

// Update updates a datastream.
// The system-derived fields (procedure, deployment, featureOfInterest, samplingFeature)
// are locked: they are always restored from the existing record and cannot be changed by the client.
func (r *DatastreamRepository) Update(datastream *domains.Datastream) error {
	var existing domains.Datastream
	if err := r.db.Select("id", "procedure_link", "procedure_id", "deployment_link", "deployment_id", "feature_of_interest", "feature_of_interest_id", "sampling_feature_link", "sampling_feature_id").
		Where("id = ?", datastream.ID).First(&existing).Error; err == nil {
		datastream.ProcedureLink = existing.ProcedureLink
		datastream.ProcedureID = existing.ProcedureID
		datastream.DeploymentLink = existing.DeploymentLink
		datastream.DeploymentID = existing.DeploymentID
		datastream.FeatureOfInterest = existing.FeatureOfInterest
		datastream.FeatureOfInterestID = existing.FeatureOfInterestID
		datastream.SamplingFeatureLink = existing.SamplingFeatureLink
		datastream.SamplingFeatureID = existing.SamplingFeatureID
	}
	normalizeDatastreamRefs(datastream)
	return r.db.Save(datastream).Error
}

// populateSystemAssociations overwrites the system-derived fields on a datastream
// using data from the parent system. These fields are server-provided and locked.
func (r *DatastreamRepository) populateSystemAssociations(datastream *domains.Datastream) {
	if datastream.SystemID == nil {
		return
	}
	systemID := *datastream.SystemID

	// Feature of interest — take the first link stored on the system
	var system domains.System
	if err := r.db.Select("id", "features_of_interest").Where("id = ?", systemID).First(&system).Error; err == nil {
		if len(system.FeaturesOfInterest) > 0 {
			foi := system.FeaturesOfInterest[0]
			datastream.FeatureOfInterest = &foi
			datastream.FeatureOfInterestID = foi.GetId("features")
		}
	}

	// Procedure — first entry in system_procedures join table
	var procID string
	if err := r.db.Table("system_procedures").Select("procedure_id").Where("system_id = ?", systemID).Limit(1).Scan(&procID).Error; err == nil && procID != "" {
		datastream.ProcedureLink = &common_shared.Link{Href: "procedures/" + procID}
		datastream.ProcedureID = &procID
	}

	// Deployment — first entry in system_deployments join table
	var depID string
	if err := r.db.Table("system_deployments").Select("deployment_id").Where("system_id = ?", systemID).Limit(1).Scan(&depID).Error; err == nil && depID != "" {
		datastream.DeploymentLink = &common_shared.Link{Href: "deployments/" + depID}
		datastream.DeploymentID = &depID
	}

	// Sampling feature — first sampling feature belonging to this system
	var sf domains.SamplingFeature
	if err := r.db.Select("id").Where("parent_system_id = ?", systemID).Limit(1).First(&sf).Error; err == nil && sf.ID != "" {
		datastream.SamplingFeatureLink = &common_shared.Link{Href: "samplingFeatures/" + sf.ID}
		datastream.SamplingFeatureID = &sf.ID
	}
}

// Delete deletes a datastream.
// If cascade is true, all observations associated with the datastream are deleted first.
func (r *DatastreamRepository) Delete(id string, cascade bool) error {
	if !cascade {
		return r.db.Delete(&domains.Datastream{}, "id = ?", id).Error
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("datastream_id = ?", id).Delete(&domains.Observation{}).Error; err != nil {
			return err
		}
		return tx.Delete(&domains.Datastream{}, "id = ?", id).Error
	})
}

// GetSchema retrieves only the schema for a datastream.
func (r *DatastreamRepository) GetSchema(id string) (*domains.DatastreamSchema, error) {
	var datastream domains.Datastream
	err := r.db.Select("id", "schema").Where("id = ?", id).First(&datastream).Error
	if err != nil {
		return nil, err
	}
	return datastream.Schema, nil
}

// UpdateSchema updates only the schema of a datastream.
func (r *DatastreamRepository) UpdateSchema(id string, schema *domains.DatastreamSchema) error {
	return r.db.Model(&domains.Datastream{}).Where("id = ?", id).Update("schema", schema).Error
}

func (r *DatastreamRepository) applyFilters(query *gorm.DB, params *queryparams.DatastreamsQueryParams, systemID *string) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ? OR unique_identifier IN ?", params.IDs, params.IDs)
	}

	if len(params.Q) > 0 {
		var clauses []string
		var args []interface{}
		for _, term := range params.Q {
			like := "%" + term + "%"
			clauses = append(clauses, "name ILIKE ?")
			args = append(args, like)
			clauses = append(clauses, "description ILIKE ?")
			args = append(args, like)
			clauses = append(clauses, "output_name ILIKE ?")
			args = append(args, like)
		}
		query = query.Where(strings.Join(clauses, " OR "), args...)
	}

	if params.PhenomenonTime != nil {
		if params.PhenomenonTime.Start != nil && params.PhenomenonTime.End != nil {
			query = query.Where("phenomenon_time_start <= ? AND (phenomenon_time_end IS NULL OR phenomenon_time_end >= ?)", params.PhenomenonTime.End, params.PhenomenonTime.Start)
		} else if params.PhenomenonTime.Start != nil {
			query = query.Where("phenomenon_time_end IS NULL OR phenomenon_time_end >= ?", params.PhenomenonTime.Start)
		} else if params.PhenomenonTime.End != nil {
			query = query.Where("phenomenon_time_start <= ?", params.PhenomenonTime.End)
		}
	}

	if params.ResultTime != nil {
		if params.ResultTime.Start != nil && params.ResultTime.End != nil {
			query = query.Where("result_time_start <= ? AND (result_time_end IS NULL OR result_time_end >= ?)", params.ResultTime.End, params.ResultTime.Start)
		} else if params.ResultTime.Start != nil {
			query = query.Where("result_time_end IS NULL OR result_time_end >= ?", params.ResultTime.Start)
		} else if params.ResultTime.End != nil {
			query = query.Where("result_time_start <= ?", params.ResultTime.End)
		}
	}

	if systemID != nil {
		query = query.Where("system_id = ?", *systemID)
	} else if len(params.System) > 0 {
		query = query.Where("system_id IN ?", params.System)
	}

	if len(params.FOI) > 0 {
		query = query.Where("feature_of_interest_id IN ?", params.FOI)
	}

	if len(params.ObservedProperty) > 0 {
		for _, observedProperty := range params.ObservedProperty {
			query = query.Where("observed_properties::text ILIKE ?", "%"+observedProperty+"%")
		}
	}

	return query
}

func normalizeDatastreamRefs(datastream *domains.Datastream) {
	if datastream == nil {
		return
	}

	if datastream.SystemLink != nil {
		datastream.SystemID = datastream.SystemLink.GetId("systems")
	}
	if datastream.ProcedureLink != nil {
		datastream.ProcedureID = datastream.ProcedureLink.GetId("procedures")
	}
	if datastream.DeploymentLink != nil {
		datastream.DeploymentID = datastream.DeploymentLink.GetId("deployments")
	}
	if datastream.FeatureOfInterest != nil {
		datastream.FeatureOfInterestID = datastream.FeatureOfInterest.GetId("features")
	}
	if datastream.SamplingFeatureLink != nil {
		datastream.SamplingFeatureID = datastream.SamplingFeatureLink.GetId("samplingFeatures")
	}
}
