package repository

// UNION makes these traversals deduplicate shared paths and terminate even if
// legacy rows contain a cycle. The root is included in each resource scope.
func systemTreeSQL(root string) string {
	return `WITH RECURSIVE members(id) AS (
 SELECT id FROM systems WHERE id = ` + root + `
 UNION SELECT s.id FROM systems s JOIN members m ON s.parent_system_id = m.id
 ) SELECT id FROM members`
}
func deploymentTreeSQL(root string) string {
	return `WITH RECURSIVE members(id) AS (
 SELECT id FROM deployments WHERE id = ` + root + `
 UNION SELECT d.id FROM deployments d JOIN members m ON d.parent_deployment_id = m.id
 ) SELECT id FROM members`
}
func deploymentSystemsSQL(root string) string {
	return `WITH RECURSIVE deps(id) AS (
 SELECT id FROM deployments WHERE id = ` + root + `
 UNION SELECT d.id FROM deployments d JOIN deps p ON d.parent_deployment_id = p.id
 ), members(id) AS (
 SELECT sd.system_id FROM system_deployments sd JOIN deps d ON d.id = sd.deployment_id
 UNION SELECT s.id FROM systems s JOIN members m ON s.parent_system_id = m.id
 ) SELECT id FROM members`
}

func deploymentSamplingCondition(root string) string {
	return `parent_system_id IN (` + deploymentSystemsSQL(root) + `) OR id IN (
 SELECT sampling_feature_id FROM datastreams WHERE system_id IN (` + deploymentSystemsSQL(root) + `) OR deployment_id IN (` + deploymentTreeSQL(root) + `)
 UNION SELECT sampling_feature_id FROM control_streams WHERE system_id IN (` + deploymentSystemsSQL(root) + `) OR deployment_id IN (` + deploymentTreeSQL(root) + `))`
}
func deploymentFeaturesCondition(root string) string {
	target := `features.canonical_path`
	return `EXISTS (SELECT 1 FROM sampling_features WHERE (` + deploymentSamplingCondition(root) + `) AND sampled_feature_id = features.id AND right('/' || (sampled_feature_link ->> 'href'), length(` + target + `)) = ` + target + `)
 OR EXISTS (SELECT 1 FROM datastreams WHERE (system_id IN (` + deploymentSystemsSQL(root) + `) OR deployment_id IN (` + deploymentTreeSQL(root) + `)) AND feature_of_interest_id = features.id AND right('/' || (feature_of_interest ->> 'href'), length(` + target + `)) = ` + target + `)
 OR EXISTS (SELECT 1 FROM control_streams WHERE (system_id IN (` + deploymentSystemsSQL(root) + `) OR deployment_id IN (` + deploymentTreeSQL(root) + `)) AND feature_of_interest_id = features.id AND right('/' || (feature_of_interest ->> 'href'), length(` + target + `)) = ` + target + `)`
}

// A GeoJSON read projection of locally stored feature resources. Canonical
// paths distinguish native resources from generic items with the same ID.
const localFOIFeaturesSQL = `
 SELECT id, collection_id, unique_identifier, name, description, date_time,
 valid_time_start, valid_time_end, geometry, properties, links, created_at, updated_at,
 '/collections/' || collection_id || '/items/' || id AS canonical_path FROM features
 UNION ALL
 SELECT id, 'systems', unique_identifier, name, description, NULL::timestamptz,
 valid_time_start, valid_time_end, geometry,
 jsonb_strip_nulls(jsonb_build_object('featureType',system_type,'assetType',asset_type,'systemKind@link',type_of)),
 jsonb_build_array(jsonb_build_object('rel','self','href','/systems/'||id,'type','application/geo+json')),
 created_at, updated_at, '/systems/' || id FROM systems
 UNION ALL
 SELECT id, 'samplingFeatures', unique_identifier, name, description, NULL::timestamptz,
 valid_time_start, valid_time_end, geometry,
 COALESCE(properties,'{}'::jsonb) || jsonb_strip_nulls(jsonb_build_object('featureType',feature_type,'sampledFeature@link',sampled_feature_link)),
 jsonb_build_array(jsonb_build_object('rel','self','href','/samplingFeatures/'||id,'type','application/geo+json')),
 created_at, updated_at, '/samplingFeatures/' || id FROM sampling_features
 UNION ALL
 SELECT id, 'deployments', unique_identifier, name, description, NULL::timestamptz,
 valid_time_start, valid_time_end, geometry,
 jsonb_build_object('featureType',deployment_type),
 jsonb_build_array(jsonb_build_object('rel','self','href','/deployments/'||id,'type','application/geo+json')),
 created_at, updated_at, '/deployments/' || id FROM deployments
 UNION ALL
 SELECT id, 'procedures', unique_identifier, name, description, NULL::timestamptz,
 valid_time_start, valid_time_end, NULL::geometry,
 COALESCE(properties,'{}'::jsonb) || jsonb_build_object('featureType',procedure_type),
 jsonb_build_array(jsonb_build_object('rel','self','href','/procedures/'||id,'type','application/geo+json')),
 created_at, updated_at, '/procedures/' || id FROM procedures`
