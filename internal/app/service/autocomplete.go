package service

import (
	"context"
	"strings"

	A "github.com/IBM/fp-go/array"
	F "github.com/IBM/fp-go/function"
	"github.com/bufbuild/protovalidate-go"
	"github.com/cockroachdb/errors"
	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/repository"
)

// AutocompleteStock returns short suggestions for a partial stock
// identifier, name or attribute value. The order of operations is:
// protovalidate validation, the trim guard, the effective limit, the
// entity mapping and the repository call.
func (s *StockService) AutocompleteStock(
	ctx context.Context,
	r *stock.StockAutocompleteParameters,
) (*stock.StockSuggestionCollection, error) {
	// The generated Validate method is a no-op for these messages.
	// protovalidate rejects nil data and nil attributes here, because
	// the proto marks both required; this server has no recovery
	// interceptor, so the boundary validation also protects the
	// process from a nil dereference.
	if err := protovalidate.Validate(r); err != nil {
		return nil, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	// The proto rule counts characters, so a whitespace-only query
	// reaches this point.
	query := strings.TrimSpace(r.GetData().GetAttributes().GetQuery())
	if query == "" {
		return nil, aphgrpc.HandleInvalidParamError(
			ctx,
			errors.New("expect a non-empty autocomplete query"),
		)
	}
	attrs := r.GetData().GetAttributes()
	effLimit := attrs.GetLimit()
	if effLimit == 0 {
		effLimit = defaultAutocompleteLimit
	}
	entity, err := autocompleteEntityFilter(attrs.GetEntity())
	if err != nil {
		return nil, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	rows, err := s.repo.AutocompleteStock(&repository.AutocompleteQuery{
		Query:  query,
		Entity: entity,
		Limit:  int(effLimit),
	})
	if err != nil {
		return nil, aphgrpc.HandleGetError(ctx, err)
	}
	return &stock.StockSuggestionCollection{
		Data: F.Pipe1(rows, A.Map(mapStockSuggestion)),
		Meta: &stock.Meta{
			NextCursor: 0,
			Limit:      effLimit,
			Total:      int64(len(rows)),
		},
	}, nil
}

// autocompleteEntityFilter maps the proto entity enum to the repository
// filter. protovalidate keeps the value within the defined enum values,
// but the default arm keeps a future proto change from opening a
// silent hole.
func autocompleteEntityFilter(
	entity stock.StockEntity,
) (repository.StockEntityFilter, error) {
	switch entity {
	case stock.StockEntity_STOCK_ENTITY_UNSPECIFIED:
		return repository.EntityBoth, nil
	case stock.StockEntity_STOCK_ENTITY_STRAIN:
		return repository.EntityStrain, nil
	case stock.StockEntity_STOCK_ENTITY_PLASMID:
		return repository.EntityPlasmid, nil
	default:
		return "", errors.Errorf(
			"expect a defined stock entity, received %d",
			entity,
		)
	}
}

// mapRepositoryField maps the repository field label to the proto
// search field enum. An unknown label maps to UNSPECIFIED.
func mapRepositoryField(field string) stock.StockSearchField {
	switch field {
	case autocompleteFieldStockID:
		return stock.StockSearchField_STOCK_SEARCH_FIELD_STOCK_ID
	case "genes":
		return stock.StockSearchField_STOCK_SEARCH_FIELD_GENES
	case "dbxrefs":
		return stock.StockSearchField_STOCK_SEARCH_FIELD_DBXREFS
	case autocompleteFieldLabel:
		return stock.StockSearchField_STOCK_SEARCH_FIELD_LABEL
	case "names":
		return stock.StockSearchField_STOCK_SEARCH_FIELD_NAMES
	case "species":
		return stock.StockSearchField_STOCK_SEARCH_FIELD_SPECIES
	case "plasmid":
		return stock.StockSearchField_STOCK_SEARCH_FIELD_PLASMID
	case autocompleteFieldName:
		return stock.StockSearchField_STOCK_SEARCH_FIELD_NAME
	default:
		return stock.StockSearchField_STOCK_SEARCH_FIELD_UNSPECIFIED
	}
}

// mapRepositoryEntity maps the repository entity filter to the proto
// entity enum. An unknown filter maps to UNSPECIFIED.
func mapRepositoryEntity(entity repository.StockEntityFilter) stock.StockEntity {
	switch entity {
	case repository.EntityStrain:
		return stock.StockEntity_STOCK_ENTITY_STRAIN
	case repository.EntityPlasmid:
		return stock.StockEntity_STOCK_ENTITY_PLASMID
	default:
		return stock.StockEntity_STOCK_ENTITY_UNSPECIFIED
	}
}

// mapStockSuggestion converts one repository suggestion into the proto
// message of the response.
func mapStockSuggestion(row *repository.Suggestion) *stock.StockSuggestion {
	return &stock.StockSuggestion{
		Id:          row.ID,
		Entity:      mapRepositoryEntity(row.Entity),
		Field:       mapRepositoryField(row.Field),
		DisplayText: row.DisplayText,
		Score:       row.Score,
	}
}

// defaultAutocompleteLimit is the effective limit for a request limit
// of 0. It mirrors the repository clamp.
const defaultAutocompleteLimit = 5
