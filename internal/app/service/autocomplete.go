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
	"github.com/dictyBase/aphgrpc"
	eithererr "github.com/dictyBase/fp-go-loom/eithererr"
	ioeutils "github.com/dictyBase/fp-go-loom/ioeitherutils"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/repository"
)

type (
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

	// autocompleteSearch carries the validated repository query and the
	// suggestions it returned. The query stays on the state because the
	// response meta reports its effective limit.
	autocompleteSearch struct {
		query *repository.AutocompleteQuery
		rows  []*repository.Suggestion
	}

	// autocompleteReply is the boundary record of one autocomplete call:
	// the proto collection and the gRPC error, exactly one of which the
	// handler returns.
	autocompleteReply struct {
		collection *stock.StockSuggestionCollection
		err        error
	}

	// invalidAutocompleteError classes a fault as a rejected request.
	// The handler reports it as an invalid parameter; every other fault
	// of the pipeline is a server error.
	invalidAutocompleteError struct{ err error }
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

	// queryLens focuses the repository query of a search.
	queryLens = L.MakeLens(
		func(s autocompleteSearch) *repository.AutocompleteQuery {
			return s.query
		},
		func(
			s autocompleteSearch,
			query *repository.AutocompleteQuery,
		) autocompleteSearch {
			s.query = query
			return s
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

// Error renders the message of the wrapped fault, so the gRPC status
// carries the reason the request was rejected.
func (e invalidAutocompleteError) Error() string {
	return e.err.Error()
}

// Unwrap keeps the wrapped fault reachable for errors.Is.
func (e invalidAutocompleteError) Unwrap() error {
	return e.err
}

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

// toAutocompleteSearch lifts the validated params into the search
// state: the repository query of the autocomplete search, with no
// suggestion yet.
func toAutocompleteSearch(p autocompleteParams) autocompleteSearch {
	return autocompleteSearch{
		query: &repository.AutocompleteQuery{
			Query:  p.query,
			Entity: p.entity,
			Limit:  int(p.limit),
		},
	}
}

// newInvalidAutocompleteError classes one validation fault as a
// rejected request.
func newInvalidAutocompleteError(err error) error {
	return invalidAutocompleteError{err: err}
}

// isInvalidAutocomplete reports whether a fault came from the request
// boundary.
func isInvalidAutocomplete(err error) bool {
	var invalid invalidAutocompleteError
	return errors.As(err, &invalid)
}

// validateAutocompleteRequest guards the proto request step by step and
// builds the search state: the generated proto rules, the normalized
// query, the effective limit and the entity filter. The proto rules
// reject nil data, nil attributes, a short query and an out-of-range
// limit; this server has no recovery interceptor, so the boundary
// validation also protects the process from a nil dereference. Every
// fault leaves as a rejected request, so the handler reports it as an
// invalid parameter.
func validateAutocompleteRequest(
	r *stock.StockAutocompleteParameters,
) IOE.IOEither[error, autocompleteSearch] {
	return F.Pipe6(
		E.TryCatchError(r, protovalidate.Validate(r)),
		E.Map[error](newAutocompleteParams),
		E.Chain(E.FromPredicate(
			hasQueryText,
			eithererr.ConstErr[autocompleteParams](msgEmptyAutocompleteQuery),
		)),
		E.Bind(entityLens.Set, mapEntityFilter),
		E.Map[error](toAutocompleteSearch),
		E.MapLeft[autocompleteSearch](newInvalidAutocompleteError),
		IOE.FromEither[error, autocompleteSearch],
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

// autocompleteStatus maps one fault to its gRPC status: a request the
// boundary rejected reports an invalid parameter, every other fault
// reports a server error. predicate.Fold selects the aphgrpc mapper
// purely by fault class, so the selected mapper runs once, here.
func autocompleteStatus(ctx context.Context, err error) error {
	selectMapper := P.Fold(
		F.Bind1st(aphgrpc.HandleGetError, ctx),
		F.Bind1st(aphgrpc.HandleInvalidParamError, ctx),
	)
	return F.Pipe1(err, selectMapper(isInvalidAutocomplete))
}

// failedAutocompleteReply is the failure arm of the terminal fold and
// the only step that turns a fault into a gRPC status. The collection
// stays absent.
func failedAutocompleteReply(
	ctx context.Context,
	err error,
) autocompleteReply {
	return autocompleteReply{err: autocompleteStatus(ctx, err)}
}

// successfulAutocompleteReply is the success arm of the terminal fold:
// the collection becomes the reply and no error is reported.
func successfulAutocompleteReply(
	collection *stock.StockSuggestionCollection,
) autocompleteReply {
	return autocompleteReply{collection: collection}
}

// AutocompleteStock returns short suggestions for a partial stock
// identifier, name or attribute value. The program is lazy until the
// edge: the request validation, the repository search and the proto
// mapping are one pipeline, and the terminal fold is the only step
// that turns the outcome into a gRPC reply.
func (s *StockService) AutocompleteStock(
	ctx context.Context,
	r *stock.StockAutocompleteParameters,
) (*stock.StockSuggestionCollection, error) {
	reply := F.Pipe5(
		r,
		validateAutocompleteRequest,
		IOE.Bind(
			rowsLens.Set,
			F.Flow2(queryLens.Get, s.repo.AutocompleteStock),
		),
		IOE.Map[error](toSuggestionCollection),
		ioeutils.ToEither[error, *stock.StockSuggestionCollection],
		E.Fold(
			F.Bind1st(failedAutocompleteReply, ctx),
			successfulAutocompleteReply,
		),
	)
	return reply.collection, reply.err
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
