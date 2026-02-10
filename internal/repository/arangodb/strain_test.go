package arangodb

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/collection"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/stretchr/testify/require"
)

const (
	filterGene     = `FILTER 'DDB_G0287317' IN s.genes`
	filterGeneMore = `FILTER 'DDB_G098058933' IN s.genes`

	filterOne = `FILTER s.depositor == 'george@costanza.com'`
	filterTwo = `FILTER s.depositor == 'george@costanza.com' 
		      AND s.depositor == 'rg@gmail.com'
	 	     `
	filterThree = `LET x = (
				FILTER 'gammaS13' IN s.names 
				RETURN 1
			)`
	filterFour          = `FILTER s.created_at <= DATE_ISO8601('2019')`
	filterFive          = `FILTER stock_prop.label =~ 'yS'`
	filterSix           = `FILTER s.summary =~ 'mutant'`
	filterRegularStrain = `FILTER cv.metadata.namespace == 'dicty_strain_property'
				AND cvterm.label == 'general strain'
	`
	filterGwdiStrain = `FILTER cv.metadata.namespace == 'dicty_strain_property'
				AND cvterm.label == 'REMI-seq'
	`
	filterBacterialStrain = `FILTER cv.metadata.namespace == 'dicty_strain_property'
				AND cvterm.label == 'bacterial strain'
	`
	filterAllStrain = `FILTER cv.metadata.namespace == 'dicty_strain_property'
				AND (
					cvterm.label == 'bacterial strain'
					OR cvterm.label == 'REMI-seq'
					OR cvterm.label == 'general strain'
				)
	`
	filterBad = `FILTER borat.acting == 'funny`
)

func createTestStrainsWithParent(
	count int,
	stype StrainType,
	repo repository.StockRepository,
	pid string,
) ([]string, error) {
	ids := make([]string, 0)
	start := 1
	for start <= count {
		ns := newTestStrain(
			fmt.Sprintf(
				"%s@kramericaindustries.com",
				arangomanager.RandomString(15, 20),
			),
			stype,
		)
		ns.Data.Attributes.Parent = pid
		nps, err := repo.AddStrain(ns)
		if err != nil {
			return ids, err
		}
		ids = append(ids, nps.StockID)
		start++
	}
	return ids, nil
}

func createTestStrainsWithIDs(
	count int,
	stype StrainType,
	repo repository.StockRepository,
) ([]string, error) {
	ids := make([]string, 0)
	start := 0
	for start < count {
		ns := newTestStrain(
			fmt.Sprintf(
				"%s@kramericaindustries.com",
				arangomanager.RandomString(15, 20),
			),
			stype,
		)
		nps, err := repo.AddStrain(ns)
		if err != nil {
			return ids, err
		}
		time.Sleep(100 * time.Millisecond)
		ids = append(ids, nps.StockID)
		start++
	}
	return ids, nil
}

func createTestStrains(
	count int,
	stype StrainType,
	repo repository.StockRepository,
) error {
	start := 1
	for start <= count {
		ns := newTestStrain(
			fmt.Sprintf(
				"%s@kramericaindustries.com",
				arangomanager.RandomString(15, 20),
			),
			stype,
		)
		_, err := repo.AddStrain(ns)
		if err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
		start++
	}
	return nil
}

// assertStrainBasicFields asserts basic strain document fields
func assertStrainBasicFields(
	assert *require.Assertions,
	doc *model.StockDoc,
	attrs *stock.ExistingStrainAttributes,
) {
	assert.Equal(doc.Key, doc.StockID, "should have identical key and stock ID")
	assert.Equal(doc.CreatedBy, attrs.CreatedBy, "should match created_by id")
	assert.Equal(doc.UpdatedBy, attrs.UpdatedBy, "should match updated_by id")
	assert.Equal(doc.Summary, attrs.Summary, "should match summary")
	assert.Equal(doc.EditableSummary, attrs.EditableSummary, "should match editable_summary")
	assert.Equal(doc.Depositor, attrs.Depositor, "should match depositor")
	assert.ElementsMatch(doc.Dbxrefs, attrs.Dbxrefs, "should match dbxrefs")
}

// assertStrainSpecificProperties asserts strain-specific property fields
func assertStrainSpecificProperties(
	assert *require.Assertions,
	doc *model.StockDoc,
	attrs *stock.ExistingStrainAttributes,
) {
	assert.Equal(doc.StrainProperties.Label, attrs.Label, "should match descriptor")
	assert.Equal(doc.StrainProperties.Species, attrs.Species, "should match species")
	assert.ElementsMatch(doc.StrainProperties.Names, attrs.Names, "should match names")
}

