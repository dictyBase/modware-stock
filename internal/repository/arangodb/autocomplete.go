package arangodb

import (
	"context"
	"slices"

	A "github.com/IBM/fp-go/v2/array"
	E "github.com/IBM/fp-go/v2/either"
	fperrors "github.com/IBM/fp-go/v2/errors"
	F "github.com/IBM/fp-go/v2/function"
	IOE "github.com/IBM/fp-go/v2/ioeither"
	L "github.com/IBM/fp-go/v2/optics/lens"
	O "github.com/IBM/fp-go/v2/option"
	S "github.com/IBM/fp-go/v2/string"
	driver "github.com/arangodb/go-driver"
	"github.com/cockroachdb/errors"
	"github.com/dictyBase/fp-go-loom/eithererr"
	predarrays "github.com/dictyBase/fp-go-loom/predicate/array"
	"github.com/dictyBase/modware-stock/internal/repository"
)

type autocompleteDisplay int

// autocompleteField is one indexed search field. The label names the
// field in the AQL, and the display selects the display expression.
type autocompleteField struct {
	label   string
	display autocompleteDisplay
}

// autocompleteRequest is the validated and normalized input of one
// autocomplete search.
type autocompleteRequest struct {
	query  string
	limit  int
	entity repository.StockEntityFilter
}

// autocompleteQueryParams carries the generated statement and its
// bind parameters for one autocomplete search.
type autocompleteQueryParams struct {
	statement  string
	bindParams map[string]any
}

// suggestionRow mirrors one object row of the autocomplete projection.
// go-driver v1 cannot decode an array row, so the projection returns
// one object per row.
type suggestionRow struct {
	Key    string  `json:"k"`
	ID     string  `json:"id"`
	Entity string  `json:"entity"`
	Field  string  `json:"f"`
	Value  string  `json:"v"`
	Score  float64 `json:"s"`
}

const (
	autocompleteScalar autocompleteDisplay = iota
	autocompleteArray
)

const (
	msgNilAutocompleteQuery   = "expect a non-nil autocomplete query"
	msgEmptyAutocompleteQuery = "expect a non-empty autocomplete query"
)

// Search asset names owned by the autocomplete feature. Every name
// belongs to this feature alone, so other search features can be
// absent, present or stale without any effect.
const (
	// autocompleteViewName is the classic arangosearch view of this
	// feature. The AQL templates name it literally, because ArangoDB
	// does not accept a bind parameter for a view name in SEARCH.
	autocompleteViewName = "stock_autocomplete"
	// autocompleteNormAnalyzer lowercases a value and removes accents.
	// It serves the prefix branches.
	autocompleteNormAnalyzer = "stock_autocomplete_norm"
	// autocompleteNgramAnalyzer chains norm and ngram. It serves the
	// fuzzy branches.
	autocompleteNgramAnalyzer = "stock_autocomplete_ngram"
	// autocompleteNgramThreshold is the minimum n-gram similarity of a
	// fuzzy match. Task 1 calibration on ArangoDB 3.11 confirmed 0.30:
	// 0.45 misses one-character typos in 4-character values (ys14 vs
	// yS13) and in 10-character identifiers (dbs0236127 vs
	// DBS0236126), while 0.30 still admits no junk.
	autocompleteNgramThreshold = 0.3
	// defaultAutocompleteLimit is the list length for a non-positive
	// limit.
	defaultAutocompleteLimit = 5
	// maxAutocompleteLimit is the hard cap of the list length.
	maxAutocompleteLimit = 50

	// autocompletePrefixScore is the score bonus of a prefix branch. A
	// prefix match always ranks above a fuzzy match, and the BM25 of a
	// pure STARTS_WITH match can be 0, so the deterministic tiebreak on
	// the stock key is required.
	autocompletePrefixScore = "1000 + BM25(d)"
	// autocompleteFuzzyScore is the score of a fuzzy branch.
	autocompleteFuzzyScore = "BM25(d)"
)

