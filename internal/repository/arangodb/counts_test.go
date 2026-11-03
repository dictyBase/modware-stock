package arangodb

import (
	"context"
	"testing"

	E "github.com/IBM/fp-go/v2/either"
	F "github.com/IBM/fp-go/v2/function"
	ioeutils "github.com/dictyBase/fp-go-loom/ioeitherutils"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/repository"
)

func TestCountStocksEmptyDatabase(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)

	res := F.Pipe1(
		repo.CountStocks(context.Background()),
		ioeutils.ToEither[error, *repository.StockCounts],
	)
	assert.True(E.IsRight(res), "expect CountStocks to succeed on empty database")
	counts := F.Pipe1(
		res,
		E.GetOrElse(F.Constant1[error]((*repository.StockCounts)(nil))),
	)
	assert.NotNil(counts)
	assert.Equal(int64(0), counts.Strains.Total)
	assert.Equal(
		[]repository.SubtypeCount{
			{Name: subtypeRegular, Count: 0},
			{Name: subtypeGwdi, Count: 0},
			{Name: subtypeBacterial, Count: 0},
		},
		counts.Strains.Subtypes,
	)
	assert.Equal(int64(0), counts.Plasmids.Total)
	assert.Equal(
		[]repository.SubtypeCount{
			{Name: subtypeRegular, Count: 0},
			{Name: subtypeGoldenBraid, Count: 0},
		},
		counts.Plasmids.Subtypes,
	)
}

func TestCountStocksGroupsStrainsAndPlasmidsLikeSearchStocks(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)

	_, err := repo.AddStrain(newTestStrain(testEmailCostanza, General))
	assert.NoError(err)
	_, err = repo.AddStrain(newTestStrain(testEmailCostanza, General))
	assert.NoError(err)
	_, err = repo.AddStrain(newTestStrain(testEmailCostanza, Gwdi))
	assert.NoError(err)
	_, err = repo.AddStrain(newTestStrain(testEmailCostanza, Bacterial))
	assert.NoError(err)

	untypedStrain, err := repo.AddStrain(newTestStrain(testEmailCostanza, General))
	assert.NoError(err)
	err = repo.Dbh().Do(
		`FOR e IN stock_term_test FILTER e._from == @from REMOVE e IN stock_term_test`,
		map[string]any{nameFrom: "stock_test/" + untypedStrain.Key},
	)
	assert.NoError(err)

	addFullSearchPlasmid(assert, repo, func(a *stock.NewPlasmidAttributes) {
		a.Name = "pVec1"
		a.DictyPlasmidProperty = "vector"
	})
	addFullSearchPlasmid(assert, repo, func(a *stock.NewPlasmidAttributes) {
		a.Name = "pClone1"
		a.DictyPlasmidProperty = OntologyTermCloningVector
	})
	addFullSearchPlasmid(assert, repo, func(a *stock.NewPlasmidAttributes) {
		a.Name = "pGB1"
		a.DictyPlasmidProperty = goldenBraidPlasmidTag
	})
	addFullSearchPlasmid(assert, repo, func(a *stock.NewPlasmidAttributes) {
		a.Name = "pGB2"
		a.DictyPlasmidProperty = goldenBraidPlasmidTag
	})

	res := F.Pipe1(
		repo.CountStocks(context.Background()),
		ioeutils.ToEither[error, *repository.StockCounts],
	)
	assert.True(E.IsRight(res), "expect CountStocks to succeed")
	counts := F.Pipe1(
		res,
		E.GetOrElse(F.Constant1[error]((*repository.StockCounts)(nil))),
	)
	assert.NotNil(counts)
	assert.Equal(int64(5), counts.Strains.Total)
	assert.Equal(
		[]repository.SubtypeCount{
			{Name: subtypeRegular, Count: 3},
			{Name: subtypeGwdi, Count: 1},
			{Name: subtypeBacterial, Count: 1},
		},
		counts.Strains.Subtypes,
	)
	assert.Equal(int64(4), counts.Plasmids.Total)
	assert.Equal(
		[]repository.SubtypeCount{
			{Name: subtypeRegular, Count: 2},
			{Name: subtypeGoldenBraid, Count: 2},
		},
		counts.Plasmids.Subtypes,
	)
}

func TestCountStocksCanceledContext(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := F.Pipe1(
		repo.CountStocks(ctx),
		ioeutils.ToEither[error, *repository.StockCounts],
	)
	assert.True(E.IsLeft(res), "expect canceled context to fail")
}
