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
	ORD "github.com/IBM/fp-go/v2/ord"
	P "github.com/IBM/fp-go/v2/predicate"
	S "github.com/IBM/fp-go/v2/string"
	"github.com/bufbuild/protovalidate-go"
	ioeutils "github.com/dictyBase/fp-go-loom/ioeitherutils"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/repository"
)

type (
	// fullSearchInput is the boundary input of one full stock search:
	// the caller context, the proto request and the repository it searches.
	fullSearchInput struct {
		ctx     context.Context
		request *stock.StockSearchParameters
		repo    repository.StockRepository
	}

	// fullSearchParams accumulates the validated fields of one full
	// search request step by step: the normalized query, the effective
	// limit, the raw proto entity and the mapped repository filter.
	fullSearchParams struct {
		query     string
		limit     int64
		rawEntity stock.StockEntity
		entity    repository.StockEntityFilter
	}

	// fullSearch carries the caller context, the repository, the validated
	// query and the hits that query returned. The query stays on the state
	// because the response meta reports its effective limit.
	fullSearch struct {
		ctx   context.Context
		repo  repository.StockRepository
		query *repository.FullSearchQuery
		hits  []*repository.FullSearchResult
	}

	// fullSearchResult is the boundary record of one full search call:
	// the proto collection and the gRPC status error, exactly one of
	// which the handler returns.
	fullSearchResult struct {
		collection *stock.StockSearchResultCollection
		err        error
	}
)

const (
	// msgEmptySearchQuery is the sentinel message of a
	// whitespace-only search query.
	msgEmptySearchQuery = "expect a non-empty search query"

	// defaultFullSearchLimit is the effective limit for a request limit of
	// 0. It mirrors the repository clamp.
	defaultFullSearchLimit = 50

	// maxFullSearchLimit is the hard cap of the returned list length. The
	// proto accepts a limit up to 100, and the handler clamps 51 to 100
	// down to this value, because the spec asks for one list of 50 items.
	maxFullSearchLimit = 50
)

var (
	// fullSearchEntityLens focuses the repository entity filter of the params.
	fullSearchEntityLens = L.MakeLens(
		func(p fullSearchParams) repository.StockEntityFilter {
			return p.entity
		},
		func(
			p fullSearchParams,
			entity repository.StockEntityFilter,
		) fullSearchParams {
			p.entity = entity
			return p
		},
	)

	// fullSearchHitsLens focuses the hits of a full search.
	fullSearchHitsLens = L.MakeLens(
		func(s fullSearch) []*repository.FullSearchResult {
			return s.hits
		},
		func(
			s fullSearch,
			hits []*repository.FullSearchResult,
		) fullSearch {
			s.hits = hits
			return s
		},
	)

	// fullSearchQueryText lifts a predicate of the normalized query onto
	// the params.
	fullSearchQueryText = P.ContraMap(
		func(param fullSearchParams) string { return param.query },
	)

	// hasSearchQueryText reports whether the normalized query has content.
	// The proto rule counts characters, so a whitespace-only query
	// reaches this guard.
	hasSearchQueryText = F.Pipe1(S.IsNonEmpty, fullSearchQueryText)

	// fullSearchLimitEq is the equality instance of the int64 request
	// limit.
	fullSearchLimitEq = EQ.FromStrictEquals[int64]()

	// fullSearchLimitOrd is the order instance of the int64 request limit.
	fullSearchLimitOrd = ORD.FromStrictCompare[int64]()

	// isDefaultFullSearchLimit reports whether a zero limit asks for the
	// default.
	isDefaultFullSearchLimit = EQ.Equals(fullSearchLimitEq)(0)

	// isAboveFullSearchCap reports whether a limit is above the hard cap.
	isAboveFullSearchCap = F.Pipe1(
		maxFullSearchLimit,
		ORD.Gt(fullSearchLimitOrd),
	)
)

// defaultedFullSearchLimit replaces a default limit request with the
// default limit. The None branch of the fold keeps the requested limit;
// the Some branch of a zero limit reports the default.
func defaultedFullSearchLimit(limit int64) int64 {
	return F.Pipe2(
		limit,
		O.FromPredicate(isDefaultFullSearchLimit),
		O.Fold(
			F.Constant(limit),
			F.Constant1[int64, int64](defaultFullSearchLimit),
		),
	)
}

// cappedFullSearchLimit brings a limit above the hard cap down to the
// cap. protovalidate rejects a limit above 100, so this covers 51 to 100.
func cappedFullSearchLimit(limit int64) int64 {
	return F.Pipe2(
		limit,
		O.FromPredicate(isAboveFullSearchCap),
		O.Fold(
			F.Constant(limit),
			F.Constant1[int64, int64](maxFullSearchLimit),
		),
	)
}

// effectiveFullSearchLimit applies both bounds of the request limit, one
// fold per bound, so an over-cap limit can never collapse into the
// default.
func effectiveFullSearchLimit(limit int64) int64 {
	return F.Pipe2(limit, defaultedFullSearchLimit, cappedFullSearchLimit)
}

