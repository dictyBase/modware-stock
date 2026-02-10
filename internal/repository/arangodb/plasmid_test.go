package arangodb

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	E "github.com/IBM/fp-go/either"
	F "github.com/IBM/fp-go/function"
	IOE "github.com/IBM/fp-go/ioeither"
	T "github.com/IBM/fp-go/tuple"
	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/stretchr/testify/require"
)

// Type aliases for test results using fp-go Tuple
type (
	// StockDocResult represents a stock document with potential error
	StockDocResult = T.Tuple2[*model.StockDoc, error]

	// StockDocListResult represents a list of stock documents with potential error
	StockDocListResult = T.Tuple2[[]*model.StockDoc, error]

	// StockDocEither represents a computation that may succeed with stock doc or fail
	StockDocEither = E.Either[error, *model.StockDoc]

	// StockDocListEither represents a computation that may succeed with stock doc list or fail
	StockDocListEither = E.Either[error, []*model.StockDoc]
)

// ToEither executes IOEither to get Either result
func ToEither[A any](ioe IOE.IOEither[error, A]) E.Either[error, A] {
	return ioe()
}

// toStockDocResult converts Either result to tuple using E.Fold pattern
func toStockDocResult(either StockDocEither) StockDocResult {
	return F.Pipe1(
		either,
		E.Fold(
			func(err error) StockDocResult {
				return T.MakeTuple2[*model.StockDoc](nil, err)
			},
			func(doc *model.StockDoc) StockDocResult {
				return T.MakeTuple2[*model.StockDoc, error](doc, nil)
			},
		),
	)
}

// toStockDocListResult converts Either result to tuple using E.Fold pattern
func toStockDocListResult(either StockDocListEither) StockDocListResult {
	return F.Pipe1(
		either,
		E.Fold(
			func(err error) StockDocListResult {
				return T.MakeTuple2[[]*model.StockDoc](nil, err)
			},
			func(docs []*model.StockDoc) StockDocListResult {
				return T.MakeTuple2[[]*model.StockDoc, error](docs, nil)
			},
		),
	)
}

const (
	georgeFilter = `FILTER s.depositor == 'george@costanza.com'`
	pfilterTwo   = `FILTER s.depositor == 'george@costanza.com' AND s.depositor == 'rg@gmail.com'`
	pfilterThree = `LET x = (
				FILTER '1348970' IN s.publications
				RETURN 1
			)`
	pfilterFour  = `FILTER s.created_at <= DATE_ISO8601('2019')`
	pfilterFive  = `FILTER stock_prop.sequence =~ 'ttttt'`
	pfilterSix   = `FILTER s.summary =~ 'test'`
	pfilterSeven = `FILTER s.depositor == 'george@costanza.com' OR stock_prop.name == 'gammaS13'`

	// Ontology term constants for testing
	OntologyTermCloningVector = "cloning vector"
	OntologyTermREMIVector    = "REMI vector"
	OntologyTermGFPMarker     = "GFP marker"
	OntologyTermGatewayVector = "Gateway vector"
	OntologyTermAct15Promoter = "act15 promoter"
	OntologyTermTetOFFVector  = "tetOFF vector"
	OntologyTermDoxONVector   = "doxON vector"
)

// assertPlasmidBasicFields asserts basic plasmid document fields
func assertPlasmidBasicFields(
	assert *require.Assertions,
	doc *model.StockDoc,
	attrs *stock.NewPlasmidAttributes,
) {
	assert.Equal(doc.CreatedBy, attrs.CreatedBy, "should match created_by id")
	assert.Equal(doc.UpdatedBy, attrs.UpdatedBy, "should match updated_by id")
	assert.Equal(doc.Summary, attrs.Summary, "should match summary")
	assert.Equal(doc.EditableSummary, attrs.EditableSummary, "should match editable_summary")
	assert.Equal(doc.Depositor, attrs.Depositor, "should match depositor")
	assert.ElementsMatch(doc.Publications, attrs.Publications, "should match publications")
}

// assertPlasmidPropertiesFields asserts plasmid-specific properties
func assertPlasmidPropertiesFields(
	assert *require.Assertions,
	doc *model.StockDoc,
	attrs *stock.NewPlasmidAttributes,
) {
	assert.Equal(doc.PlasmidProperties.ImageMap, attrs.ImageMap, "should match image_map")
	assert.Equal(doc.PlasmidProperties.Sequence, attrs.Sequence, "should match sequence")
	assert.Equal(doc.PlasmidProperties.Name, attrs.Name, "should match name")
}

