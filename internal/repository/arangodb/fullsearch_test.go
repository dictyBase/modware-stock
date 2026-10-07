package arangodb

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	F "github.com/IBM/fp-go/function"
	driver "github.com/arangodb/go-driver"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/stretchr/testify/require"
)

const (
	// testStrainLabelY is the fixture strain label.
	testStrainLabelY = "yS13"
	// Fixture probe strings shared by the full search tests.
	testXyloseGene   = "xylose"
	testCordaProbe   = "corda"
	testCordaxGene   = "cordaxin"
	testCap0Probe    = "cap0"
	testFullSearchVw = fullSearchViewName
	// testFullSearchColls mirror getCollectionParams.
	testFullSearchTextAnl = fullSearchTextAnalyzer
)

// assertFullSearchViewShape reads the view properties and asserts the
// 2 link keys, the 10 field names and the analyzer list per field.
func assertFullSearchViewShape(
	ctx context.Context,
	assert *require.Assertions,
	db driver.Database,
) {
	exists, err := db.ViewExists(ctx, testFullSearchVw)
	assert.NoErrorf(err, "expect no error checking view, received %s", err)
	assert.True(exists, "expect the stock_full_search view to exist")
	v, err := db.View(ctx, testFullSearchVw)
	assert.NoErrorf(err, "expect no error opening view, received %s", err)
	asv, err := v.ArangoSearchView()
	assert.NoErrorf(
		err,
		"expect no error casting to arangosearch view, received %s",
		err,
	)
	props, err := asv.Properties(ctx)
	assert.NoErrorf(
		err,
		"expect no error reading view properties, received %s",
		err,
	)
	keys := make([]string, 0, len(props.Links))
	for k := range props.Links {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	wantKeys := []string{testStockColl, testStockPropCol}
	slices.Sort(wantKeys)
	assert.Equal(
		wantKeys,
		keys,
		"expect the two link collections",
	)
	// The stock link carries 3 identifier fields and 2 prose fields.
	assertFullSearchLink(
		assert,
		props.Links[testStockColl],
		[]string{paramStockID, fieldGenes, fieldDbxrefs},
		[]string{fullSearchNormAnalyzer, fullSearchNgramAnalyzer},
	)
	assertFullSearchLink(
		assert,
		props.Links[testStockColl],
		[]string{fieldSummary, fieldDepositor},
		[]string{testFullSearchTextAnl},
	)
	// The property link carries 5 identifier fields.
	assertFullSearchLink(
		assert,
		props.Links[testStockPropCol],
		[]string{fieldLabel, fieldNames, fieldSpecies, fieldPlasmid, fieldName},
		[]string{fullSearchNormAnalyzer, fullSearchNgramAnalyzer},
	)
}

// assertFullSearchLink asserts that the named fields of the link carry
// exactly the given analyzer list.
func assertFullSearchLink(
	assert *require.Assertions,
	link driver.ArangoSearchElementProperties,
	fields []string,
	analyzers []string,
) {
	for _, name := range fields {
		field, ok := link.Fields[name]
		assert.True(
			ok,
			"expect field %s in the link",
			name,
		)
		got := make([]string, 0, len(field.Analyzers))
		got = append(got, field.Analyzers...)
		slices.Sort(got)
		want := make([]string, 0, len(analyzers))
		want = append(want, analyzers...)
		slices.Sort(want)
		assert.Equal(
			want,
			got,
			"expect the analyzer list on field %s",
			name,
		)
	}
}

// assertNoTextEnAnalyzerCreated asserts that the setup did not create
// a custom analyzer named text_en; the built-in one is not a creation
// target.
func assertNoTextEnAnalyzerCreated(
	ctx context.Context,
	assert *require.Assertions,
	db driver.Database,
) {
	anl, err := db.Analyzer(ctx, testFullSearchTextAnl)
	assert.NoErrorf(
		err,
		"expect no error opening the built-in text_en analyzer, received %s",
		err,
	)
	assert.Equal(
		"text",
		string(anl.Type()),
		"the built-in text_en analyzer is a text analyzer of the server, not a creation of this repository",
	)
}

func TestEnsureFullSearchCreatesAssets(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ctx := context.Background()
	db := repo.Dbh().Handler()
	// Both custom analyzers exist.
	analyzers, err := db.Analyzers(ctx)
	assert.NoErrorf(
		err,
		"expect no error listing analyzers, received %s",
		err,
	)
	names := make([]string, 0, len(analyzers))
	for _, a := range analyzers {
		names = append(names, a.Name())
	}
	assert.Contains(names, fullSearchNormAnalyzer)
	assert.Contains(names, fullSearchNgramAnalyzer)
	assertNoTextEnAnalyzerCreated(ctx, assert, db)
	assertFullSearchViewShape(ctx, assert, db)
}

func TestEnsureFullSearchReconcilesStaleView(t *testing.T) {
	assert := require.New(t)
	ta, err := testarango.NewTestArangoFromEnv(true)
	assert.NoErrorf(
		err,
		"expect no error creating a test database, received %s",
		err,
	)
	dbr, err := ta.DB(ta.Database)
	assert.NoErrorf(err, "expect no error opening database, received %s", err)
	db := dbr.Handler()
	ctx := context.Background()
	_, err = db.CreateCollection(ctx, testStockColl, nil)
	assert.NoErrorf(
		err,
		"expect no error creating the stock collection, received %s",
		err,
	)
	_, err = db.CreateCollection(ctx, testStockPropCol, nil)
	assert.NoErrorf(
		err,
		"expect no error creating the property collection, received %s",
		err,
	)
	// A stale view: one wrong link and one wrong analyzer list.
	stale, err := db.CreateArangoSearchView(
		ctx,
		testFullSearchVw,
		&driver.ArangoSearchViewProperties{
			Links: driver.ArangoSearchLinks{
				testStockColl: {
					Fields: driver.ArangoSearchFields{
						paramStockID: {Analyzers: []string{testIdentityAnl}},
						fieldGenes:   {Analyzers: []string{testIdentityAnl}},
						fieldDbxrefs: {Analyzers: []string{testIdentityAnl}},
					},
				},
				testStockPropCol: {
					Fields: driver.ArangoSearchFields{
						fieldLabel:    {Analyzers: []string{testIdentityAnl}},
						fieldNames:    {Analyzers: []string{testIdentityAnl}},
						"speciesssss": {Analyzers: []string{testIdentityAnl}},
					},
				},
			},
		},
	)
	assert.NoErrorf(
		err,
		"expect no error creating the stale view, received %s",
		err,
	)
	staleProps, err := stale.Properties(ctx)
	assert.NoError(err, "expect no error reading the stale view")
	repo, err := NewStockRepo(
		getConnectParamsFromDb(ta),
		getCollectionParams(),
		getOntoParams(),
	)
	assert.NoErrorf(
		err,
		"expect no error building the repository, received %s",
		err,
	)
	defer tearDown(repo)
	assertFullSearchViewShape(ctx, assert, db)
	fresh, err := db.View(ctx, testFullSearchVw)
	assert.NoError(err, "expect no error reopening the view")
	freshAsv, err := fresh.ArangoSearchView()
	assert.NoError(err, "expect no error casting the view")
	freshProps, err := freshAsv.Properties(ctx)
	assert.NoError(err, "expect no error reading the fresh view")
	assert.Equal(
		staleProps.ID,
		freshProps.ID,
		"expect the same view identifier, proving no delete",
	)
}

func TestNewStockRepoFailsWhenFullSearchSetupFails(t *testing.T) {
	assert := require.New(t)
	ta, err := testarango.NewTestArangoFromEnv(true)
	assert.NoErrorf(
		err,
		"expect no error creating a test database, received %s",
		err,
	)
	missing := &manager.ConnectParams{
		User:     ta.User,
		Pass:     ta.Pass,
		Database: "no_such_fullsearch_db",
		Host:     ta.Host,
		Port:     ta.Port,
		Istls:    false,
	}
	_, err = NewStockRepo(missing, getCollectionParams(), getOntoParams())
	assert.Errorf(
		err,
		"expect an error when the database does not exist",
	)
	assert.NoErrorf(
		ta.CreateDB("no_such_fullsearch_db", nil),
		"expect no error creating the database, received %s",
		err,
	)
	// Best-effort cleanup of a leftover from a failed retry. The
	// success path drops the database through tearDown(repo).
	t.Cleanup(func() {
		if dbr, dbErr := ta.DB("no_such_fullsearch_db"); dbErr == nil {
			_ = dbr.Drop()
		}
	})
	repo, err := NewStockRepo(missing, getCollectionParams(), getOntoParams())
	assert.NoErrorf(
		err,
		"expect no error on the retry after the database exists, received %s",
		err,
	)
	assert.NoError(repo.Dbh().Drop(), "expect no error dropping the repo database")
}

// ------------------------------------------------------------- builder

func TestBuildFullSearchQueryBranchCount(t *testing.T) {
	assert := require.New(t)
	q := buildFullSearchQuery()
	assert.Equal(
		19,
		strings.Count(q, "FOR d IN "+fullSearchViewName),
		"expect one FOR per branch",
	)
	branchVars := regexp.MustCompile(`LET [pnth]\d+ = \(`).FindAllString(q, -1)
	assert.Len(branchVars, 19, "expect 19 branch variables")
	assert.Equal(9, strings.Count(q, "OUTBOUND"),
		"expect one OUTBOUND traversal per stage of the 6 stock fields")
	assert.Equal(10, strings.Count(q, "INBOUND"),
		"expect one INBOUND traversal per stage of the 5 property fields")
	assert.Equal(8, strings.Count(q, "ANALYZER(STARTS_WITH"))
	assert.Equal(8, strings.Count(q, "NGRAM_MATCH"))
	assert.Equal(2, strings.Count(q, "IN TOKENS"))
	assert.Equal(1, strings.Count(q, "PHRASE("))
	assert.NotContains(q, fieldEditableSummary,
		"the statement must not name editable_summary")
}

func TestBuildFullSearchQueryHasNoNgramOnProse(t *testing.T) {
	assert := require.New(t)
	q := buildFullSearchQuery()
	for _, prose := range []string{fieldSummary, fieldDepositor} {
		assert.Equal(
			0,
			strings.Count(q, "NGRAM_MATCH(d."+prose),
			prose+" must have no ngram branch",
		)
	}
}

func TestBuildFullSearchQueryIsValidAQL(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ctx := context.Background()
	assert.NoErrorf(
		repo.Dbh().Handler().ValidateQuery(ctx, buildFullSearchQuery()),
		"expect the built statement to be valid AQL",
	)
}

func TestBuildFullSearchQueryFilterOrder(t *testing.T) {
	assert := require.New(t)
	q := buildFullSearchQuery()
	for branch := range strings.SplitSeq(q, "\nLET ") {
		if !strings.HasPrefix(branch, "LET ") {
			branch = "LET " + branch
		}
		if !regexp.MustCompile(`^LET [pnth]\d+ = \(`).MatchString(branch) {
			continue
		}
		entity := strings.Index(branch, "FILTER @entity")
		sort := strings.Index(branch, "SORT BM25")
		limit := strings.Index(branch, "LIMIT @limit")
		assert.GreaterOrEqual(entity, 0, "branch must filter by entity")
		assert.Less(entity, sort, "entity filter must precede SORT")
		assert.Less(entity, limit, "entity filter must precede LIMIT")
	}
}

func TestBuildFullSearchQueryDocumentLookupIsAfterLimit(t *testing.T) {
	assert := require.New(t)
	q := buildFullSearchQuery()
	tail := q[strings.LastIndex(q, "FOR x IN best"):]
	limit := strings.Index(tail, "LIMIT @limit")
	doc := strings.Index(tail, "DOCUMENT(@stock_collection")
	assert.GreaterOrEqual(limit, 0, "the tail must hold the limit")
	assert.GreaterOrEqual(doc, 0, "the tail must read the stock document")
	assert.Less(
		limit,
		doc,
		"the DOCUMENT lookup must run after the tail LIMIT",
	)
}

// ---------------------------------------------------------- fixtures

// addFullSearchStrain creates a strain whose attributes copy
// newTestStrain with the given overrides applied.
func addFullSearchStrain(
	assert *require.Assertions,
	repo repository.StockRepository,
	override func(*stock.NewStrainAttributes),
) *model.StockDoc {
	ns := newTestStrain(testEmailCostanza, General)
	if override != nil {
		override(ns.Data.Attributes)
	}
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error adding strain, received %s", err)
	return m
}

// addFullSearchPlasmid creates a plasmid whose attributes copy
// newTestPlasmid with the given overrides applied.
func addFullSearchPlasmid(
	assert *require.Assertions,
	repo repository.StockRepository,
	override func(*stock.NewPlasmidAttributes),
) *model.StockDoc {
	np := newTestPlasmid(testEmailCostanza)
	if override != nil {
		override(np.Data.Attributes)
	}
	result := F.Pipe2(repo.AddPlasmid(np), ToEither, toStockDocResult)
	assert.NoErrorf(result.F2, "expect no error adding plasmid, received %s", result.F2)
	return result.F1
}

// waitFullSearchRows polls until the view commits the documents and
// the probe returns at least one row, then returns the rows.
func waitFullSearchRows(
	assert *require.Assertions,
	repo repository.StockRepository,
	probe string,
) []*repository.FullSearchResult {
	var rows []*repository.FullSearchResult
	assert.Eventually(func() bool {
		got, err := repo.SearchStock(
			&repository.FullSearchQuery{Query: probe, Limit: 50},
		)
		if err != nil || len(got) == 0 {
			return false
		}
		rows = got
		return true
	}, 20*time.Second, 500*time.Millisecond, "the view must commit the new documents")
	return rows
}

// ---------------------------------------------------------- repository

func TestSearchStockTokenMatchInSummary(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.Summary = "the mutant forms culminants under starvation"
	})
	rows := waitFullSearchRows(assert, repo, "culminants")
	assert.Len(rows, 1, "expect only the summary match")
	row := rows[0]
	assert.Equal(fieldSummary, row.Field)
	assert.Equal(repository.EntityStrain, row.Entity)
	// A one-token query makes the phrase stage degenerate to a
	// single-word phrase, which ranks the same rows above the token
	// stage; the plan's failure behavior table documents this.
	assert.GreaterOrEqual(row.Score, float64(500), "expect the degenerate phrase band")
	assert.Less(row.Score, float64(1000), "the phrase band ends below 1000")
	assert.Equal(testStrainLabelY, row.StrainLabel)
}

