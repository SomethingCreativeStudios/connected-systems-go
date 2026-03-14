package repository

import (
	"fmt"

	"gorm.io/gorm"
)

// EnsureClosureSupport creates indexes, trigger functions and triggers to
// maintain a closure table for hierarchical relationships on a given table.
// Parameters:
// - db: gorm DB
// - table: the source table name (e.g. "deployments")
// - idCol: primary id column on source table (e.g. "id")
// - parentCol: parent FK column on source table (e.g. "parent_deployment_id")
// - closureTable: closure table name to maintain (e.g. "deployment_closures")
func EnsureClosureSupport(db *gorm.DB, table, idCol, parentCol, closureTable string) error {
	// unique index on ancestor/descendant
	idxSQL := fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS idx_%s_ancestor_descendant_unique ON %s (ancestor_id, descendant_id);", closureTable, closureTable)
	if err := db.Exec(idxSQL).Error; err != nil {
		return err
	}

	// insert trigger function
	insertFn := fmt.Sprintf(`
CREATE OR REPLACE FUNCTION %s_closure_after_insert() RETURNS trigger AS $$
BEGIN
    INSERT INTO %s(ancestor_id, descendant_id, depth)
    VALUES (NEW.%s, NEW.%s, 0)
    ON CONFLICT DO NOTHING;

    IF NEW.%s IS NOT NULL THEN
        INSERT INTO %s(ancestor_id, descendant_id, depth)
        SELECT ancestor_id, NEW.%s, depth + 1 FROM %s WHERE descendant_id = NEW.%s
        ON CONFLICT DO NOTHING;

        INSERT INTO %s(ancestor_id, descendant_id, depth)
        VALUES (NEW.%s, NEW.%s, 1)
        ON CONFLICT DO NOTHING;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`, table, closureTable, idCol, idCol, parentCol, closureTable, idCol, closureTable, parentCol, closureTable, parentCol, idCol)

	if err := db.Exec(insertFn).Error; err != nil {
		return err
	}

	// update trigger function
	updateFn := fmt.Sprintf(`
CREATE OR REPLACE FUNCTION %s_closure_after_update() RETURNS trigger AS $$
BEGIN
    IF OLD.%s IS DISTINCT FROM NEW.%s THEN
        DELETE FROM %s
        WHERE descendant_id IN (
            SELECT dc.descendant_id FROM %s dc WHERE dc.ancestor_id = OLD.%s
        )
        AND ancestor_id IN (
            SELECT ac.ancestor_id FROM %s ac WHERE ac.descendant_id = OLD.%s AND ac.ancestor_id != OLD.%s
        );

        IF NEW.%s IS NOT NULL THEN
            INSERT INTO %s(ancestor_id, descendant_id, depth)
            SELECT ac.ancestor_id, dc.descendant_id, ac.depth + dc.depth + 1
            FROM %s ac, %s dc
            WHERE ac.descendant_id = NEW.%s AND dc.ancestor_id = NEW.%s
            ON CONFLICT DO NOTHING;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`, table, parentCol, parentCol, closureTable, closureTable, idCol, closureTable, idCol, idCol, parentCol, closureTable, closureTable, closureTable, parentCol, idCol)

	if err := db.Exec(updateFn).Error; err != nil {
		return err
	}

	// delete trigger function
	deleteFn := fmt.Sprintf(`
CREATE OR REPLACE FUNCTION %s_closure_after_delete() RETURNS trigger AS $$
BEGIN
    DELETE FROM %s WHERE ancestor_id = OLD.%s OR descendant_id = OLD.%s;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
`, table, closureTable, idCol, idCol)

	if err := db.Exec(deleteFn).Error; err != nil {
		return err
	}

	// attach triggers
	_ = db.Exec(fmt.Sprintf("DROP TRIGGER IF EXISTS trg_%s_closure_insert ON %s;", table, table))
	if err := db.Exec(fmt.Sprintf("CREATE TRIGGER trg_%s_closure_insert AFTER INSERT ON %s FOR EACH ROW EXECUTE FUNCTION %s_closure_after_insert();", table, table, table)).Error; err != nil {
		return err
	}

	_ = db.Exec(fmt.Sprintf("DROP TRIGGER IF EXISTS trg_%s_closure_update ON %s;", table, table))
	if err := db.Exec(fmt.Sprintf("CREATE TRIGGER trg_%s_closure_update AFTER UPDATE OF %s ON %s FOR EACH ROW EXECUTE FUNCTION %s_closure_after_update();", table, parentCol, table, table)).Error; err != nil {
		return err
	}

	_ = db.Exec(fmt.Sprintf("DROP TRIGGER IF EXISTS trg_%s_closure_delete ON %s;", table, table))
	if err := db.Exec(fmt.Sprintf("CREATE TRIGGER trg_%s_closure_delete AFTER DELETE ON %s FOR EACH ROW EXECUTE FUNCTION %s_closure_after_delete();", table, table, table)).Error; err != nil {
		return err
	}

	return nil
}

// EnsureDeleteReparentSupport creates a trigger function that, when a row is
// deleted from `table`, reparents its immediate children to the deleted row's
// parent (i.e. sets child's parent_col = OLD.parent_col). This preserves the
// hierarchy for direct children when a node is removed.
func EnsureDeleteReparentSupport(db *gorm.DB, table, idCol, parentCol string) error {
	// create the trigger function
	deleteReparentFn := fmt.Sprintf(`
CREATE OR REPLACE FUNCTION %s_reparent_after_delete() RETURNS trigger AS $$
BEGIN
	UPDATE %s SET %s = OLD.%s WHERE %s = OLD.%s;
	RETURN OLD;
END;
$$ LANGUAGE plpgsql;
`, table, table, parentCol, parentCol, parentCol, idCol)

	if err := db.Exec(deleteReparentFn).Error; err != nil {
		return err
	}

	// attach trigger
	_ = db.Exec(fmt.Sprintf("DROP TRIGGER IF EXISTS trg_%s_reparent_delete ON %s;", table, table))
	if err := db.Exec(fmt.Sprintf("CREATE TRIGGER trg_%s_reparent_delete AFTER DELETE ON %s FOR EACH ROW EXECUTE FUNCTION %s_reparent_after_delete();", table, table, table)).Error; err != nil {
		return err
	}

	return nil
}
