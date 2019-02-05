package arangodb

import (
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

func newUpdatableTestStrain(createdby string) *stock.NewStock {
	return &stock.NewStock{
		Data: &stock.NewStock_Data{
			Type: "strain",
			Attributes: &stock.NewStockAttributes{
				CreatedBy:       createdby,
				UpdatedBy:       createdby,
				Depositor:       createdby,
				Summary:         "Radiation-sensitive mutant.",
				EditableSummary: "Radiation-sensitive mutant.",
				Genes:           []string{"DDB_G0348394", "DDB_G098058933"},
				Publications:    []string{"48428304983", "83943", "839434936743"},
				StrainProperties: &stock.StrainProperties{
					SystematicName: "yS13",
					Label:          "yS13",
					Species:        "Dictyostelium discoideum",
				},
			},
		},
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

func newTestPlasmidWithoutProp(createdby string) *stock.NewStock {
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

func TestEditStrain(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer func() {
		err := repo.ClearStocks()
		if err != nil {
			t.Fatalf("error in clearing stocks %s", err)
		}
	}()
	ns := newUpdatableTestStrain("todd@gagg.com")
	m, err := repo.AddStrain(ns)
	if err != nil {
		t.Fatalf("error in adding strain: %s", err)
	}
	us := &stock.StockUpdate{
		Data: &stock.StockUpdate_Data{
			Type: ns.Data.Type,
			Id:   m.StockID,
			Attributes: &stock.StockUpdateAttributes{
				UpdatedBy:       "kirby@snes.org",
				Summary:         "updated strain",
				EditableSummary: "updated strain",
				Genes:           []string{"DDB_G120987", "DDB_G45098234"},
				Dbxrefs:         []string{"FGBD9493483", "4536935", "d2578", "d0319"},
				StrainProperties: &stock.StrainUpdateProperties{
					Label:   "Ax3-pspD/lacZ",
					Plasmid: "DBP0398713",
					Names:   []string{"SP87", "AX3-PL3/gal", "AX3PL31"},
				},
			},
		},
	}
	um, err := repo.EditStrain(us)
	if err != nil {
		t.Fatalf("error in updating strain id %s: %s", m.StockID, err)
	}
	assert := assert.New(t)
	assert.Equal(um.StockID, m.StockID, "should match the stock id")
	assert.Equal(um.UpdatedBy, us.Data.Attributes.UpdatedBy, "should match updatedby")
	assert.Equal(um.Depositor, m.Depositor, "depositor name should not be updated")
	assert.Equal(um.Summary, us.Data.Attributes.Summary, "should have updated summary")
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
		"publications list should remains unchanged",
	)
	assert.Equal(
		um.StrainProperties.SystematicName,
		m.StrainProperties.SystematicName,
		"systematic name should remains unchanged",
	)
	assert.Equal(
		um.StrainProperties.Species,
		m.StrainProperties.Species,
		"species name should remains unchanged",
	)
	assert.Equal(
		um.StrainProperties.Label,
		us.Data.Attributes.StrainProperties.Label,
		"should have updated strain descriptor",
	)
	assert.Equal(
		um.StrainProperties.Plasmid,
		us.Data.Attributes.StrainProperties.Plasmid,
		"should have updated plasmid name",
	)
	assert.ElementsMatch(
		um.StrainProperties.Names,
		us.Data.Attributes.StrainProperties.Names,
		"should updated list of strain names",
	)

	// test by adding parent strain
	pm, err := repo.AddStrain(newTestParentStrain("tim@watley.org"))
	if err != nil {
		t.Fatalf("error in adding parent strain %s", err)
	}
	us2 := &stock.StockUpdate{
		Data: &stock.StockUpdate_Data{
			Type: ns.Data.Type,
			Id:   um.StockID,
			Attributes: &stock.StockUpdateAttributes{
				UpdatedBy: "mario@snes.org",
				StrainProperties: &stock.StrainUpdateProperties{
					Parent: pm.StockID,
				},
			},
		},
	}
	um2, err := repo.EditStrain(us2)
	if err != nil {
		t.Fatalf("error in updating strain id %s: %s", um.StockID, err)
	}
	assert.Equal(um2.StockID, um.StockID, "should match their id")
	assert.Equal(um2.Depositor, m.Depositor, "depositor name should not be updated")
	assert.Equal(um2.CreatedBy, m.CreatedBy, "created by should not be updated")
	assert.Equal(um2.Summary, um.Summary, "summary should not be updated")
	assert.ElementsMatch(
		um2.Publications,
		m.Publications,
		"publications list should remains unchanged",
	)
	assert.ElementsMatch(
		um2.Genes,
		um.Genes,
		"genes list should not be updated",
	)
	assert.ElementsMatch(
		um2.Dbxrefs,
		um.Dbxrefs,
		"dbxrefs list should not be updated",
	)
	assert.Equal(
		um2.StrainProperties.Label,
		um.StrainProperties.Label,
		"strain descriptor should not be updated",
	)
	assert.ElementsMatch(
		um2.StrainProperties.Names,
		um.StrainProperties.Names,
		"strain names should not be updated",
	)
	assert.Equal(
		um2.StrainProperties.Plasmid,
		um.StrainProperties.Plasmid,
		"plasmid should not be updated",
	)
	assert.Equal(
		um2.StrainProperties.Parent,
		us2.Data.Attributes.StrainProperties.Parent,
		"should have updated parent",
	)
}

func TestAddStrain(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer func() {
		err := repo.ClearStocks()
		if err != nil {
			t.Fatalf("error in clearing stocks %s", err)
		}
	}()
	nsp := newTestParentStrain("todd@gagg.com")
	m, err := repo.AddStrain(nsp)
	if err != nil {
		t.Fatalf("error in adding strain: %s", err)
	}
	assert := assert.New(t)
	assert.Regexp(regexp.MustCompile(`^DBS0\d{6,}$`), m.StockID, "should have a stock id")
	assert.Equal(m.Key, m.StockID, "should have identical key and stock ID")
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
	assert.Empty(m.StrainProperties.Parent, "should not have any parent")

	ns := newTestStrain("pennypacker@penny.com")
	ns.Data.Attributes.StrainProperties.Parent = m.StockID
	m2, err := repo.AddStrain(ns)
	if err != nil {
		t.Fatalf("error in adding strain: %s", err)
	}
	assert.Regexp(regexp.MustCompile(`^DBS0\d{6,}$`), m2.StockID, "should have a stock id")
	assert.Equal(m2.Key, m2.StockID, "should have identical key and stock ID")
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
	assert.Equal(
		m2.StrainProperties.Parent,
		ns.Data.Attributes.StrainProperties.Parent,
		"should match parent entry",
	)
}

func TestAddPlasmidWithoutProp(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer func() {
		err := repo.ClearStocks()
		if err != nil {
			t.Fatalf("error in clearing stocks %s", err)
		}
	}()
	ns := newTestPlasmidWithoutProp("george@costanza.com")
	m, err := repo.AddPlasmid(ns)
	if err != nil {
		t.Fatalf("error in adding plasmid: %s", err)
	}
	assert := assert.New(t)
	assert.Regexp(regexp.MustCompile(`^DBP0\d{6,}$`), m.StockID, "should have a plasmid stock id")
	assert.Equal(m.Key, m.StockID, "should have identical key and stock ID")
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
	defer func() {
		err := repo.ClearStocks()
		if err != nil {
			t.Fatalf("error in clearing stocks %s", err)
		}
	}()
	ns := newTestPlasmid("george@costanza.com")
	m, err := repo.AddPlasmid(ns)
	if err != nil {
		t.Fatalf("error in adding plasmid: %s", err)
	}
	assert := assert.New(t)
	assert.Regexp(regexp.MustCompile(`^DBP0\d{6,}$`), m.StockID, "should have a plasmid stock id")
	assert.Equal(m.Key, m.StockID, "should have identical key and stock ID")
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
	defer func() {
		err := repo.ClearStocks()
		if err != nil {
			t.Fatalf("error in clearing stocks %s", err)
		}
	}()
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

func TestGetStrain(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer func() {
		err := repo.ClearStocks()
		if err != nil {
			t.Fatalf("error in clearing stocks %s", err)
		}
	}()
	ns := newTestStrain("george@costanza.com")
	m, err := repo.AddStrain(ns)
	if err != nil {
		t.Fatalf("error in adding strain %s", err)
	}
	g, err := repo.GetStrain(m.StockID)
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
	assert.Equal(g.StrainProperties.Parent, ns.Data.Attributes.StrainProperties.Parent, "should match parent")
	assert.ElementsMatch(g.StrainProperties.Names, ns.Data.Attributes.StrainProperties.Names, "should match names")
	assert.ElementsMatch(g.Genes, ns.Data.Attributes.Genes, "should match genes")
	assert.Equal(g.StrainProperties.Plasmid, ns.Data.Attributes.StrainProperties.Plasmid, "should match plasmid")
	assert.Equal(len(g.Dbxrefs), 6, "should match length of six dbxrefs")
	assert.NotEmpty(g.Key, "should not have empty key")
	assert.True(m.CreatedAt.Equal(g.CreatedAt), "should match created time of stock")
	assert.True(m.UpdatedAt.Equal(g.UpdatedAt), "should match updated time of stock")

	ne, err := repo.GetStrain("DBS01")
	if err != nil {
		t.Fatalf(
			"error in fetching stock %s with ID %s",
			"DBS01",
			err,
		)
	}
	assert.True(ne.NotFound, "entry should not exist")
}

func TestGetPlasmid(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer func() {
		err := repo.ClearStocks()
		if err != nil {
			t.Fatalf("error in clearing stocks %s", err)
		}
	}()
	ns := newTestPlasmid("george@costanza.com")
	m, err := repo.AddPlasmid(ns)
	if err != nil {
		t.Fatalf("error in adding strain %s", err)
	}
	g, err := repo.GetPlasmid(m.StockID)
	if err != nil {
		t.Fatalf("error in getting stock %s with ID %s", m.StockID, err)
	}
	assert := assert.New(t)
	assert.Equal(g.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(g.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(g.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(g.EditableSummary, ns.Data.Attributes.EditableSummary, "should match editable_summary")
	assert.Equal(g.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
	assert.Equal(g.PlasmidProperties.ImageMap, ns.Data.Attributes.PlasmidProperties.ImageMap, "should match image_map")
	assert.Equal(g.PlasmidProperties.Sequence, ns.Data.Attributes.PlasmidProperties.Sequence, "should match sequence")
	assert.NotEmpty(g.Key, "should not have empty key")
	assert.True(m.CreatedAt.Equal(g.CreatedAt), "should match created time of stock")
	assert.True(m.UpdatedAt.Equal(g.UpdatedAt), "should match updated time of stock")

	ne, err := repo.GetPlasmid("DBP01")
	if err != nil {
		t.Fatalf(
			"error in fetching stock %s with ID %s",
			"DBP01",
			err,
		)
	}
	assert.True(ne.NotFound, "entry should not exist")
}

func TestRemoveStock(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to stock repository %s", err)
	}
	defer func() {
		err := repo.ClearStocks()
		if err != nil {
			t.Fatalf("error in clearing stocks %s", err)
		}
	}()
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
	ne, err := repo.GetStrain(m.Key)
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