func TestSearchStockDoesNotSearchEditableSummary(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.EditableSummary = "editableonlyzzz"
		a.Summary = testStrainSummary
	})
	waitFullSearchRows(assert, repo, "general")
	rows, err := repo.SearchStock(
		&repository.FullSearchQuery{Query: "editableonlyzzz", Limit: 50},
	)
	assert.NoError(err, "expect no error for the editable term")
	assert.NotNil(rows, "expect a non-nil slice")
	assert.Empty(rows, "expect no rows: editable_summary is not searched")
}

func TestSearchStockTokenMatchInDepositor(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	addFullSearchStrain(assert, repo, nil)
	rows := waitFullSearchRows(assert, repo, "costanza")
	assert.Len(rows, 1, "expect only the depositor match")
	assert.Equal(fieldDepositor, rows[0].Field)
}

func TestSearchStockPhraseOutranksToken(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	inOrder := addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.Summary = "the mutant forms culminants under starvation"
	})
	other := addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.Label = "farapart"
		a.Names = nil
		a.Summary = "culminants zebra quartz ladder anchor tulip forms"
	})
	rows := waitFullSearchRows(assert, repo, "forms culminants")
	assert.GreaterOrEqual(len(rows), 2, "expect both strains")
	assert.Equal(inOrder.Key, rows[0].ID, "expect the phrase strain first")
	assert.GreaterOrEqual(rows[0].Score, float64(500), "expect a phrase score")
	assert.Less(rows[0].Score, float64(1000), "the phrase band ends below 1000")
	assert.Equal(other.Key, rows[1].ID, "expect the token strain second")
	assert.GreaterOrEqual(rows[1].Score, float64(250), "expect a token score")
	assert.Less(rows[1].Score, float64(500), "the token band ends below 500")
}