func TestLoadStrainWithID(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	tm, _ := time.Parse("2006-01-02 15:04:05", "2010-03-30 14:40:58")
	nsp := &stock.ExistingStrain{
		Data: &stock.ExistingStrain_Data{
			Type: "strain",
			Attributes: &stock.ExistingStrainAttributes{
				CreatedAt:           aphgrpc.TimestampProto(tm),
				UpdatedAt:           aphgrpc.TimestampProto(tm),
				CreatedBy:           "wizard_of_loneliness@testemail.org",
				UpdatedBy:           "wizard_of_loneliness@testemail.org",
				Depositor:           "wizard_of_loneliness@testemail.org",
				Summary:             "Remi-mutant strain",
				EditableSummary:     "Remi-mutant strain.",
				Dbxrefs:             []string{"5466867", "4536935", "d2578"},
				Label:               "egeB/DDB_G0270724_ps-REMI",
				Species:             "Dictyostelium discoideum",
				Names:               []string{"gammaS13", "BCN149086"},
				DictyStrainProperty: "general strain",
			},
		},
	}
	m, err := repo.LoadStrain("DBS0252873", nsp)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.True(m.CreatedAt.Equal(tm), "should match created_at")
	assert.Equal("DBS0252873", m.StockID, "should match given stock id")
	assertStrainBasicFields(assert, m, nsp.Data.Attributes)
	assert.Empty(m.Genes, "should not be tied to any genes")
	assert.Empty(m.Publications, "should not be tied to any publications")
	assertStrainSpecificProperties(assert, m, nsp.Data.Attributes)
	assert.Empty(m.StrainProperties.Plasmid, "should not have any plasmid")
	assert.Empty(m.StrainProperties.Parent, "should not have any parent")
}

func setUpTestData(
	repo repository.StockRepository,
	assert *require.Assertions,
) *stock.ExistingStrain {
	tm, _ := time.Parse("2006-01-02 15:04:05", "2010-03-30 14:40:58")
	est := &stock.ExistingStrain{
		Data: &stock.ExistingStrain_Data{
			Type: "strain",
			Attributes: &stock.ExistingStrainAttributes{
				CreatedAt:           aphgrpc.TimestampProto(tm),
				UpdatedAt:           aphgrpc.TimestampProto(tm),
				CreatedBy:           "wizard_of_loneliness@testemail.org",
				UpdatedBy:           "wizard_of_loneliness@testemail.org",
				Depositor:           "wizard_of_loneliness@testemail.org",
				Summary:             "Remi-mutant strain",
				EditableSummary:     "Remi-mutant strain.",
				Dbxrefs:             []string{"5466867", "4536935", "d2578"},
				Label:               "egeB/DDB_G0270724_ps-REMI",
				Species:             "Dictyostelium discoideum",
				Names:               []string{"gammaS13", "BCN149086"},
				DictyStrainProperty: "general strain",
			},
		},
	}
	pst, err := repo.LoadStrain("DBS0252873", est)
	assert.NoErrorf(err, "expect no error, received %s", err)
	ns := &stock.ExistingStrain{
		Data: &stock.ExistingStrain_Data{
			Type: "strain",
			Attributes: &stock.ExistingStrainAttributes{
				CreatedAt:           aphgrpc.TimestampProto(tm),
				UpdatedAt:           aphgrpc.TimestampProto(tm),
				CreatedBy:           "wizard_of_loneliness@testemail.org",
				UpdatedBy:           "wizard_of_loneliness@testemail.org",
				Depositor:           "wizard_of_loneliness@testemail.org",
				Summary:             "Remi-mutant strain",
				EditableSummary:     "Remi-mutant strain.",
				Dbxrefs:             []string{"5466867", "4536935", "d2578"},
				Label:               "egeB/DDB_G0270724_ps-REMI",
				Species:             "Dictyostelium discoideum",
				Names:               []string{"gammaS13", "BCN149086"},
				DictyStrainProperty: "general strain",
				Parent:              pst.StockID,
			},
		},
	}
	return ns
}

