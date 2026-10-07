package arangodb

import (
	"context"
	"slices"

	"fmt"
	"regexp"
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
	testNormAnalyzer   = "stock_autocomplete_norm"
	testNgramAnalyzer  = "stock_autocomplete_ngram"
	testAutocompleteVw = "stock_autocomplete"
	// testStockColl and testStockPropColl mirror getCollectionParams.
	testStockColl    = "stock_test"
	testStockPropCol = "stock_properties_test"
	// testAutocompleteGene is the gene of the ranking test.
	testAutocompleteGene = "sada"
)

func assertAnalyzerList(
	ctx context.Context,
	assert *require.Assertions,
	db driver.Database,
) {
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
	assert.Contains(names, testNormAnalyzer)
	assert.Contains(names, testNgramAnalyzer)
}

func assertAutocompleteViewShape(
	ctx context.Context,
	assert *require.Assertions,
	db driver.Database,
	linkColls []string,
) {
	exists, err := db.ViewExists(ctx, testAutocompleteVw)
	assert.NoErrorf(err, "expect no error checking view, received %s", err)
	assert.True(exists, "expect the stock_autocomplete view to exist")
	v, err := db.View(ctx, testAutocompleteVw)
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
	slices.Sort(linkColls)
	assert.Equal(linkColls, keys, "expect the two link collections")
	assertAutocompleteLink(
		assert, props.Links[testStockColl],
		[]string{paramStockID, fieldGenes, fieldDbxrefs},
	)
	assertAutocompleteLink(
		assert, props.Links[testStockPropCol],
		[]string{fieldLabel, fieldNames, fieldSpecies, fieldPlasmid, fieldName},
	)
}

func assertAutocompleteLink(
	assert *require.Assertions,
	link driver.ArangoSearchElementProperties,
	fields []string,
) {
	names := make([]string, 0, len(link.Fields))
	for name := range link.Fields {
		names = append(names, name)
	}
	slices.Sort(names)
	slices.Sort(fields)
	assert.Equal(fields, names, "expect the indexed field names")
	for name, field := range link.Fields {
		got := make([]string, 0, len(field.Analyzers))
		got = append(got, field.Analyzers...)
		slices.Sort(got)
		assert.Equal(
			[]string{testNgramAnalyzer, testNormAnalyzer},
			got,
			"expect both analyzers on field %s",
			name,
		)
	}
}

func TestEnsureAutocompleteSearchCreatesAssets(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ctx := context.Background()
	db := repo.Dbh().Handler()
	assertAnalyzerList(ctx, assert, db)
	assertAutocompleteViewShape(
		ctx,
		assert,
		db,
		[]string{testStockColl, testStockPropCol},
	)
}

