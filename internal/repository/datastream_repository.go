package repository

import (
	"context"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
func (r *DatastreamRepository) Create(ctx context.Context, datastream *domains.Datastream) error {
	// phenomenonTime and resultTime are read-only extents derived from
	// observations; a new datastream has none.
	datastream.PhenomenonTime = nil
	datastream.ResultTime = nil
	datastream.Live = nil
	// Seed the per-format schema registry with the initial schema.
	datastream.Schemas = nil
	if datastream.Schema != nil {
		datastream.Schemas = datastream.Schemas.Upsert(*datastream.Schema)
	}
	datastream.RecomputeSchemaMetadata()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.resolveReferences(ctx, tx, datastream); err != nil {
			return err
		}
		if datastream.SystemID != nil {
			if err := requireParent(tx, &domains.System{}, "system", *datastream.SystemID); err != nil {
				return err
			}
		}
		if err := r.populateSystemAssociations(tx, datastream); err != nil {
			return err
		}
		if err := r.resolveReferences(ctx, tx, datastream); err != nil {
			return err
		}
		return tx.Create(datastream).Error
	})
}

// GetByID retrieves a datastream by ID.
func (r *DatastreamRepository) GetByID(ctx context.Context, id string) (*domains.Datastream, error) {
	var datastream domains.Datastream
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&datastream).Error
	if err != nil {
		return nil, err
	}
	return &datastream, nil
}

// List retrieves datastreams with filtering.
func (r *DatastreamRepository) List(ctx context.Context, params *queryparams.DatastreamsQueryParams, systemID *string) ([]*domains.Datastream, int64, error) {
	var datastreams []*domains.Datastream

	query := r.db.WithContext(ctx).Model(&domains.Datastream{})
	query = r.applyFilters(query, params, systemID)

	if params.PhenomenonTime != nil && params.PhenomenonTime.Latest {
		err := query.Order("phenomenon_time_start desc").Limit(1).Find(&datastreams).Error
		return datastreams, int64(len(datastreams)), err
	}

	return listPage(query, &params.QueryParams, idOrder("datastreams"), idAnchor[*domains.Datastream])
}

// Update updates a datastream.
// Writable links are replaced; system membership and schema registries are preserved.
func (r *DatastreamRepository) Update(ctx context.Context, datastream *domains.Datastream) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return replaceByIDChecked(tx, datastream, func(existing *domains.Datastream) error {
			// system@link is read-only: a datastream cannot be reparented.
			datastream.SystemID = existing.SystemID
			datastream.Live = existing.Live
			// phenomenonTime and resultTime are read-only extents derived from
			// observations; carry the stored values forward.
			datastream.PhenomenonTime = common_shared.NonEmptyTimeRange(existing.PhenomenonTime)
			datastream.ResultTime = common_shared.NonEmptyTimeRange(existing.ResultTime)

			if err := r.resolveReferences(ctx, tx, datastream); err != nil {
				return err
			}
			return mergeDatastreamSchema(tx, datastream, existing)
		})
	})
}

// populateSystemAssociations sets procedure and deployment from the parent system on create.
// Procedure comes from systems.type_of_id; deployment is inferred only when unique.
// FOI is derived from the user-supplied samplingFeature via deriveFOIFromSamplingFeature.
func (r *DatastreamRepository) populateSystemAssociations(db *gorm.DB, datastream *domains.Datastream) error {
	if datastream.SystemID == nil {
		return nil
	}
	systemID := *datastream.SystemID

	// Procedure from TypeOfID FK
	var sys domains.System
	if err := db.Select("id", "type_of_id").Where("id = ?", systemID).First(&sys).Error; err != nil {
		return err
	}
	{
		if datastream.ProcedureLink == nil && sys.TypeOfID != nil && *sys.TypeOfID != "" {
			kindID := *sys.TypeOfID
			datastream.ProcedureLink = &common_shared.Link{Href: "procedures/" + kindID}
			datastream.ProcedureID = &kindID
		}
	}

	// Infer deployment only for an unambiguous single membership.
	var depIDs []string
	if err := db.Table("system_deployments").Where("system_id = ?", systemID).Limit(2).Pluck("deployment_id", &depIDs).Error; err != nil {
		return err
	}
	if datastream.DeploymentLink == nil && len(depIDs) == 1 {
		depID := depIDs[0]
		datastream.DeploymentLink = &common_shared.Link{Href: "deployments/" + depID}
		datastream.DeploymentID = &depID
	}

	// FOI derived from user-provided samplingFeature
	return r.deriveFOIFromSamplingFeature(db, datastream)
}