func TestLoadStockWithPlasmids(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)

	ns := createTestExistingPlasmid()

	result1 := F.Pipe2(
		repo.LoadPlasmid("DBP0000098", ns),
		ToEither,
		toStockDocResult,
	)
	um, err := result1.F1, result1.F2
	assert.NoErrorf(err, "expect no error, received %s", err)

	assertLoadedPlasmidAttributes(assert, um, ns)
	assertLoadedPlasmidProperties(assert, um, ns)
	assertRetrievedPlasmidOntology(assert, repo, um.StockID)
}

func createTestExistingPlasmid() *stock.ExistingPlasmid {
	tm, _ := time.Parse("2006-01-02 15:04:05", "2010-03-30 14:40:58")
	return &stock.ExistingPlasmid{
		Data: &stock.ExistingPlasmid_Data{
			Type: "plasmid",
			Id:   "DBP0000098",
			Attributes: &stock.ExistingPlasmidAttributes{
				CreatedAt:            aphgrpc.TimestampProto(tm),
				UpdatedAt:            aphgrpc.TimestampProto(tm),
				CreatedBy:            "george@costanza.com",
				UpdatedBy:            "george@costanza.com",
				Depositor:            "george@costanza.com",
				Summary:              "this is a test plasmid",
				EditableSummary:      "this is a test plasmid",
				Publications:         []string{"1348970"},
				ImageMap:             "http://dictybase.org/data/plasmid/images/87.jpg",
				Sequence:             "tttttyyyyjkausadaaaavvvvvv",
				Name:                 "p9999",
				DictyPlasmidProperty: OntologyTermCloningVector,
			},
		},
	}
}

