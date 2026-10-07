package service

// One real round trip for the autocomplete handler over a buffer
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

func newAutocompleteArangoEnv(
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

func TestAutocompleteStockEndToEnd(t *testing.T) {
	client, assert := newAutocompleteArangoEnv(t)
	ctx := context.Background()
	created, err := client.CreateStrain(ctx, &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type: "strain",
			Attributes: &stock.NewStrainAttributes{
				CreatedBy:           testUserEmail,
				UpdatedBy:           testUserEmail,
				Depositor:           testUserEmail,
				Summary:             testStrainTag,
				EditableSummary:     testStrainTag,
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
	var res *stock.StockSuggestionCollection
	assert.Eventually(func() bool {
		got, qerr := client.AutocompleteStock(ctx,
			&stock.StockAutocompleteParameters{
				Data: &stock.StockAutocompleteParameters_Data{
					Type: autocompleteStockType,
					Attributes: &stock.StockAutocompleteAttributes{
						Query: "ys1",
					},
				},
			})
		if qerr != nil || len(got.Data) == 0 {
			return false
		}
		res = got
		return true
	}, 20*time.Second, 500*time.Millisecond,
		"the view must commit the created strain")
	assert.Len(res.Data, 1, "expect one suggestion")
	sug := res.Data[0]
	assert.Equal(created.Data.Id, sug.Id)
	assert.Equal(stock.StockEntity_STOCK_ENTITY_STRAIN, sug.Entity)
	assert.Equal(stock.StockSearchField_STOCK_SEARCH_FIELD_LABEL, sug.Field)
	assert.Equal(testStrainLabel, sug.DisplayText)
	assert.GreaterOrEqual(sug.Score, float64(1000))
	assert.Equal(int64(1), res.Meta.Total)
	assert.Equal(int64(5), res.Meta.Limit)
	assert.Equal(int64(0), res.Meta.NextCursor)
}
