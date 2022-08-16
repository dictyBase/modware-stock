package arangodb

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
)

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
}

func TestLoadStockWithParent(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
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

func TestListStrainsWithFilter(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	// add 10 new test strains
	for i := 1; i <= 10; i++ {
		ns := newTestStrain(
			fmt.Sprintf("%s@kramericaindustries.com", RandString(10)),
		)
		_, err := repo.AddStrain(ns)
		assert.NoErrorf(err, "expect no error adding strain, received %s", err)
	}
	filterOne := `FILTER s.depositor == 'george@costanza.com'`
	sf, err := repo.ListStrains(
		&stock.StockParameters{Limit: 10, Filter: filterOne},
	)
	assert.NoErrorf(
		err,
		"expect no error getting list of strains, received %s",
		err,
	)
	assert.Len(sf, 10, "should list ten strains")
	for _, m := range sf {
		assert.Equal(
			m.Summary,
			"Radiation-sensitive mutant.",
			"should match summary",
		)
		assert.Equal(m.StrainProperties.Label, "yS13", "should match label")
	}

	filterTwo := `FILTER s.depositor == 'george@costanza.com' AND s.depositor == 'rg@gmail.com'`
	n, err := repo.ListStrains(
		&stock.StockParameters{Limit: 100, Filter: filterTwo},
	)
	assert.NoErrorf(
		err,
		"expect no error getting list of stocks with two depositors with AND logic, received %s",
		err,
	)
	assert.Len(n, 0, "should list no strains")

	filterThree := `LET x = (
						FILTER 'gammaS13' IN s.names 
						RETURN 1
					)`
	// do a check for array filter
	as, err := repo.ListStrains(
		&stock.StockParameters{
			Cursor: toTimestamp(sf[5].CreatedAt),
			Limit:  10,
			Filter: filterThree,
		},
	)
	assert.NoErrorf(
		err,
		"expect no error getting list of stocks with cursor and filter, received %s",
		err,
	)
	assert.Len(as, 5, "should list five strains")

	filterFour := `FILTER s.created_at <= DATE_ISO8601('2019')`
	da, err := repo.ListStrains(
		&stock.StockParameters{
			Cursor: toTimestamp(sf[5].CreatedAt),
			Limit:  10,
			Filter: filterFour,
		},
	)
	assert.NoErrorf(
		err,
		"expect no error getting list of stocks with cursor and date filter, received %s",
		err,
	)
	assert.Len(da, 0, "should list no strains")

	filterFive := `FILTER v.label =~ 'yS'`
	ff, err := repo.ListStrains(
		&stock.StockParameters{Limit: 10, Filter: filterFive},
	)
	assert.NoErrorf(
		err,
		"expect no error getting list of strains with label substring, received %s",
		err,
	)
	assert.Len(ff, 10, "should list ten strains")

	filterSix := `FILTER s.summary =~ 'mutant'`
	fs, err := repo.ListStrains(
		&stock.StockParameters{Limit: 10, Filter: filterSix},
	)
	assert.NoErrorf(
		err,
		"expect no error getting list of strains with summary substring, received %s",
		err,
	)
	assert.Len(fs, 10, "should list ten strains")
}

func TestListStrains(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	// add 10 new test strains
	for i := 1; i <= 10; i++ {
		ns := newTestStrain(
			fmt.Sprintf("%s@kramericaindustries.com", RandString(10)),
		)
		_, err := repo.AddStrain(ns)
		assert.NoErrorf(err, "expect no error adding strain, received %s", err)
	}
	// get first five results
	ls, err := repo.ListStrains(&stock.StockParameters{Limit: 4})
	assert.NoErrorf(
		err,
		"expect no error in getting first five stocks, received %s",
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
	assert.NoErrorf(
		err,
		"expect no error in getting stocks 5-9, received %s",
		err,
	)
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

	filter := `FILTER s.depositor == 'george@costanza.com'`
	sf, err := repo.ListStrains(
		&stock.StockParameters{Limit: 10, Filter: filter},
	)
	assert.NoErrorf(
		err,
		"expect no error in getting list of strains with depositor george@costanza.com, received %s",
		err,
	)
	assert.Len(sf, 10, "should list ten strains")

	cs, err := repo.ListStrains(
		&stock.StockParameters{Cursor: toTimestamp(sf[4].CreatedAt), Limit: 10},
	)
	assert.NoErrorf(
		err,
		"expect no error getting list of strains with cursor, received %s",
		err,
	)
	assert.Len(cs, 6, "should list six strains")
}

func TestListStrainsByIds(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	// add 10 new test strains
	ids := make([]string, 0)
	for i := 1; i <= 30; i++ {
		ns := newTestStrain(
			fmt.Sprintf("%s@kramericaindustries.com", RandString(10)),
		)
		m, err := repo.AddStrain(ns)
		assert.NoErrorf(err, "expect no error adding strain, received %s", err)
		ids = append(ids, m.StockID)
	}
	// get first five results
	ls, err := repo.ListStrainsByIds(&stock.StockIdList{Id: ids})
	assert.NoErrorf(
		err,
		"expect no error in getting 30 stocks, received %s",
		err,
	)
	assert.Len(ls, 30, "should match the provided limit number")

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
		assert.Empty(
			stock.StrainProperties.Parent,
			"parent field should be empty",
		)
		assert.Equal(
			stock.StrainProperties.DictyStrainProperty,
			"general strain",
			"should match ontology strain property",
		)
	}
	// strain with parents
	pm, err := repo.AddStrain(newTestParentStrain("j@peterman.org"))
	assert.NoErrorf(
		err,
		"expect no error in creating parent strain, received %s",
		err,
	)
	pids := make([]string, 0)
	for i := 1; i <= 30; i++ {
		ns := newTestStrain(fmt.Sprintf("%s@mailman.com", RandString(10)))
		ns.Data.Attributes.Parent = pm.StockID
		m, err := repo.AddStrain(ns)
		assert.NoErrorf(
			err,
			"expect no error adding strain with parent, received %s",
			err,
		)
		pids = append(pids, m.StockID)
	}
	pls, err := repo.ListStrainsByIds(&stock.StockIdList{Id: pids})
	assert.NoErrorf(
		err,
		"expect no error in getting 30 stocks with parents, received %s",
		err,
	)
	assert.Len(pls, 30, "should match the provided limit number")

	for _, stock := range pls {
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
		assert.Equal(
			stock.StrainProperties.Parent,
			pm.StockID,
			"should match parent id",
		)
		assert.Equal(
			stock.StrainProperties.DictyStrainProperty,
			"general strain",
			"should match ontology strain property",
		)
	}
	// Non-existing ids
	els, err := repo.ListStrainsByIds(
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
	ns := newTestStrain("george@costanza.com")
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	g, err := repo.GetStrain(m.StockID)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Regexp(
		regexp.MustCompile(`^DBS0\d{6,}$`),
		g.StockID,
		"should have a strain stock id",
	)
	assert.Equal(
		g.CreatedBy,
		ns.Data.Attributes.CreatedBy,
		"should match created_by id",
	)
	assert.Equal(
		g.UpdatedBy,
		ns.Data.Attributes.UpdatedBy,
		"should match updated_by id",
	)
	assert.Equal(g.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(
		g.EditableSummary,
		ns.Data.Attributes.EditableSummary,
		"should match editable_summary",
	)
	assert.Equal(
		g.Depositor,
		ns.Data.Attributes.Depositor,
		"should match depositor",
	)
	assert.ElementsMatch(
		g.Dbxrefs,
		ns.Data.Attributes.Dbxrefs,
		"should match dbxrefs",
	)
	assert.ElementsMatch(
		g.StrainProperties.Names,
		ns.Data.Attributes.Names,
		"should match names",
	)
	assert.ElementsMatch(
		g.Genes,
		ns.Data.Attributes.Genes,
		"should match genes",
	)
	assert.Equal(
		g.StrainProperties.Label,
		ns.Data.Attributes.Label,
		"should match descriptor",
	)
	assert.Equal(
		g.StrainProperties.Species,
		ns.Data.Attributes.Species,
		"should match species",
	)
	assert.Equal(
		g.StrainProperties.Plasmid,
		ns.Data.Attributes.Plasmid,
		"should match plasmid",
	)
	assert.Empty(g.StrainProperties.Parent, "should not have parent")
	assert.Equal(
		g.StrainProperties.DictyStrainProperty,
		"general strain",
		"should match ontology strain property",
	)
	assert.Len(g.Dbxrefs, 6, "should match length of six dbxrefs")
	assert.True(
		m.CreatedAt.Equal(g.CreatedAt),
		"should match created time of stock",
	)
	assert.True(
		m.UpdatedAt.Equal(g.UpdatedAt),
		"should match updated time of stock",
	)

	ns2 := newTestStrain("dead@cells.com")
	ns2.Data.Attributes.Parent = m.StockID
	m2, err := repo.AddStrain(ns2)
	assert.NoErrorf(err, "expect no error, received %s", err)
	g2, err := repo.GetStrain(m2.StockID)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(g2.StrainProperties.Parent, m.StockID, "should match parent")
	assert.Equal(
		g2.StrainProperties.DictyStrainProperty,
		"general strain",
		"should match ontology strain property",
	)
	ne, err := repo.GetStrain("DBS01")
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.True(ne.NotFound, "entry should not exist")
}

func TestAddStrain(t *testing.T) {
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

	ns := newTestStrain("pennypacker@penny.com")
	ns.Data.Attributes.Parent = m.StockID
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

func TestEditStrain(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	ns := newUpdatableTestStrain("todd@gagg.com")
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	us := &stock.StrainUpdate{
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
	um, err := repo.EditStrain(us)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(um.StockID, m.StockID, "should match the stock id")
	assert.Equal(
		um.UpdatedBy,
		us.Data.Attributes.UpdatedBy,
		"should match updatedby",
	)
	assert.Equal(
		um.Depositor,
		m.Depositor,
		"depositor name should not be updated",
	)
	assert.Equal(
		um.Summary,
		us.Data.Attributes.Summary,
		"should have updated summary",
	)
	assert.Equal(
		um.EditableSummary,
		us.Data.Attributes.EditableSummary,
		"should have updated editable summary",
	)
	assert.ElementsMatch(
		um.Genes,
		us.Data.Attributes.Genes,
		"should match updated list of genes",
	)
	assert.ElementsMatch(
		um.Dbxrefs,
		us.Data.Attributes.Dbxrefs,
		"should match updated list of dbxrefs",
	)
	assert.ElementsMatch(
		um.Publications,
		m.Publications,
		"publications list should remain unchanged",
	)
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

func TestEditStrainWithParent(t *testing.T) {
	t.Parallel()
	assert, repo := setUp(t)
	defer tearDown(repo)
	pm, err := repo.AddStrain(newTestParentStrain("tim@watley.org"))
	assert.NoErrorf(err, "expect no error, received %s", err)
	ns := newUpdatableTestStrain("todd@gagg.com")
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
	assert.Equal(um2.StockID, ust.StockID, "should match their id")
	assert.Equal(
		um2.Depositor,
		us2.Data.Attributes.Depositor,
		"depositor name should be updated",
	)
	assert.Equal(
		um2.CreatedBy,
		ust.CreatedBy,
		"created by should not be updated",
	)
	assert.Equal(um2.Summary, ust.Summary, "summary should not be updated")
	assert.ElementsMatch(
		um2.Publications,
		ust.Publications,
		"publications list should remains unchanged",
	)
	assert.ElementsMatch(
		um2.Genes,
		ust.Genes,
		"genes list should not be updated",
	)
	assert.ElementsMatch(
		um2.Dbxrefs,
		ust.Dbxrefs,
		"dbxrefs list should not be updated",
	)
	assert.Equal(
		um2.StrainProperties.Label,
		ust.StrainProperties.Label,
		"strain descriptor should not be updated",
	)
	assert.ElementsMatch(
		um2.StrainProperties.Names,
		ust.StrainProperties.Names,
		"strain names should not be updated",
	)
	assert.Equal(
		um2.StrainProperties.Plasmid,
		ust.StrainProperties.Plasmid,
		"plasmid should not be updated",
	)
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
	// add another new strain, let's make this one a parent
	// so we can test updating parent if one already exists
	pu, err := repo.AddStrain(newUpdatableTestStrain("castle@vania.org"))
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
