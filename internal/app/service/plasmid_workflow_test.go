package service

import (
	"errors"
	"testing"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/stretchr/testify/require"
)

func TestPlasmidWorkflowPredicates(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	t.Run("isNotFoundError", func(_ *testing.T) {
		assert.True(isNotFoundError(errors.New("could not find plasmid with ID 123")))
		assert.False(isNotFoundError(errors.New("some other error")))
		assert.False(isNotFoundError(nil))
	})

	t.Run("hasEnoughResults", func(_ *testing.T) {
		limit := int64(10)
		lctx := withPlasmidCollectionData{
			withStockDocList: withStockDocList{
				withValidatedFilter: withValidatedFilter{
					listPlasmidsContext: listPlasmidsContext{limit: limit},
				},
			},
		}

		// Not enough (exactly limit — no next page)
		lctx.collectionData = make([]*stock.PlasmidCollection_Data, 10)
		assert.False(hasEnoughResults(lctx))

		// Just enough (limit + 1 — next page exists)
		lctx.collectionData = make([]*stock.PlasmidCollection_Data, 11)
		assert.True(hasEnoughResults(lctx))

		// Empty
		lctx.collectionData = []*stock.PlasmidCollection_Data{}
		assert.False(hasEnoughResults(lctx))
	})

	t.Run("shouldTrimLastItem", func(_ *testing.T) {
		ctx := withNextCursor{
			nextCursor: 12345,
			withPlasmidCollectionData: withPlasmidCollectionData{
				collectionData: []*stock.PlasmidCollection_Data{{}},
			},
		}
		assert.True(shouldTrimLastItem(ctx))

		ctx.nextCursor = 0
		assert.False(shouldTrimLastItem(ctx))

		ctx.nextCursor = 12345
		ctx.collectionData = []*stock.PlasmidCollection_Data{}
		assert.False(shouldTrimLastItem(ctx))
	})
}