// stockFields lists the indexed fields of the stock collection. Each
// field gets a prefix branch and a fuzzy branch in the statement.
var stockFields = []autocompleteField{
	{label: paramStockID, display: autocompleteScalar},
	{label: fieldGenes, display: autocompleteArray},
	{label: fieldDbxrefs, display: autocompleteArray},
}

// propFields lists the indexed fields of the stock property
// collection. Each field gets a prefix branch and a fuzzy branch in
// the statement.
var propFields = []autocompleteField{
	{label: fieldLabel, display: autocompleteScalar},
	{label: fieldNames, display: autocompleteArray},
	{label: fieldSpecies, display: autocompleteScalar},
	{label: fieldPlasmid, display: autocompleteScalar},
	{label: fieldName, display: autocompleteScalar},
}

// entityLens focuses the entity filter of a validated request.
var entityLens = L.MakeLens(
	func(req autocompleteRequest) repository.StockEntityFilter {
		return req.entity
	},
	func(
		req autocompleteRequest,
		entity repository.StockEntityFilter,
	) autocompleteRequest {
		req.entity = entity
		return req
	},
)

// ensureAutocompleteSearch creates both analyzers and the view. It is
// idempotent, and a failure stops the start of the service.
func (ar *arangorepository) ensureAutocompleteSearch(ctx context.Context) error {
	_, _, err := ar.database.Handler().EnsureCreatedAnalyzer(
		ctx,
		autocompleteNormDef(),
	)
	if err != nil {
		return errors.Errorf(
			"error in creating analyzer %s %s",
			autocompleteNormAnalyzer,
			err,
		)
	}
	_, _, err = ar.database.Handler().EnsureCreatedAnalyzer(
		ctx,
		autocompleteNgramDef(),
	)
	if err != nil {
		return errors.Errorf(
			"error in creating analyzer %s %s",
			autocompleteNgramAnalyzer,
			err,
		)
	}
	return ar.ensureAutocompleteView(ctx)
}

// autocompleteNormDef is the norm analyzer of the prefix branches. It
// lowercases and removes accents.
func autocompleteNormDef() *driver.ArangoSearchAnalyzerDefinition {
	return &driver.ArangoSearchAnalyzerDefinition{
		Name: autocompleteNormAnalyzer,
		Type: driver.ArangoSearchAnalyzerTypeNorm,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Locale: analyzerNormLocale,
			Case:   driver.ArangoSearchCaseLower,
			Accent: new(false),
		},
	}
}

