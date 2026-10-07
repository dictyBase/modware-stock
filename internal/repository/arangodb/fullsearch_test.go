package arangodb

import (
	"context"
	"slices"
	"testing"

	driver "github.com/arangodb/go-driver"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/stretchr/testify/require"
)

const (
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
