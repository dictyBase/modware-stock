package arangodb

import (
	"context"
	"strings"

	A "github.com/IBM/fp-go/v2/array"
	E "github.com/IBM/fp-go/v2/either"
	fperrors "github.com/IBM/fp-go/v2/errors"
	F "github.com/IBM/fp-go/v2/function"
	IOE "github.com/IBM/fp-go/v2/ioeither"
	L "github.com/IBM/fp-go/v2/optics/lens"
	O "github.com/IBM/fp-go/v2/option"
	P "github.com/IBM/fp-go/v2/predicate"
	S "github.com/IBM/fp-go/v2/string"
	driver "github.com/arangodb/go-driver"
	"github.com/cockroachdb/errors"
	"github.com/dictyBase/fp-go-loom/eithererr"
	predarrays "github.com/dictyBase/fp-go-loom/predicate/array"
	"github.com/dictyBase/modware-stock/internal/repository"
)

type (
	// searchResultRow mirrors one object row of the full search projection.
	// go-driver v1 cannot decode an array row, so the projection returns one
	// object per row.
	searchResultRow struct {
		Key         string  `json:"k"`
		ID          string  `json:"id"`
		Entity      string  `json:"entity"`
		Field       string  `json:"f"`
		Value       string  `json:"v"`
		Summary     string  `json:"sm"`
		StrainLabel string  `json:"sl"`
		Score       float64 `json:"s"`
	}
	fullSearchColl    int
	fullSearchDisplay int

	// fullSearchField is one indexed search field. The label names the
	// field in the AQL, the collection selects the branch template and
	// the display selects the display expression.
	fullSearchField struct {
		label   string
		coll    fullSearchColl
		display fullSearchDisplay
	}

	// fullSearchRequest is the validated and normalized input of one
	// full search.
	fullSearchRequest struct {
		ctx    context.Context
		query  string
		limit  int
		entity repository.StockEntityFilter
	}

	// fullSearchQueryParams carries the generated statement and its bind
	// parameters for one full search.
	fullSearchQueryParams struct {
		ctx        context.Context
		statement  string
		bindParams map[string]any
	}
)

const (
	fullSearchCollStock fullSearchColl = iota
	fullSearchCollProp
	fullSearchScalar fullSearchDisplay = iota
	fullSearchArray

	// Search asset names owned by the full search feature. Every name
	// belongs to this feature alone, so other search features can be
	// absent, present or stale without any effect.
	// fullSearchViewName is the classic arangosearch view of this
	// feature. The AQL templates name it literally, because ArangoDB
	// does not accept a bind parameter for a view name in SEARCH.
	fullSearchViewName = "stock_full_search"
	// fullSearchNormAnalyzer lowercases a value and removes accents. It
	// serves the prefix branches.
	fullSearchNormAnalyzer = "stock_search_norm"
	// fullSearchNgramAnalyzer chains norm and ngram. It serves the fuzzy
	// branches.
	fullSearchNgramAnalyzer = "stock_search_ngram"
	// fullSearchTextAnalyzer is the built-in English text analyzer. It
	// serves the token branches and the phrase branches. It is built in,
	// so the setup must not try to create it.
	fullSearchTextAnalyzer = "text_en"
	// defaultFullSearchLimit is the list length for a non-positive limit.
	defaultFullSearchLimit = 50
	// maxFullSearchLimit is the hard cap of the returned list length. The
	// proto accepts a limit up to 100, and the repository clamps it to
	// this value, because the spec asks for one list of 50 items.
	maxFullSearchLimit = 50
	// fullSearchNgramThreshold is the minimum n-gram similarity of a
	// fuzzy match. Task 1 calibration on ArangoDB 3.11.14 corrected the
	// assumed 0.45 to 0.30: 0.45 misses the one-character typo
	// dbs0236127 against DBS0236126 entirely, while 0.30 matches it and
	// still admits no junk. This mirrors the autocomplete calibration.
	fullSearchNgramThreshold = 0.3

	// Score bands. The bands are 250 apart and carry the stage order:
	// prefix above phrase above token above fuzzy. The BM25 of a pure
	// STARTS_WITH match can be 0, so the constants carry the order and
	// the deterministic tiebreak on the key carries the rest. The Task 1
	// calibration measured a BM25 maximum of 39.5 on a 1071-document
	// fixture, far below the band width of 250.

	fullSearchPrefixBand = "1000 + BM25(d)"
	fullSearchPhraseBand = "500 + BM25(d)"
	fullSearchProseBand  = "250 + BM25(d)"
	fullSearchFuzzyBand  = "BM25(d)"

	// msgNilFullSearchQuery is the sentinel message of an absent request.
	msgNilFullSearchQuery = "expect a non-nil full search query"
	// msgEmptyFullSearchQuery is the sentinel message of a query that
	// normalizes to nothing.
	msgEmptyFullSearchQuery = "expect a non-empty full search query"
)