func assertLoadedPlasmidAttributes(
	assert *require.Assertions,
	doc *model.StockDoc,
	ns *stock.ExistingPlasmid,
) {
	assert.Equal("DBP0000098", doc.StockID, "should match given plasmid stock id")
	assert.Equal(doc.Key, doc.StockID, "should have identical key and stock ID")
	assert.Equal(doc.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(doc.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(doc.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(doc.EditableSummary, ns.Data.Attributes.EditableSummary, "should match editable_summary")
	assert.Equal(doc.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
	assert.Empty(doc.Genes, "should have empty genes field")
	assert.Empty(doc.Dbxrefs, "should have empty dbxrefs field")
	assert.ElementsMatch(doc.Publications, ns.Data.Attributes.Publications, "should match publications")
}

func assertLoadedPlasmidProperties(
	assert *require.Assertions,
	doc *model.StockDoc,
	ns *stock.ExistingPlasmid,
) {
	assert.Equal(doc.PlasmidProperties.ImageMap, ns.Data.Attributes.ImageMap, "should match image_map")
	assert.Equal(doc.PlasmidProperties.Sequence, ns.Data.Attributes.Sequence, "should match sequence")
}

func assertRetrievedPlasmidOntology(
	assert *require.Assertions,
	repo repository.StockRepository,
	stockID string,
) {
	result := F.Pipe2(repo.GetPlasmid(stockID), ToEither, toStockDocResult)
	doc, err := result.F1, result.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(
		OntologyTermCloningVector,
		doc.PlasmidProperties.DictyPlasmidProperty,
		"GetPlasmid should retrieve the ontology term label that was stored during LoadPlasmid for plasmid %s",
		stockID,
	)
}

// createTestPlasmids creates n test plasmids for testing
func createTestPlasmids(
	assert *require.Assertions,
	repo repository.StockRepository,
	count int,
) {
	for i := 1; i <= count; i++ {
		np := newTestPlasmid(
			fmt.Sprintf("%s@cye.com", arangomanager.RandomString(15, 25)),
		)
		result := F.Pipe2(repo.AddPlasmid(np), ToEither, toStockDocResult)
		_, err := result.F1, result.F2
		assert.NoErrorf(err, "expect no error, received %s", err)
		time.Sleep(100 * time.Millisecond)
	}
}

// testPlasmidListFilter tests a plasmid list with specific filter
func testPlasmidListFilter(
	assert *require.Assertions,
	repo repository.StockRepository,
	params *stock.StockParameters,
	expectedLen int,
) {
	result := F.Pipe2(repo.ListPlasmids(params), ToEither, toStockDocListResult)
	docs, err := result.F1, result.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(docs, expectedLen, "should match expected length")
}

// runPlasmidFilterTests runs multiple plasmid filter tests
func runPlasmidFilterTests(
	assert *require.Assertions,
	repo repository.StockRepository,
	cursor int64,
) {
	testPlasmidListFilter(assert, repo, &stock.StockParameters{
		Limit: 100, Filter: pfilterTwo}, 0)
	testPlasmidListFilter(assert, repo, &stock.StockParameters{
		Cursor: cursor, Limit: 10, Filter: pfilterThree}, 5)
	testPlasmidListFilter(assert, repo, &stock.StockParameters{
		Cursor: cursor, Limit: 10, Filter: pfilterFour}, 0)
	testPlasmidListFilter(assert, repo, &stock.StockParameters{
		Limit: 10, Filter: pfilterFive}, 10)
	testPlasmidListFilter(assert, repo, &stock.StockParameters{
		Limit: 10, Filter: pfilterSix}, 10)
	testPlasmidListFilter(assert, repo, &stock.StockParameters{
		Cursor: cursor, Limit: 10, Filter: pfilterSeven}, 5)
}

func TestListPlasmidsWithFilter(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	createTestPlasmids(assert, repo, 10)

	result4 := F.Pipe2(repo.ListPlasmids(
		&stock.StockParameters{Limit: 10, Filter: georgeFilter},
	), ToEither, toStockDocListResult)

	sf, err := result4.F1, result4.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(sf, 10, "should list ten plasmids")
	for _, um := range sf {
		assert.Equal(um.Summary, "this is a test plasmid", "should match summary")
		assert.Equal(um.PlasmidProperties.Name, "p123456", "should match name")
	}

	runPlasmidFilterTests(assert, repo, toTimestamp(sf[5].CreatedAt))
}

func TestListPlasmids(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	for i := 1; i <= 10; i++ {
		np := newTestPlasmid(
			fmt.Sprintf("%s@cye.com", arangomanager.RandomString(15, 20)),
		)
		result8 := F.Pipe2(repo.AddPlasmid(np), ToEither, toStockDocResult)

		_, err := result8.F1, result8.F2
		assert.NoErrorf(err, "expect no error adding plasmid, received %s", err)
		time.Sleep(100 * time.Millisecond)
	}
	result9 := F.Pipe2(
		repo.ListPlasmids(&stock.StockParameters{Limit: 4}),
		ToEither,
		toStockDocListResult,
	)

	ls, err := result9.F1, result9.F2
	assert.NoErrorf(
		err,
		"expect no error getting first five plasmids, received %s",
		err,
	)
	assert.Len(ls, 5, "should match the provided limit number + 1")
	for _, stock := range ls {
		assert.Equal(
			stock.Depositor,
			"george@costanza.com",
			"should match the depositor",
		)
		assert.Equal(stock.Key, stock.StockID, "stock key and ID should match")
		assert.Regexp(
			regexp.MustCompile(`^DBP0\d{6,}$`),
			stock.StockID,
			"should have a plasmid stock id",
		)
	}
	assert.NotEqual(
		ls[0].CreatedBy,
		ls[1].CreatedBy,
		"should have different created_by",
	)
	testMoreListPlasmids(repo, assert, t, ls)
}

func testMoreListPlasmids(
	repo repository.StockRepository,
	assert *require.Assertions,
	t *testing.T,
	ls []*model.StockDoc,
) {
	ls2 := testPaginatedPlasmidList(repo, assert, ls, 5, "5-9")
	ls3 := testPaginatedPlasmidList(repo, assert, ls2, 2, "9-10")

	testModelListSort(ls, t)
	testModelListSort(ls2, t)
	testModelListSort(ls3, t)

	testFilteredPlasmidList(repo, assert)
}

func testPaginatedPlasmidList(
	repo repository.StockRepository,
	assert *require.Assertions,
	previousList []*model.StockDoc,
	expectedLen int,
	rangeDesc string,
) []*model.StockDoc {
	cursor := toTimestamp(previousList[len(previousList)-1].CreatedAt)
	result := F.Pipe2(
		repo.ListPlasmids(&stock.StockParameters{Cursor: cursor, Limit: 4}),
		ToEither,
		toStockDocListResult,
	)

	docs, err := result.F1, result.F2
	assert.NoErrorf(err, "expect no error getting plasmids %s, received %s", rangeDesc, err)
	assert.Len(docs, expectedLen, "should match expected length for range %s", rangeDesc)
	assert.Exactly(
		docs[0],
		previousList[len(previousList)-1],
		"last item from previous results and first item from current results should be the same",
	)

	return docs
}

func testFilteredPlasmidList(
	repo repository.StockRepository,
	assert *require.Assertions,
) {
	result := F.Pipe2(
		repo.ListPlasmids(&stock.StockParameters{Limit: 100, Filter: georgeFilter}),
		ToEither,
		toStockDocListResult,
	)

	docs, err := result.F1, result.F2
	assert.NoErrorf(err, "expect no error getting list of plasmids, received %s", err)
	assert.Len(docs, 10, "should list ten plasmids")

	cursorResult := F.Pipe2(
		repo.ListPlasmids(&stock.StockParameters{
			Cursor: toTimestamp(docs[4].CreatedAt),
			Limit:  10,
		}),
		ToEither,
		toStockDocListResult,
	)
	cursorDocs, err := cursorResult.F1, cursorResult.F2
	assert.NoErrorf(err, "expect no error getting list of plasmids with cursor, received %s", err)
	assert.Len(cursorDocs, 6, "should list six plasmids")
}

// assertPlasmidStockID asserts plasmid stock ID format
func assertPlasmidStockID(assert *require.Assertions, stockID string) {
	assert.Regexp(
		regexp.MustCompile(`^DBP0\d{6,}$`),
		stockID,
		"should have a plasmid stock id",
	)
}

// assertPlasmidTimestamps asserts plasmid timestamp equality
func assertPlasmidTimestamps(
	assert *require.Assertions,
	expected *model.StockDoc,
	actual *model.StockDoc,
) {
	assert.True(
		expected.CreatedAt.Equal(actual.CreatedAt),
		"should match created time of stock",
	)
	assert.True(
		expected.UpdatedAt.Equal(actual.UpdatedAt),
		"should match updated time of stock",
	)
}

func TestGetPlasmid(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ns := newTestPlasmid("george@costanza.com")
	result13 := F.Pipe2(repo.AddPlasmid(ns), ToEither, toStockDocResult)

	um, err := result13.F1, result13.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	result14 := F.Pipe2(repo.GetPlasmid(um.StockID), ToEither, toStockDocResult)

	g, err := result14.F1, result14.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	assertPlasmidStockID(assert, g.StockID)
	assertPlasmidBasicFields(assert, g, ns.Data.Attributes)
	assertPlasmidPropertiesFields(assert, g, ns.Data.Attributes)
	assertPlasmidTimestamps(assert, um, g)

	result15 := F.Pipe2(repo.GetPlasmid("DBP01"), ToEither, toStockDocResult)
	_, err = result15.F1, result15.F2
	assert.Error(err, "expect error for non-existent plasmid")
	assert.Contains(
		err.Error(),
		"not found",
		"error should indicate plasmid was not found",
	)
}

// assertPlasmidUpdate asserts plasmid update results
func assertPlasmidUpdate(
	assert *require.Assertions,
	updated *model.StockDoc,
	original *stock.NewPlasmid,
	updateAttrs *stock.PlasmidUpdateAttributes,
) {
	assert.Equal(updated.UpdatedBy, updateAttrs.UpdatedBy, "should match updatedby")
	assert.Equal(
		updated.Depositor,
		original.Data.Attributes.Depositor,
		"depositor name should not be updated",
	)
	assert.Equal(updated.Summary, updateAttrs.Summary, "should have updated summary")
	assert.Equal(
		updated.EditableSummary,
		updateAttrs.EditableSummary,
		"should have updated editable summary",
	)
	assert.ElementsMatch(
		updated.Publications,
		updateAttrs.Publications,
		"should match updated list of publications",
	)
	assert.ElementsMatch(
		updated.Dbxrefs,
		original.Data.Attributes.Dbxrefs,
		"dbxrefs should remain unchanged",
	)
}

func TestEditPlasmid(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ns := newUpdatableTestPlasmid("art@vandelay.org")
	result16 := F.Pipe2(repo.AddPlasmid(ns), ToEither, toStockDocResult)

	m, err := result16.F1, result16.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	us := &stock.PlasmidUpdate{
		Data: &stock.PlasmidUpdate_Data{
			Type: ns.Data.Type,
			Id:   m.StockID,
			Attributes: &stock.PlasmidUpdateAttributes{
				UpdatedBy:       "varnes@seinfeld.org",
				Summary:         "updated plasmid",
				EditableSummary: "updated plasmid",
				Publications:    []string{"8394839", "583989343", "853983948"},
				Genes:           []string{"DDB_G0270724", "DDB_G027489343"},
				ImageMap:        "http://dictybase.org/data/plasmid/images/87.jpg",
			},
		},
	}
	result17 := F.Pipe2(repo.EditPlasmid(us), ToEither, toStockDocResult)

	um, err := result17.F1, result17.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(um.StockID, m.StockID, "should match the stock id")
	assertPlasmidUpdate(assert, um, ns, us.Data.Attributes)
	assert.Equal(
		um.PlasmidProperties.ImageMap,
		us.Data.Attributes.ImageMap,
		"should match image map",
	)
	assert.Equal(
		um.PlasmidProperties.Name,
		us.Data.Attributes.Name,
		"should match name",
	)
}

func PlasmidUpdateInstance(
	um *model.StockDoc,
	ns *stock.NewPlasmid,
) *stock.PlasmidUpdate {
	return &stock.PlasmidUpdate{
		Data: &stock.PlasmidUpdate_Data{
			Type: ns.Data.Type,
			Id:   um.StockID,
			Attributes: &stock.PlasmidUpdateAttributes{
				UpdatedBy: "puddy@seinfeld.org",
				Genes: []string{
					"DDB_G0270851",
					"DDB_G02748",
					"DDB_G7392222",
				},
				Sequence: "atgctagagaagacttt",
			},
		},
	}
}

// assertPlasmidGeneUpdate asserts gene-related fields after update
func assertPlasmidGeneUpdate(
	assert *require.Assertions,
	updated *model.StockDoc,
	previous *model.StockDoc,
	updateAttrs *stock.PlasmidUpdateAttributes,
) {
	assert.Equal(updated.StockID, previous.StockID, "should match the previous stock id")
	assert.Equal(
		updated.UpdatedBy,
		updateAttrs.UpdatedBy,
		"should have updated the updatedby field",
	)
	assert.ElementsMatch(updated.Genes, updateAttrs.Genes, "should have the genes list")
	assert.ElementsMatch(
		updated.Publications,
		previous.Publications,
		"publications list should remain the same",
	)
	assert.ElementsMatch(
		updated.Dbxrefs,
		previous.Dbxrefs,
		"dbxrefs list should remain the same",
	)
}

func TestEditPlasmidGene(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ns := newUpdatableTestPlasmid("art@vandelay.org")
	result18 := F.Pipe2(repo.AddPlasmid(ns), ToEither, toStockDocResult)

	um, err := result18.F1, result18.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	us2 := PlasmidUpdateInstance(um, ns)
	result19 := F.Pipe2(repo.EditPlasmid(us2), ToEither, toStockDocResult)

	um2, err := result19.F1, result19.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	assertPlasmidGeneUpdate(assert, um2, um, us2.Data.Attributes)
	assert.Equal(
		um2.PlasmidProperties.ImageMap,
		um.PlasmidProperties.ImageMap,
		"image map should remain the same",
	)
	assert.Equal(
		um2.PlasmidProperties.Sequence,
		us2.Data.Attributes.Sequence,
		"sequence plasmid property should have been updated",
	)

	us3 := &stock.PlasmidUpdate{
		Data: &stock.PlasmidUpdate_Data{
			Type: ns.Data.Type,
			Id:   um.StockID,
			Attributes: &stock.PlasmidUpdateAttributes{
				UpdatedBy:       "seven@costanza.org",
				Summary:         "this is an updated summary",
				EditableSummary: "this is an updated summary",
			},
		},
	}
	result20 := F.Pipe2(repo.EditPlasmid(us3), ToEither, toStockDocResult)

	um3, err := result20.F1, result20.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(um3.StockID, um.StockID, "should match the original stock id")
	assert.Equal(
		um3.UpdatedBy,
		us3.Data.Attributes.UpdatedBy,
		"should have updated the updatedby field",
	)
	assert.Equal(
		um3.Summary,
		us3.Data.Attributes.Summary,
		"should have updated the summary field",
	)
	assert.Equal(
		um3.EditableSummary,
		us3.Data.Attributes.EditableSummary,
		"should have updated the editable summary field",
	)
	assert.ElementsMatch(
		um3.Dbxrefs,
		um2.Dbxrefs,
		"dbxrefs list should remain the same",
	)
}

// assertAddedPlasmidFields asserts fields after adding a plasmid
func assertAddedPlasmidFields(
	assert *require.Assertions,
	doc *model.StockDoc,
	attrs *stock.NewPlasmidAttributes,
) {
	assertPlasmidStockID(assert, doc.StockID)
	assert.Equal(doc.Key, doc.StockID, "should have identical key and stock ID")
	assertPlasmidBasicFields(assert, doc, attrs)
	assert.Empty(doc.Genes, "should have empty genes field")
	assert.Empty(doc.Dbxrefs, "should have empty dbxrefs field")
	assertPlasmidPropertiesFields(assert, doc, attrs)
}

func TestAddPlasmid(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ns := newTestPlasmid("george@costanza.com")
	result21 := F.Pipe2(repo.AddPlasmid(ns), ToEither, toStockDocResult)

	um, err := result21.F1, result21.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	assertAddedPlasmidFields(assert, um, ns.Data.Attributes)

	result22 := F.Pipe2(repo.GetPlasmid(um.StockID), ToEither, toStockDocResult)
	gm, err := result22.F1, result22.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(
		ns.Data.Attributes.DictyPlasmidProperty,
		gm.PlasmidProperties.DictyPlasmidProperty,
		"should include ontology label",
	)
}

func TestAddPlasmidWithOntologyTerms(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)

	testCases := []struct {
		name string
		term string
	}{
		{"REMI vector", OntologyTermREMIVector},
		{"GFP marker", OntologyTermGFPMarker},
		{"Gateway vector", OntologyTermGatewayVector},
		{"act15 promoter", OntologyTermAct15Promoter},
		{"tetOFF vector", OntologyTermTetOFFVector},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			np := newTestPlasmid("pfey@dictybase.org")
			np.Data.Attributes.DictyPlasmidProperty = tc.term
			result23 := F.Pipe2(repo.AddPlasmid(np), ToEither, toStockDocResult)

			um, err := result23.F1, result23.F2
			assert.NoErrorf(
				err,
				"expect no error adding plasmid with ontology term %s, received %s",
				tc.term,
				err,
			)
			result24 := F.Pipe2(
				repo.GetPlasmid(um.StockID),
				ToEither,
				toStockDocResult,
			)

			gm, err := result24.F1, result24.F2
			assert.NoErrorf(
				err,
				"expect no error retrieving plasmid %s, received %s",
				um.StockID,
				err,
			)
			assert.Equal(
				tc.term,
				gm.PlasmidProperties.DictyPlasmidProperty,
				"AddPlasmid should store and GetPlasmid should retrieve ontology term label '%s' for plasmid %s",
				tc.term,
				um.StockID,
			)
		})
	}
}

func TestEditPlasmidOntologyUpdate(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ns := newUpdatableTestPlasmid("art@vandelay.org")
	result25 := F.Pipe2(repo.AddPlasmid(ns), ToEither, toStockDocResult)

	m, err := result25.F1, result25.F2
	assert.NoErrorf(err, "expect no error, received %s", err)
	us := &stock.PlasmidUpdate{
		Data: &stock.PlasmidUpdate_Data{
			Type: ns.Data.Type,
			Id:   m.StockID,
			Attributes: &stock.PlasmidUpdateAttributes{
				UpdatedBy:            "peterman@jpeterman.com",
				DictyPlasmidProperty: OntologyTermDoxONVector,
			},
		},
	}
	result26 := F.Pipe2(repo.EditPlasmid(us), ToEither, toStockDocResult)

	_, err = result26.F1, result26.F2
	assert.NoErrorf(
		err,
		"expect no error updating plasmid ontology, received %s",
		err,
	)
	result27 := F.Pipe2(repo.GetPlasmid(m.StockID), ToEither, toStockDocResult)

	gm, err := result27.F1, result27.F2
	assert.NoErrorf(
		err,
		"expect no error retrieving updated plasmid %s, received %s",
		m.StockID,
		err,
	)
	assert.Equal(
		OntologyTermDoxONVector,
		gm.PlasmidProperties.DictyPlasmidProperty,
		"EditPlasmid should update ontology term from '%s' to '%s' for plasmid %s",
		ns.Data.Attributes.DictyPlasmidProperty,
		OntologyTermDoxONVector,
		m.StockID,
	)
}

func TestAddPlasmidInvalidOntologyTerm(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	np := newTestPlasmid("pfey@dictybase.org")
	invalidTerm := "not a real ontology term"
	np.Data.Attributes.DictyPlasmidProperty = invalidTerm
	result28 := F.Pipe2(repo.AddPlasmid(np), ToEither, toStockDocResult)

	_, err := result28.F1, result28.F2
	assert.Error(
		err,
		"AddPlasmid should return an error when attempting to add plasmid with invalid ontology term '%s'",
		invalidTerm,
	)
}
