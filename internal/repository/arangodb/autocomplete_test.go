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
	testNormAnalyzer   = "stock_autocomplete_norm"
	testNgramAnalyzer  = "stock_autocomplete_ngram"
	testAutocompleteVw = "stock_autocomplete"
	// testStockColl and testStockPropColl mirror getCollectionParams.
	testStockColl    = "stock_test"
	testStockPropCol = "stock_properties_test"
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