// deriveFOIFromSamplingFeature sets FeatureOfInterest and FeatureOfInterestID on the datastream
// by looking up the sampledFeature link stored on the referenced SamplingFeature.
func (r *DatastreamRepository) deriveFOIFromSamplingFeature(db *gorm.DB, datastream *domains.Datastream) error {
	if datastream.FeatureOfInterest != nil || datastream.SamplingFeatureID == nil {
		return nil
	}
	var sf domains.SamplingFeature
	if err := db.Select("id", "sampled_feature_link", "sampled_feature_id").
		Where("id = ?", *datastream.SamplingFeatureID).First(&sf).Error; err != nil {
		return err
	}
	{
		datastream.FeatureOfInterest = sf.SampledFeatureLink
		datastream.FeatureOfInterestID = sf.SampledFeatureID
	}
	return nil
}

// Delete deletes a datastream.
// If cascade is true, all observations associated with the datastream are deleted first.
// The checks and every delete share one transaction, so a refused or failed
// delete leaves the datastream exactly as it was.
func (r *DatastreamRepository) Delete(ctx context.Context, id string, cascade bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockDatastream(tx, id); err != nil {
			return err
		}
		if cascade {
			if err := tx.Where("datastream_id = ?", id).Delete(&domains.Observation{}).Error; err != nil {
				return err
			}
		} else if has, err := hasRows(tx, &domains.Observation{}, "datastream_id = ?", id); err != nil {
			return err
		} else if has {
			return ErrHasChildren
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

// GetSchema retrieves the current schema and the per-format schema registry
// for a datastream.
func (r *DatastreamRepository) GetSchema(ctx context.Context, id string) (*domains.DatastreamSchema, domains.DatastreamSchemas, error) {
	var datastream domains.Datastream
	err := r.db.WithContext(ctx).Select("id", "schema", "schemas").Where("id = ?", id).First(&datastream).Error
	if err != nil {
		return nil, nil, err
	}
	// Legacy rows predate the registry: expose the current schema through it.
	if len(datastream.Schemas) == 0 && datastream.Schema != nil {
		datastream.Schemas = datastream.Schemas.Upsert(*datastream.Schema)
	}
	if datastream.Schema == nil && len(datastream.Schemas) > 0 {
		datastream.Schema = datastream.Schemas.ForFormat("application/json")
		if datastream.Schema == nil {
			datastream.Schema = &datastream.Schemas[0]
		}
	}
	return datastream.Schema, datastream.Schemas, nil
}

// UpdateSchema upserts the schema into the datastream's per-format registry
// (replacing the entry with the same obsFormat, adding it otherwise) and makes
// it the current schema.
func (r *DatastreamRepository) UpdateSchema(ctx context.Context, id string, schema *domains.DatastreamSchema) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domains.Datastream
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&existing).Error; err != nil {
			return err
		}
		replacement := existing
		replacement.Schema = schema
		if err := mergeDatastreamSchema(tx, &replacement, &existing); err != nil {
			return err
		}
		return tx.Model(&replacement).Select("schema", "schemas", "formats", "observed_properties", "result_type").Updates(&replacement).Error
	})
}

func (r *DatastreamRepository) applyFilters(query *gorm.DB, params *queryparams.DatastreamsQueryParams, systemID *string) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ?", params.IDs)
	}

	query = applyKeywordFilter(query, params.Q, "name", "description", "output_name")

	query = applyRangeFilter(query, "datastreams.phenomenon_time_start", "datastreams.phenomenon_time_end", params.PhenomenonTime)

	query = applyRangeFilter(query, "datastreams.result_time_start", "datastreams.result_time_end", params.ResultTime)

	if systemID != nil {
		query = query.Where("system_id IN ("+systemTreeSQL("?")+")", *systemID)
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

	return applyCQLFilter(query, &params.QueryParams)
}

func (r *DatastreamRepository) resolveReferences(ctx context.Context, tx *gorm.DB, datastream *domains.Datastream) error {
	var err error
	datastream.ProcedureID, datastream.DeploymentID, datastream.SamplingFeatureID, datastream.FeatureOfInterestID, err = resolveStreamReferences(ctx, tx, datastream.ProcedureLink, datastream.DeploymentLink, datastream.SamplingFeatureLink, datastream.FeatureOfInterest)
	return err
}