var (
	// fullSearchIdentifierFields lists the 8 identifier and name fields
	// that the view indexes with both custom analyzers. The first three
	// live on the stock collection, the last five on the stock property
	// collection. Each field serves one prefix branch and one fuzzy branch.
	fullSearchIdentifierFields = []fullSearchField{
		{label: paramStockID, coll: fullSearchCollStock, display: fullSearchScalar},
		{label: fieldGenes, coll: fullSearchCollStock, display: fullSearchArray},
		{label: fieldDbxrefs, coll: fullSearchCollStock, display: fullSearchArray},
		{label: fieldLabel, coll: fullSearchCollProp, display: fullSearchScalar},
		{label: fieldNames, coll: fullSearchCollProp, display: fullSearchArray},
		{label: fieldSpecies, coll: fullSearchCollProp, display: fullSearchScalar},
		{label: fieldPlasmid, coll: fullSearchCollProp, display: fullSearchScalar},
		{label: fieldName, coll: fullSearchCollProp, display: fullSearchScalar},
	}

	// fullSearchProseFields lists the 2 prose fields that the view indexes
	// with the built-in text analyzer only. The plan does not index
	// editable_summary: an n-gram index over long prose grows fast and
	// returns weak matches. Each field serves one token branch, and the
	// summary field serves the single phrase branch.
	fullSearchProseFields = []fullSearchField{
		{label: fieldSummary, coll: fullSearchCollStock, display: fullSearchScalar},
		{label: fieldDepositor, coll: fullSearchCollStock, display: fullSearchScalar},
	}

	// fullSearchStopwords is the short stopword list of the token stage.
	// Task 1 probe 18 recorded that the built-in text_en analyzer does not
	// remove stopwords, so the repository rejects a query whose tokens are
	// all stopwords before any AQL runs.
	fullSearchStopwords = map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {},
		"be": {}, "but": {}, "by": {}, "for": {}, "from": {}, "had": {},
		"has": {}, "have": {}, "if": {}, "in": {}, "is": {}, "it": {},
		"its": {}, "not": {}, "of": {}, "on": {}, "or": {}, "so": {},
		"than": {}, "that": {}, "the": {}, "their": {}, "then": {},
		"there": {}, "these": {}, "this": {}, "to": {}, "too": {},
		"very": {}, "was": {}, "were": {}, "what": {}, "when": {},
		"which": {}, "while": {}, "who": {}, "will": {}, "with": {},
	}

	// fullSearchEntityLens focuses the entity filter of a validated
	// request.
	fullSearchEntityLens = L.MakeLens(
		func(req fullSearchRequest) repository.StockEntityFilter {
			return req.entity
		},
		func(
			req fullSearchRequest,
			entity repository.StockEntityFilter,
		) fullSearchRequest {
			req.entity = entity
			return req
		},
	)

	// fullSearchCtxLens focuses the caller context of a validated request.
	fullSearchCtxLens = L.MakeLens(
		func(req fullSearchRequest) context.Context { return req.ctx },
		func(req fullSearchRequest, ctx context.Context) fullSearchRequest {
			req.ctx = ctx
			return req
		},
	)

	// fullSearchQueryLens focuses the normalized query of a validated
	// request.
	fullSearchQueryLens = L.MakeLens(
		func(req fullSearchRequest) string { return req.query },
		func(req fullSearchRequest, query string) fullSearchRequest {
			req.query = query
			return req
		},
	)

	// queryAllStopwords lifts the stopword test onto a validated request.
	queryAllStopwords = F.Pipe1(
		fullSearchAllStopwords,
		P.ContraMap(fullSearchQueryLens.Get),
	)

	// keepsNonStopword is the guard of the stopword rule: the repository
	// rejects a query whose tokens are all stopwords before any AQL runs.
	keepsNonStopword = F.Pipe1(queryAllStopwords, P.Not)
)

