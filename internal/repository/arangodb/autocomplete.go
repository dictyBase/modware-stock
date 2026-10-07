package arangodb

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"

	A "github.com/IBM/fp-go/array"
	F "github.com/IBM/fp-go/function"
	driver "github.com/arangodb/go-driver"
	"github.com/cockroachdb/errors"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
	"golang.org/x/text/unicode/norm"
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

// autocompleteStockFields lists the 8 indexed fields in the order of the
// branch matrix. The first three run over the stock collection, the last
// five over the stock property collection.
var autocompleteStockFields = []autocompleteField{
	{label: paramStockID, coll: autocompleteCollStock, display: autocompleteScalar},
	{label: fieldGenes, coll: autocompleteCollStock, display: autocompleteArray},
	{label: fieldDbxrefs, coll: autocompleteCollStock, display: autocompleteArray},
	{label: fieldLabel, coll: autocompleteCollProp, display: autocompleteScalar},
	{label: fieldNames, coll: autocompleteCollProp, display: autocompleteArray},
	{label: fieldSpecies, coll: autocompleteCollProp, display: autocompleteScalar},
	{label: fieldPlasmid, coll: autocompleteCollProp, display: autocompleteScalar},
	{label: fieldName, coll: autocompleteCollProp, display: autocompleteScalar},
}

type autocompleteColl int

const (
	autocompleteCollStock autocompleteColl = iota
	autocompleteCollProp
)

type autocompleteDisplay int

const (
	autocompleteScalar autocompleteDisplay = iota
	autocompleteArray
)

type autocompleteField struct {
	label   string
	coll    autocompleteColl
	display autocompleteDisplay
}

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
			Locale: "en.utf-8",
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
	stockFields := driver.ArangoSearchFields{}
	propFields := driver.ArangoSearchFields{}
	analyzers := []string{autocompleteNormAnalyzer, autocompleteNgramAnalyzer}
	for _, f := range autocompleteStockFields {
		if f.coll == autocompleteCollStock {
			stockFields[f.label] = driver.ArangoSearchElementProperties{
				Analyzers: analyzers,
			}
		} else {
			propFields[f.label] = driver.ArangoSearchElementProperties{
				Analyzers: analyzers,
			}
		}
	}
	return driver.ArangoSearchLinks{
		ar.stockc.stock.Name():     {Fields: stockFields},
		ar.stockc.stockProp.Name(): {Fields: propFields},
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

// AutocompleteStock returns suggestions for a partial stock
// identifier, name or attribute value. The repository normalizes the
// query, clamps the limit, and validates the entity filter before any
// AQL runs.
func (ar *arangorepository) AutocompleteStock(
	params *repository.AutocompleteQuery,
) ([]*repository.Suggestion, error) {
	if params == nil {
		return nil, errors.New("expect a non-nil autocomplete query")
	}
	query := normalizeAutocompleteQuery(params.Query)
	if query == "" {
		return nil, errors.New("expect a non-empty autocomplete query")
	}
	limit := autocompleteLimit(params.Limit)
	entity, err := autocompleteEntity(params.Entity)
	if err != nil {
		return nil, err
	}
	rows, err := ar.database.SearchRows(autocompleteQuery, map[string]any{
		"q":                query,
		"th":               autocompleteNgramThreshold,
		"limit":            limit,
		"entity":           string(entity),
		nameStockPropGraph: ar.stockc.stockPropType.Name(),
	})
	if err != nil {
		return nil, errors.Errorf("error in running autocomplete query %s", err)
	}
	defer func() { _ = rows.Close() }()
	matches := make([]suggestionRow, 0, limit)
	for rows.Scan() {
		var row suggestionRow
		if err := rows.Read(&row); err != nil {
			return nil, errors.Errorf("error in reading autocomplete row %s", err)
		}
		matches = append(matches, row)
	}
	return F.Pipe1(
		matches,
		A.Map(suggestionRow.suggestion),
	), nil
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

// suggestion converts one query row into a repository suggestion.
func (r suggestionRow) suggestion() *repository.Suggestion {
	return &repository.Suggestion{
		ID:          r.ID,
		Field:       r.Field,
		DisplayText: r.Value,
		Entity:      repository.StockEntityFilter(r.Entity),
		Score:       r.Score,
	}
}

// normalizeAutocompleteQuery trims, lowercases and strips combining
// diacritical marks from the query. The calibration probe 10 decided
// this Go-side form: a punctuation-only query keeps its tokens safe,
// where an AQL TOKENS normalization would produce an empty token list
// and a null prefix argument.
func normalizeAutocompleteQuery(q string) string {
	done := F.Pipe1(
		q,
		F.Flow3(strings.TrimSpace, strings.ToLower, norm.NFD.String),
	)
	var b strings.Builder
	b.Grow(len(done))
	for _, r := range done {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
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

// autocompleteQuery is the full 16-branch statement, built once per
// process.
var autocompleteQuery = buildAutocompleteQuery()

// buildAutocompleteQuery assembles the statement from the two branch
// templates and the merge template. It emits a prefix branch and a
// fuzzy branch per field of the branch matrix, and appends the merge.
func buildAutocompleteQuery() string {
	branches := make([]string, 0, len(autocompleteStockFields)*2)
	names := make([]string, 0, len(autocompleteStockFields)*2)
	for i, f := range autocompleteStockFields {
		prefixName := fmt.Sprintf("p%d", i)
		fuzzyName := fmt.Sprintf("n%d", i)
		tpl := statement.AutocompleteStockBranch
		if f.coll == autocompleteCollProp {
			tpl = statement.AutocompletePropBranch
		}
		display := autocompleteDisplayExpr(f)
		branches = append(branches,
			fmt.Sprintf(
				tpl,
				prefixName,
				autocompletePrefixExpr(f.label),
				f.label,
				display,
				autocompletePrefixScore,
			),
			fmt.Sprintf(
				tpl,
				fuzzyName,
				autocompleteFuzzyExpr(f.label),
				f.label,
				display,
				autocompleteFuzzyScore,
			),
		)
		names = append(names, prefixName, fuzzyName)
	}
	return fmt.Sprintf(
		"%s\n%s",
		strings.Join(branches, "\n"),
		fmt.Sprintf(statement.AutocompleteMerge, strings.Join(names, ", ")),
	)
}

// autocompletePrefixExpr is the SEARCH expression of a prefix branch.
// STARTS_WITH inside SEARCH needs the ANALYZER wrapper: without it the
// comparison uses the identity analyzer and matches nothing.
func autocompletePrefixExpr(field string) string {
	return fmt.Sprintf(
		"ANALYZER(STARTS_WITH(d.%s, @q), %q)",
		field,
		autocompleteNormAnalyzer,
	)
}

// autocompleteFuzzyExpr is the SEARCH expression of a fuzzy branch.
// NGRAM_MATCH takes the analyzer as its fourth argument.
func autocompleteFuzzyExpr(field string) string {
	return fmt.Sprintf(
		"NGRAM_MATCH(d.%s, @q, @th, %q)",
		field,
		autocompleteNgramAnalyzer,
	)
}

// autocompleteDisplayExpr returns the display expression of a field.
// A scalar field returns its value; an array field returns the first
// element that contains the query, or the joined list as a fallback.
// The expressions run only on the rows that survive the branch LIMIT,
// never inside the index.
func autocompleteDisplayExpr(f autocompleteField) string {
	if f.display == autocompleteScalar {
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
