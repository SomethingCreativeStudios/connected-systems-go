package repository

import (
	"context"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

func (r *DatastreamRepository) ListDeployment(ctx context.Context, id string, params *queryparams.DatastreamsQueryParams) ([]*domains.Datastream, int64, error) {
	query := r.applyFilters(r.db.WithContext(ctx).Model(&domains.Datastream{}), params, nil)
	query = query.Where("system_id IN ("+deploymentSystemsSQL("?")+") OR deployment_id IN ("+deploymentTreeSQL("?")+")", id, id)
	return listPage(query, &params.QueryParams, idOrder("datastreams"), idAnchor[*domains.Datastream])
}
func (r *ControlStreamRepository) ListDeployment(ctx context.Context, id string, params *queryparams.ControlStreamsQueryParams) ([]*domains.ControlStream, int64, error) {
	query := r.applyFilters(r.db.WithContext(ctx).Model(&domains.ControlStream{}), params, nil)
	query = query.Where("system_id IN ("+deploymentSystemsSQL("?")+") OR deployment_id IN ("+deploymentTreeSQL("?")+")", id, id)
	return listPage(query, &params.QueryParams, idOrder("control_streams"), idAnchor[*domains.ControlStream])
}
func deploymentSamplingScope(query *gorm.DB, id string) *gorm.DB {
	return query.Where(deploymentSamplingCondition("?"), id, id, id, id, id)
}
func (r *SamplingFeatureRepository) ListDeployment(ctx context.Context, id string, params *queryparams.SamplingFeatureQueryParams) ([]*domains.SamplingFeature, int64, error) {
	query := r.applyFilters(r.db.WithContext(ctx).Model(&domains.SamplingFeature{}), params, nil)
	query = deploymentSamplingScope(query, id)
	return listPage(query, &params.QueryParams, idOrder("sampling_features"), idAnchor[*domains.SamplingFeature])
}
func (r *FeatureRepository) ListDeployment(ctx context.Context, id string, params *queryparams.FeatureQueryParams) ([]*domains.Feature, int64, error) {
	query := r.applyFilters(r.db.WithContext(ctx).Table("("+localFOIFeaturesSQL+") AS features"), params)
	// Match the collection as well as the ID. Generic collection item IDs are
	// scoped to their collection; a basename alone can identify another feature.
	query = query.Where(deploymentFeaturesCondition("?"), id, id, id, id, id, id, id, id, id)
	return listPage(query, &params.QueryParams, idOrder("features"), idAnchor[*domains.Feature])
}

// DeployedSystemsByIDs builds a read projection for a whole page. Direct
// memberships remain owned by the deployment that stores them.
func (r *DeploymentRepository) DeployedSystemsByIDs(ctx context.Context, ids []string) (map[string]domains.DeployedSystemItems, error) {
	result := make(map[string]domains.DeployedSystemItems)
	if len(ids) == 0 {
		return result, nil
	}
	type row struct {
		Root            string
		DeployedSystems domains.DeployedSystemItems
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`WITH RECURSIVE tree(root, id) AS (
 SELECT id, id FROM deployments WHERE id IN ?
 UNION SELECT t.root, d.id FROM deployments d JOIN tree t ON d.parent_deployment_id = t.id
 ) SELECT t.root, d.deployed_systems FROM tree t JOIN deployments d ON d.id = t.id ORDER BY t.root, d.id`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	seen := map[string]map[string]bool{}
	for _, row := range rows {
		if seen[row.Root] == nil {
			seen[row.Root] = map[string]bool{}
		}
		for _, item := range row.DeployedSystems {
			key := item.System.Href
			if path, local, err := localLinkPath(ctx, &item.System); err == nil && local {
				key = "/" + path
			}
			if !seen[row.Root][key] {
				seen[row.Root][key] = true
				result[row.Root] = append(result[row.Root], item)
			}
		}
	}
	return result, nil
}