func TestLoadStockWithParent(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)

	ns := setUpTestData(repo, assert)
	m2, err := repo.LoadStrain("DBS0235412", ns)
	assert.NoErrorf(err, "expect no error, received %s", err)

	assert.Equal("DBS0235412", m2.StockID, "should match given stock id")
	assert.Equal(m2.Key, m2.StockID, "should have identical key and stock ID")
	assert.Equal(
		m2.CreatedBy,
		ns.Data.Attributes.CreatedBy,
		"should match created_by id",
	)
	assert.Equal(
		m2.UpdatedBy,
		ns.Data.Attributes.UpdatedBy,
		"should match updated_by id",
	)
	assert.Equal(
		m2.Depositor,
		ns.Data.Attributes.Depositor,
		"should match depositor",
	)
	assert.ElementsMatch(
		m2.Dbxrefs,
		ns.Data.Attributes.Dbxrefs,
		"should match dbxrefs",
	)
	assert.ElementsMatch(
		m2.Genes,
		ns.Data.Attributes.Genes,
		"should match gene ids",
	)
	assert.Equal(
		m2.StrainProperties.Label,
		ns.Data.Attributes.Label,
		"should match descriptor",
	)
	assert.Equal(
		m2.StrainProperties.Species,
		ns.Data.Attributes.Species,
		"should match species",
	)
	assert.ElementsMatch(
		m2.StrainProperties.Names,
		ns.Data.Attributes.Names,
		"should match names",
	)
	assert.Equal(
		m2.StrainProperties.Plasmid,
		ns.Data.Attributes.Plasmid,
		"should match plasmid entry",
	)
	assert.Equal(
		m2.StrainProperties.Parent,
		ns.Data.Attributes.Parent,
		"should match parent entry",
	)
}

func TestListStrainsWithType(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	gwids, err := createTestStrainsWithIDs(15, Gwdi, repo)
	assert.NoError(err, "expect no error from creating gwdi strains")
	gwStrains, err := repo.ListStrains(
		&stock.StockParameters{Limit: 100, Filter: filterGwdiStrain},
	)
	assert.NoError(err, "expect no error in getting list of strains")
	assert.Lenf(
		gwStrains,
		15,
		"expect to have 15 gwdi strains, received %d",
		len(gwStrains),
	)
	rids, err := createTestStrainsWithIDs(10, General, repo)
	assert.NoError(err, "expect no error from creating regular strains")
	regStrains, err := repo.ListStrains(
		&stock.StockParameters{Limit: 100, Filter: filterRegularStrain},
	)
	assert.NoError(err, "expect no error in getting list of strains")
	assert.Lenf(
		regStrains,
		10,
		"expect to have 15 gwdi strains, received %d",
		len(regStrains),
	)
	bids, err := createTestStrainsWithIDs(10, Bacterial, repo)
	assert.NoError(err, "expect no error from creating bacterial strains")
	bacStrains, err := repo.ListStrains(
		&stock.StockParameters{Limit: 100, Filter: filterBacterialStrain},
	)
	assert.NoError(err, "expect no error in getting list of strains")
	assert.Lenf(
		bacStrains,
		10,
		"expect to have 15 gwdi strains, received %d",
		len(bacStrains),
	)
	allStrains, err := repo.ListStrains(
		&stock.StockParameters{Limit: 99, Filter: filterAllStrain},
	)
	assert.NoError(err, "expect no error in getting list of strains")
	assert.Lenf(
		allStrains,
		35,
		"expect to have 35 strains received %d",
		len(allStrains),
	)
	assert.Equal(
		len(allStrains),
		len(gwids)+len(rids)+len(bids),
		"should match all types of strains count",
	)
	assert.ElementsMatch(
		collection.Map(allStrains, stockToID),
		append(bids, append(rids, gwids...)...),
		"should match all stock ids",
	)
}

// testStrainListFilterScenario tests a specific filter scenario
func testStrainListFilterScenario(
	assert *require.Assertions,
	repo repository.StockRepository,
	params *stock.StockParameters,
	expectedLen int,
	description string,
	shouldError bool,
) {
	strains, err := repo.ListStrains(params)
	if shouldError {
		assert.Error(err, description)
		return
	}
	assert.NoError(err, description)
	assert.Len(strains, expectedLen, description)
}

// runStrainListFilters runs multiple strain list filter tests
func runStrainListFilters(
	assert *require.Assertions,
	repo repository.StockRepository,
	cursor int64,
) {
	testStrainListFilterScenario(
		assert,
		repo,
		&stock.StockParameters{Limit: 100, Filter: filterTwo},
		0,
		"expect no error getting list of stocks with two depositors with AND logic",
		false,
	)
	testStrainListFilterScenario(
		assert,
		repo,
		&stock.StockParameters{Cursor: cursor, Limit: 10, Filter: filterThree},
		5,
		"expect no error getting list of stocks with cursor and filter",
		false,
	)
	testStrainListFilterScenario(
		assert,
		repo,
		&stock.StockParameters{Cursor: cursor, Limit: 10, Filter: filterFour},
		0,
		"expect no error getting list of stocks with cursor and date filter",
		false,
	)
	testStrainListFilterScenario(
		assert,
		repo,
		&stock.StockParameters{Limit: 10, Filter: filterFive},
		10,
		"expect no error getting list of strains with label substring",
		false,
	)
	testStrainListFilterScenario(
		assert,
		repo,
		&stock.StockParameters{Limit: 10, Filter: filterSix},
		10,
		"expect no error in matching summary substring",
		false,
	)
	testStrainListFilterScenario(
		assert,
		repo,
		&stock.StockParameters{Limit: 2, Filter: filterBad},
		0,
		"expect have error with the query",
		true,
	)
}

