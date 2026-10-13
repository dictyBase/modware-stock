package service

import (
	"context"
	"strings"

	A "github.com/IBM/fp-go/v2/array"
	E "github.com/IBM/fp-go/v2/either"
	EQ "github.com/IBM/fp-go/v2/eq"
	F "github.com/IBM/fp-go/v2/function"
	IOE "github.com/IBM/fp-go/v2/ioeither"
	L "github.com/IBM/fp-go/v2/optics/lens"
	O "github.com/IBM/fp-go/v2/option"
	P "github.com/IBM/fp-go/v2/predicate"
	R "github.com/IBM/fp-go/v2/record"
	S "github.com/IBM/fp-go/v2/string"
	"github.com/bufbuild/protovalidate-go"
	"github.com/cockroachdb/errors"
	eithererr "github.com/dictyBase/fp-go-loom/eithererr"
	ioeutils "github.com/dictyBase/fp-go-loom/ioeitherutils"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// autocompleteInput is the boundary input of one autocomplete call:
	// the proto request and the repository it searches.
	autocompleteInput struct {
		request *stock.StockAutocompleteParameters
		repo    repository.StockRepository
	}

	// autocompleteParams accumulates the validated fields of one
	// autocomplete request step by step: the normalized query, the
	// effective limit, the raw proto entity and the mapped repository
	// filter.
	autocompleteParams struct {
		query     string
		limit     int64
		rawEntity stock.StockEntity
		entity    repository.StockEntityFilter
	}

	// autocompleteSearch carries the repository, the validated query and
	// the suggestions that query returned. The query stays on the state
	// because the response meta reports its effective limit.
	autocompleteSearch struct {
		repo  repository.StockRepository
		query *repository.AutocompleteQuery
		rows  []*repository.Suggestion
	}

	// autocompleteResult is the boundary record of one autocomplete
	// call: the proto collection and the gRPC status error, exactly one
	// of which the handler returns.
	autocompleteResult struct {
		collection *stock.StockSuggestionCollection
		err        error
	}
)

const (
	// msgEmptyAutocompleteQuery is the sentinel message of a
	// whitespace-only autocomplete query.
	msgEmptyAutocompleteQuery = "expect a non-empty autocomplete query"

	// defaultAutocompleteLimit is the effective limit for a request
	// limit of 0. It mirrors the repository clamp.
	defaultAutocompleteLimit = 5

	// unspecifiedSearchField is the proto search field of a repository
	// label that the field map does not cover.
	unspecifiedSearchField = stock.StockSearchField_STOCK_SEARCH_FIELD_UNSPECIFIED
)

var (
	// entityLens focuses the repository entity filter of the params.
	entityLens = L.MakeLens(
		func(p autocompleteParams) repository.StockEntityFilter {
			return p.entity
		},
		func(
			p autocompleteParams,
			entity repository.StockEntityFilter,
		) autocompleteParams {
			p.entity = entity
			return p
		},
	)

	// rowsLens focuses the suggestions of a search.
	rowsLens = L.MakeLens(
		func(s autocompleteSearch) []*repository.Suggestion {
			return s.rows
		},
		func(
			s autocompleteSearch,
			rows []*repository.Suggestion,
		) autocompleteSearch {
			s.rows = rows
			return s
		},
	)

	// limitEq is the equality instance of the int64 request limit.
	limitEq = EQ.FromStrictEquals[int64]()

	// isDefaultLimit reports whether a zero limit asks for the default.
	isDefaultLimit = EQ.Equals(limitEq)(0)

	// queryText lifts a predicate of the normalized query onto the
	// params.
	queryText = P.ContraMap(
		func(param autocompleteParams) string { return param.query },
	)

	// hasQueryText reports whether the normalized query has content.
	// The proto rule counts characters, so a whitespace-only query
	// reaches this guard.
	hasQueryText = F.Pipe1(S.IsNonEmpty, queryText)

	// repositoryFieldMap maps the repository field label to the proto
	// search field enum. The map covers the 8 identifier labels of the
	// autocomplete feature and the 2 prose labels of the full search.
	repositoryFieldMap = map[string]stock.StockSearchField{
		autocompleteFieldStockID: stock.StockSearchField_STOCK_SEARCH_FIELD_STOCK_ID,
		searchFieldGenes:         stock.StockSearchField_STOCK_SEARCH_FIELD_GENES,
		searchFieldDbxrefs:       stock.StockSearchField_STOCK_SEARCH_FIELD_DBXREFS,
		autocompleteFieldLabel:   stock.StockSearchField_STOCK_SEARCH_FIELD_LABEL,
		searchFieldNames:         stock.StockSearchField_STOCK_SEARCH_FIELD_NAMES,
		searchFieldSpecies:       stock.StockSearchField_STOCK_SEARCH_FIELD_SPECIES,
		searchFieldPlasmid:       stock.StockSearchField_STOCK_SEARCH_FIELD_PLASMID,
		autocompleteFieldName:    stock.StockSearchField_STOCK_SEARCH_FIELD_NAME,
		fullSearchFieldSummary:   stock.StockSearchField_STOCK_SEARCH_FIELD_SUMMARY,
		fullSearchFieldDepositor: stock.StockSearchField_STOCK_SEARCH_FIELD_DEPOSITOR,
	}
)

