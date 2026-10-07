package arangodb

import (
	"context"
	"fmt"
	"strings"

	A "github.com/IBM/fp-go/array"
	F "github.com/IBM/fp-go/function"
	driver "github.com/arangodb/go-driver"
	"github.com/cockroachdb/errors"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

// Search asset names owned by the full search feature. Every name
// belongs to this feature alone, so other search features can be
// absent, present or stale without any effect.
const (
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
)

// fullSearchIdentifierFields lists the 8 identifier and name fields
// that the view indexes with both custom analyzers. The first three
// live on the stock collection, the last five on the stock property
// collection. Each field serves one prefix branch and one fuzzy branch.
var fullSearchIdentifierFields = []struct {
	label   string
	coll    fullSearchColl
	display fullSearchDisplay
}{
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
var fullSearchProseFields = []struct {
	label   string
	coll    fullSearchColl
	display fullSearchDisplay
}{
	{label: fieldSummary, coll: fullSearchCollStock, display: fullSearchScalar},
	{label: fieldDepositor, coll: fullSearchCollStock, display: fullSearchScalar},
}

type fullSearchColl int

const (
	fullSearchCollStock fullSearchColl = iota
	fullSearchCollProp
)

type fullSearchDisplay int

const (
	fullSearchScalar fullSearchDisplay = iota
	fullSearchArray
)

// SearchStock returns at most 50 ranked results for the normalized
// query of params. The repository trims and lowercases the query,
// rejects an empty, punctuation-only or all-stopword query, clamps the
// limit, and validates the entity filter before any AQL runs. An empty
// result is not an error.
func (ar *arangorepository) SearchStock(
	params *repository.FullSearchQuery,
) ([]*repository.FullSearchResult, error) {
	if params == nil {
		return nil, errors.New("expect a non-nil full search query")
	}
	query := normalizeAutocompleteQuery(params.Query)
	if query == "" {
		return nil, errors.New("expect a non-empty full search query")
	}
	if fullSearchAllStopwords(query) {
		return nil, errors.Errorf(
			"expect a query with at least one non-stopword, received %q",
			params.Query,
		)
	}
	limit := fullSearchLimit(params.Limit)
	entity, err := autocompleteEntity(params.Entity)
	if err != nil {
		return nil, err
	}
	rows, err := ar.database.SearchRows(fullSearchQuery, map[string]any{
		"q":                 query,
		"th":                fullSearchNgramThreshold,
		"limit":             limit,
		"entity":            string(entity),
		nameStockPropGraph:  ar.stockc.stockPropType.Name(),
		nameStockCollection: ar.stockc.stock.Name(),
	})
	if err != nil {
		return nil, errors.Errorf("error in running full search query %s", err)
	}
	defer func() { _ = rows.Close() }()
	matches := make([]searchResultRow, 0, limit)
	for rows.Scan() {
		var row searchResultRow
		if err := rows.Read(&row); err != nil {
			return nil, errors.Errorf("error in reading full search row %s", err)
		}
		matches = append(matches, row)
	}
	return F.Pipe1(
		matches,
		A.Map(searchResultRow.result),
	), nil
}

// searchResultRow mirrors one object row of the full search projection.
// go-driver v1 cannot decode an array row, so the projection returns one
// object per row.
type searchResultRow struct {
	Key         string  `json:"k"`
	ID          string  `json:"id"`
	Entity      string  `json:"entity"`
	Field       string  `json:"f"`
	Value       string  `json:"v"`
	Summary     string  `json:"sm"`
	StrainLabel string  `json:"sl"`
	Score       float64 `json:"s"`
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

// fullSearchStopwords is the short stopword list of the token stage.
// Task 1 probe 18 recorded that the built-in text_en analyzer does not
// remove stopwords, so the repository rejects a query whose tokens are
// all stopwords before any AQL runs.
var fullSearchStopwords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {},
	"be": {}, "but": {}, "by": {}, "for": {}, "from": {}, "had": {},
	"has": {}, "have": {}, "if": {}, "in": {}, "is": {}, "it": {},
	"its": {}, "not": {}, "of": {}, "on": {}, "or": {}, "so": {},
	"than": {}, "that": {}, "the": {}, "their": {}, "then": {},
	"there": {}, "these": {}, "this": {}, "to": {}, "too": {},
	"very": {}, "was": {}, "were": {}, "what": {}, "when": {},
	"which": {}, "while": {}, "who": {}, "will": {}, "with": {},
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

// fullSearchQuery is the full 19-branch statement, built once per
// process.
var fullSearchQuery = buildFullSearchQuery()

// buildFullSearchQuery assembles the statement from the two branch
// templates and the merge template. It emits a prefix branch and a
// fuzzy branch per identifier field, a token branch per prose field,
// the single phrase branch over summary, and appends the merge.
func buildFullSearchQuery() string {
	branches := make([]string, 0, len(fullSearchIdentifierFields)*2+3)
	names := make([]string, 0, len(fullSearchIdentifierFields)*2+3)
	for i, f := range fullSearchIdentifierFields {
		display := fullSearchDisplayExpr(f)
		tpl := statement.FullSearchStockBranch
		if f.coll == fullSearchCollProp {
			tpl = statement.FullSearchPropBranch
		}
		branches = append(branches,
			fmt.Sprintf(
				tpl,
				fmt.Sprintf("p%d", i),
				fullSearchPrefixExpr(f.label),
				f.label,
				display,
				fullSearchPrefixBand,
			),
			fmt.Sprintf(
				tpl,
				fmt.Sprintf("n%d", i),
				fullSearchFuzzyExpr(f.label),
				f.label,
				display,
				fullSearchFuzzyBand,
			),
		)
		names = append(names, fmt.Sprintf("p%d", i), fmt.Sprintf("n%d", i))
	}
	for i, f := range fullSearchProseFields {
		display := fullSearchDisplayExpr(f)
		branches = append(branches, fmt.Sprintf(
			statement.FullSearchStockBranch,
			fmt.Sprintf("t%d", i),
			fullSearchTokenExpr(f.label),
			f.label,
			display,
			fullSearchProseBand,
		))
		names = append(names, fmt.Sprintf("t%d", i))
	}
	// The single phrase branch runs over summary only.
	branches = append(branches, fmt.Sprintf(
		statement.FullSearchStockBranch,
		"h0",
		fullSearchPhraseExpr(fieldSummary),
		fieldSummary,
		fullSearchDisplayExpr(fullSearchProseFields[0]),
		fullSearchPhraseBand,
	))
	names = append(names, "h0")
	return fmt.Sprintf(
		"%s\n%s",
		strings.Join(branches, "\n"),
		fmt.Sprintf(statement.FullSearchMerge, strings.Join(names, ", ")),
	)
}

// fullSearchPrefixExpr is the SEARCH expression of a prefix branch.
// STARTS_WITH inside SEARCH needs the ANALYZER wrapper: without it the
// comparison uses the identity analyzer and matches nothing.
func fullSearchPrefixExpr(field string) string {
	return fmt.Sprintf(
		"ANALYZER(STARTS_WITH(d.%s, @q), %q)",
		field,
		fullSearchNormAnalyzer,
	)
}

// fullSearchFuzzyExpr is the SEARCH expression of a fuzzy branch.
// NGRAM_MATCH takes the analyzer as its fourth argument.
func fullSearchFuzzyExpr(field string) string {
	return fmt.Sprintf(
		"NGRAM_MATCH(d.%s, @q, @th, %q)",
		field,
		fullSearchNgramAnalyzer,
	)
}

// fullSearchTokenExpr is the SEARCH expression of a token branch. The
// IN TOKENS form needs the ANALYZER wrapper, because the comparison of
// the indexed value against the token list must run under the same
// analyzer that produced the index.
func fullSearchTokenExpr(field string) string {
	return fmt.Sprintf(
		"ANALYZER(d.%s IN TOKENS(@q, %q), %q)",
		field,
		fullSearchTextAnalyzer,
		fullSearchTextAnalyzer,
	)
}

// fullSearchPhraseExpr is the SEARCH expression of the phrase branch.
// PHRASE takes the analyzer as its last argument.
func fullSearchPhraseExpr(field string) string {
	return fmt.Sprintf(
		"PHRASE(d.%s, @q, %q)",
		field,
		fullSearchTextAnalyzer,
	)
}

// fullSearchDisplayExpr returns the display expression of a field.
// A scalar field returns its value; an array field returns the first
// element that contains the query, or the joined list as a fallback.
// The expressions run only on the rows that survive the branch LIMIT,
// never inside the index.
func fullSearchDisplayExpr(f struct {
	label   string
	coll    fullSearchColl
	display fullSearchDisplay
}) string {
	if f.display == fullSearchScalar {
		return fmt.Sprintf(`NOT_NULL(d.%s, "")`, f.label)
	}
	return fmt.Sprintf(
		`NOT_NULL(FIRST(FOR item IN NOT_NULL(d.%s, []) `+
			`FILTER CONTAINS(LOWER(item), @q) RETURN item), `+
			`CONCAT_SEPARATOR(", ", NOT_NULL(d.%s, [])))`,
		f.label,
		f.label,
	)
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