func TestListStrainsWithFilter(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	err := createTestStrains(10, General, repo)
	assert.NoError(err, "expect no error from creating strains")

	sf, err := repo.ListStrains(&stock.StockParameters{Limit: 10, Filter: filterOne})
	assert.NoError(err, "expect no error in getting list of strains")
	assert.Len(sf, 10, "should list ten strains")
	for _, m := range sf {
		assert.Equal(m.Summary, "Radiation-sensitive mutant.", "should match summary")
		assert.Equal(m.StrainProperties.Label, "yS13", "should match label")
	}

	runStrainListFilters(assert, repo, toTimestamp(sf[5].CreatedAt))
}

func TestListStrains(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	// add 10 new test strains
	err := createTestStrains(10, General, repo)
	assert.NoError(err, "expect no error from creating strains")
	// get first five results
	ls, err := repo.ListStrains(&stock.StockParameters{Limit: 4})
	assert.NoError(err, "expect no error in getting first five stocks")
	assert.Len(ls, 5, "should match the provided limit number + 1")
	for _, stock := range ls {
		assert.Equal(
			stock.Depositor,
			"george@costanza.com",
			"should match the depositor",
		)
		assert.Equal(stock.Key, stock.StockID, "stock key and ID should match")
		assert.Regexp(
			regexp.MustCompile(`^DBS0\d{6,}$`),
			stock.StockID,
			"should have a strain stock id",
		)
	}
	assert.NotEqual(
		ls[0].CreatedBy,
		ls[1].CreatedBy,
		"should have different created_by",
	)
	// convert fifth result to numeric timestamp in milliseconds
	// so we can use this as cursor
	ti := toTimestamp(ls[4].CreatedAt)

	// get next five results (5-9)
	ls2, err := repo.ListStrains(&stock.StockParameters{Cursor: ti, Limit: 4})
	assert.NoError(err, "expect no error in getting stocks 5-9")
	assert.Len(ls2, 5, "should match the provided limit number + 1")
	assert.Exactly(
		ls2[0],
		ls[len(ls)-1],
		"last item from first five results and first item from next five results should be the same",
	)
	assert.NotEqual(
		ls2[0].CreatedBy,
		ls2[1].CreatedBy,
		"should have different created_by fields",
	)

	// convert ninth result to numeric timestamp
	ti2 := toTimestamp(ls2[len(ls2)-1].CreatedAt)
	// get last results (9-10)
	ls3, err := repo.ListStrains(&stock.StockParameters{Cursor: ti2, Limit: 4})
	assert.NoErrorf(
		err,
		"expect no error in getting stocks 9-10, received %s",
		err,
	)
	assert.Len(ls3, 2, "should retrieve the last two results")
	assert.Exactly(
		ls3[0],
		ls2[len(ls2)-1],
		"last item from previous five results and first item from next five results should be the same",
	)

	// sort all of the results
	testModelListSort(ls, t)
	testModelListSort(ls2, t)
	testModelListSort(ls3, t)
}

// assertStrainListItems asserts common properties across strain list items
func assertStrainListItems(
	assert *require.Assertions,
	strains []*model.StockDoc,
	expectedParent string,
	expectedProperty string,
) {
	for _, stock := range strains {
		assert.Equal(stock.Depositor, "george@costanza.com", "should match the depositor")
		assert.Equal(stock.Key, stock.StockID, "stock key and ID should match")
		assert.Regexp(
			regexp.MustCompile(`^DBS0\d{6,}$`),
			stock.StockID,
			"should have a strain stock id",
		)
		assert.Equal(
			stock.StrainProperties.Parent,
			expectedParent,
			"parent field should match expected",
		)
		assert.Equal(
			stock.StrainProperties.DictyStrainProperty,
			expectedProperty,
			"should match ontology strain property",
		)
	}
}

func TestListStrainsByIDs(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)

	ids, err := createTestStrainsWithIDs(30, General, repo)
	assert.NoError(err, "expect no error from creating strains")

	ls, err := repo.ListStrainsByIDs(&stock.StockIdList{Id: ids})
	assert.NoError(err, "expect no error in getting strains")
	assert.Len(ls, 30, "should match the provided limit number")
	assertStrainListItems(assert, ls, "", "general strain")

	pm, err := repo.AddStrain(newTestParentStrain("j@peterman.org"))
	assert.NoErrorf(
		err,
		"expect no error in creating parent strain, received %s",
		err,
	)
	pids, err := createTestStrainsWithParent(30, General, repo, pm.StockID)
	assert.NoError(err, "expect no error from creating strains")
	pls, err := repo.ListStrainsByIDs(&stock.StockIdList{Id: pids})
	assert.NoError(err, "expect no error in getting 30 stocks with parents")
	assert.Len(pls, 30, "should match the provided limit number")
	assertStrainListItems(assert, pls, pm.StockID, "general strain")

	els, err := repo.ListStrainsByIDs(
		&stock.StockIdList{Id: []string{"DBN589343", "DBN48473232"}},
	)
	assert.NoErrorf(
		err,
		"expect no error in getting first five stocks, received %s",
		err,
	)
	assert.Len(els, 0, "should get empty list of strain")
}