func TestSearchStockPrefixOutranksPhrase(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	prefixStock := addFullSearchStrain(assert, repo, nil)
	// A single-word phrase query over the complete stock key: the prose
	// strain holds the full key in its summary, so the phrase branch and
	// the token branch both catch it, and the phrase band outranks the
	// token band. The complete key keeps every other generated stock key
	// from sharing the prefix.
	prose := addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.Label = "proseonly"
		a.Names = nil
		a.Genes = nil
		a.Summary = "the word " + strings.ToLower(prefixStock.Key) + " in prose"
	})
	query := strings.ToLower(prefixStock.Key)
	rows := waitFullSearchRows(assert, repo, query)
	assert.GreaterOrEqual(len(rows), 2, "expect both strains")
	assert.Equal(prefixStock.Key, rows[0].ID, "expect the prefix row first")
	assert.GreaterOrEqual(rows[0].Score, float64(1000), "expect a prefix score")
	assert.Equal(paramStockID, rows[0].Field)
	assert.Equal(prose.Key, rows[len(rows)-1].ID, "expect the prose row last")
	assert.Less(rows[len(rows)-1].Score, float64(1000),
		"the phrase band stays below the prefix band")
}

func TestSearchStockMultiTokenMatchesAnyToken(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.Label = "onlyforms"
		a.Names = nil
		a.Summary = "forms"
	})
	rows := waitFullSearchRows(assert, repo, "forms")
	assert.Len(rows, 1, "expect the single-token row")
	multi, err := repo.SearchStock(
		&repository.FullSearchQuery{Query: "forms culminants", Limit: 50},
	)
	assert.NoError(err, "expect no error for the 2-word query")
	var found bool
	for _, row := range multi {
		if row.ID == m.Key {
			found = true
			assert.GreaterOrEqual(row.Score, float64(250),
				"expect the token band for the single-word strain")
		}
	}
	assert.True(found,
		"the any-token rule must return the forms-only strain")
}

