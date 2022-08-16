package arangodb

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
)

const (
	georgeFilter = `FILTER s.depositor == 'george@costanza.com'`
)

func TestLoadStockWithPlasmids(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	tm, _ := time.Parse("2006-01-02 15:04:05", "2010-03-30 14:40:58")
	ns := &stock.ExistingPlasmid{
		Data: &stock.ExistingPlasmid_Data{
			Type: "plasmid",
			Id:   "DBP0000098",
			Attributes: &stock.ExistingPlasmidAttributes{
				CreatedAt:       aphgrpc.TimestampProto(tm),
				UpdatedAt:       aphgrpc.TimestampProto(tm),
				CreatedBy:       "george@costanza.com",
				UpdatedBy:       "george@costanza.com",
				Depositor:       "george@costanza.com",
				Summary:         "this is a test plasmid",
				EditableSummary: "this is a test plasmid",
				Publications:    []string{"1348970"},
				ImageMap:        "http://dictybase.org/data/plasmid/images/87.jpg",
				Sequence:        "tttttyyyyjkausadaaaavvvvvv",
				Name:            "p9999",
			},
		},
	}
	um, err := repo.LoadPlasmid("DBP0000098", ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal("DBP0000098", um.StockID, "should match given plasmid stock id")
	assert.Equal(um.Key, um.StockID, "should have identical key and stock ID")
	assert.Equal(
		um.CreatedBy,
		ns.Data.Attributes.CreatedBy,
		"should match created_by id",
	)
	assert.Equal(
		um.UpdatedBy,
		ns.Data.Attributes.UpdatedBy,
		"should match updated_by id",
	)
	assert.Equal(um.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(
		um.EditableSummary,
		ns.Data.Attributes.EditableSummary,
		"should match editable_summary",
	)
	assert.Equal(
		um.Depositor,
		ns.Data.Attributes.Depositor,
		"should match depositor",
	)
	assert.Empty(um.Genes, "should have empty genes field")
	assert.Empty(um.Dbxrefs, "should have empty dbxrefs field")
	assert.ElementsMatch(
		um.Publications,
		ns.Data.Attributes.Publications,
		"should match publications",
	)
	assert.Equal(
		um.PlasmidProperties.ImageMap,
		ns.Data.Attributes.ImageMap,
		"should match image_map",
	)
	assert.Equal(
		um.PlasmidProperties.Sequence,
		ns.Data.Attributes.Sequence,
		"should match sequence",
	)
}

func TestListPlasmidsWithFilter(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	// add 10 new test plasmids
	for i := 1; i <= 10; i++ {
		np := newTestPlasmid(fmt.Sprintf("%s@cye.com", RandString(10)))
		_, err := repo.AddPlasmid(np)
		assert.NoErrorf(err, "expect no error, received %s", err)
	}
	sf, err := repo.ListPlasmids(
		&stock.StockParameters{Limit: 10, Filter: georgeFilter},
	)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(sf, 10, "should list ten plasmids")
	for _, um := range sf {
		assert.Equal(
			um.Summary,
			"this is a test plasmid",
			"should match summary",
		)
		assert.Equal(um.PlasmidProperties.Name, "p123456", "should match name")
	}

	filterTwo := `FILTER s.depositor == 'george@costanza.com' AND s.depositor == 'rg@gmail.com'`
	n, err := repo.ListPlasmids(
		&stock.StockParameters{Limit: 100, Filter: filterTwo},
	)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(n, 0, "should list no plasmids")

	filterThree := `LET x = (
						FILTER '1348970' IN s.publications 
						RETURN 1
					)`
	// do a check for array filter
	as, err := repo.ListPlasmids(
		&stock.StockParameters{
			Cursor: toTimestamp(sf[5].CreatedAt),
			Limit:  10,
			Filter: filterThree,
		},
	)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(as, 5, "should list five plasmids")

	filterFour := `FILTER s.created_at <= DATE_ISO8601('2019')`
	da, err := repo.ListPlasmids(
		&stock.StockParameters{
			Cursor: toTimestamp(sf[5].CreatedAt),
			Limit:  10,
			Filter: filterFour,
		},
	)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(da, 0, "should list no plasmids")

	filterFive := `FILTER v.sequence =~ 'ttttt'`
	ff, err := repo.ListPlasmids(
		&stock.StockParameters{Limit: 10, Filter: filterFive},
	)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(ff, 10, "should list ten plasmids")

	filterSix := `FILTER s.summary =~ 'test'`
	fs, err := repo.ListPlasmids(
		&stock.StockParameters{Limit: 10, Filter: filterSix},
	)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(fs, 10, "should list ten plasmids")

	filterSeven := `FILTER s.depositor == 'george@costanza.com' OR v.name == 'gammaS13'`
	fv, err := repo.ListPlasmids(
		&stock.StockParameters{
			Cursor: toTimestamp(sf[5].CreatedAt),
			Limit:  10,
			Filter: filterSeven,
		},
	)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(fv, 5, "should list five plasmids")
}

func TestListPlasmids(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	for i := 1; i <= 10; i++ {
		np := newTestPlasmid(fmt.Sprintf("%s@cye.com", RandString(10)))
		_, err := repo.AddPlasmid(np)
		assert.NoErrorf(err, "expect no error adding plasmid, received %s", err)
	}
	ls, err := repo.ListPlasmids(&stock.StockParameters{Limit: 4})
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
	// convert fifth result to numeric timestamp in milliseconds
	// so we can use this as cursor
	ti := toTimestamp(ls[len(ls)-1].CreatedAt)

	// get next five results (5-9)
	ls2, err := repo.ListPlasmids(&stock.StockParameters{Cursor: ti, Limit: 4})
	assert.NoErrorf(
		err,
		"expect no error getting plasmids 5-9, received %s",
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
	ls3, err := repo.ListPlasmids(&stock.StockParameters{Cursor: ti2, Limit: 4})
	assert.NoErrorf(
		err,
		"expect no error getting plasmids 9-10, received %s",
		err,
	)
	assert.Len(ls3, 2, "should retrieve the last two results")
	assert.Exactly(
		ls3[0].CreatedBy,
		ls2[len(ls2)-1].CreatedBy,
		"last item from previous five results and first item from next five results should be the same",
	)

	testModelListSort(ls, t)
	testModelListSort(ls2, t)
	testModelListSort(ls3, t)

	sf, err := repo.ListPlasmids(
		&stock.StockParameters{Limit: 100, Filter: georgeFilter},
	)
	assert.NoErrorf(
		err,
		"expect no error getting list of plasmids, received %s",
		err,
	)
	assert.Len(sf, 10, "should list ten plasmids")

	cs, err := repo.ListPlasmids(
		&stock.StockParameters{Cursor: toTimestamp(sf[4].CreatedAt), Limit: 10},
	)
	assert.NoErrorf(
		err,
		"expect no error getting list of plasmids with cursor, received %s",
		err,
	)
	assert.Len(cs, 6, "should list six plasmids")
}

func TestGetPlasmid(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ns := newTestPlasmid("george@costanza.com")
	um, err := repo.AddPlasmid(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	g, err := repo.GetPlasmid(um.StockID)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Regexp(
		regexp.MustCompile(`^DBP0\d{6,}$`),
		g.StockID,
		"should have a plasmid stock id",
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
	assert.Equal(
		g.Depositor,
		ns.Data.Attributes.Depositor,
		"should match depositor",
	)
	assert.Equal(g.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(
		g.EditableSummary,
		ns.Data.Attributes.EditableSummary,
		"should match editable_summary",
	)
	assert.Equal(
		g.PlasmidProperties.ImageMap,
		ns.Data.Attributes.ImageMap,
		"should match image_map",
	)
	assert.Equal(
		g.PlasmidProperties.Sequence,
		ns.Data.Attributes.Sequence,
		"should match sequence",
	)
	assert.Equal(
		g.PlasmidProperties.Name,
		ns.Data.Attributes.Name,
		"should match name",
	)
	assert.True(
		um.CreatedAt.Equal(g.CreatedAt),
		"should match created time of stock",
	)
	assert.True(
		um.UpdatedAt.Equal(g.UpdatedAt),
		"should match updated time of stock",
	)

	ne, err := repo.GetPlasmid("DBP01")
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.True(ne.NotFound, "entry should not exist")
}

func TestEditPlasmid(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ns := newUpdatableTestPlasmid("art@vandelay.org")
	m, err := repo.AddPlasmid(ns)
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
	um, err := repo.EditPlasmid(us)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(um.StockID, um.StockID, "should match the stock id")
	assert.Equal(
		um.UpdatedBy,
		us.Data.Attributes.UpdatedBy,
		"should match updatedby",
	)
	assert.Equal(
		um.Depositor,
		ns.Data.Attributes.Depositor,
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
		um.Publications,
		us.Data.Attributes.Publications,
		"should match updated list of publications",
	)
	assert.ElementsMatch(
		um.Dbxrefs,
		ns.Data.Attributes.Dbxrefs,
		"dbxrefs should remain unchanged",
	)
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

func TestEditPlasmidGene(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ns := newUpdatableTestPlasmid("art@vandelay.org")
	um, err := repo.AddPlasmid(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	us2 := &stock.PlasmidUpdate{
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
	um2, err := repo.EditPlasmid(us2)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(um2.StockID, um.StockID, "should match the previous stock id")
	assert.Equal(
		um2.UpdatedBy,
		us2.Data.Attributes.UpdatedBy,
		"should have updated the updatedby field",
	)
	assert.ElementsMatch(
		um2.Genes,
		us2.Data.Attributes.Genes,
		"should have the genes list",
	)
	assert.ElementsMatch(
		um2.Publications,
		um.Publications,
		"publications list should remain the same",
	)
	assert.ElementsMatch(
		um2.Dbxrefs,
		um.Dbxrefs,
		"dbxrefs list should remain the same",
	)
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
	um3, err := repo.EditPlasmid(us3)
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

func TestAddPlasmid(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	ns := newTestPlasmid("george@costanza.com")
	um, err := repo.AddPlasmid(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Regexp(
		regexp.MustCompile(`^DBP0\d{6,}$`),
		um.StockID,
		"should have a plasmid stock id",
	)
	assert.Equal(um.Key, um.StockID, "should have identical key and stock ID")
	assert.Equal(
		um.CreatedBy,
		ns.Data.Attributes.CreatedBy,
		"should match created_by id",
	)
	assert.Equal(
		um.UpdatedBy,
		ns.Data.Attributes.UpdatedBy,
		"should match updated_by id",
	)
	assert.Equal(um.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(
		um.EditableSummary,
		ns.Data.Attributes.EditableSummary,
		"should match editable_summary",
	)
	assert.Equal(
		um.Depositor,
		ns.Data.Attributes.Depositor,
		"should match depositor",
	)
	assert.Empty(um.Genes, "should have empty genes field")
	assert.Empty(um.Dbxrefs, "should have empty dbxrefs field")
	assert.ElementsMatch(
		um.Publications,
		ns.Data.Attributes.Publications,
		"should match publications",
	)
	assert.Equal(
		um.PlasmidProperties.ImageMap,
		ns.Data.Attributes.ImageMap,
		"should match image_map",
	)
	assert.Equal(
		um.PlasmidProperties.Sequence,
		ns.Data.Attributes.Sequence,
		"should match sequence",
	)
	assert.Equal(
		um.PlasmidProperties.Name,
		ns.Data.Attributes.Name,
		"should match name",
	)
}