func TestGetStrain(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	ns := newTestStrain("george@costanza.com", General)
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	g, err := repo.GetStrain(m.StockID)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assertRegexp(assert, g.StockID)
	assertStrainProperties(assert, g, ns)
	assert.Equal(
		g.StrainProperties.Parent,
		"",
		"should not have parent",
	)
	assert.Len(g.Dbxrefs, 6, "should match length of six dbxrefs")
	assert.Equal(
		g.CreatedAt,
		m.CreatedAt,
		"should match created time of stock",
	)
	assert.Equal(
		g.UpdatedAt,
		m.UpdatedAt,
		"should match updated time of stock",
	)
}

func assertRegexp(assert *require.Assertions, stockID string) {
	assert.Regexp(
		regexp.MustCompile(`^DBS0\d{6,}$`),
		stockID,
		"should have a strain stock id",
	)
}

// assertNewStrainBasicFields asserts basic fields for NewStrain
func assertNewStrainBasicFields(
	assert *require.Assertions,
	doc *model.StockDoc,
	attrs *stock.NewStrainAttributes,
) {
	assert.Equal(doc.CreatedBy, attrs.CreatedBy, "should match created_by id")
	assert.Equal(doc.UpdatedBy, attrs.UpdatedBy, "should match updated_by id")
	assert.Equal(doc.Summary, attrs.Summary, "should match summary")
	assert.Equal(doc.EditableSummary, attrs.EditableSummary, "should match editable_summary")
	assert.Equal(doc.Depositor, attrs.Depositor, "should match depositor")
	assert.ElementsMatch(doc.Dbxrefs, attrs.Dbxrefs, "should match dbxrefs")
	assert.ElementsMatch(doc.Genes, attrs.Genes, "should match genes")
}

// assertNewStrainPropertiesFields asserts strain-specific properties for NewStrain
func assertNewStrainPropertiesFields(
	assert *require.Assertions,
	doc *model.StockDoc,
	attrs *stock.NewStrainAttributes,
) {
	assert.ElementsMatch(doc.StrainProperties.Names, attrs.Names, "should match names")
	assert.Equal(doc.StrainProperties.Label, attrs.Label, "should match descriptor")
	assert.Equal(doc.StrainProperties.Species, attrs.Species, "should match species")
	assert.Equal(doc.StrainProperties.Plasmid, attrs.Plasmid, "should match plasmid")
	assert.Equal(
		doc.StrainProperties.DictyStrainProperty,
		"general strain",
		"should match ontology strain property",
	)
}

func assertStrainProperties(
	assert *require.Assertions,
	g *model.StockDoc,
	ns *stock.NewStrain,
) {
	assertNewStrainBasicFields(assert, g, ns.Data.Attributes)
	assertNewStrainPropertiesFields(assert, g, ns.Data.Attributes)
}

func TestAddParentStrain(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	nsp := newTestParentStrain("todd@gagg.com")
	m, err := repo.AddStrain(nsp)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Regexp(
		regexp.MustCompile(`^DBS0\d{6,}$`),
		m.StockID,
		"should have a stock id",
	)
	assert.Equal(m.Key, m.StockID, "should have identical key and stock ID")
	assert.Equal(
		m.CreatedBy,
		nsp.Data.Attributes.CreatedBy,
		"should match created_by id",
	)
	assert.Equal(
		m.UpdatedBy,
		nsp.Data.Attributes.UpdatedBy,
		"should match updated_by id",
	)
	assert.Equal(m.Summary, nsp.Data.Attributes.Summary, "should match summary")
	assert.Equal(
		m.EditableSummary,
		nsp.Data.Attributes.EditableSummary,
		"should match editable_summary",
	)
	assert.Equal(
		m.Depositor,
		nsp.Data.Attributes.Depositor,
		"should match depositor",
	)
	assert.ElementsMatch(
		m.Dbxrefs,
		nsp.Data.Attributes.Dbxrefs,
		"should match dbxrefs",
	)
	assert.Empty(m.Genes, "should not be tied to any genes")
	assert.Empty(m.Publications, "should not be tied to any publications")
	assert.Equal(
		m.StrainProperties.Label,
		nsp.Data.Attributes.Label,
		"should match descriptor",
	)
	assert.Equal(
		m.StrainProperties.Species,
		nsp.Data.Attributes.Species,
		"should match species",
	)
	assert.ElementsMatch(
		m.StrainProperties.Names,
		nsp.Data.Attributes.Names,
		"should match names",
	)
	assert.Empty(m.StrainProperties.Plasmid, "should not have any plasmid")
	assert.Empty(m.StrainProperties.Parent, "should not have any parent")
	testAddChildStrain(repo, assert, m.StockID)
}