// nonNilFullSearchParams reports whether the raw request is present.
func nonNilFullSearchParams(params *repository.FullSearchQuery) bool {
	return params != nil
}

// hasFullSearchQueryText reports whether the normalized query has
// content.
func hasFullSearchQueryText(req fullSearchRequest) bool {
	return S.IsNonEmpty(req.query)
}

// stopwordFullSearchError reports a query whose every token is a
// stopword. The message names the normalized query, because that is the
// form the statement searches.
func stopwordFullSearchError(req fullSearchRequest) error {
	return errors.Errorf(
		"expect a query with at least one non-stopword, received %q",
		req.query,
	)
}

// newFullSearchRequest normalizes the query and clamps the limit of the
// raw request.
func newFullSearchRequest(
	params *repository.FullSearchQuery,
) fullSearchRequest {
	return fullSearchRequest{
		query:  normalizeAutocompleteQuery(params.Query),
		limit:  fullSearchLimit(params.Limit),
		entity: params.Entity,
	}
}

// validateFullSearchRequest guards the request before any AQL runs: the
// parameters must exist, the normalized query must be non-empty and
// must keep at least one non-stopword, and the entity filter must be
// defined.
func validateFullSearchRequest(
	params *repository.FullSearchQuery,
) E.Either[error, fullSearchRequest] {
	return F.Pipe5(
		params,
		E.FromPredicate(
			nonNilFullSearchParams,
			eithererr.ConstErr[*repository.FullSearchQuery](
				msgNilFullSearchQuery,
			),
		),
		E.Map[error](newFullSearchRequest),
		E.Chain(E.FromPredicate(
			hasFullSearchQueryText,
			eithererr.ConstErr[fullSearchRequest](
				msgEmptyFullSearchQuery,
			),
		)),
		E.Chain(E.FromPredicate(keepsNonStopword, stopwordFullSearchError)),
		E.BindL(fullSearchEntityLens, E.Eitherize1(autocompleteEntity)),
	)
}

// buildFullSearchQueryParams assembles the statement and the bind
// parameters from the validated request.
func (ar *arangorepository) buildFullSearchQueryParams(
	req fullSearchRequest,
) fullSearchQueryParams {
	return fullSearchQueryParams{
		ctx:       req.ctx,
		statement: buildFullSearchQuery(),
		bindParams: map[string]any{
			"q":                 req.query,
			"th":                fullSearchNgramThreshold,
			"limit":             req.limit,
			"entity":            string(req.entity),
			nameStockPropGraph:  ar.stockc.stockPropType.Name(),
			nameStockCollection: ar.stockc.stock.Name(),
		},
	}
}

// executeFullSearchQuery runs the generated statement and returns the
// raw result rows.
func (ar *arangorepository) executeFullSearchQuery(
	params fullSearchQueryParams,
) IOE.IOEither[error, dbRows] {
	return F.Pipe1(
		IOE.TryCatchError(
			func() (dbRows, error) {
				return ar.searchRows(params.ctx, params.statement, params.bindParams)
			},
		),
		IOE.MapLeft[dbRows](
			fperrors.OnError("error in running full search query"),
		),
	)
}

// emptyFullSearchResults is the None branch of toFullSearchResults: it
// reports an empty list, not a missing one.
func emptyFullSearchResults() []*repository.FullSearchResult {
	return []*repository.FullSearchResult{}
}

// toFullSearchResults converts projection rows into full search results.
// An empty input stays a non-nil empty slice, because the callers report
// an empty list and not a missing one.
func toFullSearchResults(
	rows []searchResultRow,
) []*repository.FullSearchResult {
	return F.Pipe2(
		rows,
		O.FromPredicate(predarrays.IsNonEmpty[searchResultRow]()),
		O.Fold(emptyFullSearchResults, A.Map(searchResultRow.result)),
	)
}

