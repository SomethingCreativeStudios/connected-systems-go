package repository

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// childCollection is a nested collection of a resource and the SQL condition,
// evaluated against the resource's row aliased "parent", under which it has
// members.
type childCollection struct {
	rel    string
	exists string
}

var (
	systemChildCollections = []childCollection{
		{"subsystems", "EXISTS (SELECT 1 FROM systems c WHERE c.parent_system_id = parent.id)"},
		{"samplingFeatures", "EXISTS (SELECT 1 FROM sampling_features c WHERE c.parent_system_id IN (" + systemTreeSQL("parent.id") + "))"},
		{"datastreams", "EXISTS (SELECT 1 FROM datastreams c WHERE c.system_id IN (" + systemTreeSQL("parent.id") + "))"},
		{"controlstreams", "EXISTS (SELECT 1 FROM control_streams c WHERE c.system_id IN (" + systemTreeSQL("parent.id") + "))"},
		{"deployments", "EXISTS (SELECT 1 FROM deployments c WHERE EXISTS (SELECT 1 FROM system_deployments sd WHERE sd.deployment_id = c.id AND sd.system_id = parent.id))"},
		{"procedures", "TRIM(COALESCE(parent.type_of_id, '')) <> ''"},
	}
	deploymentChildCollections = []childCollection{
		{"subdeployments", "EXISTS (SELECT 1 FROM deployments c WHERE c.parent_deployment_id = parent.id)"},
		{"samplingFeatures", "EXISTS (SELECT 1 FROM sampling_features WHERE " + deploymentSamplingCondition("parent.id") + ")"},
		{"featuresOfInterest", "EXISTS (SELECT 1 FROM (" + localFOIFeaturesSQL + ") AS features WHERE " + deploymentFeaturesCondition("parent.id") + ")"},
		{"datastreams", "EXISTS (SELECT 1 FROM datastreams c WHERE c.system_id IN (" + deploymentSystemsSQL("parent.id") + ") OR c.deployment_id IN (" + deploymentTreeSQL("parent.id") + "))"},
		{"controlstreams", "EXISTS (SELECT 1 FROM control_streams c WHERE c.system_id IN (" + deploymentSystemsSQL("parent.id") + ") OR c.deployment_id IN (" + deploymentTreeSQL("parent.id") + "))"},
	}
	procedureChildCollections = []childCollection{
		{"implementingSystems", "EXISTS (SELECT 1 FROM systems c WHERE c.type_of_id = parent.id)"},
	}
)

// ChildCollections reports, for each of the given systems, which nested
// collections have members.
func (r *SystemRepository) ChildCollections(ctx context.Context, ids []string) (map[string][]string, error) {
	return queryChildCollections(ctx, r.db, "systems", ids, systemChildCollections)
}

// ChildCollections reports, for each of the given deployments, which nested
// collections have members.
func (r *DeploymentRepository) ChildCollections(ctx context.Context, ids []string) (map[string][]string, error) {
	return queryChildCollections(ctx, r.db, "deployments", ids, deploymentChildCollections)
}

// ChildCollections reports, for each of the given procedures, which nested
// collections have members.
func (r *ProcedureRepository) ChildCollections(ctx context.Context, ids []string) (map[string][]string, error) {
	return queryChildCollections(ctx, r.db, "procedures", ids, procedureChildCollections)
}

// queryChildCollections answers for a whole page of resources in one query,
// with one boolean column per collection, instead of one query per resource
// and collection.
func queryChildCollections(ctx context.Context, db *gorm.DB, table string, ids []string, collections []childCollection) (map[string][]string, error) {
	result := make(map[string][]string, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	columns := []string{"parent.id"}
	for i, collection := range collections {
		columns = append(columns, fmt.Sprintf("(%s) AS has_%d", collection.exists, i))
	}
	rows, err := db.WithContext(ctx).
		Table(table+" AS parent").
		Select(strings.Join(columns, ", ")).
		Where("parent.id IN ?", ids).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id string
	has := make([]bool, len(collections))
	dest := []any{&id}
	for i := range has {
		dest = append(dest, &has[i])
	}
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		for i, collection := range collections {
			if has[i] {
				result[id] = append(result[id], collection.rel)
			}
		}
	}
	return result, rows.Err()
}
