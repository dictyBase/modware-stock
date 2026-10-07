package service

// One real round trip for the full search handler over a buffer
// connection. The helpers come from strain_test_helpers.go and
// bufconn_test_helpers.go of this package; the buffer connection goes
// through the shared newBufconnClient helper.

import (
	"context"
	"testing"
	"time"

	"github.com/dictyBase/arangomanager/testarango"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	arangodb "github.com/dictyBase/modware-stock/internal/repository/arangodb"
	"github.com/stretchr/testify/require"
)

func newFullSearchArangoEnv(
	t *testing.T,
) (stock.StockServiceClient, *require.Assertions) {
	t.Helper()
	assert := require.New(t)
	tra, err := testarango.NewTestArangoFromEnv(true)
	assert.NoErrorf(err,
		"expect no error creating a test database, received %s", err)
	repo, err := arangodb.NewStockRepo(
		getConnectParamsFromDb(tra),
		getCollectionParams(),
		getOntoParams(),
	)
	assert.NoErrorf(err,
		"expect no error building the repository, received %s", err)
	assert.NoError(loadData(tra), "expect no error loading the ontology")
	svc := setupTestService(repo)
	client := newBufconnClient(t, svc)
	t.Cleanup(func() {
		_ = repo.Dbh().Drop()
	})

	return client, assert
}

func TestSearchStockEndToEnd(t *testing.T) {
	client, assert := newFullSearchArangoEnv(t)
	ctx := context.Background()
	created, err := client.CreateStrain(ctx, &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type: "strain",
			Attributes: &stock.NewStrainAttributes{
				CreatedBy:           testUserEmail,
				UpdatedBy:           testUserEmail,
				Depositor:           testUserEmail,
				Summary:             "the end to end forms culminants summary",
				EditableSummary:     "the end to end forms culminants summary",
				Label:               testStrainLabel,
				Species:             testSpecies,
				Genes:               []string{"DDB_G0348394"},
				Names:               []string{"gammaS13"},
				DictyStrainProperty: "general strain",
			},
		},
	})
	assert.NoErrorf(err, "expect no error creating the strain, received %s", err)
	assert.NotEmpty(created.Data.Id, "expect a created strain")
	var res *stock.StockSearchResultCollection
	assert.Eventually(func() bool {
		got, serr := client.SearchStock(ctx,
			&stock.StockSearchParameters{
				Data: &stock.StockSearchParameters_Data{
					Type: autocompleteStockType,
					Attributes: &stock.StockSearchAttributes{
						Query: "ys1",
					},
				},
			})
		if serr != nil || len(got.Data) == 0 {
			return false
		}
		res = got
		return true
	}, 20*time.Second, 500*time.Millisecond,
		"the view must commit the created strain")
	assert.Len(res.Data, 1, "expect one result")
	row := res.Data[0]
	assert.Equal(created.Data.Id, row.Id)
	assert.Equal(stock.StockEntity_STOCK_ENTITY_STRAIN, row.Entity)
	assert.Equal(stock.StockSearchField_STOCK_SEARCH_FIELD_LABEL, row.Field)
	assert.Equal(testStrainLabel, row.DisplayText)
	assert.Equal(testStrainLabel, row.StrainLabel)
	assert.GreaterOrEqual(row.Score, float64(1000))
	assert.Equal(
		"the end to end forms culminants summary",
		row.Summary,
		"expect the complete stored summary",
	)
	assert.Equal(int64(1), res.Meta.Total)
	assert.Equal(int64(50), res.Meta.Limit)
	assert.Equal(int64(0), res.Meta.NextCursor)
	// A phrase query over the summary ranks the created strain through
	// the phrase branch.
	phrase, perr := client.SearchStock(ctx,
		&stock.StockSearchParameters{
			Data: &stock.StockSearchParameters_Data{
				Type: autocompleteStockType,
				Attributes: &stock.StockSearchAttributes{
					Query: "forms culminants",
				},
			},
		})
	assert.NoErrorf(perr, "expect no error for the phrase query, received %s", perr)
	assert.NotEmpty(phrase.Data, "expect the phrase query to match")
	assert.Equal(created.Data.Id, phrase.Data[0].Id)
	assert.Equal(stock.StockSearchField_STOCK_SEARCH_FIELD_SUMMARY,
		phrase.Data[0].Field)
	assert.GreaterOrEqual(phrase.Data[0].Score, float64(500))
}
