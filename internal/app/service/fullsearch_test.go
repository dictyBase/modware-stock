package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/stretchr/testify/require"
	"io"

	IOE "github.com/IBM/fp-go/ioeither"
	IOEV2 "github.com/IBM/fp-go/v2/ioeither"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-obograph/storage"
	"github.com/dictyBase/modware-stock/internal/model"
)

// fullSearchStubRepo implements every method of
// repository.StockRepository with no-ops, so the handler tests run
// without a database.
type fullSearchStubRepo struct {
	hits   []*repository.FullSearchResult
	docs   []*model.StockDoc
	err    error
	got    *repository.FullSearchQuery
	gotCtx context.Context
}

func (sr *fullSearchStubRepo) GetStrain(string) (*model.StockDoc, error) {
	return nil, nil
}
func (sr *fullSearchStubRepo) GetPlasmid(string) IOE.IOEither[error, *model.StockDoc] {
	return IOE.Left[*model.StockDoc](errors.New("not used"))
}
func (sr *fullSearchStubRepo) AddStrain(*stock.NewStrain) (*model.StockDoc, error) {
	return nil, nil
}
func (sr *fullSearchStubRepo) AddPlasmid(*stock.NewPlasmid) IOE.IOEither[error, *model.StockDoc] {
	return IOE.Left[*model.StockDoc](errors.New("not used"))
}
func (sr *fullSearchStubRepo) EditStrain(*stock.StrainUpdate) (*model.StockDoc, error) {
	return nil, nil
}
func (sr *fullSearchStubRepo) EditPlasmid(*stock.PlasmidUpdate) IOE.IOEither[error, *model.StockDoc] {
	return IOE.Left[*model.StockDoc](errors.New("not used"))
}
func (sr *fullSearchStubRepo) ListStrains(
	ctx context.Context,
	_ *stock.StockParameters,
) ([]*model.StockDoc, error) {
	sr.gotCtx = ctx
	return sr.docs, sr.err
}
func (sr *fullSearchStubRepo) ListStrainsByIDs(
	ctx context.Context,
	_ *stock.StockIdList,
) ([]*model.StockDoc, error) {
	sr.gotCtx = ctx
	return sr.docs, sr.err
}
func (sr *fullSearchStubRepo) ListPlasmids(
	ctx context.Context,
	_ *stock.StockParameters,
) IOE.IOEither[error, []*model.StockDoc] {
	sr.gotCtx = ctx
	if sr.err != nil {
		return IOE.Left[[]*model.StockDoc](sr.err)
	}
	return IOE.Of[error](sr.docs)
}
func (sr *fullSearchStubRepo) LoadStrain(string, *stock.ExistingStrain) (*model.StockDoc, error) {
	return nil, nil
}
func (sr *fullSearchStubRepo) LoadPlasmid(string, *stock.ExistingPlasmid) IOE.IOEither[error, *model.StockDoc] {
	return IOE.Left[*model.StockDoc](errors.New("not used"))
}
func (sr *fullSearchStubRepo) RemoveStock(string) error { return nil }
func (sr *fullSearchStubRepo) SearchStock(
	ctx context.Context,
	params *repository.FullSearchQuery,
) IOEV2.IOEither[error, []*repository.FullSearchResult] {
	sr.gotCtx = ctx
	sr.got = params
	if sr.err != nil {
		return IOEV2.Left[[]*repository.FullSearchResult](sr.err)
	}
	return IOEV2.Of[error](sr.hits)
}
func (sr *fullSearchStubRepo) AutocompleteStock(
	ctx context.Context,
	_ *repository.AutocompleteQuery,
) IOEV2.IOEither[error, []*repository.Suggestion] {
	sr.gotCtx = ctx
	return IOEV2.Of[error]([]*repository.Suggestion(nil))
}
func (sr *fullSearchStubRepo) Dbh() *manager.Database { return nil }
func (sr *fullSearchStubRepo) LoadOboJSON(io.Reader) (*storage.UploadInformation, error) {
	return nil, nil
}

// fullSearchRequest builds a search request with the given attributes.
func fullSearchRequest(
	query string,
	limit int64,
	entity stock.StockEntity,
) *stock.StockSearchParameters {
	return &stock.StockSearchParameters{
		Data: &stock.StockSearchParameters_Data{
			Type: autocompleteStockType,
			Attributes: &stock.StockSearchAttributes{
				Query:  query,
				Limit:  limit,
				Entity: entity,
			},
		},
	}
}

