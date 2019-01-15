package arangodb

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"testing"
	"time"

	driver "github.com/arangodb/go-driver"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/stretchr/testify/assert"
)

var gta *testarango.TestArango

func getConnectParams() *manager.ConnectParams {
	return &manager.ConnectParams{
		User:     gta.User,
		Pass:     gta.Pass,
		Database: gta.Database,
		Host:     gta.Host,
		Port:     gta.Port,
		Istls:    false,
	}
}

func getCollectionParams() *CollectionParams {
	return &CollectionParams{
		Stock:              "stock",
		StockProp:          "stockprop",
		StockKeyGenerator:  "stock_key_generator",
		StockType:          "stock_type",
		ParentStrain:       "parent_strain",
		StockPropTypeGraph: "stockprop_type",
		Strain2ParentGraph: "strain2parent",
		KeyOffset:          270000,
	}
}

func newTestStrain(createdby string) *stock.NewStock {
	return &stock.NewStock{
		Data: &stock.NewStock_Data{
			Type: "strain",
			Attributes: &stock.NewStockAttributes{
				CreatedBy:       createdby,
				UpdatedBy:       createdby,
				Depositor:       createdby,
				Summary:         "Radiation-sensitive mutant.",
				EditableSummary: "Radiation-sensitive mutant.",
				Dbxrefs:         []string{"5466867", "4536935", "d2578", "d0319", "d2020/1033268", "d2580"},
				Genes:           []string{"DDB_G0348394", "DDB_G098058933"},
				Publications:    []string{"4849343943", "48394394"},
				StrainProperties: &stock.StrainProperties{
					SystematicName: "yS13",
					Label:          "yS13",
					Species:        "Dictyostelium discoideum",
					Plasmid:        "DBP0000027",
					Names:          []string{"gammaS13", "gammaS-13", "γS-13"},
				},
			},
		},
	}
}

func newTestParentStrain(createdby string) *stock.NewStock {
	return &stock.NewStock{
		Data: &stock.NewStock_Data{
			Type: "strain",
			Attributes: &stock.NewStockAttributes{
				CreatedBy:       createdby,
				UpdatedBy:       createdby,
				Depositor:       createdby,
				Summary:         "Remi-mutant strain",
				EditableSummary: "Remi-mutant strain.",
				Dbxrefs:         []string{"5466867", "4536935", "d2578"},
				StrainProperties: &stock.StrainProperties{
					SystematicName: "AK40107",
					Label:          "egeB/DDB_G0270724_ps-REMI",
					Species:        "Dictyostelium discoideum",
					Names:          []string{"gammaS13", "BCN149086"},
				},
			},
		},
	}
}

func newUpdatableTestPlasmid(createdby string) *stock.NewStock {
	return &stock.NewStock{
		Data: &stock.NewStock_Data{
			Type: "plasmid",
			Attributes: &stock.NewStockAttributes{
				CreatedBy:       createdby,
				UpdatedBy:       createdby,
				Depositor:       createdby,
				Summary:         "update this plasmid",
				EditableSummary: "update this plasmid",
				Publications:    []string{"1348970", "48493483"},
				Dbxrefs:         []string{"5466867", "4536935", "d2578"},
			},
		},
	}
}

func newTestPlasmidWithProp(createdby string) *stock.NewStock {
	return &stock.NewStock{
		Data: &stock.NewStock_Data{
			Type: "plasmid",
			Attributes: &stock.NewStockAttributes{
				CreatedBy:       createdby,
				UpdatedBy:       createdby,
				Depositor:       createdby,
				Summary:         "this is a test plasmid",
				EditableSummary: "this is a test plasmid",
				Publications:    []string{"1348970"},
			},
		},
	}
}

func newTestPlasmid(createdby string) *stock.NewStock {
	return &stock.NewStock{
		Data: &stock.NewStock_Data{
			Type: "plasmid",
			Attributes: &stock.NewStockAttributes{
				CreatedBy:       createdby,
				UpdatedBy:       createdby,
				Depositor:       createdby,
				Summary:         "this is a test plasmid",
				EditableSummary: "this is a test plasmid",
				Publications:    []string{"1348970"},
				PlasmidProperties: &stock.PlasmidProperties{
					ImageMap: "http://dictybase.org/data/plasmid/images/87.jpg",
					Sequence: "tttttyyyyjkausadaaaavvvvvv",
				},
			},
		},
	}
}