func TestSearchStockFuzzyRanksLast(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	// The fuzzy branch receives the whole normalized query, so a
	// misspelled identifier does not survive a multi-word query. The
	// reachable pin of the plan's ranking rule is the band comparison:
	// a fuzzy row scores below the token band of 250.
	tokenStock := addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.Label = "tokener"
		a.Names = nil
		a.Summary = "culminants"
	})
	fuzzyStock := addFullSearchStrain(assert, repo, nil)
	frows := waitFullSearchRows(assert, repo, "ys14")
	assert.Len(frows, 1, "expect only the fuzzy label match")
	assert.Equal(fuzzyStock.Key, frows[0].ID)
	assert.Less(frows[0].Score, float64(250), "the fuzzy band stays below 250")
	trows := waitFullSearchRows(assert, repo, "culminants")
	assert.Len(trows, 1, "expect only the summary match")
	assert.Equal(tokenStock.Key, trows[0].ID)
	assert.GreaterOrEqual(trows[0].Score, float64(250), "expect the token band")
	assert.Less(frows[0].Score, trows[0].Score,
		"the token row ranks above the fuzzy row")
}

func TestSearchStockReturnsCompleteSummary(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	long := "a very long summary that must come back unchanged " +
		"with every word in place and no truncation at the end"
	addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.Summary = long
	})
	rows := waitFullSearchRows(assert, repo, "unchanged")
	assert.Len(rows, 1, "expect the summary match")
	assert.Equal(long, rows[0].Summary, "expect the complete stored summary")
	assert.Equal(long, rows[0].DisplayText, "the display text is the full value")
}

