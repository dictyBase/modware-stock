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
		KeyOffset:          370000,
	}
}

func newUpdatableTestStrain(createdby string) *stock.NewStrain {
	return &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type: "strain",
			Attributes: &stock.NewStrainAttributes{
				CreatedBy:       createdby,
				UpdatedBy:       createdby,
				Depositor:       createdby,
				Summary:         "Radiation-sensitive mutant.",
				EditableSummary: "Radiation-sensitive mutant.",
				Genes:           []string{"DDB_G0348394", "DDB_G098058933"},
				Publications:    []string{"48428304983", "83943", "839434936743"},
				Label:           "yS13",
				Species:         "Dictyostelium discoideum",
			},
		},
	}
}

func newTestStrain(createdby string) *stock.NewStrain {
	return &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type: "strain",
			Attributes: &stock.NewStrainAttributes{
				CreatedBy:       createdby,
				UpdatedBy:       createdby,
				Depositor:       "george@costanza.com",
				Summary:         "Radiation-sensitive mutant.",
				EditableSummary: "Radiation-sensitive mutant.",
				Dbxrefs:         []string{"5466867", "4536935", "d2578", "d0319", "d2020/1033268", "d2580"},
				Genes:           []string{"DDB_G0348394", "DDB_G098058933"},
				Publications:    []string{"4849343943", "48394394"},
				Label:           "yS13",
				Species:         "Dictyostelium discoideum",
				Plasmid:         "DBP0000027",
				Names:           []string{"gammaS13", "gammaS-13", "γS-13"},
			},
		},
	}
}

func newTestParentStrain(createdby string) *stock.NewStrain {
	return &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type: "strain",
			Attributes: &stock.NewStrainAttributes{
				CreatedBy:       createdby,
				UpdatedBy:       createdby,
				Depositor:       createdby,
				Summary:         "Remi-mutant strain",
				EditableSummary: "Remi-mutant strain.",
				Dbxrefs:         []string{"5466867", "4536935", "d2578"},
				Label:           "egeB/DDB_G0270724_ps-REMI",
				Species:         "Dictyostelium discoideum",
				Names:           []string{"gammaS13", "BCN149086"},
			},
		},
	}
}