// effectiveLimit replaces a default limit request with the default and
// keeps every other limit as-is. protovalidate bounds the raw limit
// from zero through the cap. The None branch of the fold keeps the
// requested limit; the Some branch of a zero limit reports the default.
func effectiveLimit(limit int64) int64 {
	return F.Pipe2(
		limit,
		O.FromPredicate(isDefaultLimit),
		O.Fold(
			F.Constant(limit),
			F.Constant1[int64, int64](defaultAutocompleteLimit),
		),
	)
}

// newAutocompleteParams accumulates the normalized query and the
// effective limit of a validated request. The raw entity waits for the
// filter mapping step.
func newAutocompleteParams(
	r *stock.StockAutocompleteParameters,
) autocompleteParams {
	attrs := r.GetData().GetAttributes()
	query := attrs.GetQuery()
	limit := attrs.GetLimit()
	return autocompleteParams{
		query:     F.Pipe1(query, strings.TrimSpace),
		limit:     effectiveLimit(limit),
		rawEntity: attrs.GetEntity(),
	}
}

// mapEntityFilter maps the raw proto entity of the params into the
// repository filter.
func mapEntityFilter(
	p autocompleteParams,
) E.Either[error, repository.StockEntityFilter] {
	return F.Pipe1(
		p.rawEntity,
		E.Eitherize1(autocompleteEntityFilter),
	)
}

// newAutocompleteSearch lifts the validated params into the search
// state: the repository of the call and its query, with no suggestion
// yet.
func newAutocompleteSearch(
	repo repository.StockRepository,
	p autocompleteParams,
) autocompleteSearch {
	return autocompleteSearch{
		repo: repo,
		query: &repository.AutocompleteQuery{
			Query:  p.query,
			Entity: p.entity,
			Limit:  int(p.limit),
		},
	}
}

// invalidParamStatus is the gRPC status of a request that the boundary
// rejected. It replaces aphgrpc.HandleInvalidParamError and reports the
// code alone, without a trailer.
func invalidParamStatus(err error) error {
	return status.Error(codes.InvalidArgument, err.Error())
}

// serverErrorStatus is the gRPC status of a fault that the repository
// search raised. It replaces aphgrpc.HandleGetError and reports the
// code alone, without a trailer.
func serverErrorStatus(err error) error {
	return status.Error(codes.Internal, err.Error())
}

