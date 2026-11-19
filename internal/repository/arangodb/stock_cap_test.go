package arangodb

import (
	"strings"
	"testing"

	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/stretchr/testify/require"
)

// assertStockBranchCapOrder checks that a stock branch ranks and caps the
// view rows before its owner traversal, and trims after the entity filter.
func assertStockBranchCapOrder(t *testing.T, branch string) {
	t.Helper()
	assert := require.New(t)
	sort := strings.Index(branch, "SORT BM25")
	capAt := strings.Index(branch, "LIMIT @stock_cap")
	traverse := strings.Index(branch, "OUTBOUND d GRAPH")
	entity := strings.Index(branch, "FILTER @entity")
	trim := strings.Index(branch, "LIMIT @limit")
	assert.GreaterOrEqual(sort, 0, "stock branch must sort")
	assert.Less(sort, capAt, "SORT must precede the cap")
	assert.Less(capAt, traverse, "the cap must precede the owner traversal")
	assert.Less(traverse, entity, "the entity filter must follow the traversal")
	assert.Less(entity, trim, "the trim must follow the entity filter")
}

func TestStockBranchCap(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		entity repository.StockEntityFilter
		want   int
	}{
		{"no filter keeps the limit", 50, "", 50},
		{"strain filter over-fetches", 50, repository.EntityStrain, 250},
		{"plasmid filter over-fetches", 10, repository.EntityPlasmid, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, stockBranchCap(tt.limit, tt.entity))
		})
	}
}