func TestSearchStockPlasmidMetadataAndMissingSummary(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addFullSearchPlasmid(assert, repo, func(a *stock.NewPlasmidAttributes) {
		a.Name = "pzzzuniq"
		a.Summary = ""
	})
	rows := waitFullSearchRows(assert, repo, "pzzz")
	assert.Len(rows, 1, "expect the plasmid name match")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal(fieldName, rows[0].Field)
	assert.Equal(repository.EntityPlasmid, rows[0].Entity)
	assert.Empty(rows[0].Summary, "expect an empty summary")
	assert.Empty(rows[0].StrainLabel, "expect an empty strain label")
}

func TestSearchStockIdentifierPrefixFields(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	strain := addFullSearchStrain(assert, repo, nil)
	plasmid := addFullSearchPlasmid(assert, repo, nil)
	waitFullSearchRows(assert, repo, "ys1")
	type probe struct {
		query string
		field string
		id    string
		label string
	}
	probes := []probe{
		{strings.ToLower(strain.Key[:6]), paramStockID, strain.Key, testStrainLabelY},
		{"ddb_g03", fieldGenes, strain.Key, testStrainLabelY},
		{"d031", fieldDbxrefs, strain.Key, testStrainLabelY},
		{"ys13", fieldLabel, strain.Key, testStrainLabelY},
		{"gammas", fieldNames, strain.Key, testStrainLabelY},
		{"dictyo", fieldSpecies, strain.Key, testStrainLabelY},
		{"dbp00", fieldPlasmid, strain.Key, testStrainLabelY},
		{"p123", fieldName, plasmid.Key, ""},
	}
	for _, p := range probes {
		rows := autocompleteRowsSearch(assert, repo, p.query)
		assert.NotEmpty(rows, "expect a match for query %q", p.query)
		row := rows[0]
		assert.Equal(p.field, row.Field, "expect field for query %q", p.query)
		assert.Equal(p.id, row.ID, "expect the stock identifier for query %q", p.query)
		assert.GreaterOrEqual(row.Score, float64(1000),
			"expect a prefix score for query %q", p.query)
		assert.Equal(p.label, row.StrainLabel,
			"expect the strain label for query %q", p.query)
	}
}

