-- +goose Up
-- Backfill single-valued legacy relationships before removing the duplicate
-- tables. Ambiguity fails the transaction and retains the legacy evidence.
-- +goose StatementBegin
DO $$
DECLARE relation text; child_table text; child_key text; parent_key text; parent_table text; conflict_count bigint;
BEGIN
 FOR relation, child_table, child_key, parent_key IN VALUES
  ('system_datastreams', 'datastreams', 'datastream_id', 'system_id'),
  ('system_controlstreams', 'control_streams', 'control_stream_id', 'system_id'),
  ('system_procedures', 'systems', 'system_id', 'type_of_id'),
  ('deployment_closures', 'deployments', 'descendant_id', 'parent_deployment_id')
 LOOP
  IF to_regclass(relation) IS NOT NULL THEN
   IF relation = 'deployment_closures' THEN
    EXECUTE 'CREATE TEMP TABLE legacy_relationships ON COMMIT DROP AS SELECT descendant_id AS child_id, ancestor_id AS parent_id FROM deployment_closures WHERE depth = 1';
   ELSIF relation = 'system_procedures' THEN
    EXECUTE format('CREATE TEMP TABLE legacy_relationships ON COMMIT DROP AS SELECT %I AS child_id, procedure_id AS parent_id FROM %I', child_key, relation);
   ELSE
    EXECUTE format('CREATE TEMP TABLE legacy_relationships ON COMMIT DROP AS SELECT %I AS child_id, system_id AS parent_id FROM %I', child_key, relation);
   END IF;
   parent_table := CASE relation WHEN 'system_procedures' THEN 'procedures' WHEN 'deployment_closures' THEN 'deployments' ELSE 'systems' END;
   EXECUTE format($query$SELECT COUNT(*) FROM legacy_relationships l LEFT JOIN %I c ON c.id = l.child_id LEFT JOIN %I p ON p.id = l.parent_id WHERE c.id IS NULL OR p.id IS NULL$query$, child_table, parent_table) INTO conflict_count;
   IF conflict_count > 0 THEN RAISE EXCEPTION 'Orphaned legacy relationships in %', relation; END IF;
   IF EXISTS (SELECT child_id FROM legacy_relationships GROUP BY child_id HAVING COUNT(DISTINCT parent_id) > 1) THEN
    RAISE EXCEPTION 'Ambiguous legacy relationships in %', relation;
   END IF;
   EXECUTE format($query$SELECT COUNT(*) FROM legacy_relationships l JOIN %I c ON c.id = l.child_id WHERE COALESCE(c.%I, '') <> '' AND c.%I <> l.parent_id$query$, child_table, parent_key, parent_key) INTO conflict_count;
   IF conflict_count > 0 THEN RAISE EXCEPTION 'Conflicting legacy relationships in %', relation; END IF;
   EXECUTE format($query$UPDATE %I c SET %I = l.parent_id FROM legacy_relationships l WHERE c.id = l.child_id AND COALESCE(c.%I, '') = ''$query$, child_table, parent_key, parent_key);
   DROP TABLE legacy_relationships;
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd

-- A system's procedure is systems.type_of_id, and its datastreams and control
-- streams are found by their system_id, so these copies are no longer kept.
DROP TABLE IF EXISTS system_datastreams;
DROP TABLE IF EXISTS system_controlstreams;
DROP TABLE IF EXISTS system_procedures;

-- Nothing reads the deployment closure table; subdeployments are found
-- through parent_deployment_id.
DROP TRIGGER IF EXISTS trg_deployments_closure_insert ON deployments;
DROP TRIGGER IF EXISTS trg_deployments_closure_update ON deployments;
DROP TRIGGER IF EXISTS trg_deployments_closure_delete ON deployments;
DROP FUNCTION IF EXISTS deployments_closure_after_insert();
DROP FUNCTION IF EXISTS deployments_closure_after_update();
DROP FUNCTION IF EXISTS deployments_closure_after_delete();
DROP TABLE IF EXISTS deployment_closures;