func TestSearchStockHandlerMapsResults(t *testing.T) {
	assert := require.New(t)
	stub := &fullSearchStubRepo{
		hits: []*repository.FullSearchResult{
			{
				ID:          testStockID,
				Field:       fullSearchFieldSummary,
				DisplayText: "the mutant forms culminants under starvation",
				Summary:     "the mutant forms culminants under starvation",
				StrainLabel: testStrainLabel,
				StockType:   "REMI-seq",
				Entity:      repository.EntityStrain,
				Score:       500.5,
			},
			{
				ID:        testPlasmidID,
				Field:     autocompleteFieldName,
				StockType: "GB vector",
				Entity:    repository.EntityPlasmid,
				Score:     1000,
			},
		},
	}
	svc := setupTestService(stub)
	res, err := svc.SearchStock(
		context.Background(),
		fullSearchRequest("culminants", 50, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.NoError(err, "expect no error")
	assert.Len(res.Data, 2, "expect two results")
	first := res.Data[0]
	assert.Equal(testStockID, first.Id)
	assert.Equal(stock.StockSearchField_STOCK_SEARCH_FIELD_SUMMARY, first.Field)
	assert.Equal(stock.StockEntity_STOCK_ENTITY_STRAIN, first.Entity)
	assert.Equal("the mutant forms culminants under starvation", first.DisplayText)
	assert.Equal("the mutant forms culminants under starvation", first.Summary)
	assert.Equal(testStrainLabel, first.StrainLabel)
	assert.Equal("REMI-seq", first.StockType)
	assert.Equal(500.5, first.Score)
	second := res.Data[1]
	assert.Equal(stock.StockSearchField_STOCK_SEARCH_FIELD_NAME, second.Field)
	assert.Equal(stock.StockEntity_STOCK_ENTITY_PLASMID, second.Entity)
	assert.Empty(second.StrainLabel, "a plasmid result keeps an empty label")
	assert.Equal("GB vector", second.StockType)
	assert.Equal(int64(2), res.Meta.Total)
	assert.Equal(int64(50), res.Meta.Limit)
	assert.Equal(int64(0), res.Meta.NextCursor)
}

func TestSearchStockHandlerPassesThroughQueryAndEntity(t *testing.T) {
	assert := require.New(t)
	stub := &fullSearchStubRepo{}
	svc := setupTestService(stub)
	_, err := svc.SearchStock(
		context.Background(),
		fullSearchRequest(
			"  culminants  ", 10, stock.StockEntity_STOCK_ENTITY_PLASMID,
		),
	)
	assert.NoError(err, "expect no error")
	assert.Equal("culminants", stub.got.Query, "expect the trimmed query")
	assert.Equal(repository.EntityPlasmid, stub.got.Entity,
		"expect the mapped entity filter")
	assert.Equal(10, stub.got.Limit, "expect the requested limit")
}

func TestSearchStockHandlerDefaultsLimitToFifty(t *testing.T) {
	assert := require.New(t)
	stub := &fullSearchStubRepo{}
	svc := setupTestService(stub)
	res, err := svc.SearchStock(
		context.Background(),
		fullSearchRequest("culminants", 0, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.NoError(err, "expect no error")
	assert.Equal(50, stub.got.Limit, "expect the default limit of 50")
	assert.Equal(int64(50), res.Meta.Limit)
}

func TestSearchStockHandlerClampsLimitAboveFifty(t *testing.T) {
	assert := require.New(t)
	stub := &fullSearchStubRepo{}
	svc := setupTestService(stub)
	res, err := svc.SearchStock(
		context.Background(),
		fullSearchRequest("culminants", 80, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.NoError(err, "expect no error")
	assert.Equal(50, stub.got.Limit, "expect the clamped limit of 50")
	assert.Equal(int64(50), res.Meta.Limit)
}

func TestSearchStockHandlerRejectsNilData(t *testing.T) {
	assert := require.New(t)
	svc := setupTestService(&fullSearchStubRepo{})
	_, err := svc.SearchStock(context.Background(), &stock.StockSearchParameters{})
	assert.Equal(
		codes.InvalidArgument,
		status.Code(err),
		"expect an invalid-argument error for a nil data",
	)
}

func TestSearchStockHandlerRejectsNilAttributes(t *testing.T) {
	assert := require.New(t)
	svc := setupTestService(&fullSearchStubRepo{})
	_, err := svc.SearchStock(context.Background(), &stock.StockSearchParameters{
		Data: &stock.StockSearchParameters_Data{Type: autocompleteStockType},
	})
	assert.Equal(
		codes.InvalidArgument,
		status.Code(err),
		"expect an invalid-argument error for nil attributes",
	)
}

func TestSearchStockHandlerRejectsShortQuery(t *testing.T) {
	assert := require.New(t)
	svc := setupTestService(&fullSearchStubRepo{})
	_, err := svc.SearchStock(
		context.Background(),
		fullSearchRequest("a", 50, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.Equal(
		codes.InvalidArgument,
		status.Code(err),
		"expect an invalid-argument error for a 1-character query",
	)
}

func TestSearchStockHandlerRejectsWhitespaceQuery(t *testing.T) {
	assert := require.New(t)
	svc := setupTestService(&fullSearchStubRepo{})
	_, err := svc.SearchStock(
		context.Background(),
		fullSearchRequest("  ", 50, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.Equal(
		codes.InvalidArgument,
		status.Code(err),
		"expect an invalid-argument error for a whitespace-only query",
	)
}

func TestSearchStockHandlerRejectsLimitAboveHundred(t *testing.T) {
	assert := require.New(t)
	svc := setupTestService(&fullSearchStubRepo{})
	_, err := svc.SearchStock(
		context.Background(),
		fullSearchRequest("culminants", 101, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.Equal(
		codes.InvalidArgument,
		status.Code(err),
		"expect an invalid-argument error for a limit above 100",
	)
}

func TestSearchStockHandlerRejectsUndefinedEntity(t *testing.T) {
	assert := require.New(t)
	svc := setupTestService(&fullSearchStubRepo{})
	_, err := svc.SearchStock(
		context.Background(),
		fullSearchRequest("culminants", 50, stock.StockEntity(9)),
	)
	assert.Equal(
		codes.InvalidArgument,
		status.Code(err),
		"expect an invalid-argument error for an undefined entity",
	)
}

func TestSearchStockHandlerReturnsEmptyCollection(t *testing.T) {
	assert := require.New(t)
	svc := setupTestService(&fullSearchStubRepo{})
	res, err := svc.SearchStock(
		context.Background(),
		fullSearchRequest("zzzqqq", 50, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.NoError(err, "an empty result is not an error")
	assert.Empty(res.Data, "expect an empty data list")
	assert.Equal(int64(0), res.Meta.Total)
	assert.Equal(int64(50), res.Meta.Limit)
	assert.Equal(int64(0), res.Meta.NextCursor)
}

func TestSearchStockHandlerKeepsEmptySummary(t *testing.T) {
	assert := require.New(t)
	stub := &fullSearchStubRepo{
		hits: []*repository.FullSearchResult{
			{
				ID:          testPlasmidID,
				Field:       autocompleteFieldName,
				DisplayText: "pDM304",
				Entity:      repository.EntityPlasmid,
				Score:       1000,
			},
		},
	}
	svc := setupTestService(stub)
	res, err := svc.SearchStock(
		context.Background(),
		fullSearchRequest("pdm", 50, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.NoError(err, "expect no error")
	require.Len(t, res.Data, 1)
	assert.Empty(res.Data[0].Summary, "expect an empty summary string")
	assert.NotPanics(func() { _ = res.Data[0].Summary })
}

func TestSearchStockHandlerMapsRepositoryError(t *testing.T) {
	assert := require.New(t)
	svc := setupTestService(&fullSearchStubRepo{
		err: errors.New("stub repository error"),
	})
	_, err := svc.SearchStock(
		context.Background(),
		fullSearchRequest("culminants", 50, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.Error(err, "expect the repository error to surface")
	// aphgrpc.HandleGetError produces codes.Internal for a plain error;
	// the same helper is what the other read handlers use.
	assert.Equal(codes.Internal, status.Code(err),
		"expect the code that aphgrpc.HandleGetError produces")
}

// TestFullSearchFieldMapping proves that every one of the 10 searched
// field labels maps to a defined StockSearchField and an unknown label
// maps to UNSPECIFIED.
func TestFullSearchFieldMapping(t *testing.T) {
	assert := require.New(t)
	expected := map[string]stock.StockSearchField{
		fullSearchFieldSummary:   stock.StockSearchField_STOCK_SEARCH_FIELD_SUMMARY,
		fullSearchFieldDepositor: stock.StockSearchField_STOCK_SEARCH_FIELD_DEPOSITOR,
		autocompleteFieldStockID: stock.StockSearchField_STOCK_SEARCH_FIELD_STOCK_ID,
		searchFieldGenes:         stock.StockSearchField_STOCK_SEARCH_FIELD_GENES,
		searchFieldDbxrefs:       stock.StockSearchField_STOCK_SEARCH_FIELD_DBXREFS,
		autocompleteFieldLabel:   stock.StockSearchField_STOCK_SEARCH_FIELD_LABEL,
		searchFieldNames:         stock.StockSearchField_STOCK_SEARCH_FIELD_NAMES,
		searchFieldSpecies:       stock.StockSearchField_STOCK_SEARCH_FIELD_SPECIES,
		searchFieldPlasmid:       stock.StockSearchField_STOCK_SEARCH_FIELD_PLASMID,
		autocompleteFieldName:    stock.StockSearchField_STOCK_SEARCH_FIELD_NAME,
	}
	for label, want := range expected {
		got := mapRepositoryField(label)
		assert.NotEqual(
			stock.StockSearchField_STOCK_SEARCH_FIELD_UNSPECIFIED,
			got,
			"field %s must map to a defined value",
			label,
		)
		assert.Equal(want, got, "field %s maps to the wrong value", label)
	}
	assert.Equal(
		stock.StockSearchField_STOCK_SEARCH_FIELD_UNSPECIFIED,
		mapRepositoryField("nosuchfield"),
		"an unknown label maps to UNSPECIFIED",
	)
}

func TestSearchStockHandlerForwardsContextToRepository(t *testing.T) {
	assert := require.New(t)
	stub := &fullSearchStubRepo{}
	svc := setupTestService(stub)
	ctx := t.Context()

	_, err := svc.SearchStock(
		ctx,
		fullSearchRequest("culminants", 10, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.NoError(err)
	assert.Equal(ctx, stub.gotCtx, "expect handler to forward caller context to SearchStock")

	canceledSvc := setupTestService(&fullSearchStubRepo{
		err: fmt.Errorf("wrapped: %w", context.Canceled),
	})
	_, err = canceledSvc.SearchStock(
		ctx,
		fullSearchRequest("culminants", 10, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.Equal(
		codes.Canceled,
		status.Code(err),
		"expect Canceled status code for context.Canceled",
	)

	deadlineSvc := setupTestService(&fullSearchStubRepo{
		err: fmt.Errorf("wrapped: %w", context.DeadlineExceeded),
	})
	_, err = deadlineSvc.SearchStock(
		ctx,
		fullSearchRequest("culminants", 10, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.Equal(
		codes.DeadlineExceeded,
		status.Code(err),
		"expect DeadlineExceeded status code for context.DeadlineExceeded",
	)
}

func TestListHandlersForwardContextToRepository(t *testing.T) {
	doc := &model.StockDoc{
		StrainProperties:  &model.StrainProperties{Label: testStrainLabel},
		PlasmidProperties: &model.PlasmidProperties{Name: testPlasmidID},
	}
	stub := &fullSearchStubRepo{docs: []*model.StockDoc{doc}}
	svc := setupTestService(stub)

	t.Run("ListStrains", func(t *testing.T) {
		ctx := t.Context()
		_, err := svc.ListStrains(ctx, &stock.StockParameters{Limit: 5})
		require.NoError(t, err)
		require.Equal(t, ctx, stub.gotCtx, "expect ListStrains to forward caller context")
	})

	t.Run("ListStrainsByIDs", func(t *testing.T) {
		ctx := t.Context()
		_, err := svc.ListStrainsByIDs(ctx, &stock.StockIdList{Id: []string{testStockID}})
		require.NoError(t, err)
		require.Equal(t, ctx, stub.gotCtx, "expect ListStrainsByIDs to forward caller context")
	})

	t.Run("ListPlasmids", func(t *testing.T) {
		ctx := t.Context()
		_, err := svc.ListPlasmids(ctx, &stock.StockParameters{Limit: 5})
		require.NoError(t, err)
		require.Equal(t, ctx, stub.gotCtx, "expect ListPlasmids to forward caller context")
	})
}