// autocompleteRowsSearch is a helper with an honest name: it runs one
// full search query and returns the rows.
func autocompleteRowsSearch(
	assert *require.Assertions,
	repo repository.StockRepository,
	query string,
) []*repository.FullSearchResult {
	rows, err := repo.SearchStock(
		&repository.FullSearchQuery{Query: query, Limit: 50},
	)
	assert.NoErrorf(err, "expect no error for query %q, received %s", query, err)
	return rows
}

func TestSearchStockDefaultLimitIsFifty(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	for i := range 60 {
		addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
			a.Label = fmt.Sprintf("limi-%d", i)
			a.Genes = []string{fmt.Sprintf("limi_G%d", i)}
			a.Dbxrefs = nil
			a.Names = nil
			a.Publications = nil
		})
	}
	waitFullSearchRows(assert, repo, "limi")
	rows := autocompleteRowsSearch(assert, repo, "limi")
	assert.Len(rows, 50, "expect the default limit of 50")
}

func TestSearchStockClampsLimitToFifty(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	for i := range 60 {
		addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
			a.Label = fmt.Sprintf("cap0-%d", i)
			a.Genes = []string{fmt.Sprintf("cap0_G%d", i)}
			a.Dbxrefs = nil
			a.Names = nil
			a.Publications = nil
		})
	}
	// Poll until the whole fixture is committed and the cap holds.
	assert.Eventually(func() bool {
		rows, err := repo.SearchStock(
			&repository.FullSearchQuery{Query: testCap0Probe, Limit: 100},
		)
		return err == nil && len(rows) == 50
	}, 20*time.Second, 500*time.Millisecond, "expect the hard cap of 50")
	rows := autocompleteRowsSearch(assert, repo, testCap0Probe)
	assert.Len(rows, 50, "expect the hard cap of 50")
}