func testAddChildStrain(
	repo repository.StockRepository,
	assert *require.Assertions,
	id string,
) {
	ns := newTestStrain("pennypacker@penny.com", General)
	ns.Data.Attributes.Parent = id
	m2, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(
		m2.StrainProperties.Parent,
		ns.Data.Attributes.Parent,
		"should match parent entry",
	)
	assert.Regexp(
		regexp.MustCompile(`^DBS0\d{6,}$`),
		m2.StockID,
		"should have a stock id",
	)
	assert.Equal(m2.Key, m2.StockID, "should have identical key and stock ID")
	assert.Equal(
		m2.CreatedBy,
		ns.Data.Attributes.CreatedBy,
		"should match created_by id",
	)
	assert.Equal(
		m2.UpdatedBy,
		ns.Data.Attributes.UpdatedBy,
		"should match updated_by id",
	)
	assert.ElementsMatch(
		m2.Dbxrefs,
		ns.Data.Attributes.Dbxrefs,
		"should match dbxrefs",
	)
	assert.ElementsMatch(
		m2.Genes,
		ns.Data.Attributes.Genes,
		"should match gene ids",
	)
	assert.Equal(
		m2.StrainProperties.Label,
		ns.Data.Attributes.Label,
		"should match descriptor",
	)
	assert.Equal(
		m2.StrainProperties.Species,
		ns.Data.Attributes.Species,
		"should match species",
	)
	assert.Equal(
		m2.StrainProperties.Plasmid,
		ns.Data.Attributes.Plasmid,
		"should match plasmid entry",
	)
}

func strainUpdateInstance(
	ns *stock.NewStrain,
	m *model.StockDoc,
) *stock.StrainUpdate {
	return &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: ns.Data.Type,
			Id:   m.StockID,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy:       "kirby@snes.org",
				Summary:         "updated strain",
				EditableSummary: "updated strain",
				Genes:           []string{"DDB_G120987", "DDB_G45098234"},
				Dbxrefs: []string{
					"FGBD9493483",
					"4536935",
					"d2578",
					"d0319",
				},
				Label:   "Ax3-pspD/lacZ",
				Plasmid: "DBP0398713",
				Names:   []string{"SP87", "AX3-PL3/gal", "AX3PL31"},
			},
		},
	}
}

// assertStrainUpdate asserts strain update results
func assertStrainUpdate(
	assert *require.Assertions,
	updated *model.StockDoc,
	original *model.StockDoc,
	updateAttrs *stock.StrainUpdateAttributes,
) {
	assert.Equal(updated.StockID, original.StockID, "should match the stock id")
	assert.Equal(updated.UpdatedBy, updateAttrs.UpdatedBy, "should match updatedby")
	assert.Equal(
		updated.Depositor,
		original.Depositor,
		"depositor name should not be updated",
	)
	assert.Equal(updated.Summary, updateAttrs.Summary, "should have updated summary")
	assert.Equal(
		updated.EditableSummary,
		updateAttrs.EditableSummary,
		"should have updated editable summary",
	)
	assert.ElementsMatch(
		updated.Genes,
		updateAttrs.Genes,
		"should match updated list of genes",
	)
	assert.ElementsMatch(
		updated.Dbxrefs,
		updateAttrs.Dbxrefs,
		"should match updated list of dbxrefs",
	)
	assert.ElementsMatch(
		updated.Publications,
		original.Publications,
		"publications list should remain unchanged",
	)
}

func TestEditStrain(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	ns := newUpdatableTestStrain("todd@gagg.com", General)
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	us := strainUpdateInstance(ns, m)
	um, err := repo.EditStrain(us)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assertStrainUpdate(assert, um, m, us.Data.Attributes)
	assert.Equal(
		um.StrainProperties.Species,
		m.StrainProperties.Species,
		"species name should remain unchanged",
	)
	assert.Equal(
		um.StrainProperties.Label,
		us.Data.Attributes.Label,
		"should have updated strain descriptor",
	)
	assert.Equal(
		um.StrainProperties.Plasmid,
		us.Data.Attributes.Plasmid,
		"should have updated plasmid name",
	)
	assert.ElementsMatch(
		um.StrainProperties.Names,
		us.Data.Attributes.Names,
		"should have updated list of strain names",
	)
}

