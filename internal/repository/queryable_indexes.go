package repository

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/queryables"
	"gorm.io/gorm"
)

// ensureCollectionQueryableIndexes provisions collection-scoped expression
// indexes for the only JSON paths the queryables catalog permits. Identifiers
// are deterministic and all literals are generated from validated metadata.
func ensureCollectionQueryableIndexes(db *gorm.DB, collectionID string, catalog common_shared.Properties) error {
	if err := queryables.ValidateCustomCatalog(catalog); err != nil {
		return err
	}
	properties, _ := catalog["properties"].(map[string]any)
	for name, raw := range properties {
		schema := raw.(map[string]any)
		typ, _ := schema["type"].(string)
		format, _ := schema["format"].(string)
		index := customQueryableIndexName(collectionID, name)
		key := sqlString(name)
		collection := sqlString(collectionID)
		var statement string
		switch typ {
		case "string":
			if format == "date-time" {
				// PostgreSQL's text-to-timestamptz cast is STABLE rather than
				// IMMUTABLE (it observes TimeZone), so it is illegal in an
				// expression index. RFC 3339 UTC strings have chronological
				// lexicographic order; index their validated stored representation.
				statement = fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON features ((properties ->> %s)) WHERE collection_id = %s`, index, key, collection)
				break
			}
			statement = fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON features USING GIN ((properties ->> %s) gin_trgm_ops) WHERE collection_id = %s`, index, key, collection)
		case "array":
			statement = fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON features USING GIN (cql2_jsonb_text_array(properties -> %s)) WHERE collection_id = %s`, index, key, collection)
		case "number":
			statement = fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON features ((NULLIF(properties ->> %s, '')::numeric)) WHERE collection_id = %s`, index, key, collection)
		case "integer":
			statement = fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON features ((NULLIF(properties ->> %s, '')::bigint)) WHERE collection_id = %s`, index, key, collection)
		case "boolean":
			statement = fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON features ((NULLIF(properties ->> %s, '')::boolean)) WHERE collection_id = %s`, index, key, collection)
		default:
			return fmt.Errorf("queryable %q cannot be safely indexed", name)
		}
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("create queryable index for %q: %w", name, err)
		}
	}
	return nil
}

func customQueryableIndexName(collectionID, property string) string {
	sum := sha256.Sum256([]byte(collectionID + "\x00" + property))
	return "idx_features_q_" + fmt.Sprintf("%x", sum[:8])
}

func sqlString(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