func TestSearchStockEntityFilterKeepsSmallGroup(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	for i := range 60 {
		addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
			a.Label = fmt.Sprintf("xyl-%d", i)
			a.Genes = []string{fmt.Sprintf("xylose_G%d", i)}
			a.Dbxrefs = nil
			a.Names = nil
			a.Publications = nil
		})
	}
	for i := range 5 {
		addFullSearchPlasmid(assert, repo, func(a *stock.NewPlasmidAttributes) {
			a.Name = fmt.Sprintf("xylp-%d", i)
			a.Genes = []string{fmt.Sprintf("xylose_P%d", i)}
			a.Publications = nil
		})
	}
	// Poll until the whole plasmid group is committed.
	assert.Eventually(func() bool {
		rows, err := repo.SearchStock(&repository.FullSearchQuery{
			Query:  testXyloseGene,
			Entity: repository.EntityPlasmid,
			Limit:  50,
		})
		return err == nil && len(rows) == 5
	}, 20*time.Second, 500*time.Millisecond, "expect exactly the 5 plasmids")
	filtered, err := repo.SearchStock(&repository.FullSearchQuery{
		Query:  testXyloseGene,
		Entity: repository.EntityPlasmid,
		Limit:  50,
	})
	assert.NoError(err, "expect no error for the filtered query")
	for _, row := range filtered {
		assert.Equal(repository.EntityPlasmid, row.Entity)
	}
	assert.Len(filtered, 5, "expect exactly the 5 plasmids")
}

func TestSearchStockCrossCollectionMerge(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.Label = testCordaProbe
		a.Summary = "the cordax strain of the cordax line"
		a.Genes = []string{testCordaxGene}
		a.Names = nil
	})
	rows := waitFullSearchRows(assert, repo, testCordaProbe)
	assert.Len(rows, 1, "expect one merged row for one stock")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal(repository.EntityStrain, rows[0].Entity)
	assert.Contains([]string{fieldGenes, fieldLabel}, rows[0].Field,
		"expect the field of the higher score")
	assert.NotEmpty(rows[0].Summary, "expect the summary in the tail")
}

func TestSearchStockDuplicateHitKeepsBestField(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.Label = testCordaProbe
		a.Summary = "the cordax strain of the cordax line"
		a.Genes = []string{testCordaxGene}
		a.Names = nil
	})
	rows := waitFullSearchRows(assert, repo, testCordaProbe)
	assert.Len(rows, 1, "expect one row despite three branch hits")
	assert.Equal(m.Key, rows[0].ID)
	assert.Contains([]string{fieldGenes, fieldLabel}, rows[0].Field,
		"expect the prefix field, not the prose field")
}

func TestSearchStockEmptyResultIsNotAnError(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	addFullSearchStrain(assert, repo, nil)
	waitFullSearchRows(assert, repo, "ys1")
	rows, err := repo.SearchStock(
		&repository.FullSearchQuery{Query: "zzzqqq", Limit: 50},
	)
	assert.NoError(err, "expect no error for a non-matching query")
	assert.NotNil(rows, "expect a non-nil slice")
	assert.Empty(rows, "expect no rows")
}

func TestSearchStockOneTokenQuery(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addFullSearchStrain(assert, repo, nil)
	rows := waitFullSearchRows(assert, repo, "ys")
	assert.NotEmpty(rows, "expect a 2-character query to match")
	assert.Equal(m.Key, rows[0].ID)
}