func TestMain(m *testing.M) {
	ta, err := testarango.NewTestArangoFromEnv(true)
	if err != nil {
		log.Fatalf("unable to construct new TestArango instance %s", err)
	}
	gta = ta
	dbh, err := ta.DB(ta.Database)
	if err != nil {
		log.Fatalf("unable to get database %s", err)
	}
	cp := getCollectionParams()
	_, err = dbh.CreateCollection(cp.Stock, &driver.CreateCollectionOptions{})
	if err != nil {
		dbh.Drop()
		log.Fatalf("unable to create collection %s %s", cp.Stock, err)
	}
	code := m.Run()
	dbh.Drop()
	os.Exit(code)
}

func TestAddStrain(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer repo.ClearStocks()
	nsp := newTestParentStrain("todd@gagg.com")
	m, err := repo.AddStrain(nsp)
	if err != nil {
		t.Fatalf("error in adding strain: %s", err)
	}
	assert := assert.New(t)
	assert.Regexp(regexp.MustCompile(`^\d+$`), m.Key, "should have a key with numbers")
	assert.Regexp(regexp.MustCompile(`^DBS0\d{6,}$`), m.StockID, "should have a stock id")
	assert.Equal(m.CreatedBy, nsp.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(m.UpdatedBy, nsp.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(m.Summary, nsp.Data.Attributes.Summary, "should match summary")
	assert.Equal(m.EditableSummary, nsp.Data.Attributes.EditableSummary, "should match editable_summary")
	assert.Equal(m.Depositor, nsp.Data.Attributes.Depositor, "should match depositor")
	assert.ElementsMatch(m.Dbxrefs, nsp.Data.Attributes.Dbxrefs, "should match dbxrefs")
	assert.Empty(m.Genes, "should not be tied to any genes")
	assert.Empty(m.Publications, "should not be tied to any publications")
	assert.Equal(m.StrainProperties.SystematicName, nsp.Data.Attributes.StrainProperties.SystematicName, "should match systematic_name")
	assert.Equal(m.StrainProperties.Label, nsp.Data.Attributes.StrainProperties.Label, "should match descriptor")
	assert.Equal(m.StrainProperties.Species, nsp.Data.Attributes.StrainProperties.Species, "should match species")
	assert.ElementsMatch(m.StrainProperties.Names, nsp.Data.Attributes.StrainProperties.Names, "should match names")
	assert.Empty(m.StrainProperties.Plasmid, "should not have any plasmid")
	assert.Empty(m.StrainProperties.Parents, "should not have any parent")

	ns := newTestStrain("pennypacker@penny.com")
	ns.Data.Attributes.StrainProperties.Parents = []string{m.StockID}
	m2, err := repo.AddStrain(ns)
	if err != nil {
		t.Fatalf("error in adding strain: %s", err)
	}
	assert.Regexp(regexp.MustCompile(`^\d+$`), m2.Key, "should have a key with numbers")
	assert.Regexp(regexp.MustCompile(`^DBS0\d{6,}$`), m2.StockID, "should have a stock id")
	assert.Equal(m2.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(m2.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(m2.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(m2.EditableSummary, ns.Data.Attributes.EditableSummary, "should match editable_summary")
	assert.Equal(m2.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
	assert.ElementsMatch(m2.Dbxrefs, ns.Data.Attributes.Dbxrefs, "should match dbxrefs")
	assert.ElementsMatch(m2.Genes, ns.Data.Attributes.Genes, "should match gene ids")
	assert.ElementsMatch(
		m2.Publications,
		ns.Data.Attributes.Publications,
		"should match list of publications",
	)
	assert.Equal(
		m2.StrainProperties.SystematicName,
		ns.Data.Attributes.StrainProperties.SystematicName,
		"should match systematic_name",
	)
	assert.Equal(
		m2.StrainProperties.Label,
		ns.Data.Attributes.StrainProperties.Label,
		"should match descriptor",
	)
	assert.Equal(
		m2.StrainProperties.Species,
		ns.Data.Attributes.StrainProperties.Species,
		"should match species",
	)
	assert.ElementsMatch(
		m2.StrainProperties.Names,
		ns.Data.Attributes.StrainProperties.Names,
		"should match names",
	)
	assert.Equal(
		m2.StrainProperties.Plasmid,
		ns.Data.Attributes.StrainProperties.Plasmid,
		"should match plasmid entry",
	)
	assert.ElementsMatch(
		m2.StrainProperties.Parents,
		ns.Data.Attributes.StrainProperties.Parents,
		"should match parent entries",
	)
}

func TestAddPlasmidWithoutProp(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer repo.ClearStocks()
	ns := newTestPlasmidWithProp("george@costanza.com")
	m, err := repo.AddPlasmid(ns)
	if err != nil {
		t.Fatalf("error in adding plasmid: %s", err)
	}
	assert := assert.New(t)
	assert.Regexp(regexp.MustCompile(`^\d+$`), m.Key, "should have a key with numbers")
	assert.Regexp(regexp.MustCompile(`^DBP0\d{6,}$`), m.StockID, "should have a plasmid stock id")
	assert.Equal(m.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(m.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(m.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(
		m.EditableSummary,
		ns.Data.Attributes.EditableSummary,
		"should match editable_summary",
	)
	assert.Equal(m.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
	assert.Empty(m.Genes, "should have empty genes field")
	assert.Empty(m.Dbxrefs, "should have empty dbxrefs field")
	assert.ElementsMatch(
		m.Publications,
		ns.Data.Attributes.Publications,
		"should match publications",
	)
}

func TestAddPlasmid(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer repo.ClearStocks()
	ns := newTestPlasmid("george@costanza.com")
	m, err := repo.AddPlasmid(ns)
	if err != nil {
		t.Fatalf("error in adding plasmid: %s", err)
	}
	assert := assert.New(t)
	assert.Regexp(regexp.MustCompile(`^\d+$`), m.Key, "should have a key with numbers")
	assert.Regexp(regexp.MustCompile(`^DBP0\d{6,}$`), m.StockID, "should have a plasmid stock id")
	assert.Equal(m.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(m.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(m.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(
		m.EditableSummary,
		ns.Data.Attributes.EditableSummary,
		"should match editable_summary",
	)
	assert.Equal(m.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
	assert.Empty(m.Genes, "should have empty genes field")
	assert.Empty(m.Dbxrefs, "should have empty dbxrefs field")
	assert.ElementsMatch(
		m.Publications,
		ns.Data.Attributes.Publications,
		"should match publications",
	)
	assert.Equal(
		m.PlasmidProperties.ImageMap,
		ns.Data.Attributes.PlasmidProperties.ImageMap,
		"should match image_map",
	)
	assert.Equal(
		m.PlasmidProperties.Sequence,
		ns.Data.Attributes.PlasmidProperties.Sequence,
		"should match sequence",
	)
}

func TestEditPlasmid(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer repo.ClearStocks()
	ns := newUpdatableTestPlasmid("art@vandeley.org")
	m, err := repo.AddPlasmid(ns)
	if err != nil {
		t.Fatalf("error in adding plasmid: %s", err)
	}
	us := &stock.StockUpdate{
		Data: &stock.StockUpdate_Data{
			Type: ns.Data.Type,
			Id:   m.StockID,
			Attributes: &stock.StockUpdateAttributes{
				UpdatedBy:       "varnes@seinfeld.org",
				Summary:         "updated plasmid",
				EditableSummary: "updated plasmid",
				Publications:    []string{"8394839", "583989343", "853983948"},
				Genes:           []string{"DDB_G0270724", "DDB_G027489343"},
				PlasmidProperties: &stock.PlasmidProperties{
					ImageMap: "http://dictybase.org/data/plasmid/images/87.jpg",
				},
			},
		},
	}
	um, err := repo.EditPlasmid(us)
	if err != nil {
		t.Fatalf("error in updating plasmid %s", err)
	}
	assert := assert.New(t)
	assert.Equal(um.StockID, m.StockID, "should match the stock id")
	assert.Equal(um.UpdatedBy, us.Data.Attributes.UpdatedBy, "should match updatedby")
	assert.Equal(um.Depositor, ns.Data.Attributes.Depositor, "depositor name should not be updated")
	assert.Equal(um.Summary, us.Data.Attributes.Summary, "should have updated summary")
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
	us2 := &stock.StockUpdate{
		Data: &stock.StockUpdate_Data{
			Type: ns.Data.Type,
			Id:   um.StockID,
			Attributes: &stock.StockUpdateAttributes{
				UpdatedBy: "puddy@seinfeld.org",
				Genes:     []string{"DDB_G0270851", "DDB_G02748", "DDB_G7392222"},
				PlasmidProperties: &stock.PlasmidProperties{
					Sequence: "atgctagagaagacttt",
				},
			},
		},
	}
	um2, err := repo.EditPlasmid(us2)
	if err != nil {
		t.Fatalf("error in re updating the plasmid %s", err)
	}
	assert.Equal(um2.StockID, um.StockID, "should match the previous stock id")
	assert.Equal(um2.UpdatedBy, us2.Data.Attributes.UpdatedBy, "should have updated the updatedby field")
	assert.ElementsMatch(um2.Genes, us2.Data.Attributes.Genes, "should have the genes list")
	assert.ElementsMatch(um2.Publications, um.Publications, "publications list should remain the same")
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
		us2.Data.Attributes.PlasmidProperties.Sequence,
		"sequence plasmid property should have been updated",
	)
}

func TestGetStock(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer repo.ClearStocks()
	ns := newTestStrain("george@costanza.com")
	m, err := repo.AddStrain(ns)
	if err != nil {
		t.Fatalf("error in adding strain %s", err)
	}
	g, err := repo.GetStock(m.StockID)
	if err != nil {
		t.Fatalf("error in getting stock %s with ID %s", m.StockID, err)
	}
	assert := assert.New(t)
	assert.Equal(g.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(g.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(g.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(g.EditableSummary, ns.Data.Attributes.EditableSummary, "should match editable_summary")
	assert.Equal(g.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
	assert.Equal(g.Dbxrefs, ns.Data.Attributes.Dbxrefs, "should match dbxrefs")
	assert.Equal(g.StrainProperties.SystematicName, ns.Data.Attributes.StrainProperties.SystematicName, "should match systematic_name")
	assert.Equal(g.StrainProperties.Label, ns.Data.Attributes.StrainProperties.Label, "should match descriptor")
	assert.Equal(g.StrainProperties.Species, ns.Data.Attributes.StrainProperties.Species, "should match species")
	assert.Equal(g.StrainProperties.Parents, ns.Data.Attributes.StrainProperties.Parents, "should match parents")
	assert.Equal(g.StrainProperties.Names, ns.Data.Attributes.StrainProperties.Names, "should match names")
	assert.Empty(g.Genes, ns.Data.Attributes.Genes, "should have empty genes field")
	assert.Empty(g.StrainProperties.Plasmid, ns.Data.Attributes.StrainProperties.Plasmid, "should have empty plasmid field")
	assert.Equal(len(g.Dbxrefs), 6, "should match length of six dbxrefs")
	assert.NotEmpty(g.Key, "should not have empty key")
	assert.True(m.CreatedAt.Equal(g.CreatedAt), "should match created time of stock")
	assert.True(m.UpdatedAt.Equal(g.UpdatedAt), "should match updated time of stock")

	ne, err := repo.GetStock("DBS01")
	if err != nil {
		t.Fatalf(
			"error in fetching stock %s with ID %s",
			"DBS01",
			err,
		)
	}
	assert.True(ne.NotFound, "entry should not exist")
}

// func TestEditStock(t *testing.T) {

// }

func TestListStocks(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer repo.ClearStocks()
	// add 10 new test strains
	for i := 1; i <= 10; i++ {
		ns := newTestStrain(fmt.Sprintf("%s@kramericaindustries.com", RandString(10)))
		_, err := repo.AddStrain(ns)
		if err != nil {
			t.Fatalf("error in adding strain %s", err)
		}
	}
	// add 5 new test plasmids
	for i := 1; i <= 5; i++ {
		np := newTestPlasmid(fmt.Sprintf("%s@cye.com", RandString(10)))
		_, err := repo.AddPlasmid(np)
		if err != nil {
			t.Fatalf("error in adding plasmid %s", err)
		}
	}
	// get first five results
	ls, err := repo.ListStocks(0, 4)
	if err != nil {
		t.Fatalf("error in getting first five stocks %s", err)
	}
	assert := assert.New(t)
	assert.Equal(len(ls), 5, "should match the provided limit number + 1")

	for _, stock := range ls {
		assert.Equal(stock.Depositor, "george@costanza.com", "should match the depositor")
		assert.NotEmpty(stock.Key, "should not have empty key")
	}
	assert.NotEqual(ls[0].CreatedBy, ls[1].CreatedBy, "should have different created_by")
	// convert fifth result to numeric timestamp in milliseconds
	// so we can use this as cursor
	ti := toTimestamp(ls[4].CreatedAt)

	// get next five results (5-9)
	ls2, err := repo.ListStocks(ti, 4)
	if err != nil {
		t.Fatalf("error in getting stocks 5-9 %s", err)
	}
	assert.Equal(len(ls2), 5, "should match the provided limit number + 1")
	assert.Equal(ls2[0], ls[4], "last item from first five results and first item from next five results should be the same")
	assert.NotEqual(ls2[0].CreatedBy, ls2[1].CreatedBy, "should have different consumers")

	// convert ninth result to numeric timestamp
	ti2 := toTimestamp(ls2[4].CreatedAt)
	// get last five results (9-13)
	ls3, err := repo.ListStocks(ti2, 4)
	if err != nil {
		t.Fatalf("error in getting stocks 9-13 %s", err)
	}
	assert.Equal(len(ls3), 5, "should match the provided limit number + 1")
	assert.Equal(ls3[0].CreatedBy, ls2[4].CreatedBy, "last item from previous five results and first item from next five results should be the same")

	// convert 13th result to numeric timestamp
	ti3 := toTimestamp(ls3[4].CreatedAt)
	// get last results
	ls4, err := repo.ListStocks(ti3, 4)
	if err != nil {
		t.Fatalf("error in getting stocks 13-15 %s", err)
	}
	assert.Equal(len(ls4), 3, "should only bring last three results")
	assert.Equal(ls3[4].CreatedBy, ls4[0].CreatedBy, "last item from previous five results and first item from next three results should be the same")
	testModelListSort(ls, t)
	testModelListSort(ls2, t)
	testModelListSort(ls3, t)
	testModelListSort(ls4, t)
}

// func TestListStrains(t *testing.T) {

// }

// func TestListPlasmids(t *testing.T) {

// }

func TestRemoveStock(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer repo.ClearStocks()
	ns := newTestStrain("george@costanza.com")
	m, err := repo.AddStrain(ns)
	if err != nil {
		t.Fatalf("error in adding strain: %s", err)
	}
	err = repo.RemoveStock(m.Key)
	if err != nil {
		t.Fatalf("error in removing stock %s with key %s",
			m.Key,
			err)
	}
	ne, err := repo.GetStock(m.Key)
	if err != nil {
		t.Fatalf(
			"error in fetching stock %s with ID %s",
			m.Key,
			err,
		)
	}
	assert := assert.New(t)
	assert.True(ne.NotFound, "entry should not exist")
}

func testModelListSort(m []*model.StockDoc, t *testing.T) {
	it, err := NewPairWiseIterator(m)
	if err != nil {
		t.Fatal(err)
	}
	assert := assert.New(t)
	for it.NextPair() {
		cm, nm := it.Pair()
		assert.Truef(
			nm.CreatedAt.Before(cm.CreatedAt),
			"date %s should be before %s",
			nm.CreatedAt.String(),
			cm.CreatedAt.String(),
		)
	}
}

const (
	charSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var seedRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	var b []byte
	for i := 0; i < length; i++ {
		b = append(
			b,
			charset[seedRand.Intn(len(charset))],
		)
	}
	return string(b)
}

func RandString(length int) string {
	return stringWithCharset(length, charSet)
}

func toTimestamp(t time.Time) int64 {
	return t.UnixNano() / 1000000
}