// newFullSearchParams accumulates the normalized query and the
// effective limit of a validated request. The raw entity waits for the
// filter mapping step.
func newFullSearchParams(
	r *stock.StockSearchParameters,
) fullSearchParams {
	attrs := r.GetData().GetAttributes()
	query := attrs.GetQuery()
	limit := attrs.GetLimit()
	return fullSearchParams{
		query:     F.Pipe1(query, strings.TrimSpace),
		limit:     effectiveFullSearchLimit(limit),
		rawEntity: attrs.GetEntity(),
	}
}

// mapFullSearchEntityFilter maps the raw proto entity of the params into
// the repository filter.
func mapFullSearchEntityFilter(
	p fullSearchParams,
) E.Either[error, repository.StockEntityFilter] {
	return F.Pipe1(
		p.rawEntity,
		E.Eitherize1(autocompleteEntityFilter),
	)
}

// newFullSearch lifts the validated params into the search state: the
// caller context, the repository of the call and its query, with no hits yet.
func newFullSearch(
	in fullSearchInput,
	p fullSearchParams,
) fullSearch {
	return fullSearch{
		ctx:  in.ctx,
		repo: in.repo,
		query: &repository.FullSearchQuery{
			Query:  p.query,
			Entity: p.entity,
			Limit:  int(p.limit),
		},
	}
}

// validatedFullSearch guards the proto request step by step and builds
// the search state: the generated proto rules, the normalized query,
// the effective limit and the entity filter. The proto rules reject nil
// data, nil attributes, a short query and an out-of-range limit; this
// server has no recovery interceptor, so the boundary validation also
// protects the process from a nil dereference. Every fault leaves as an
// invalid-parameter status, decided here, where the class of the fault
// is known.
func validatedFullSearch(
	in fullSearchInput,
) E.Either[error, fullSearch] {
	return F.Pipe5(
		E.TryCatchError(in.request, protovalidate.Validate(in.request)),
		E.Map[error](newFullSearchParams),
		E.Chain(E.FromPredicate(
			hasSearchQueryText,
			invalidParamStatusOf[fullSearchParams](msgEmptySearchQuery),
		)),
		E.Bind(fullSearchEntityLens.Set, mapFullSearchEntityFilter),
		E.Map[error](F.Bind1st(newFullSearch, in)),
		E.MapLeft[fullSearch](invalidParamStatus),
	)
}

// searchStockHits runs the lazy repository search of the validated
// query. A repository fault leaves as a server-error status, decided
// here, where the class of the fault is known.
func searchStockHits(
	s fullSearch,
) IOE.IOEither[error, []*repository.FullSearchResult] {
	return F.Pipe1(
		s.repo.SearchStock(s.ctx, s.query),
		IOE.MapLeft[[]*repository.FullSearchResult](serverErrorStatus),
	)
}

// toSearchResultCollection builds the proto response of one full search:
// every hit maps into the collection and the meta reports the effective
// limit.
func toSearchResultCollection(
	s fullSearch,
) *stock.StockSearchResultCollection {
	return &stock.StockSearchResultCollection{
		Data: F.Pipe1(s.hits, A.Map(mapFullSearchResult)),
		Meta: &stock.Meta{
			NextCursor: 0,
			Limit:      int64(s.query.Limit),
			Total:      int64(len(s.hits)),
		},
	}
}

// failedFullSearchResult is the failure arm of the terminal fold. The
// fault already carries its status code, so the arm only pairs it with
// an absent collection.
func failedFullSearchResult(err error) fullSearchResult {
	return fullSearchResult{err: err}
}

// successfulFullSearchResult is the success arm of the terminal fold:
// the collection becomes the result and no error is reported.
func successfulFullSearchResult(
	collection *stock.StockSearchResultCollection,
) fullSearchResult {
	return fullSearchResult{collection: collection}
}

// SearchStock returns at most 50 ranked results for a search text of
// at least 2 characters. The program is an IOEither from the boundary
// input to the edge: the pure request validation enters it through
// ChainEitherK, the repository search and the proto mapping follow, and
// the last step forces it into the collection and the status error.
// Every fault carries its own status code from the step that knows its
// class, so no fault classification happens at the edge. The request
// context stays unused: status.Error is pure, and the repository search
// takes no context.
func (s *StockService) SearchStock(
	ctx context.Context,
	r *stock.StockSearchParameters,
) (*stock.StockSearchResultCollection, error) {
	result := F.Pipe5(
		IOE.Of[error](fullSearchInput{ctx: ctx, request: r, repo: s.repo}),
		IOE.ChainEitherK(validatedFullSearch),
		IOE.Bind(fullSearchHitsLens.Set, searchStockHits),
		IOE.Map[error](toSearchResultCollection),
		ioeutils.ToEither[error, *stock.StockSearchResultCollection],
		E.Fold(failedFullSearchResult, successfulFullSearchResult),
	)
	return result.collection, result.err
}

// mapFullSearchResult converts one repository result into the proto
// message of the response.
func mapFullSearchResult(row *repository.FullSearchResult) *stock.StockSearchResult {
	return &stock.StockSearchResult{
		Id:          row.ID,
		Entity:      mapRepositoryEntity(row.Entity),
		Field:       mapRepositoryField(row.Field),
		DisplayText: row.DisplayText,
		Score:       row.Score,
		Summary:     row.Summary,
		StrainLabel: row.StrainLabel,
		StockType:   row.StockType,
	}
}
