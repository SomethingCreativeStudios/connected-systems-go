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
	if err := r.db.Create(datastream).Error; err != nil {
		return err
	}
	if datastream.SystemID != nil && datastream.ID != "" {
		r.db.Exec(
			"INSERT INTO system_datastreams (system_id, datastream_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
			*datastream.SystemID, datastream.ID,
		)
	}
	return nil
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
// Procedure and deployment are locked (server-derived from system). SamplingFeature can be
// changed by the client; featureOfInterest is re-derived from the updated samplingFeature.
func (r *DatastreamRepository) Update(datastream *domains.Datastream) error {
	var existing domains.Datastream
	if err := r.db.Select("id", "procedure_link", "procedure_id", "deployment_link", "deployment_id").
		Where("id = ?", datastream.ID).First(&existing).Error; err == nil {
		datastream.ProcedureLink = existing.ProcedureLink
		datastream.ProcedureID = existing.ProcedureID
		datastream.DeploymentLink = existing.DeploymentLink
		datastream.DeploymentID = existing.DeploymentID
	}
	normalizeDatastreamRefs(datastream)
	r.deriveFOIFromSamplingFeature(datastream)
	return r.db.Save(datastream).Error
}

// populateSystemAssociations sets procedure and deployment from the parent system on create.
// Procedure comes from system.SystemKindID; deployment from system_deployments.
// FOI is derived from the user-supplied samplingFeature via deriveFOIFromSamplingFeature.
func (r *DatastreamRepository) populateSystemAssociations(datastream *domains.Datastream) {
	if datastream.SystemID == nil {
		return
	}
	systemID := *datastream.SystemID

	// Procedure from TypeOfID FK
	var sys domains.System
	if err := r.db.Select("id", "type_of_id").Where("id = ?", systemID).First(&sys).Error; err == nil {
		if sys.TypeOfID != nil && *sys.TypeOfID != "" {
			kindID := *sys.TypeOfID
			datastream.ProcedureLink = &common_shared.Link{Href: "procedures/" + kindID}
			datastream.ProcedureID = &kindID
		}
	}

	// Deployment — first entry in system_deployments join table
	var depID string
	if err := r.db.Table("system_deployments").Select("deployment_id").Where("system_id = ?", systemID).Limit(1).Scan(&depID).Error; err == nil && depID != "" {
		datastream.DeploymentLink = &common_shared.Link{Href: "deployments/" + depID}
		datastream.DeploymentID = &depID
	}

	// FOI derived from user-provided samplingFeature
	r.deriveFOIFromSamplingFeature(datastream)
}

// deriveFOIFromSamplingFeature sets FeatureOfInterest and FeatureOfInterestID on the datastream
// by looking up the sampledFeature link stored on the referenced SamplingFeature.
func (r *DatastreamRepository) deriveFOIFromSamplingFeature(datastream *domains.Datastream) {
	if datastream.SamplingFeatureID == nil {
		return
	}
	var sf domains.SamplingFeature
	if err := r.db.Select("id", "sampled_feature_link", "sampled_feature_id").
		Where("id = ?", *datastream.SamplingFeatureID).First(&sf).Error; err == nil {
		datastream.FeatureOfInterest = sf.SampledFeatureLink
		datastream.FeatureOfInterestID = sf.SampledFeatureID
	}
}

// Delete deletes a datastream.
// If cascade is true, all observations associated with the datastream are deleted first.
func (r *DatastreamRepository) Delete(id string, cascade bool) error {
	// Always clean join table first to unblock FK constraints.
	if err := r.db.Exec("DELETE FROM system_datastreams WHERE datastream_id = ?", id).Error; err != nil {
		return err
	}

	if !cascade {
		var obsCount int64
		if err := r.db.Model(&domains.Observation{}).Where("datastream_id = ?", id).Count(&obsCount).Error; err != nil {
			return err
		}
		if obsCount > 0 {
			return ErrHasChildren
		}

		result := r.db.Delete(&domains.Datastream{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("datastream_id = ?", id).Delete(&domains.Observation{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&domains.Datastream{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
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
		query = query.Where("sampling_feature_id IN ? OR feature_of_interest_id IN ?", params.FOI, params.FOI)
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