// autocompleteNgramDef is the pipeline analyzer of the fuzzy branches.
// The three features are mandatory: NGRAM_MATCH fails without them.
func autocompleteNgramDef() *driver.ArangoSearchAnalyzerDefinition {
	normStep := *autocompleteNormDef()
	return &driver.ArangoSearchAnalyzerDefinition{
		Name: autocompleteNgramAnalyzer,
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

// autocompleteLinks builds the view links from the collection names of
// the repository. It never hardcodes a production name.
func (ar *arangorepository) autocompleteLinks() driver.ArangoSearchLinks {
	stockIndex := driver.ArangoSearchFields{}
	propIndex := driver.ArangoSearchFields{}
	analyzers := []string{autocompleteNormAnalyzer, autocompleteNgramAnalyzer}
	for _, f := range stockFields {
		stockIndex[f.label] = driver.ArangoSearchElementProperties{
			Analyzers: analyzers,
		}
	}
	for _, f := range propFields {
		propIndex[f.label] = driver.ArangoSearchElementProperties{
			Analyzers: analyzers,
		}
	}
	return driver.ArangoSearchLinks{
		ar.stockc.stock.Name():     {Fields: stockIndex},
		ar.stockc.stockProp.Name(): {Fields: propIndex},
	}
}

// ensureAutocompleteView creates the view or reconciles a stale one.
// A stale view converges through SetProperties, which is a full
// replacement; the view itself is never deleted.
func (ar *arangorepository) ensureAutocompleteView(ctx context.Context) error {
	db := ar.database.Handler()
	exists, err := db.ViewExists(ctx, autocompleteViewName)
	if err != nil {
		return errors.Errorf(
			"error in checking view %s %s",
			autocompleteViewName,
			err,
		)
	}
	if !exists {
		_, err := db.CreateArangoSearchView(
			ctx,
			autocompleteViewName,
			&driver.ArangoSearchViewProperties{Links: ar.autocompleteLinks()},
		)
		// A second repository instance can win the race.
		if err != nil && !driver.IsConflict(err) {
			return errors.Errorf(
				"error in creating view %s %s",
				autocompleteViewName,
				err,
			)
		}
		return nil
	}
	v, err := db.View(ctx, autocompleteViewName)
	if err != nil {
		return errors.Errorf("error in opening view %s %s", autocompleteViewName, err)
	}
	asv, err := v.ArangoSearchView()
	if err != nil {
		// A view of another type holds the name. Do not delete it.
		return errors.Errorf(
			"view %s is not an arangosearch view %s",
			autocompleteViewName,
			err,
		)
	}
	props, err := asv.Properties(ctx)
	if err != nil {
		return errors.Errorf(
			"error in reading view %s properties %s",
			autocompleteViewName,
			err,
		)
	}
	want := ar.autocompleteLinks()
	if sameLinkShape(props.Links, want) {
		return nil
	}
	if err := asv.SetProperties(
		ctx,
		driver.ArangoSearchViewProperties{Links: want},
	); err != nil {
		return errors.Errorf(
			"error in reconciling view %s %s",
			autocompleteViewName,
			err,
		)
	}
	return nil
}

// sameLinkShape compares only the shape that this feature controls: the
// link keys, the field names per link and the analyzer names per field.
// A deep comparison would always report a difference, because the
// server fills defaults into the response.
func sameLinkShape(got, want driver.ArangoSearchLinks) bool {
	if len(got) != len(want) {
		return false
	}
	for coll, gl := range got {
		wl, ok := want[coll]
		if !ok || !sameFieldShape(gl.Fields, wl.Fields) {
			return false
		}
	}
	return true
}

func sameFieldShape(got, want driver.ArangoSearchFields) bool {
	if len(got) != len(want) {
		return false
	}
	for field, gf := range got {
		wf, ok := want[field]
		if !ok || !slices.Equal(gf.Analyzers, wf.Analyzers) {
			return false
		}
	}
	return true
}

// nonNilQueryParams reports whether the raw request is present.
func nonNilQueryParams(params *repository.AutocompleteQuery) bool {
	return params != nil
}

// hasQueryText reports whether the normalized query has content.
func hasQueryText(req autocompleteRequest) bool {
	return S.IsNonEmpty(req.query)
}

// newAutocompleteRequest normalizes the query and clamps the limit of
// the raw request.
func newAutocompleteRequest(
	params *repository.AutocompleteQuery,
) autocompleteRequest {
	return autocompleteRequest{
		query:  normalizeAutocompleteQuery(params.Query),
		limit:  autocompleteLimit(params.Limit),
		entity: params.Entity,
	}
}

// validateAutocompleteRequest guards the request before any AQL runs:
// the parameters must exist, the normalized query must be non-empty
// and the entity filter must be defined.
func validateAutocompleteRequest(
	params *repository.AutocompleteQuery,
) IOE.IOEither[error, autocompleteRequest] {
	return F.Pipe5(
		params,
		E.FromPredicate(
			nonNilQueryParams,
			eithererr.ConstErr[*repository.AutocompleteQuery](
				msgNilAutocompleteQuery,
			),
		),
		E.Map[error](newAutocompleteRequest),
		E.Chain(E.FromPredicate(
			hasQueryText,
			eithererr.ConstErr[autocompleteRequest](
				msgEmptyAutocompleteQuery,
			),
		)),
		E.BindL(entityLens, E.Eitherize1(autocompleteEntity)),
		IOE.FromEither[error, autocompleteRequest],
	)
}

// buildAutocompleteQueryParams assembles the statement and the bind
// parameters from the validated request.
func (ar *arangorepository) buildAutocompleteQueryParams(
	req autocompleteRequest,
) autocompleteQueryParams {
	return autocompleteQueryParams{
		statement: buildAutocompleteQuery(),
		bindParams: map[string]any{
			"q":                req.query,
			"th":               autocompleteNgramThreshold,
			"limit":            req.limit,
			"entity":           string(req.entity),
			nameStockPropGraph: ar.stockc.stockPropType.Name(),
		},
	}
}

// executeAutocompleteQuery runs the generated statement and returns
// the raw result rows.
func (ar *arangorepository) executeAutocompleteQuery(
	params autocompleteQueryParams,
) IOE.IOEither[error, dbRows] {
	return F.Pipe1(
		IOE.TryCatchError(
			func() (dbRows, error) {
				return ar.database.SearchRows(params.statement, params.bindParams)
			},
		),
		IOE.MapLeft[dbRows](
			fperrors.OnError("error in running autocomplete query"),
		),
	)
}

// scanAutocompleteRows drains the cursor into projection rows. The
// go-driver cursor is a pull iterator, so the raw callback keeps the
// scan loop and the defer close.
//
// fp-go:allow-trycatch-iferr the go-driver cursor reports a read
// error only inside the scan loop
func scanAutocompleteRows(rows dbRows) IOE.IOEither[error, []suggestionRow] {
	return F.Pipe1(
		IOE.TryCatchError(
			func() ([]suggestionRow, error) {
				defer func() { _ = rows.Close() }()
				matches := []suggestionRow{}
				for rows.Scan() {
					var row suggestionRow
					if err := rows.Read(&row); err != nil {
						return nil, err
					}
					matches = append(matches, row)
				}
				return matches, nil
			},
		),
		IOE.MapLeft[[]suggestionRow](
			fperrors.OnError("error in reading autocomplete row"),
		),
	)
}

// AutocompleteStock returns suggestions for a partial stock
// identifier, name or attribute value. The repository normalizes the
// query, clamps the limit, and validates the entity filter before any
// AQL runs.
func (ar *arangorepository) AutocompleteStock(
	params *repository.AutocompleteQuery,
) IOE.IOEither[error, []*repository.Suggestion] {
	return F.Pipe5(
		params,
		validateAutocompleteRequest,
		IOE.Map[error](ar.buildAutocompleteQueryParams),
		IOE.Chain(ar.executeAutocompleteQuery),
		IOE.Chain(scanAutocompleteRows),
		IOE.Map[error](toSuggestions),
	)
}

// toSuggestions converts projection rows into suggestions. An empty
// input stays a non-nil empty slice, because the callers report an
// empty list and not a missing one.
func toSuggestions(rows []suggestionRow) []*repository.Suggestion {
	return F.Pipe2(
		rows,
		O.FromPredicate(predarrays.IsNonEmpty[suggestionRow]()),
		O.Fold(emptySuggestions, A.Map(toSuggestion)),
	)
}

// emptySuggestions is the None branch of toSuggestions: it reports an
// empty list, not a missing one.
func emptySuggestions() []*repository.Suggestion {
	return []*repository.Suggestion{}
}

// toSuggestion converts one projection row into a repository
// suggestion.
func toSuggestion(r suggestionRow) *repository.Suggestion {
	return &repository.Suggestion{
		ID:          r.ID,
		Field:       r.Field,
		DisplayText: r.Value,
		Entity:      repository.StockEntityFilter(r.Entity),
		Score:       r.Score,
	}
}

// autocompleteLimit clamps the requested limit: at or below 0 becomes
// the default, above 50 the hard cap.
func autocompleteLimit(limit int) int {
	if limit <= 0 {
		return defaultAutocompleteLimit
	}
	if limit > maxAutocompleteLimit {
		return maxAutocompleteLimit
	}
	return limit
}

// autocompleteEntity validates the entity filter against the three
// repository constants.
func autocompleteEntity(entity repository.StockEntityFilter) (repository.StockEntityFilter, error) {
	switch entity {
	case repository.EntityBoth, repository.EntityStrain, repository.EntityPlasmid:
		return entity, nil
	default:
		return "", errors.Errorf(
			"expect a valid stock entity, received %s",
			entity,
		)
	}
}
