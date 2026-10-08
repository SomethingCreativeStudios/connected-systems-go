package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGORMPlaceholdersPreserveBoundArgumentOrder(t *testing.T) {
	sql, args := gormPlaceholders(`("systems"."name" = $2 OR "systems"."id" = $1 OR "systems"."name" = $2)`, []any{"id-1", "north"})
	require.Equal(t, `("systems"."name" = ? OR "systems"."id" = ? OR "systems"."name" = ?)`, sql)
	require.Equal(t, []any{"north", "id-1", "north"}, args)
}
