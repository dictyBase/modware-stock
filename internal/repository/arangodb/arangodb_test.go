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
	"github.com/dictyBase/apihelpers/aphgrpc"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/query"
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
				Depositor:       "george@costanza.com",
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
				Depositor:       "george@costanza.com",
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
		"publications list should remain unchanged",
	)
	assert.Equal(
		um.StrainProperties.SystematicName,
		m.StrainProperties.SystematicName,
		"systematic name should remain unchanged",
	)
	assert.Equal(
		um.StrainProperties.Species,
		m.StrainProperties.Species,
		"species name should remain unchanged",
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
		"should have updated list of strain names",
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
				Depositor: "mario@snes.org",
				StrainProperties: &stock.StrainUpdateProperties{
					Parent:         pm.StockID,
					SystematicName: "y99",
					Species:        "updated species",
				},
			},
		},
	}
	um2, err := repo.EditStrain(us2)
	if err != nil {
		t.Fatalf("error in updating strain id %s: %s", um.StockID, err)
	}
	assert.Equal(um2.StockID, um.StockID, "should match their id")
	assert.Equal(um2.Depositor, us2.Data.Attributes.Depositor, "depositor name should be updated")
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
	assert.Equal(um2.StrainProperties.SystematicName, us2.Data.Attributes.StrainProperties.SystematicName, "systematic name should be updated")
	assert.Equal(um2.StrainProperties.Species, us2.Data.Attributes.StrainProperties.Species, "species should be updated")

	// add another new strain, let's make this one a parent
	// so we can test updating parent if one already exists
	pu, err := repo.AddStrain(newUpdatableTestStrain("castle@vania.org"))
	if err != nil {
		t.Fatalf("error in adding strain: %s", err)
	}
	us3 := &stock.StockUpdate{
		Data: &stock.StockUpdate_Data{
			Type: ns.Data.Type,
			Id:   um.StockID,
			Attributes: &stock.StockUpdateAttributes{
				UpdatedBy: "mario@snes.org",
				StrainProperties: &stock.StrainUpdateProperties{
					Parent: pu.StockID,
				},
			},
		},
	}
	um3, err := repo.EditStrain(us3)
	if err != nil {
		t.Fatalf("error in updating strain id %s: %s", um.StockID, err)
	}
	assert.Equal(
		um3.StrainProperties.Parent,
		us3.Data.Attributes.StrainProperties.Parent,
		"should have updated parent",
	)
	assert.Equal(um.StockID, um3.StockID, "should have same stock ID")
	assert.Equal(um2.StrainProperties.Plasmid, um3.StrainProperties.Plasmid, "plasmid should not have been updated")
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
	assert.Empty(m.PlasmidProperties, "should have empty plasmid properties field")
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
	ns := newUpdatableTestPlasmid("art@vandelay.org")
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
	assert.Equal(um.PlasmidProperties.ImageMap, us.Data.Attributes.PlasmidProperties.ImageMap, "should match image map")
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
		t.Fatalf("error in reupdating the plasmid %s", err)
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
	us3 := &stock.StockUpdate{
		Data: &stock.StockUpdate_Data{
			Type: ns.Data.Type,
			Id:   um.StockID,
			Attributes: &stock.StockUpdateAttributes{
				UpdatedBy:       "seven@costanza.org",
				Summary:         "this is an updated summary",
				EditableSummary: "this is an updated summary",
			},
		},
	}
	um3, err := repo.EditPlasmid(us3)
	if err != nil {
		t.Fatalf("error in reupdating the plasmid %s", err)
	}
	assert.Equal(um3.StockID, um.StockID, "should match the original stock id")
	assert.Equal(um3.UpdatedBy, us3.Data.Attributes.UpdatedBy, "should have updated the updatedby field")
	assert.Equal(um3.Summary, us3.Data.Attributes.Summary, "should have updated the summary field")
	assert.Equal(um3.EditableSummary, us3.Data.Attributes.EditableSummary, "should have updated the editable summary field")
	assert.ElementsMatch(
		um3.Dbxrefs,
		um2.Dbxrefs,
		"dbxrefs list should remain the same",
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
	assert.Regexp(regexp.MustCompile(`^DBS0\d{6,}$`), g.StockID, "should have a strain stock id")
	assert.Equal(g.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(g.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(g.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(g.EditableSummary, ns.Data.Attributes.EditableSummary, "should match editable_summary")
	assert.Equal(g.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
	assert.ElementsMatch(g.Dbxrefs, ns.Data.Attributes.Dbxrefs, "should match dbxrefs")
	assert.ElementsMatch(g.StrainProperties.Names, ns.Data.Attributes.StrainProperties.Names, "should match names")
	assert.ElementsMatch(g.Genes, ns.Data.Attributes.Genes, "should match genes")
	assert.Equal(g.StrainProperties.SystematicName, ns.Data.Attributes.StrainProperties.SystematicName, "should match systematic_name")
	assert.Equal(g.StrainProperties.Label, ns.Data.Attributes.StrainProperties.Label, "should match descriptor")
	assert.Equal(g.StrainProperties.Species, ns.Data.Attributes.StrainProperties.Species, "should match species")
	assert.Equal(g.StrainProperties.Plasmid, ns.Data.Attributes.StrainProperties.Plasmid, "should match plasmid")
	assert.Empty(g.StrainProperties.Parent, "should not have parent")
	assert.Len(g.Dbxrefs, 6, "should match length of six dbxrefs")
	assert.True(m.CreatedAt.Equal(g.CreatedAt), "should match created time of stock")
	assert.True(m.UpdatedAt.Equal(g.UpdatedAt), "should match updated time of stock")

	ns2 := newTestStrain("dead@cells.com")
	ns2.Data.Attributes.StrainProperties.Parent = m.StockID
	m2, err := repo.AddStrain(ns2)
	if err != nil {
		t.Fatalf("error in adding strain %s", err)
	}
	g2, err := repo.GetStrain(m2.StockID)
	if err != nil {
		t.Fatalf("error in getting stock %s with ID %s", m2.StockID, err)
	}
	assert.Equal(g2.StrainProperties.Parent, m.StockID, "should match parent")

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
	assert.Regexp(regexp.MustCompile(`^DBP0\d{6,}$`), g.StockID, "should have a plasmid stock id")
	assert.Equal(g.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(g.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(g.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
	assert.Equal(g.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(g.EditableSummary, ns.Data.Attributes.EditableSummary, "should match editable_summary")
	assert.Equal(g.PlasmidProperties.ImageMap, ns.Data.Attributes.PlasmidProperties.ImageMap, "should match image_map")
	assert.Equal(g.PlasmidProperties.Sequence, ns.Data.Attributes.PlasmidProperties.Sequence, "should match sequence")
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

func TestListStrains(t *testing.T) {
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
	// add 10 new test strains
	for i := 1; i <= 10; i++ {
		ns := newTestStrain(fmt.Sprintf("%s@kramericaindustries.com", RandString(10)))
		_, err := repo.AddStrain(ns)
		if err != nil {
			t.Fatalf("error in adding strain %s", err)
		}
	}
	// get first five results
	ls, err := repo.ListStrains(&stock.StockParameters{Limit: 4})
	if err != nil {
		t.Fatalf("error in getting first five stocks %s", err)
	}
	assert := assert.New(t)
	assert.Len(ls, 5, "should match the provided limit number + 1")

	for _, stock := range ls {
		assert.Equal(stock.Depositor, "george@costanza.com", "should match the depositor")
		assert.Equal(stock.Key, stock.StockID, "stock key and ID should match")
		assert.Regexp(regexp.MustCompile(`^DBS0\d{6,}$`), stock.StockID, "should have a strain stock id")
	}
	assert.NotEqual(ls[0].CreatedBy, ls[1].CreatedBy, "should have different created_by")
	// convert fifth result to numeric timestamp in milliseconds
	// so we can use this as cursor
	ti := toTimestamp(ls[4].CreatedAt)

	// get next five results (5-9)
	ls2, err := repo.ListStrains(&stock.StockParameters{Cursor: ti, Limit: 4})
	if err != nil {
		t.Fatalf("error in getting stocks 5-9 %s", err)
	}
	assert.Len(ls2, 5, "should match the provided limit number + 1")
	assert.Exactly(ls2[0], ls[len(ls)-1], "last item from first five results and first item from next five results should be the same")
	assert.NotEqual(ls2[0].CreatedBy, ls2[1].CreatedBy, "should have different created_by fields")

	// convert ninth result to numeric timestamp
	ti2 := toTimestamp(ls2[len(ls2)-1].CreatedAt)
	// get last results (9-10)
	ls3, err := repo.ListStrains(&stock.StockParameters{Cursor: ti2, Limit: 4})
	if err != nil {
		t.Fatalf("error in getting stocks 9-10 %s", err)
	}
	assert.Len(ls3, 2, "should retrieve the last two results")
	assert.Exactly(ls3[0], ls2[len(ls2)-1], "last item from previous five results and first item from next five results should be the same")

	// sort all of the results
	testModelListSort(ls, t)
	testModelListSort(ls2, t)
	testModelListSort(ls3, t)

	sf, err := repo.ListStrains(&stock.StockParameters{Limit: 10, Filter: convertFilterToQuery("depositor===george@costanza.com")})
	if err != nil {
		t.Fatalf("error in getting list of strains with depositor george@costanza.com %s", err)
	}
	assert.Len(sf, 10, "should list ten strains")

	n, err := repo.ListStrains(&stock.StockParameters{Limit: 100, Filter: convertFilterToQuery("depositor===george@costanza.com;depositor===rg@gmail.com")})
	if err != nil {
		t.Fatalf("error in getting list of stocks with two depositors with AND logic %s", err)
	}
	assert.Len(n, 0, "should list no strains")

	cs, err := repo.ListStrains(&stock.StockParameters{Cursor: toTimestamp(sf[4].CreatedAt), Limit: 10})
	if err != nil {
		t.Fatalf("error in getting list of strains with cursor %s", err)
	}
	assert.Len(cs, 6, "should list six strains")

	// do a check for array filter
	as, err := repo.ListStrains(&stock.StockParameters{Cursor: toTimestamp(sf[5].CreatedAt), Limit: 10, Filter: convertFilterToQuery("name@==gammaS13")})
	if err != nil {
		t.Fatalf("error in getting list of stocks with cursor and filter %s", err)
	}
	assert.Len(as, 5, "should list five strains")

	da, err := repo.ListStrains(&stock.StockParameters{Cursor: toTimestamp(sf[5].CreatedAt), Limit: 10, Filter: convertFilterToQuery("created_at$<=2019")})
	if err != nil {
		t.Fatalf("error in getting list of stocks with cursor and date filter %s", err)
	}
	assert.Len(da, 0, "should list no strains")
}

func TestListPlasmids(t *testing.T) {
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
	// add ten new test plasmids
	for i := 1; i <= 10; i++ {
		np := newTestPlasmid(fmt.Sprintf("%s@cye.com", RandString(10)))
		_, err := repo.AddPlasmid(np)
		if err != nil {
			t.Fatalf("error in adding plasmid %s", err)
		}
	}
	// get first five results
	ls, err := repo.ListPlasmids(&stock.StockParameters{Limit: 4})
	if err != nil {
		t.Fatalf("error in getting first five plasmids %s", err)
	}
	assert := assert.New(t)
	assert.Len(ls, 5, "should match the provided limit number + 1")

	for _, stock := range ls {
		assert.Equal(stock.Depositor, "george@costanza.com", "should match the depositor")
		assert.Equal(stock.Key, stock.StockID, "stock key and ID should match")
		assert.Regexp(regexp.MustCompile(`^DBP0\d{6,}$`), stock.StockID, "should have a plasmid stock id")
	}
	assert.NotEqual(ls[0].CreatedBy, ls[1].CreatedBy, "should have different created_by")
	// convert fifth result to numeric timestamp in milliseconds
	// so we can use this as cursor
	ti := toTimestamp(ls[len(ls)-1].CreatedAt)

	// get next five results (5-9)
	ls2, err := repo.ListPlasmids(&stock.StockParameters{Cursor: ti, Limit: 4})
	if err != nil {
		t.Fatalf("error in getting plasmids 5-9 %s", err)
	}
	assert.Len(ls2, 5, "should match the provided limit number + 1")
	assert.Exactly(ls2[0], ls[len(ls)-1], "last item from first five results and first item from next five results should be the same")
	assert.NotEqual(ls2[0].CreatedBy, ls2[1].CreatedBy, "should have different created_by fields")

	// convert ninth result to numeric timestamp
	ti2 := toTimestamp(ls2[len(ls2)-1].CreatedAt)
	// get last results (9-10)
	ls3, err := repo.ListPlasmids(&stock.StockParameters{Cursor: ti2, Limit: 4})
	if err != nil {
		t.Fatalf("error in getting plasmids 9-10 %s", err)
	}
	assert.Len(ls3, 2, "should retrieve the last two results")
	assert.Exactly(ls3[0].CreatedBy, ls2[len(ls2)-1].CreatedBy, "last item from previous five results and first item from next five results should be the same")

	testModelListSort(ls, t)
	testModelListSort(ls2, t)
	testModelListSort(ls3, t)

	sf, err := repo.ListPlasmids(&stock.StockParameters{Limit: 100, Filter: convertFilterToQuery("depositor===george@costanza.com")})
	if err != nil {
		t.Fatalf("error in getting list of plasmids with depositor george@costanza.com %s", err)
	}
	assert.Len(sf, 10, "should list ten plasmids")

	n, err := repo.ListPlasmids(&stock.StockParameters{Limit: 100, Filter: convertFilterToQuery("depositor===george@costanza.com;depositor===rg@gmail.com")})
	if err != nil {
		t.Fatalf("error in getting list of plasmids with two depositors with AND logic %s", err)
	}
	assert.Len(n, 0, "should list no plasmids")

	cs, err := repo.ListPlasmids(&stock.StockParameters{Cursor: toTimestamp(sf[4].CreatedAt), Limit: 10})
	if err != nil {
		t.Fatalf("error in getting list of plasmids with cursor %s", err)
	}
	assert.Len(cs, 6, "should list six plasmids")

	as, err := repo.ListPlasmids(&stock.StockParameters{Cursor: toTimestamp(sf[5].CreatedAt), Limit: 10, Filter: convertFilterToQuery("depositor===george@costanza.com,name@==gammaS13")})
	if err != nil {
		t.Fatalf("error in getting list of plasmids with cursor and OR filter %s", err)
	}
	assert.Len(as, 5, "should list five plasmids")

	da, err := repo.ListStrains(&stock.StockParameters{Cursor: toTimestamp(sf[5].CreatedAt), Limit: 10, Filter: convertFilterToQuery("created_at$<=2019")})
	if err != nil {
		t.Fatalf("error in getting list of stocks with cursor and date filter %s", err)
	}
	assert.Len(da, 0, "should list no plasmids")
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
	// try removing nonexistent stock
	e := repo.RemoveStock("xyz")
	assert.Error(e)
}

func TestLoadStockWithStrains(t *testing.T) {
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
	tm, _ := time.Parse("2006-01-02 15:04:05", "2010-03-30 14:40:58")
	nsp := &stock.ExistingStock{
		Data: &stock.ExistingStock_Data{
			Type: "strain",
			Id:   "DBS0252873",
			Attributes: &stock.ExistingStockAttributes{
				CreatedAt:       aphgrpc.TimestampProto(tm),
				UpdatedAt:       aphgrpc.TimestampProto(tm),
				CreatedBy:       "wizard_of_loneliness@testemail.org",
				UpdatedBy:       "wizard_of_loneliness@testemail.org",
				Depositor:       "wizard_of_loneliness@testemail.org",
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
	m, err := repo.LoadStock("DBS0252873", nsp)
	if err != nil {
		t.Fatalf("error in adding strain: %s", err)
	}
	assert := assert.New(t)
	assert.True(m.CreatedAt.Equal(tm), "should match created_at")
	assert.Equal("DBS0252873", m.StockID, "should match given stock id")
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

	ns := &stock.ExistingStock{
		Data: &stock.ExistingStock_Data{
			Type: "strain",
			Id:   "DBS0252873",
			Attributes: &stock.ExistingStockAttributes{
				CreatedAt:       aphgrpc.TimestampProto(tm),
				UpdatedAt:       aphgrpc.TimestampProto(tm),
				CreatedBy:       "wizard_of_loneliness@testemail.org",
				UpdatedBy:       "wizard_of_loneliness@testemail.org",
				Depositor:       "wizard_of_loneliness@testemail.org",
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
	ns.Data.Attributes.StrainProperties.Parent = m.StockID
	m2, err := repo.LoadStock("DBS0235412", ns)
	if err != nil {
		t.Fatalf("error in adding strain: %s", err)
	}
	assert.Equal("DBS0235412", m2.StockID, "should match given stock id")
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

func TestLoadStockWithPlasmids(t *testing.T) {
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
	tm, _ := time.Parse("2006-01-02 15:04:05", "2010-03-30 14:40:58")
	ns := &stock.ExistingStock{
		Data: &stock.ExistingStock_Data{
			Type: "plasmid",
			Id:   "DBP0000098",
			Attributes: &stock.ExistingStockAttributes{
				CreatedAt:       aphgrpc.TimestampProto(tm),
				UpdatedAt:       aphgrpc.TimestampProto(tm),
				CreatedBy:       "george@costanza.com",
				UpdatedBy:       "george@costanza.com",
				Depositor:       "george@costanza.com",
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
	m, err := repo.LoadStock("DBP0000098", ns)
	if err != nil {
		t.Fatalf("error in adding plasmid: %s", err)
	}
	assert := assert.New(t)
	assert.Equal("DBP0000098", m.StockID, "should match given plasmid stock id")
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

func convertFilterToQuery(s string) string {
	// parse filter logic
	// this needs to be done here since it is implemented in the service, not repository
	p, err := query.ParseFilterString(s)
	if err != nil {
		log.Printf("error parsing filter string %s", err)
		return s
	}
	str, err := query.GenAQLFilterStatement(&query.StatementParameters{Fmap: FMap, Filters: p, Doc: "s", Vert: "v"})
	if err != nil {
		log.Printf("error generating AQL filter statement %s", err)
		return s
	}
	return str
}