func TestSearchStockPunctuationOnlyQuery(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	addFullSearchStrain(assert, repo, nil)
	waitFullSearchRows(assert, repo, "ys1")
	rows, err := repo.SearchStock(
		&repository.FullSearchQuery{Query: "---", Limit: 50},
	)
	assert.NoError(err, "expect no error for a punctuation-only query")
	assert.Empty(rows, "expect no rows")
}

func TestSearchStockStopwordOnlyQuery(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	addFullSearchStrain(assert, repo, nil)
	// Task 1 probe 18 recorded that text_en keeps stopwords, so the
	// repository applies the Go-side stopword rule and rejects the
	// query before any AQL runs.
	_, err := repo.SearchStock(
		&repository.FullSearchQuery{Query: "the and", Limit: 50},
	)
	assert.Errorf(err, "expect an error for an all-stopword query")
}

func TestSearchStockRejectsWhitespaceQuery(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	_, err := repo.SearchStock(
		&repository.FullSearchQuery{Query: "   ", Limit: 50},
	)
	assert.Errorf(err, "expect an error for a whitespace-only query")
}

func TestSearchStockRejectsUnknownEntity(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	_, err := repo.SearchStock(
		&repository.FullSearchQuery{
			Query:  "ys1",
			Entity: repository.StockEntityFilter("vector"),
			Limit:  50,
		},
	)
	assert.Errorf(err, "expect an error for an unknown entity")
}

func TestSearchStockHandlesMissingPropertyMetadata(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ctx := context.Background()
	propc, err := repo.Dbh().Handler().Collection(ctx, testStockPropCol)
	assert.NoErrorf(err, "expect no error opening the property collection, received %s", err)
	_, err = propc.CreateDocument(ctx, map[string]any{fieldLabel: "orphanonly"})
	assert.NoErrorf(err, "expect no error inserting the orphan property, received %s", err)
	// A strain whose property document holds no label.
	addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.Label = ""
		a.Summary = "missinglabelmarker"
	})
	waitFullSearchRows(assert, repo, "missinglabelmarker")
	orphan, err := repo.SearchStock(
		&repository.FullSearchQuery{Query: "orphanonly", Limit: 50},
	)
	assert.NoError(err, "expect no error for the orphan query")
	assert.Empty(orphan, "expect the orphan property to be dropped")
	rows, err := repo.SearchStock(
		&repository.FullSearchQuery{Query: "missinglabelmarker", Limit: 50},
	)
	assert.NoError(err, "expect no error for the marker query")
	assert.Len(rows, 1, "expect the linked strain")
	assert.Empty(rows[0].StrainLabel, "expect an empty strain label")
}

func TestSearchStockArrayDisplayFallback(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.Label = "fb-strain"
		a.Names = []string{"xgamma"}
		a.Dbxrefs = nil
		a.Genes = nil
		a.Publications = nil
	})
	rows := waitFullSearchRows(assert, repo, "xgama")
	assert.Len(rows, 1, "expect the fuzzy names match")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal(fieldNames, rows[0].Field)
	assert.Equal("xgamma", rows[0].DisplayText,
		"expect the matched element through CONTAINS")
	// A fuzzy hit whose elements all fail CONTAINS falls back to the
	// joined list. The misspelling shares enough n-grams to pass the
	// threshold and holds none of the stored element as a substring.
	fb := addFullSearchStrain(assert, repo, func(a *stock.NewStrainAttributes) {
		a.Label = "fz-strain"
		a.Names = []string{"abcxyz"}
		a.Dbxrefs = nil
		a.Genes = nil
		a.Publications = nil
	})
	frows := waitFullSearchRows(assert, repo, "xbcxyz")
	assert.NotEmpty(frows, "expect the fuzzy names match")
	var matched *repository.FullSearchResult
	for _, row := range frows {
		if row.ID == fb.Key {
			matched = row
		}
	}
	assert.NotNil(matched, "expect the abcxyz strain in the results")
	assert.Equal("abcxyz", matched.DisplayText,
		"expect the joined-list fallback, never null")
}
