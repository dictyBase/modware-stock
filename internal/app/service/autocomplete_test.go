package service

import (
	"context"
	"errors"
	"io"
	"testing"

	IOE "github.com/IBM/fp-go/ioeither"
	E "github.com/IBM/fp-go/v2/either"
	F "github.com/IBM/fp-go/v2/function"
	IOEV2 "github.com/IBM/fp-go/v2/ioeither"
	manager "github.com/dictyBase/arangomanager"
	ioeutils "github.com/dictyBase/fp-go-loom/ioeitherutils"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/go-obograph/storage"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// autocompleteStubRepo implements every method of
// repository.StockRepository for the handler unit tests.
type autocompleteStubRepo struct {
	sugs []*repository.Suggestion
	err  error
	got  *repository.AutocompleteQuery
}

func (sr *autocompleteStubRepo) GetStrain(string) (*model.StockDoc, error) {
	return nil, nil
}
func (sr *autocompleteStubRepo) GetPlasmid(string) IOE.IOEither[error, *model.StockDoc] {
	return IOE.Left[*model.StockDoc](errors.New("not used"))
}
func (sr *autocompleteStubRepo) AddStrain(*stock.NewStrain) (*model.StockDoc, error) {
	return nil, nil
}
func (sr *autocompleteStubRepo) AddPlasmid(*stock.NewPlasmid) IOE.IOEither[error, *model.StockDoc] {
	return IOE.Left[*model.StockDoc](errors.New("not used"))
}
func (sr *autocompleteStubRepo) EditStrain(*stock.StrainUpdate) (*model.StockDoc, error) {
	return nil, nil
}
func (sr *autocompleteStubRepo) EditPlasmid(*stock.PlasmidUpdate) IOE.IOEither[error, *model.StockDoc] {
	return IOE.Left[*model.StockDoc](errors.New("not used"))
}
func (sr *autocompleteStubRepo) ListStrains(*stock.StockParameters) ([]*model.StockDoc, error) {
	return nil, nil
}
func (sr *autocompleteStubRepo) ListStrainsByIDs(*stock.StockIdList) ([]*model.StockDoc, error) {
	return nil, nil
}
func (sr *autocompleteStubRepo) ListPlasmids(*stock.StockParameters) IOE.IOEither[error, []*model.StockDoc] {
	return IOE.Left[[]*model.StockDoc](errors.New("not used"))
}
func (sr *autocompleteStubRepo) LoadStrain(string, *stock.ExistingStrain) (*model.StockDoc, error) {
	return nil, nil
}
func (sr *autocompleteStubRepo) LoadPlasmid(string, *stock.ExistingPlasmid) IOE.IOEither[error, *model.StockDoc] {
	return IOE.Left[*model.StockDoc](errors.New("not used"))
}
func (sr *autocompleteStubRepo) RemoveStock(string) error { return nil }
func (sr *autocompleteStubRepo) SearchStock(
	_ *repository.FullSearchQuery,
) ([]*repository.FullSearchResult, error) {
	return nil, nil
}
func (sr *autocompleteStubRepo) AutocompleteStock(
	params *repository.AutocompleteQuery,
) IOEV2.IOEither[error, []*repository.Suggestion] {
	sr.got = params
	if sr.err != nil {
		return IOEV2.Left[[]*repository.Suggestion](sr.err)
	}
	return IOEV2.Of[error](sr.sugs)
}
func (sr *autocompleteStubRepo) Dbh() *manager.Database { return nil }
func (sr *autocompleteStubRepo) LoadOboJSON(io.Reader) (*storage.UploadInformation, error) {
	return nil, nil
}

func autocompleteRequest(query string, limit int64, entity stock.StockEntity) *stock.StockAutocompleteParameters {
	return &stock.StockAutocompleteParameters{
		Data: &stock.StockAutocompleteParameters_Data{
			Type: autocompleteStockType,
			Attributes: &stock.StockAutocompleteAttributes{
				Query:  query,
				Limit:  limit,
				Entity: entity,
			},
		},
	}
}

func TestAutocompleteStockHandlerMapsSuggestions(t *testing.T) {
	assert := require.New(t)
	stub := &autocompleteStubRepo{
		sugs: []*repository.Suggestion{
			{
				ID:          testStockID,
				Field:       autocompleteFieldLabel,
				DisplayText: testStrainLabel,
				Entity:      repository.EntityStrain,
				Score:       1000.5,
			},
			{
				ID:          testPlasmidID,
				Field:       autocompleteFieldName,
				DisplayText: "pDM304",
				Entity:      repository.EntityPlasmid,
				Score:       12.25,
			},
		},
	}
	svc := setupTestService(stub)
	res, err := svc.AutocompleteStock(
		context.Background(),
		autocompleteRequest("ys1", 5, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.NoError(err, "expect no error")
	assert.Len(res.Data, 2, "expect two suggestions")
	first := res.Data[0]
	assert.Equal(testStockID, first.Id)
	assert.Equal(stock.StockSearchField_STOCK_SEARCH_FIELD_LABEL, first.Field)
	assert.Equal(stock.StockEntity_STOCK_ENTITY_STRAIN, first.Entity)
	assert.Equal("yS13", first.DisplayText)
	assert.Equal(1000.5, first.Score)
	second := res.Data[1]
	assert.Equal(stock.StockSearchField_STOCK_SEARCH_FIELD_NAME, second.Field)
	assert.Equal(stock.StockEntity_STOCK_ENTITY_PLASMID, second.Entity)
	assert.Equal("pDM304", second.DisplayText)
	assert.Equal(12.25, second.Score)
	assert.Equal(int64(2), res.Meta.Total)
	assert.Equal(int64(5), res.Meta.Limit)
	assert.Equal(int64(0), res.Meta.NextCursor)
}

func TestAutocompleteStockHandlerPassesThroughQueryAndLimit(t *testing.T) {
	assert := require.New(t)
	stub := &autocompleteStubRepo{
		sugs: []*repository.Suggestion{
			{
				ID:          testStockID,
				Field:       autocompleteFieldLabel,
				DisplayText: testStrainLabel,
				Entity:      repository.EntityStrain,
				Score:       1000,
			},
		},
	}
	svc := setupTestService(stub)
	_, err := svc.AutocompleteStock(
		context.Background(),
		autocompleteRequest(
			"  ys1  ", 10, stock.StockEntity_STOCK_ENTITY_PLASMID,
		),
	)
	assert.NoError(err, "expect no error")
	assert.Equal("ys1", stub.got.Query, "expect the trimmed query")
	assert.Equal(10, stub.got.Limit, "expect the request limit")
	assert.Equal(repository.EntityPlasmid, stub.got.Entity)
}

func TestAutocompleteStockHandlerDefaultsLimitToFive(t *testing.T) {
	assert := require.New(t)
	stub := &autocompleteStubRepo{}
	svc := setupTestService(stub)
	res, err := svc.AutocompleteStock(
		context.Background(),
		autocompleteRequest(
			"ys1", 0, stock.StockEntity_STOCK_ENTITY_STRAIN,
		),
	)
	assert.NoError(err, "expect no error")
	assert.Equal(5, stub.got.Limit, "expect the effective limit")
	assert.Equal(repository.EntityStrain, stub.got.Entity)
	assert.Equal(int64(5), res.Meta.Limit)
}

func TestAutocompleteStockHandlerRejectsNilData(t *testing.T) {
	assert := require.New(t)
	stub := &autocompleteStubRepo{}
	svc := setupTestService(stub)
	_, err := svc.AutocompleteStock(
		context.Background(),
		&stock.StockAutocompleteParameters{},
	)
	assert.Error(err, "expect an error for nil data")
	assert.Equal(codes.InvalidArgument, status.Code(err))
}

func TestAutocompleteStockHandlerRejectsNilAttributes(t *testing.T) {
	assert := require.New(t)
	stub := &autocompleteStubRepo{}
	svc := setupTestService(stub)
	_, err := svc.AutocompleteStock(
		context.Background(),
		&stock.StockAutocompleteParameters{
			Data: &stock.StockAutocompleteParameters_Data{Type: "stock"},
		},
	)
	assert.Error(err, "expect an error for nil attributes")
	assert.Equal(codes.InvalidArgument, status.Code(err))
}

func TestAutocompleteStockHandlerRejectsShortQuery(t *testing.T) {
	assert := require.New(t)
	stub := &autocompleteStubRepo{}
	svc := setupTestService(stub)
	_, err := svc.AutocompleteStock(
		context.Background(),
		autocompleteRequest("ys", 5, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.Error(err, "expect an error for a 2-character query")
	assert.Equal(codes.InvalidArgument, status.Code(err))
}

func TestAutocompleteStockHandlerRejectsWhitespaceQuery(t *testing.T) {
	assert := require.New(t)
	stub := &autocompleteStubRepo{}
	svc := setupTestService(stub)
	_, err := svc.AutocompleteStock(
		context.Background(),
		autocompleteRequest("   ", 5, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.Error(err, "expect an error for a whitespace-only query")
	assert.Equal(codes.InvalidArgument, status.Code(err))
}

func TestAutocompleteStockHandlerRejectsLimitAboveCap(t *testing.T) {
	assert := require.New(t)
	stub := &autocompleteStubRepo{}
	svc := setupTestService(stub)
	_, err := svc.AutocompleteStock(
		context.Background(),
		autocompleteRequest("ys1", 51, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.Error(err, "expect an error for a limit above the cap")
	assert.Equal(codes.InvalidArgument, status.Code(err))
}

func TestAutocompleteStockHandlerRejectsUndefinedEntity(t *testing.T) {
	assert := require.New(t)
	stub := &autocompleteStubRepo{}
	svc := setupTestService(stub)
	_, err := svc.AutocompleteStock(
		context.Background(),
		autocompleteRequest("ys1", 5, stock.StockEntity(7)),
	)
	assert.Error(err, "expect an error for an undefined entity")
	assert.Equal(codes.InvalidArgument, status.Code(err))
}

func TestAutocompleteStockHandlerReturnsEmptyCollection(t *testing.T) {
	assert := require.New(t)
	stub := &autocompleteStubRepo{sugs: []*repository.Suggestion{}}
	svc := setupTestService(stub)
	res, err := svc.AutocompleteStock(
		context.Background(),
		autocompleteRequest("zzzqqq", 5, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.NoError(err, "expect no error for an empty result")
	assert.Empty(res.Data, "expect an empty list")
	assert.Equal(int64(0), res.Meta.Total)
	assert.Equal(int64(5), res.Meta.Limit)
}

func TestAutocompleteStockHandlerMapsRepositoryError(t *testing.T) {
	assert := require.New(t)
	stub := &autocompleteStubRepo{err: errors.New("database failure")}
	svc := setupTestService(stub)
	_, err := svc.AutocompleteStock(
		context.Background(),
		autocompleteRequest("ys1", 5, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED),
	)
	assert.Error(err, "expect the repository error to map")
	assert.Equal(codes.Internal, status.Code(err),
		"aphgrpc.HandleGetError maps to Internal")
}

func TestAutocompleteFieldMapping(t *testing.T) {
	assert := require.New(t)
	mapping := map[string]stock.StockSearchField{
		autocompleteFieldStockID: stock.StockSearchField_STOCK_SEARCH_FIELD_STOCK_ID,
		searchFieldGenes:         stock.StockSearchField_STOCK_SEARCH_FIELD_GENES,
		searchFieldDbxrefs:       stock.StockSearchField_STOCK_SEARCH_FIELD_DBXREFS,
		autocompleteFieldLabel:   stock.StockSearchField_STOCK_SEARCH_FIELD_LABEL,
		searchFieldNames:         stock.StockSearchField_STOCK_SEARCH_FIELD_NAMES,
		searchFieldSpecies:       stock.StockSearchField_STOCK_SEARCH_FIELD_SPECIES,
		searchFieldPlasmid:       stock.StockSearchField_STOCK_SEARCH_FIELD_PLASMID,
		autocompleteFieldName:    stock.StockSearchField_STOCK_SEARCH_FIELD_NAME,
		// The full search feature extended the field map with the 2
		// prose labels.
		fullSearchFieldSummary:   stock.StockSearchField_STOCK_SEARCH_FIELD_SUMMARY,
		fullSearchFieldDepositor: stock.StockSearchField_STOCK_SEARCH_FIELD_DEPOSITOR,
	}
	for field, want := range mapping {
		assert.Equal(want, mapRepositoryField(field),
			"field %s must map to the right enum", field)
	}
	entity := map[repository.StockEntityFilter]stock.StockEntity{
		repository.EntityStrain:  stock.StockEntity_STOCK_ENTITY_STRAIN,
		repository.EntityPlasmid: stock.StockEntity_STOCK_ENTITY_PLASMID,
		repository.EntityBoth:    stock.StockEntity_STOCK_ENTITY_UNSPECIFIED,
	}
	for filter, want := range entity {
		assert.Equal(want, mapRepositoryEntity(filter),
			"entity %s must map to the right enum", filter)
	}
}

func TestAutocompleteEntityFilterRejectsUnknown(t *testing.T) {
	assert := require.New(t)
	_, err := autocompleteEntityFilter(stock.StockEntity(7))
	assert.Error(err, "expect an error for an undefined stock entity")
}

// forcedValidation runs the lazy validation pipeline of one request.
func forcedValidation(
	r *stock.StockAutocompleteParameters,
) (autocompleteSearch, error) {
	return E.UnwrapError(F.Pipe2(
		r,
		validateAutocompleteRequest,
		ioeutils.ToEither[error, autocompleteSearch],
	))
}

func TestValidateAutocompleteRequestBuildsSearchState(t *testing.T) {
	assert := require.New(t)
	state, err := forcedValidation(autocompleteRequest(
		"  ys1  ", 0, stock.StockEntity_STOCK_ENTITY_STRAIN,
	))
	assert.NoError(err, "expect no error for a valid request")
	assert.Equal("ys1", state.query.Query, "expect the trimmed query")
	assert.Equal(defaultAutocompleteLimit, state.query.Limit,
		"expect the default limit for a limit of 0")
	assert.Equal(repository.EntityStrain, state.query.Entity,
		"expect the mapped entity filter")
	assert.Nil(state.rows, "expect no suggestion before the search")
}

func TestValidateAutocompleteRequestClassesRejectedRequest(t *testing.T) {
	assert := require.New(t)
	_, err := forcedValidation(&stock.StockAutocompleteParameters{})
	assert.Error(err, "expect an error for nil data")
	assert.True(isInvalidAutocomplete(err),
		"expect a boundary fault to carry the rejected request class")
	_, err = forcedValidation(autocompleteRequest(
		"   ", 5, stock.StockEntity_STOCK_ENTITY_UNSPECIFIED,
	))
	assert.Error(err, "expect an error for a whitespace-only query")
	assert.True(isInvalidAutocomplete(err),
		"expect a guard fault to carry the rejected request class")
	assert.Equal(msgEmptyAutocompleteQuery, err.Error(),
		"expect the class to keep the sentinel message")
}

func TestAutocompleteStatusMapsFaultClasses(t *testing.T) {
	assert := require.New(t)
	ctx := context.Background()
	rejected := newInvalidAutocompleteError(
		errors.New(msgEmptyAutocompleteQuery),
	)
	failed := errors.New("database failure")
	rejectedStatus := autocompleteStatus(ctx, rejected)
	failedStatus := autocompleteStatus(ctx, failed)
	assert.Equal(codes.InvalidArgument, status.Code(rejectedStatus),
		"a rejected request reports an invalid parameter")
	assert.Equal(msgEmptyAutocompleteQuery,
		status.Convert(rejectedStatus).Message(),
		"expect the status to carry the sentinel message")
	assert.Equal(codes.Internal, status.Code(failedStatus),
		"a repository fault reports a server error")
	assert.Equal("database failure",
		status.Convert(failedStatus).Message(),
		"expect the status to carry the repository message")
}

func TestEffectiveLimit(t *testing.T) {
	assert := require.New(t)
	assert.Equal(int64(defaultAutocompleteLimit), effectiveLimit(0),
		"expect the default for a limit of 0")
	assert.Equal(int64(1), effectiveLimit(1),
		"expect a positive limit to pass through")
	assert.Equal(int64(50), effectiveLimit(50),
		"expect the cap to pass through")
}