// SearchStock returns at most 50 ranked results for the normalized query
// of params as a lazy IOEither pipeline, forced by the caller. The
// repository trims and lowercases the query, rejects an empty,
// punctuation-only or all-stopword query, clamps the limit, and validates
// the entity filter before any AQL runs. An empty result is not an error.
func (ar *arangorepository) SearchStock(
	ctx context.Context,
	params *repository.FullSearchQuery,
) IOE.IOEither[error, []*repository.FullSearchResult] {
	baseCtx := normalizeQueryContext(ctx)
	withCtx := fullSearchCtxLens.Set(baseCtx)
	return F.Pipe6(
		IOE.Of[error](params),
		IOE.ChainEitherK(validateFullSearchRequest),
		IOE.Map[error](withCtx),
		IOE.Map[error](ar.buildFullSearchQueryParams),
		IOE.Chain(ar.executeFullSearchQuery),
		IOE.Chain(
			scanSearchRows[searchResultRow]("error in reading full search row"),
		),
		IOE.Map[error](toFullSearchResults),
	)
}

// result converts one query row into a repository full search result.
func (r searchResultRow) result() *repository.FullSearchResult {
	return &repository.FullSearchResult{
		ID:          r.ID,
		Field:       r.Field,
		DisplayText: r.Value,
		Summary:     r.Summary,
		StrainLabel: r.StrainLabel,
		Entity:      repository.StockEntityFilter(r.Entity),
		Score:       r.Score,
	}
}

// fullSearchAllStopwords reports whether every whitespace-separated
// word of the normalized query is a stopword.
func fullSearchAllStopwords(query string) bool {
	words := strings.Fields(query)
	if len(words) == 0 {
		return true
	}
	for _, w := range words {
		if _, ok := fullSearchStopwords[w]; !ok {
			return false
		}
	}
	return true
}

// fullSearchLimit clamps the requested limit: at or below 0 becomes
// the default, above 50 the hard cap.
func fullSearchLimit(limit int) int {
	if limit <= 0 {
		return defaultFullSearchLimit
	}
	if limit > maxFullSearchLimit {
		return maxFullSearchLimit
	}
	return limit
}

// ensureFullSearch creates both custom analyzers and the view. It is
// idempotent, and a failure stops the start of the service.
func (ar *arangorepository) ensureFullSearch(ctx context.Context) error {
	if _, _, err := ar.database.Handler().EnsureCreatedAnalyzer(
		ctx,
		fullSearchNormDef(),
	); err != nil {
		return errors.Errorf(
			"error in creating analyzer %s %s",
			fullSearchNormAnalyzer,
			err,
		)
	}
	if _, _, err := ar.database.Handler().EnsureCreatedAnalyzer(
		ctx,
		fullSearchNgramDef(),
	); err != nil {
		return errors.Errorf(
			"error in creating analyzer %s %s",
			fullSearchNgramAnalyzer,
			err,
		)
	}
	return ar.ensureFullSearchView(ctx)
}

// fullSearchNormDef is the norm analyzer of the prefix branches. It
// lowercases and removes accents.
func fullSearchNormDef() *driver.ArangoSearchAnalyzerDefinition {
	return &driver.ArangoSearchAnalyzerDefinition{
		Name: fullSearchNormAnalyzer,
		Type: driver.ArangoSearchAnalyzerTypeNorm,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Locale: analyzerNormLocale,
			Case:   driver.ArangoSearchCaseLower,
			Accent: new(false),
		},
	}
}