// validatedAutocompleteSearch guards the proto request step by step and
// builds the search state: the generated proto rules, the normalized
// query, the effective limit and the entity filter. The proto rules
// reject nil data, nil attributes, a short query and an out-of-range
// limit; this server has no recovery interceptor, so the boundary
// validation also protects the process from a nil dereference. Every
// fault leaves as an invalid-parameter status, decided here, where the
// class of the fault is known.
func validatedAutocompleteSearch(
	in autocompleteInput,
) E.Either[error, autocompleteSearch] {
	return F.Pipe5(
		E.TryCatchError(in.request, protovalidate.Validate(in.request)),
		E.Map[error](newAutocompleteParams),
		E.Chain(E.FromPredicate(
			hasQueryText,
			eithererr.ConstErr[autocompleteParams](msgEmptyAutocompleteQuery),
		)),
		E.Bind(entityLens.Set, mapEntityFilter),
		E.Map[error](F.Bind1st(newAutocompleteSearch, in.repo)),
		E.MapLeft[autocompleteSearch](invalidParamStatus),
	)
}

// searchSuggestions runs the lazy repository search of the validated
// query. A repository fault leaves as a server-error status, decided
// here, where the class of the fault is known.
func searchSuggestions(
	s autocompleteSearch,
) IOE.IOEither[error, []*repository.Suggestion] {
	return F.Pipe1(
		s.repo.AutocompleteStock(s.query),
		IOE.MapLeft[[]*repository.Suggestion](serverErrorStatus),
	)
}

// toSuggestionCollection builds the proto response of one autocomplete
// search: every suggestion maps into the collection and the meta
// reports the effective limit.
func toSuggestionCollection(
	s autocompleteSearch,
) *stock.StockSuggestionCollection {
	return &stock.StockSuggestionCollection{
		Data: F.Pipe1(s.rows, A.Map(mapStockSuggestion)),
		Meta: &stock.Meta{
			NextCursor: 0,
			Limit:      int64(s.query.Limit),
			Total:      int64(len(s.rows)),
		},
	}
}

// failedAutocompleteResult is the failure arm of the terminal fold. The
// fault already carries its status code, so the arm only pairs it with
// an absent collection.
func failedAutocompleteResult(err error) autocompleteResult {
	return autocompleteResult{err: err}
}

// successfulAutocompleteResult is the success arm of the terminal fold:
// the collection becomes the result and no error is reported.
func successfulAutocompleteResult(
	collection *stock.StockSuggestionCollection,
) autocompleteResult {
	return autocompleteResult{collection: collection}
}

// toAutocompleteResult forces the lazy program at the edge and folds it
// into the boundary record of the handler.
func toAutocompleteResult(
	program IOE.IOEither[error, *stock.StockSuggestionCollection],
) autocompleteResult {
	return F.Pipe2(
		program,
		ioeutils.ToEither[error, *stock.StockSuggestionCollection],
		E.Fold(failedAutocompleteResult, successfulAutocompleteResult),
	)
}

// AutocompleteStock returns short suggestions for a partial stock
// identifier, name or attribute value. The program is an IOEither from
// the boundary input to the edge: the pure request validation enters it
// through ChainEitherK, the repository search and the proto mapping
// follow, and the last step forces it into the collection and the
// status error. Every fault carries its own status code from the step
// that knows its class, so no fault classification happens at the edge.
// The request context stays unused: status.Error is pure, and the
// repository search takes no context.
func (s *StockService) AutocompleteStock(
	_ context.Context,
	r *stock.StockAutocompleteParameters,
) (*stock.StockSuggestionCollection, error) {
	result := F.Pipe5(
		IOE.Of[error](autocompleteInput{request: r, repo: s.repo}),
		IOE.ChainEitherK(validatedAutocompleteSearch),
		IOE.Bind(rowsLens.Set, searchSuggestions),
		IOE.Map[error](toSuggestionCollection),
		ioeutils.ToEither[error, *stock.StockSuggestionCollection],
		E.Fold(failedAutocompleteResult, successfulAutocompleteResult),
	)
	return result.collection, result.err
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
// search field enum. A label the map does not cover maps to
// UNSPECIFIED.
func mapRepositoryField(field string) stock.StockSearchField {
	return F.Pipe2(
		repositoryFieldMap,
		R.Lookup[stock.StockSearchField](field),
		O.GetOrElse(F.Constant(unspecifiedSearchField)),
	)
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
