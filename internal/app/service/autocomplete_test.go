package service

import (
	"context"
	"errors"
	"io"
	"testing"

	IOE "github.com/IBM/fp-go/ioeither"
	manager "github.com/dictyBase/arangomanager"
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
func (sr *autocompleteStubRepo) AutocompleteStock(
	params *repository.AutocompleteQuery,
) ([]*repository.Suggestion, error) {
	sr.got = params
	return sr.sugs, sr.err
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
				ID:          "DBS0236126",
				Field:       autocompleteFieldLabel,
				DisplayText: testStrainLabel,
				Entity:      repository.EntityStrain,
				Score:       1000.5,
			},
			{
				ID:          "DBP0000027",
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
	assert.Equal("DBS0236126", first.Id)
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
				ID:          "DBS0236126",
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
		"stock_id": stock.StockSearchField_STOCK_SEARCH_FIELD_STOCK_ID,
		"genes":    stock.StockSearchField_STOCK_SEARCH_FIELD_GENES,
		"dbxrefs":  stock.StockSearchField_STOCK_SEARCH_FIELD_DBXREFS,
		"label":    stock.StockSearchField_STOCK_SEARCH_FIELD_LABEL,
		"names":    stock.StockSearchField_STOCK_SEARCH_FIELD_NAMES,
		"species":  stock.StockSearchField_STOCK_SEARCH_FIELD_SPECIES,
		"plasmid":  stock.StockSearchField_STOCK_SEARCH_FIELD_PLASMID,
		"name":     stock.StockSearchField_STOCK_SEARCH_FIELD_NAME,
		"summary":  stock.StockSearchField_STOCK_SEARCH_FIELD_UNSPECIFIED,
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