func TestEnsureAutocompleteSearchReconcilesStaleView(t *testing.T) {
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
		testAutocompleteVw,
		&driver.ArangoSearchViewProperties{
			Links: driver.ArangoSearchLinks{
				"stock_test": {
					Fields: driver.ArangoSearchFields{
						"stock_id": {Analyzers: []string{"identity"}},
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
	assertAutocompleteViewShape(
		ctx,
		assert,
		db,
		[]string{testStockColl, testStockPropCol},
	)
	fresh, err := db.View(ctx, testAutocompleteVw)
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

func TestNewStockRepoFailsWhenSetupFails(t *testing.T) {
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
		Database: "no_such_autocomplete_db",
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
		ta.CreateDB("no_such_autocomplete_db", nil),
		"expect no error creating the database, received %s",
		err,
	)
	// Best-effort cleanup of a leftover from a failed retry. The
	// success path drops the database through tearDown(repo).
	t.Cleanup(func() {
		if dbr, dbErr := ta.DB("no_such_autocomplete_db"); dbErr == nil {
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

// addTestStrainAttributes creates a strain with custom attributes and
// returns its stored document.
func addTestStrainAttributes(
	assert *require.Assertions,
	repo repository.StockRepository,
	attr *stock.NewStrainAttributes,
) *model.StockDoc {
	ns := &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type:       testStrainName,
			Attributes: attr,
		},
	}
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error adding strain, received %s", err)
	return m
}

// addTestStrain creates a strain whose attributes copy newTestStrain
// with the given overrides applied.
func addTestStrain(
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

// addTestPlasmid creates a plasmid whose attributes copy newTestPlasmid
// with the given overrides applied.
func addTestPlasmid(
	assert *require.Assertions,
	repo repository.StockRepository,
	createdby string,
	override func(*stock.NewPlasmidAttributes),
) *model.StockDoc {
	np := newTestPlasmid(createdby)
	if override != nil {
		override(np.Data.Attributes)
	}
	result := F.Pipe2(repo.AddPlasmid(np), ToEither, toStockDocResult)
	assert.NoErrorf(result.F2, "expect no error adding plasmid, received %s", result.F2)
	return result.F1
}

// autocompleteRows runs the query and requires no error.
func autocompleteRows(
	assert *require.Assertions,
	repo repository.StockRepository,
	params *repository.AutocompleteQuery,
) []*repository.Suggestion {
	rows, err := repo.AutocompleteStock(params)
	assert.NoErrorf(err, "expect no error for query %q, received %s", params.Query, err)
	return rows
}

// waitAutocompleteRows polls until the view commits the documents
// and the probe returns at least one row, then returns the rows.
func waitAutocompleteRows(
	assert *require.Assertions,
	repo repository.StockRepository,
	probe string,
) []*repository.Suggestion {
	var rows []*repository.Suggestion
	assert.Eventually(func() bool {
		got, err := repo.AutocompleteStock(
			&repository.AutocompleteQuery{Query: probe, Limit: 5},
		)
		if err != nil || len(got) == 0 {
			return false
		}
		rows = got
		return true
	}, 20*time.Second, 500*time.Millisecond, "the view must commit the new documents")
	return rows
}

// waitAutocompleteIndexed polls until a query that must match returns
// at least one row, ignoring the returned rows.
func waitAutocompleteIndexed(
	assert *require.Assertions,
	repo repository.StockRepository,
	params *repository.AutocompleteQuery,
) {
	assert.Eventually(func() bool {
		got, err := repo.AutocompleteStock(params)
		return err == nil && len(got) > 0
	}, 20*time.Second, 500*time.Millisecond,
		"the view must commit the new documents for query %q", params.Query)
}

// ---------------------------------------------------------------- builder

func TestBuildAutocompleteQueryBranchCount(t *testing.T) {
	assert := require.New(t)
	q := buildAutocompleteQuery()
	assert.Equal(
		16,
		strings.Count(q, "FOR d IN stock_autocomplete"),
		"expect one FOR per branch",
	)
	branchVars := regexp.MustCompile(`LET [pn]\d+ = \(`).FindAllString(q, -1)
	assert.Len(branchVars, 16, "expect 16 branch variables")
	assert.Equal(6, strings.Count(q, "OUTBOUND"),
		"expect one OUTBOUND traversal per stage of the 3 stock fields")
	assert.Equal(10, strings.Count(q, "INBOUND"),
		"expect one INBOUND traversal per stage of the 5 property fields")
	assert.Equal(8, strings.Count(q, "ANALYZER(STARTS_WITH"))
	assert.Equal(8, strings.Count(q, "NGRAM_MATCH"))
}

func TestBuildAutocompleteQueryIsValidAQL(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ctx := context.Background()
	assert.NoErrorf(
		repo.Dbh().Handler().ValidateQuery(ctx, buildAutocompleteQuery()),
		"expect the built statement to be valid AQL",
	)
}

func TestBuildAutocompleteQueryFilterOrder(t *testing.T) {
	assert := require.New(t)
	q := buildAutocompleteQuery()
	for branch := range strings.SplitSeq(q, "\nLET ") {
		if !strings.HasPrefix(branch, "LET ") {
			branch = "LET " + branch
		}
		if !regexp.MustCompile(`^LET [pn]\d+ = \(`).MatchString(branch) {
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

// ------------------------------------------------------------ repository

func TestAutocompleteStockLabelPrefix(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestStrain(assert, repo, nil)
	rows := waitAutocompleteRows(assert, repo, "ys1")
	assert.Len(rows, 1, "expect only the label match")
	sug := rows[0]
	assert.Equal(m.Key, sug.ID, "expect the stock identifier, not the property key")
	assert.Equal("label", sug.Field)
	assert.Equal(repository.EntityStrain, sug.Entity)
	assert.GreaterOrEqual(sug.Score, float64(1000), "expect a prefix score")
	assert.Equal("yS13", sug.DisplayText)
}

func TestAutocompleteStockPlasmidNamePrefix(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestPlasmid(assert, repo, testEmailCostanza, nil)
	rows := waitAutocompleteRows(assert, repo, "p123")
	assert.Len(rows, 1, "expect only the name match")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal("name", rows[0].Field)
	assert.Equal(repository.EntityPlasmid, rows[0].Entity)
}

func TestAutocompleteStockGenePrefix(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestStrain(assert, repo, nil)
	rows := waitAutocompleteRows(assert, repo, "ddb_g03")
	assert.Len(rows, 1, "expect only the genes match")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal("genes", rows[0].Field)
	assert.Equal("DDB_G0348394", rows[0].DisplayText)
}

func TestAutocompleteStockDbxrefPrefix(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestStrain(assert, repo, nil)
	rows := waitAutocompleteRows(assert, repo, "d031")
	assert.Len(rows, 1, "expect only the dbxrefs match")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal("dbxrefs", rows[0].Field)
	assert.Equal("d0319", rows[0].DisplayText)
}

func TestAutocompleteStockStockIDPrefix(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestStrain(assert, repo, nil)
	rows := waitAutocompleteRows(assert, repo, m.Key[:6])
	assert.Len(rows, 1, "expect the stock_id match")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal("stock_id", rows[0].Field)
}

func TestAutocompleteStockSpeciesPrefix(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestStrain(assert, repo, nil)
	rows := waitAutocompleteRows(assert, repo, "dictyo")
	assert.GreaterOrEqual(len(rows), 1, "expect the species match")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal("species", rows[0].Field)
}

func TestAutocompleteStockStrainPlasmidPrefix(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestStrain(assert, repo, nil)
	rows := waitAutocompleteRows(assert, repo, "dbp00")
	assert.Len(rows, 1, "expect the plasmid attribute match")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal("plasmid", rows[0].Field)
	assert.Equal(repository.EntityStrain, rows[0].Entity)
}

func TestAutocompleteStockNamesPrefix(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestStrain(assert, repo, nil)
	rows := waitAutocompleteRows(assert, repo, "gammas")
	assert.Len(rows, 1, "expect the names match")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal("names", rows[0].Field)
	assert.Equal(testNameGammaS13, rows[0].DisplayText)
}

func TestAutocompleteStockTypoFuzzy(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestStrain(assert, repo, nil)
	rows := waitAutocompleteRows(assert, repo, "ys14")
	assert.Len(rows, 1, "expect the fuzzy label match")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal("label", rows[0].Field)
	assert.Less(rows[0].Score, float64(1000), "expect a fuzzy score")
}

func TestAutocompleteStockEntityFilterKeepsSmallGroup(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	for i := range 60 {
		addTestStrainAttributes(assert, repo, &stock.NewStrainAttributes{
			CreatedBy:           testEmailCostanza,
			UpdatedBy:           testEmailCostanza,
			Depositor:           testEmailCostanza,
			Summary:             testStrainSummary,
			EditableSummary:     testStrainSummary,
			Label:               fmt.Sprintf("xyl-%d", i),
			Species:             testSpecies,
			Genes:               []string{fmt.Sprintf("xylose_G%d", i)},
			DictyStrainProperty: testStrainSummary,
		})
	}
	for i := range 5 {
		np := &stock.NewPlasmid{
			Data: &stock.NewPlasmid_Data{
				Type: fieldPlasmid,
				Attributes: &stock.NewPlasmidAttributes{
					CreatedBy:            testEmailCostanza,
					UpdatedBy:            testEmailCostanza,
					Depositor:            testEmailCostanza,
					Summary:              testPlasmidSummary,
					EditableSummary:      testPlasmidSummary,
					Name:                 fmt.Sprintf("xylp-%d", i),
					Genes:                []string{fmt.Sprintf("xylose_P%d", i)},
					DictyPlasmidProperty: OntologyTermCloningVector,
				},
			},
		}
		result := F.Pipe2(repo.AddPlasmid(np), ToEither, toStockDocResult)
		assert.NoErrorf(result.F2, "expect no error adding plasmid %d, received %s", i, result.F2)
	}
	// Poll until the whole plasmid group is committed: the view
	// commits in the background, so a single wait for one row does not
	// prove that all five plasmids are visible.
	assert.Eventually(func() bool {
		rows, err := repo.AutocompleteStock(&repository.AutocompleteQuery{
			Query:  "xylose",
			Entity: repository.EntityPlasmid,
			Limit:  10,
		})
		return err == nil && len(rows) == 5
	}, 20*time.Second, 500*time.Millisecond, "expect exactly the 5 plasmids")
	rows := autocompleteRows(assert, repo, &repository.AutocompleteQuery{
		Query:  "xylose",
		Entity: repository.EntityPlasmid,
		Limit:  10,
	})
	for _, row := range rows {
		assert.Equal(repository.EntityPlasmid, row.Entity)
	}
}

func TestAutocompleteStockCrossCollectionMerge(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestStrainAttributes(assert, repo, &stock.NewStrainAttributes{
		CreatedBy:           testEmailCostanza,
		UpdatedBy:           testEmailCostanza,
		Depositor:           testEmailCostanza,
		Summary:             testStrainSummary,
		EditableSummary:     testStrainSummary,
		Label:               "corda",
		Species:             testSpecies,
		Genes:               []string{"cordaxin"},
		DictyStrainProperty: testStrainSummary,
	})
	rows := waitAutocompleteRows(assert, repo, "corda")
	assert.Len(rows, 1, "expect one merged row for one stock")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal(repository.EntityStrain, rows[0].Entity)
	assert.Contains([]string{"genes", "label"}, rows[0].Field,
		"expect the field of the higher score")
}

func TestAutocompleteStockDefaultLimitIsFive(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	for i := range 8 {
		addTestStrain(assert, repo, func(a *stock.NewStrainAttributes) {
			a.Label = fmt.Sprintf("limi-%d", i)
			a.Genes = []string{fmt.Sprintf("limi_G%d", i)}
			a.Dbxrefs = nil
			a.Names = nil
			a.Publications = nil
		})
	}
	waitAutocompleteIndexed(assert, repo,
		&repository.AutocompleteQuery{Query: "limi", Limit: 50})
	rows := autocompleteRows(assert, repo,
		&repository.AutocompleteQuery{Query: "limi", Limit: 0})
	assert.Len(rows, 5, "expect the default limit")
}

func TestAutocompleteStockLimitCapIsFifty(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	for i := range 60 {
		addTestStrain(assert, repo, func(a *stock.NewStrainAttributes) {
			a.Label = fmt.Sprintf("cap0-%d", i)
			a.Genes = []string{fmt.Sprintf("cap0_G%d", i)}
			a.Dbxrefs = nil
			a.Names = nil
			a.Publications = nil
		})
	}
	// Poll until the whole fixture is committed and the cap holds.
	assert.Eventually(func() bool {
		rows, err := repo.AutocompleteStock(
			&repository.AutocompleteQuery{Query: "cap0", Limit: 500},
		)
		return err == nil && len(rows) == 50
	}, 20*time.Second, 500*time.Millisecond, "expect the hard cap of 50")
	rows := autocompleteRows(assert, repo,
		&repository.AutocompleteQuery{Query: "cap0", Limit: 500})
	assert.Len(rows, 50, "expect the hard cap")
}

func TestAutocompleteStockRankingUnderSmallLimit(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestStrainAttributes(assert, repo, &stock.NewStrainAttributes{
		CreatedBy:           testEmailCostanza,
		UpdatedBy:           testEmailCostanza,
		Depositor:           testEmailCostanza,
		Summary:             testStrainSummary,
		EditableSummary:     testStrainSummary,
		Label:               "rank-strong",
		Species:             testSpecies,
		Genes:               []string{testAutocompleteGene},
		DictyStrainProperty: testStrainSummary,
	})
	for i := range 7 {
		addTestStrainAttributes(assert, repo, &stock.NewStrainAttributes{
			CreatedBy:           testEmailCostanza,
			UpdatedBy:           testEmailCostanza,
			Depositor:           testEmailCostanza,
			Summary:             testStrainSummary,
			EditableSummary:     testStrainSummary,
			Label:               fmt.Sprintf("rank-%d", i),
			Species:             testSpecies,
			Genes:               []string{testAutocompleteGene + "XYZ" + fmt.Sprintf("%d", i)},
			DictyStrainProperty: testStrainSummary,
		})
	}
	waitAutocompleteIndexed(assert, repo,
		&repository.AutocompleteQuery{Query: testAutocompleteGene, Limit: 50})
	rows := autocompleteRows(assert, repo,
		&repository.AutocompleteQuery{Query: testAutocompleteGene, Limit: 3})
	assert.NotEmpty(rows, "expect rows")
	assert.Equal(m.Key, rows[0].ID, "expect the prefix match first")
	assert.GreaterOrEqual(rows[0].Score, float64(1000), "expect a prefix score")
}

func TestAutocompleteStockEmptyResultIsNotAnError(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	addTestStrain(assert, repo, nil)
	waitAutocompleteRows(assert, repo, "ys1")
	rows, err := repo.AutocompleteStock(
		&repository.AutocompleteQuery{Query: "zzzqqq", Limit: 5},
	)
	assert.NoError(err, "expect no error for a non-matching query")
	assert.NotNil(rows, "expect a non-nil slice")
	assert.Empty(rows, "expect no rows")
}

func TestAutocompleteStockArrayDisplayFallback(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestStrainAttributes(assert, repo, &stock.NewStrainAttributes{
		CreatedBy:           testEmailCostanza,
		UpdatedBy:           testEmailCostanza,
		Depositor:           testEmailCostanza,
		Summary:             testStrainSummary,
		EditableSummary:     testStrainSummary,
		Label:               "fb-strain",
		Species:             testSpecies,
		Names:               []string{"xgamma"},
		DictyStrainProperty: testStrainSummary,
	})
	rows := waitAutocompleteRows(assert, repo, "xgama")
	assert.Len(rows, 1, "expect the fuzzy names match")
	assert.Equal(m.Key, rows[0].ID)
	assert.Equal("names", rows[0].Field)
	assert.NotEmpty(rows[0].DisplayText, "display text must never be null")
	assert.Equal("xgamma", rows[0].DisplayText,
		"expect the joined-list fallback")
}

func TestAutocompleteStockRejectsWhitespaceQuery(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	_, err := repo.AutocompleteStock(
		&repository.AutocompleteQuery{Query: "   ", Limit: 5},
	)
	assert.Errorf(err, "expect an error for a whitespace-only query")
}

func TestAutocompleteStockRejectsUnknownEntity(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	_, err := repo.AutocompleteStock(
		&repository.AutocompleteQuery{
			Query:  "ys1",
			Entity: repository.StockEntityFilter("vector"),
			Limit:  5,
		},
	)
	assert.Errorf(err, "expect an error for an unknown entity")
}

func TestAutocompleteStockSkipsStockWithoutEdge(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ctx := context.Background()
	propc, err := repo.Dbh().Handler().Collection(ctx, testStockPropCol)
	assert.NoErrorf(err, "expect no error opening the property collection, received %s", err)
	_, err = propc.CreateDocument(ctx, map[string]any{"label": "orphanedge"})
	assert.NoErrorf(err, "expect no error inserting the orphan property, received %s", err)
	// Wait until the view indexes the orphan document. The poll
	// returns the label and not a scalar: a RETURN 1 row never reaches
	// the go-driver cursor (Scan stays false even for a match).
	assert.Eventually(func() bool {
		search, serr := repo.Dbh().SearchRows(
			`FOR d IN stock_autocomplete SEARCH ANALYZER(STARTS_WITH(d.label, @q), "`+
				testNormAnalyzer+`") LIMIT 1 RETURN d.label`,
			map[string]any{"q": "orphan"},
		)
		if serr != nil {
			return false
		}
		found := false
		for search.Scan() {
			var v string
			if rerr := search.Read(&v); rerr == nil {
				found = true
			}
		}
		_ = search.Close()
		return found
	}, 20*time.Second, 500*time.Millisecond, "the view must index the orphan document")
	rows, err := repo.AutocompleteStock(
		&repository.AutocompleteQuery{Query: "orphan", Limit: 5},
	)
	assert.NoError(err, "expect no error for the orphan query")
	assert.Empty(rows, "expect the orphan property to be dropped")
}

func TestNormalizeAutocompleteQuery(t *testing.T) {
	assert := require.New(t)
	assert.Equal("ys1", normalizeAutocompleteQuery("  YS1  "))
	assert.Equal("ax2", normalizeAutocompleteQuery("Áx2"))
	assert.Equal("ys1", normalizeAutocompleteQuery("Ÿs1"))
	assert.Equal("", normalizeAutocompleteQuery("   "))
}

func TestAutocompleteStockAccentedQuery(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	m := addTestStrain(assert, repo, nil)
	rows := waitAutocompleteRows(assert, repo, "ys1")
	assert.Equal(m.Key, rows[0].ID)
	acc := autocompleteRows(assert, repo,
		&repository.AutocompleteQuery{Query: "Ÿs1", Limit: 5})
	assert.Len(acc, 1, "expect the accented query to match")
	assert.Equal(m.Key, acc[0].ID)
}