// fullSearchNgramDef is the pipeline analyzer of the fuzzy branches.
// The three features are mandatory: NGRAM_MATCH fails without them.
func fullSearchNgramDef() *driver.ArangoSearchAnalyzerDefinition {
	normStep := *fullSearchNormDef()
	return &driver.ArangoSearchAnalyzerDefinition{
		Name: fullSearchNgramAnalyzer,
		Type: driver.ArangoSearchAnalyzerTypePipeline,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Pipeline: []driver.ArangoSearchAnalyzerPipeline{
				{
					Type:       driver.ArangoSearchAnalyzerTypeNorm,
					Properties: normStep.Properties,
				},
				{
					Type: driver.ArangoSearchAnalyzerTypeNGram,
					Properties: driver.ArangoSearchAnalyzerProperties{
						Min:              new(int64(2)),
						Max:              new(int64(3)),
						PreserveOriginal: new(true),
						StreamType:       new(driver.ArangoSearchNGramStreamUTF8),
					},
				},
			},
		},
		Features: []driver.ArangoSearchAnalyzerFeature{
			driver.ArangoSearchAnalyzerFeatureFrequency,
			driver.ArangoSearchAnalyzerFeatureNorm,
			driver.ArangoSearchAnalyzerFeaturePosition,
		},
	}
}

// fullSearchLinks builds the view links from the collection names of
// the repository. It never hardcodes a production name. The identifier
// fields carry both custom analyzers; the prose fields carry the
// built-in text analyzer only, and no field carries both.
func (ar *arangorepository) fullSearchLinks() driver.ArangoSearchLinks {
	stockFields := driver.ArangoSearchFields{}
	propFields := driver.ArangoSearchFields{}
	identifier := []string{fullSearchNormAnalyzer, fullSearchNgramAnalyzer}
	prose := []string{fullSearchTextAnalyzer}
	for _, f := range fullSearchIdentifierFields {
		if f.coll == fullSearchCollStock {
			stockFields[f.label] = driver.ArangoSearchElementProperties{
				Analyzers: identifier,
			}
		} else {
			propFields[f.label] = driver.ArangoSearchElementProperties{
				Analyzers: identifier,
			}
		}
	}
	for _, f := range fullSearchProseFields {
		if f.coll == fullSearchCollStock {
			stockFields[f.label] = driver.ArangoSearchElementProperties{
				Analyzers: prose,
			}
		} else {
			propFields[f.label] = driver.ArangoSearchElementProperties{
				Analyzers: prose,
			}
		}
	}
	return driver.ArangoSearchLinks{
		ar.stockc.stock.Name():     {Fields: stockFields},
		ar.stockc.stockProp.Name(): {Fields: propFields},
	}
}

// ensureFullSearchView creates the view or reconciles a stale one.
// A stale view converges through SetProperties, which is a full
// replacement; the view itself is never deleted.
func (ar *arangorepository) ensureFullSearchView(ctx context.Context) error {
	db := ar.database.Handler()
	exists, err := db.ViewExists(ctx, fullSearchViewName)
	if err != nil {
		return errors.Errorf(
			"error in checking view %s %s",
			fullSearchViewName,
			err,
		)
	}
	if !exists {
		_, err := db.CreateArangoSearchView(
			ctx,
			fullSearchViewName,
			&driver.ArangoSearchViewProperties{Links: ar.fullSearchLinks()},
		)
		// A second repository instance can win the race.
		if err != nil && !driver.IsConflict(err) {
			return errors.Errorf(
				"error in creating view %s %s",
				fullSearchViewName,
				err,
			)
		}
		return nil
	}
	v, err := db.View(ctx, fullSearchViewName)
	if err != nil {
		return errors.Errorf("error in opening view %s %s", fullSearchViewName, err)
	}
	asv, err := v.ArangoSearchView()
	if err != nil {
		// A view of another type holds the name. Do not delete it.
		return errors.Errorf(
			"view %s is not an arangosearch view %s",
			fullSearchViewName,
			err,
		)
	}
	props, err := asv.Properties(ctx)
	if err != nil {
		return errors.Errorf(
			"error in reading view %s properties %s",
			fullSearchViewName,
			err,
		)
	}
	want := ar.fullSearchLinks()
	if sameLinkShape(props.Links, want) {
		return nil
	}
	if err := asv.SetProperties(
		ctx,
		driver.ArangoSearchViewProperties{Links: want},
	); err != nil {
		return errors.Errorf(
			"error in reconciling view %s %s",
			fullSearchViewName,
			err,
		)
	}
	return nil
}