func TestListStrainsWithGeneFilter(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	// Create test strain with specific gene
	ns := newTestStrain("george@costanza.com", General)
	ns.Data.Attributes.Genes = []string{"DDB_G0287317", "DDB_G0287318"}
	_, err := repo.AddStrain(ns)
	assert.NoError(err, "expect no error from creating strain with genes")

	// Create additional strains without the specific gene
	err = createTestStrains(5, General, repo)
	assert.NoError(err, "expect no error from creating additional strains")

	// Test filtering by gene
	ls, err := repo.ListStrains(&stock.StockParameters{
		Limit:  10,
		Filter: filterGene,
	})
	assert.NoError(err, "expect no error in getting strains filtered by gene")
	assert.Len(ls, 1, "should find exactly one strain with the specific gene")
	assert.Contains(
		ls[0].Genes,
		"DDB_G0287317",
		"returned strain should contain the filtered gene",
	)
	ls2, err := repo.ListStrains(&stock.StockParameters{
		Limit:  10,
		Filter: filterGeneMore,
	})
	assert.NoError(err, "expect no error in getting strains filtered by gene")
	assert.Len(
		ls2,
		5,
		"should find exactly five strains with the specific gene",
	)
	assert.Contains(
		ls2[1].Genes,
		"DDB_G098058933",
		"returned strain should contain the filtered gene",
	)
}

// assertStrainParentUpdate asserts parent-related update results
func assertStrainParentUpdate(
	assert *require.Assertions,
	updated *model.StockDoc,
	original *model.StockDoc,
	updateAttrs *stock.StrainUpdateAttributes,
) {
	assert.Equal(updated.StockID, original.StockID, "should match their id")
	assert.Equal(updated.Depositor, updateAttrs.Depositor, "depositor name should be updated")
	assert.Equal(updated.CreatedBy, original.CreatedBy, "created by should not be updated")
	assert.Equal(updated.Summary, original.Summary, "summary should not be updated")
	assert.ElementsMatch(
		updated.Publications,
		original.Publications,
		"publications list should remains unchanged",
	)
	assert.ElementsMatch(updated.Genes, original.Genes, "genes list should not be updated")
	assert.ElementsMatch(updated.Dbxrefs, original.Dbxrefs, "dbxrefs list should not be updated")
}

// assertStrainPropertiesUnchanged asserts strain properties remain unchanged
func assertStrainPropertiesUnchanged(
	assert *require.Assertions,
	updated *model.StockDoc,
	original *model.StockDoc,
) {
	assert.Equal(
		updated.StrainProperties.Label,
		original.StrainProperties.Label,
		"strain descriptor should not be updated",
	)
	assert.ElementsMatch(
		updated.StrainProperties.Names,
		original.StrainProperties.Names,
		"strain names should not be updated",
	)
	assert.Equal(
		updated.StrainProperties.Plasmid,
		original.StrainProperties.Plasmid,
		"plasmid should not be updated",
	)
}

func TestEditStrainWithParent(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	pm, err := repo.AddStrain(newTestParentStrain("tim@watley.org"))
	assert.NoErrorf(err, "expect no error, received %s", err)
	ns := newUpdatableTestStrain("todd@gagg.com", General)
	ust, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	us2 := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: ns.Data.Type,
			Id:   ust.StockID,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy: "mario@snes.org",
				Depositor: "mario@snes.org",
				Parent:    pm.StockID,
				Species:   "updated species",
			},
		},
	}
	um2, err := repo.EditStrain(us2)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assertStrainParentUpdate(assert, um2, ust, us2.Data.Attributes)
	assertStrainPropertiesUnchanged(assert, um2, ust)
	assert.Equal(
		um2.StrainProperties.Parent,
		us2.Data.Attributes.Parent,
		"should have updated parent",
	)
	assert.Equal(
		um2.StrainProperties.Species,
		us2.Data.Attributes.Species,
		"species should be updated",
	)
	testMoreEditWithParent(repo, assert, ust, um2, ns)
}

func testMoreEditWithParent(
	repo repository.StockRepository,
	assert *require.Assertions,
	ust *model.StockDoc,
	um2 *model.StockDoc,
	ns *stock.NewStrain,
) {
	// add another new strain, let's make this one a parent
	// so we can test updating parent if one already exists
	pu, err := repo.AddStrain(
		newUpdatableTestStrain("castle@vania.org", General),
	)
	assert.NoErrorf(err, "expect no error, received %s", err)
	us3 := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: ns.Data.Type,
			Id:   ust.StockID,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy: "mario@snes.org",
				Parent:    pu.StockID,
			},
		},
	}
	um3, err := repo.EditStrain(us3)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(
		um3.StrainProperties.Parent,
		us3.Data.Attributes.Parent,
		"should have updated parent",
	)
	assert.Equal(ust.StockID, um3.StockID, "should have same stock ID")
	assert.Equal(
		um2.StrainProperties.Plasmid,
		um3.StrainProperties.Plasmid,
		"plasmid should not have been updated",
	)
}