func newUpdatableTestPlasmid(createdby string) *stock.NewPlasmid {
	return &stock.NewPlasmid{
		Data: &stock.NewPlasmid_Data{
			Type: "plasmid",
			Attributes: &stock.NewPlasmidAttributes{
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

func newTestPlasmid(createdby string) *stock.NewPlasmid {
	return &stock.NewPlasmid{
		Data: &stock.NewPlasmid_Data{
			Type: "plasmid",
			Attributes: &stock.NewPlasmidAttributes{
				CreatedBy:       createdby,
				UpdatedBy:       createdby,
				Depositor:       "george@costanza.com",
				Summary:         "this is a test plasmid",
				EditableSummary: "this is a test plasmid",
				Publications:    []string{"1348970"},
				ImageMap:        "http://dictybase.org/data/plasmid/images/87.jpg",
				Sequence:        "tttttyyyyjkausadaaaavvvvvv",
				Name:            "p123456",
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
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
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
				Dbxrefs:         []string{"FGBD9493483", "4536935", "d2578", "d0319"},
				Label:           "Ax3-pspD/lacZ",
				Plasmid:         "DBP0398713",
				Names:           []string{"SP87", "AX3-PL3/gal", "AX3PL31"},
			},
		},
	}
	um, err := repo.EditStrain(us)
	assert.NoErrorf(err, "expect no error, received %s", err)
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

	// test by adding parent strain
	pm, err := repo.AddStrain(newTestParentStrain("tim@watley.org"))
	assert.NoErrorf(err, "expect no error, received %s", err)
	us2 := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: ns.Data.Type,
			Id:   um.StockID,
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
		us2.Data.Attributes.Parent,
		"should have updated parent",
	)
	assert.Equal(um2.StrainProperties.Species, us2.Data.Attributes.Species, "species should be updated")

	// add another new strain, let's make this one a parent
	// so we can test updating parent if one already exists
	pu, err := repo.AddStrain(newUpdatableTestStrain("castle@vania.org"))
	assert.NoErrorf(err, "expect no error, received %s", err)
	us3 := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: ns.Data.Type,
			Id:   um.StockID,
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
	assert.Equal(um.StockID, um3.StockID, "should have same stock ID")
	assert.Equal(um2.StrainProperties.Plasmid, um3.StrainProperties.Plasmid, "plasmid should not have been updated")
}

func TestAddStrain(t *testing.T) {
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
	nsp := newTestParentStrain("todd@gagg.com")
	m, err := repo.AddStrain(nsp)
	assert.NoErrorf(err, "expect no error, received %s", err)
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
	assert.Equal(m.StrainProperties.Label, nsp.Data.Attributes.Label, "should match descriptor")
	assert.Equal(m.StrainProperties.Species, nsp.Data.Attributes.Species, "should match species")
	assert.ElementsMatch(m.StrainProperties.Names, nsp.Data.Attributes.Names, "should match names")
	assert.Empty(m.StrainProperties.Plasmid, "should not have any plasmid")
	assert.Empty(m.StrainProperties.Parent, "should not have any parent")

	ns := newTestStrain("pennypacker@penny.com")
	ns.Data.Attributes.Parent = m.StockID
	m2, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
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

func TestAddPlasmid(t *testing.T) {
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
	ns := newTestPlasmid("george@costanza.com")
	m, err := repo.AddPlasmid(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
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
		ns.Data.Attributes.ImageMap,
		"should match image_map",
	)
	assert.Equal(
		m.PlasmidProperties.Sequence,
		ns.Data.Attributes.Sequence,
		"should match sequence",
	)
	assert.Equal(m.PlasmidProperties.Name, ns.Data.Attributes.Name, "should match name")
}

func TestEditPlasmid(t *testing.T) {
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
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
	assert.Equal(um.PlasmidProperties.ImageMap, us.Data.Attributes.ImageMap, "should match image map")
	assert.Equal(um.PlasmidProperties.Name, us.Data.Attributes.Name, "should match name")
	us2 := &stock.PlasmidUpdate{
		Data: &stock.PlasmidUpdate_Data{
			Type: ns.Data.Type,
			Id:   um.StockID,
			Attributes: &stock.PlasmidUpdateAttributes{
				UpdatedBy: "puddy@seinfeld.org",
				Genes:     []string{"DDB_G0270851", "DDB_G02748", "DDB_G7392222"},
				Sequence:  "atgctagagaagacttt",
			},
		},
	}
	um2, err := repo.EditPlasmid(us2)
	assert.NoErrorf(err, "expect no error, received %s", err)
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
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
	ns := newTestStrain("george@costanza.com")
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	g, err := repo.GetStrain(m.StockID)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Regexp(regexp.MustCompile(`^DBS0\d{6,}$`), g.StockID, "should have a strain stock id")
	assert.Equal(g.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(g.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(g.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(g.EditableSummary, ns.Data.Attributes.EditableSummary, "should match editable_summary")
	assert.Equal(g.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
	assert.ElementsMatch(g.Dbxrefs, ns.Data.Attributes.Dbxrefs, "should match dbxrefs")
	assert.ElementsMatch(g.StrainProperties.Names, ns.Data.Attributes.Names, "should match names")
	assert.ElementsMatch(g.Genes, ns.Data.Attributes.Genes, "should match genes")
	assert.Equal(g.StrainProperties.Label, ns.Data.Attributes.Label, "should match descriptor")
	assert.Equal(g.StrainProperties.Species, ns.Data.Attributes.Species, "should match species")
	assert.Equal(g.StrainProperties.Plasmid, ns.Data.Attributes.Plasmid, "should match plasmid")
	assert.Empty(g.StrainProperties.Parent, "should not have parent")
	assert.Len(g.Dbxrefs, 6, "should match length of six dbxrefs")
	assert.True(m.CreatedAt.Equal(g.CreatedAt), "should match created time of stock")
	assert.True(m.UpdatedAt.Equal(g.UpdatedAt), "should match updated time of stock")

	ns2 := newTestStrain("dead@cells.com")
	ns2.Data.Attributes.Parent = m.StockID
	m2, err := repo.AddStrain(ns2)
	assert.NoErrorf(err, "expect no error, received %s", err)
	g2, err := repo.GetStrain(m2.StockID)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Equal(g2.StrainProperties.Parent, m.StockID, "should match parent")
	ne, err := repo.GetStrain("DBS01")
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.True(ne.NotFound, "entry should not exist")
}

func TestGetPlasmid(t *testing.T) {
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
	ns := newTestPlasmid("george@costanza.com")
	m, err := repo.AddPlasmid(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	g, err := repo.GetPlasmid(m.StockID)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Regexp(regexp.MustCompile(`^DBP0\d{6,}$`), g.StockID, "should have a plasmid stock id")
	assert.Equal(g.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(g.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(g.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
	assert.Equal(g.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(g.EditableSummary, ns.Data.Attributes.EditableSummary, "should match editable_summary")
	assert.Equal(g.PlasmidProperties.ImageMap, ns.Data.Attributes.ImageMap, "should match image_map")
	assert.Equal(g.PlasmidProperties.Sequence, ns.Data.Attributes.Sequence, "should match sequence")
	assert.Equal(g.PlasmidProperties.Name, ns.Data.Attributes.Name, "should match name")
	assert.True(m.CreatedAt.Equal(g.CreatedAt), "should match created time of stock")
	assert.True(m.UpdatedAt.Equal(g.UpdatedAt), "should match updated time of stock")

	ne, err := repo.GetPlasmid("DBP01")
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.True(ne.NotFound, "entry should not exist")
}

func TestListStrains(t *testing.T) {
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
	// add 10 new test strains
	for i := 1; i <= 10; i++ {
		ns := newTestStrain(fmt.Sprintf("%s@kramericaindustries.com", RandString(10)))
		_, err := repo.AddStrain(ns)
		assert.NoErrorf(err, "expect no error adding strain, received %s", err)
	}
	// get first five results
	ls, err := repo.ListStrains(&stock.StockParameters{Limit: 4})
	assert.NoErrorf(err, "expect no error in getting first five stocks, received %s", err)
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
	assert.NoErrorf(err, "expect no error in getting stocks 5-9, received %s", err)
	assert.Len(ls2, 5, "should match the provided limit number + 1")
	assert.Exactly(ls2[0], ls[len(ls)-1], "last item from first five results and first item from next five results should be the same")
	assert.NotEqual(ls2[0].CreatedBy, ls2[1].CreatedBy, "should have different created_by fields")

	// convert ninth result to numeric timestamp
	ti2 := toTimestamp(ls2[len(ls2)-1].CreatedAt)
	// get last results (9-10)
	ls3, err := repo.ListStrains(&stock.StockParameters{Cursor: ti2, Limit: 4})
	assert.NoErrorf(err, "expect no error in getting stocks 9-10, received %s", err)
	assert.Len(ls3, 2, "should retrieve the last two results")
	assert.Exactly(ls3[0], ls2[len(ls2)-1], "last item from previous five results and first item from next five results should be the same")

	// sort all of the results
	testModelListSort(ls, t)
	testModelListSort(ls2, t)
	testModelListSort(ls3, t)

	filter := `FILTER s.depositor == 'george@costanza.com'`
	sf, err := repo.ListStrains(&stock.StockParameters{Limit: 10, Filter: filter})
	assert.NoErrorf(err, "expect no error in getting list of strains with depositor george@costanza.com, received %s", err)
	assert.Len(sf, 10, "should list ten strains")

	cs, err := repo.ListStrains(&stock.StockParameters{Cursor: toTimestamp(sf[4].CreatedAt), Limit: 10})
	assert.NoErrorf(err, "expect no error getting list of strains with cursor, received %s", err)
	assert.Len(cs, 6, "should list six strains")
}

func TestListStrainsWithFilter(t *testing.T) {
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
	// add 10 new test strains
	for i := 1; i <= 10; i++ {
		ns := newTestStrain(fmt.Sprintf("%s@kramericaindustries.com", RandString(10)))
		_, err := repo.AddStrain(ns)
		assert.NoErrorf(err, "expect no error adding strain, received %s", err)
	}
	filterOne := `FILTER s.depositor == 'george@costanza.com'`
	sf, err := repo.ListStrains(&stock.StockParameters{Limit: 10, Filter: filterOne})
	assert.NoErrorf(err, "expect no error getting list of strains, received %s", err)
	assert.Len(sf, 10, "should list ten strains")
	for _, m := range sf {
		assert.Equal(m.Summary, "Radiation-sensitive mutant.", "should match summary")
		assert.Equal(m.StrainProperties.Label, "yS13", "should match label")
	}

	filterTwo := `FILTER s.depositor == 'george@costanza.com' AND s.depositor == 'rg@gmail.com'`
	n, err := repo.ListStrains(&stock.StockParameters{Limit: 100, Filter: filterTwo})
	assert.NoErrorf(err, "expect no error getting list of stocks with two depositors with AND logic, received %s", err)
	assert.Len(n, 0, "should list no strains")

	filterThree := `LET x = (
						FILTER 'gammaS13' IN s.names 
						RETURN 1
					)`
	// do a check for array filter
	as, err := repo.ListStrains(&stock.StockParameters{Cursor: toTimestamp(sf[5].CreatedAt), Limit: 10, Filter: filterThree})
	assert.NoErrorf(err, "expect no error getting list of stocks with cursor and filter, received %s", err)
	assert.Len(as, 5, "should list five strains")

	filterFour := `FILTER s.created_at <= DATE_ISO8601('2019')`
	da, err := repo.ListStrains(&stock.StockParameters{Cursor: toTimestamp(sf[5].CreatedAt), Limit: 10, Filter: filterFour})
	assert.NoErrorf(err, "expect no error getting list of stocks with cursor and date filter, received %s", err)
	assert.Len(da, 0, "should list no strains")

	filterFive := `FILTER v.label =~ 'yS'`
	ff, err := repo.ListStrains(&stock.StockParameters{Limit: 10, Filter: filterFive})
	assert.NoErrorf(err, "expect no error getting list of strains with label substring, received %s", err)
	assert.Len(ff, 10, "should list ten strains")

	filterSix := `FILTER s.summary =~ 'mutant'`
	fs, err := repo.ListStrains(&stock.StockParameters{Limit: 10, Filter: filterSix})
	assert.NoErrorf(err, "expect no error getting list of strains with summary substring, received %s", err)
	assert.Len(fs, 10, "should list ten strains")
}

func TestListPlasmids(t *testing.T) {
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
	// add ten new test plasmids
	for i := 1; i <= 10; i++ {
		np := newTestPlasmid(fmt.Sprintf("%s@cye.com", RandString(10)))
		_, err := repo.AddPlasmid(np)
		assert.NoErrorf(err, "expect no error adding plasmid, received %s", err)
	}
	// get first five results
	ls, err := repo.ListPlasmids(&stock.StockParameters{Limit: 4})
	assert.NoErrorf(err, "expect no error getting first five plasmids, received %s", err)
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
	assert.NoErrorf(err, "expect no error getting plasmids 5-9, received %s", err)
	assert.Len(ls2, 5, "should match the provided limit number + 1")
	assert.Exactly(ls2[0], ls[len(ls)-1], "last item from first five results and first item from next five results should be the same")
	assert.NotEqual(ls2[0].CreatedBy, ls2[1].CreatedBy, "should have different created_by fields")

	// convert ninth result to numeric timestamp
	ti2 := toTimestamp(ls2[len(ls2)-1].CreatedAt)
	// get last results (9-10)
	ls3, err := repo.ListPlasmids(&stock.StockParameters{Cursor: ti2, Limit: 4})
	assert.NoErrorf(err, "expect no error getting plasmids 9-10, received %s", err)
	assert.Len(ls3, 2, "should retrieve the last two results")
	assert.Exactly(ls3[0].CreatedBy, ls2[len(ls2)-1].CreatedBy, "last item from previous five results and first item from next five results should be the same")

	testModelListSort(ls, t)
	testModelListSort(ls2, t)
	testModelListSort(ls3, t)

	filter := `FILTER s.depositor == 'george@costanza.com'`
	sf, err := repo.ListPlasmids(&stock.StockParameters{Limit: 100, Filter: filter})
	assert.NoErrorf(err, "expect no error getting list of plasmids, received %s", err)
	assert.Len(sf, 10, "should list ten plasmids")

	cs, err := repo.ListPlasmids(&stock.StockParameters{Cursor: toTimestamp(sf[4].CreatedAt), Limit: 10})
	assert.NoErrorf(err, "expect no error getting list of plasmids with cursor, received %s", err)
	assert.Len(cs, 6, "should list six plasmids")
}

func TestListPlasmidsWithFilter(t *testing.T) {
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
	// add 10 new test plasmids
	for i := 1; i <= 10; i++ {
		np := newTestPlasmid(fmt.Sprintf("%s@cye.com", RandString(10)))
		_, err := repo.AddPlasmid(np)
		assert.NoErrorf(err, "expect no error, received %s", err)
	}
	filterOne := `FILTER s.depositor == 'george@costanza.com'`
	sf, err := repo.ListPlasmids(&stock.StockParameters{Limit: 10, Filter: filterOne})
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(sf, 10, "should list ten plasmids")
	for _, m := range sf {
		assert.Equal(m.Summary, "this is a test plasmid", "should match summary")
		assert.Equal(m.PlasmidProperties.Name, "p123456", "should match name")
	}

	filterTwo := `FILTER s.depositor == 'george@costanza.com' AND s.depositor == 'rg@gmail.com'`
	n, err := repo.ListPlasmids(&stock.StockParameters{Limit: 100, Filter: filterTwo})
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(n, 0, "should list no plasmids")

	filterThree := `LET x = (
						FILTER '1348970' IN s.publications 
						RETURN 1
					)`
	// do a check for array filter
	as, err := repo.ListPlasmids(&stock.StockParameters{Cursor: toTimestamp(sf[5].CreatedAt), Limit: 10, Filter: filterThree})
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(as, 5, "should list five plasmids")

	filterFour := `FILTER s.created_at <= DATE_ISO8601('2019')`
	da, err := repo.ListPlasmids(&stock.StockParameters{Cursor: toTimestamp(sf[5].CreatedAt), Limit: 10, Filter: filterFour})
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(da, 0, "should list no plasmids")

	filterFive := `FILTER v.sequence =~ 'ttttt'`
	ff, err := repo.ListPlasmids(&stock.StockParameters{Limit: 10, Filter: filterFive})
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(ff, 10, "should list ten plasmids")

	filterSix := `FILTER s.summary =~ 'test'`
	fs, err := repo.ListPlasmids(&stock.StockParameters{Limit: 10, Filter: filterSix})
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(fs, 10, "should list ten plasmids")

	filterSeven := `FILTER s.depositor == 'george@costanza.com' OR v.name == 'gammaS13'`
	fv, err := repo.ListPlasmids(&stock.StockParameters{Cursor: toTimestamp(sf[5].CreatedAt), Limit: 10, Filter: filterSeven})
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.Len(fv, 5, "should list five plasmids")
}

func TestRemoveStock(t *testing.T) {
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
	ns := newTestStrain("george@costanza.com")
	m, err := repo.AddStrain(ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
	err = repo.RemoveStock(m.Key)
	assert.NoErrorf(err, "expect no error, received %s", err)
	ne, err := repo.GetStrain(m.Key)
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.True(ne.NotFound, "entry should not exist")
	// try removing nonexistent stock
	e := repo.RemoveStock("xyz")
	assert.Error(e)
}

func TestLoadStockWithStrains(t *testing.T) {
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
	tm, _ := time.Parse("2006-01-02 15:04:05", "2010-03-30 14:40:58")
	nsp := &stock.ExistingStrain{
		Data: &stock.ExistingStrain_Data{
			Type: "strain",
			Id:   "DBS0252873",
			Attributes: &stock.ExistingStrainAttributes{
				CreatedAt:       aphgrpc.TimestampProto(tm),
				UpdatedAt:       aphgrpc.TimestampProto(tm),
				CreatedBy:       "wizard_of_loneliness@testemail.org",
				UpdatedBy:       "wizard_of_loneliness@testemail.org",
				Depositor:       "wizard_of_loneliness@testemail.org",
				Summary:         "Remi-mutant strain",
				EditableSummary: "Remi-mutant strain.",
				Dbxrefs:         []string{"5466867", "4536935", "d2578"},
				Label:           "egeB/DDB_G0270724_ps-REMI",
				Species:         "Dictyostelium discoideum",
				Names:           []string{"gammaS13", "BCN149086"},
			},
		},
	}
	m, err := repo.LoadStrain("DBS0252873", nsp)
	assert.NoErrorf(err, "expect no error, received %s", err)
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
	assert.Equal(m.StrainProperties.Label, nsp.Data.Attributes.Label, "should match descriptor")
	assert.Equal(m.StrainProperties.Species, nsp.Data.Attributes.Species, "should match species")
	assert.ElementsMatch(m.StrainProperties.Names, nsp.Data.Attributes.Names, "should match names")
	assert.Empty(m.StrainProperties.Plasmid, "should not have any plasmid")
	assert.Empty(m.StrainProperties.Parent, "should not have any parent")

	ns := &stock.ExistingStrain{
		Data: &stock.ExistingStrain_Data{
			Type: "strain",
			Id:   "DBS0252873",
			Attributes: &stock.ExistingStrainAttributes{
				CreatedAt:       aphgrpc.TimestampProto(tm),
				UpdatedAt:       aphgrpc.TimestampProto(tm),
				CreatedBy:       "wizard_of_loneliness@testemail.org",
				UpdatedBy:       "wizard_of_loneliness@testemail.org",
				Depositor:       "wizard_of_loneliness@testemail.org",
				Summary:         "Remi-mutant strain",
				EditableSummary: "Remi-mutant strain.",
				Dbxrefs:         []string{"5466867", "4536935", "d2578"},
				Label:           "egeB/DDB_G0270724_ps-REMI",
				Species:         "Dictyostelium discoideum",
				Names:           []string{"gammaS13", "BCN149086"},
			},
		},
	}
	ns.Data.Attributes.Parent = m.StockID
	m2, err := repo.LoadStrain("DBS0235412", ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
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

func TestLoadStockWithPlasmids(t *testing.T) {
	assert := assert.New(t)
	connP := getConnectParams()
	collP := getCollectionParams()
	repo, err := NewStockRepo(connP, collP)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	defer func() {
		err := repo.ClearStocks()
		assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
	}()
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
	m, err := repo.LoadPlasmid("DBP0000098", ns)
	assert.NoErrorf(err, "expect no error, received %s", err)
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
		ns.Data.Attributes.ImageMap,
		"should match image_map",
	)
	assert.Equal(
		m.PlasmidProperties.Sequence,
		ns.Data.Attributes.Sequence,
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