func stockToID(model *model.StockDoc) string {
	return model.StockID
}

func TestEditStrainOntologyUpdate(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)

	// Create and add initial strain with "general strain" property
	ns := newUpdatableTestStrain("art@vandelay.org", General)
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(
		"general strain",
		m.StrainProperties.DictyStrainProperty,
		"initial strain should have general strain property",
	)

	// Update with different ontology term
	us := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: ns.Data.Type,
			Id:   m.StockID,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy:           "peterman@jpeterman.com",
				DictyStrainProperty: "bacterial strain",
			},
		},
	}

	_, err = repo.EditStrain(us)
	assert.NoErrorf(err, "expect no error, received %s", err)

	// Verify the update
	gm, err := repo.GetStrain(m.StockID)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(
		"bacterial strain",
		gm.StrainProperties.DictyStrainProperty,
		"should update ontology term to bacterial strain",
	)
	assert.Equal(
		"peterman@jpeterman.com",
		gm.UpdatedBy,
		"should update updatedby field",
	)
}

func TestEditStrainOntologyUpdateToGwdi(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)

	// Create and add initial strain with "general strain" property
	ns := newUpdatableTestStrain("art@vandelay.org", General)
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)

	// Update with REMI-seq ontology term
	us := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: ns.Data.Type,
			Id:   m.StockID,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy:           "peterman@jpeterman.com",
				DictyStrainProperty: "REMI-seq",
			},
		},
	}

	_, err = repo.EditStrain(us)
	assert.NoErrorf(err, "expect no error, received %s", err)

	// Verify the update
	gm, err := repo.GetStrain(m.StockID)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(
		"REMI-seq",
		gm.StrainProperties.DictyStrainProperty,
		"should update ontology term to REMI-seq",
	)
}

func TestEditStrainInvalidOntologyTerm(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)

	// Create and add initial strain
	ns := newUpdatableTestStrain("art@vandelay.org", General)
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)

	// Try to update with invalid ontology term
	us := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: ns.Data.Type,
			Id:   m.StockID,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy:           "peterman@jpeterman.com",
				DictyStrainProperty: "invalid ontology term",
			},
		},
	}

	_, err = repo.EditStrain(us)
	assert.Error(err, "should error on invalid ontology term")
	assert.Contains(
		err.Error(),
		"invalid ontology term",
		"error should mention invalid ontology term",
	)
}

func TestEditStrainOntologyUpdateWithOtherFields(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)

	// Create and add initial strain
	ns := newUpdatableTestStrain("art@vandelay.org", General)
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)

	// Update ontology term along with other fields
	us := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: ns.Data.Type,
			Id:   m.StockID,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy:           "peterman@jpeterman.com",
				DictyStrainProperty: "bacterial strain",
				Summary:             "updated summary with ontology",
				Label:               "updated-label",
				Species:             "updated species",
			},
		},
	}

	_, err = repo.EditStrain(us)
	assert.NoErrorf(err, "expect no error, received %s", err)

	// Verify all updates
	gm, err := repo.GetStrain(m.StockID)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(
		"bacterial strain",
		gm.StrainProperties.DictyStrainProperty,
		"should update ontology term",
	)
	assert.Equal(
		"updated summary with ontology",
		gm.Summary,
		"should update summary",
	)
	assert.Equal(
		"updated-label",
		gm.StrainProperties.Label,
		"should update label",
	)
	assert.Equal(
		"updated species",
		gm.StrainProperties.Species,
		"should update species",
	)
}

func TestEditStrainWithoutOntologyUpdate(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)

	// Create and add initial strain with bacterial strain property
	ns := newUpdatableTestStrain("art@vandelay.org", Bacterial)
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(
		"bacterial strain",
		m.StrainProperties.DictyStrainProperty,
		"initial strain should have bacterial strain property",
	)

	// Update without specifying ontology term (empty string)
	us := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: ns.Data.Type,
			Id:   m.StockID,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy: "peterman@jpeterman.com",
				Summary:   "updated summary only",
			},
		},
	}

	_, err = repo.EditStrain(us)
	assert.NoErrorf(err, "expect no error, received %s", err)

	// Verify ontology term remains unchanged
	gm, err := repo.GetStrain(m.StockID)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(
		"bacterial strain",
		gm.StrainProperties.DictyStrainProperty,
		"ontology term should remain unchanged when not specified",
	)
	assert.Equal(
		"updated summary only",
		gm.Summary,
		"summary should be updated",
	)
}
