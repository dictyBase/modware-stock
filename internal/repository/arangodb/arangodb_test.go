package arangodb

import (
	"log"
	"os"
	"testing"

	driver "github.com/arangodb/go-driver"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
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
		Strain:             "strain",
		Plasmid:            "plasmid",
		StockKeyGenerator:  "stock_key_generator",
		StockPlasmid:       "stock_plasmid",
		StockStrain:        "stock_strain",
		ParentStrain:       "parent_strain",
		Stock2PlasmidGraph: "stock2plasmid",
		Stock2StrainGraph:  "stock2strain",
		Strain2ParentGraph: "strain2parent",
		KeyOffset:          270000,
	}
}

func newTestStrain(createdby string) *stock.NewStock {
	return &stock.NewStock{
		Data: &stock.NewStock_Data{
			Type: "strain",
			Id:   "DBS0238532",
			Attributes: &stock.NewStockAttributes{
				CreatedBy:       createdby,
				UpdatedBy:       createdby,
				Summary:         "Radiation-sensitive mutant.",
				EditableSummary: "Radiation-sensitive mutant.",
				Depositor:       "Rob Guyer (Reg Deering)",
				Dbxrefs:         []string{"5466867", "4536935", "d2578", "d0319", "d2020/1033268", "d2580"},
				StrainProperties: &stock.StrainProperties{
					SystematicName: "yS13",
					Descriptor_:    "yS13",
					Species:        "Dictyostelium discoideum",
					Parents:        []string{"stock/NC4(DdB)"},
					Names:          []string{"gammaS13", "gammaS-13", "γS-13"},
				},
			},
		},
	}
}

func newTestPlasmid(createdby string) *stock.NewStock {
	return &stock.NewStock{
		Data: &stock.NewStock_Data{
			Type: "plasmid",
			Id:   "DBP0999999",
			Attributes: &stock.NewStockAttributes{
				CreatedBy:       createdby,
				UpdatedBy:       createdby,
				Summary:         "this is a test plasmid",
				EditableSummary: "this is a test plasmid",
				Depositor:       "george@costanza.com",
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
	ns := newTestStrain("george@costanza.com")
	m, err := repo.AddStrain(ns)
	if err != nil {
		t.Fatalf("error in adding strain: %s", err)
	}
	assert := assert.New(t)
	assert.Equal(m.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(m.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(m.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(m.EditableSummary, ns.Data.Attributes.EditableSummary, "should match editable_summary")
	assert.Equal(m.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
	assert.Equal(m.Dbxrefs, ns.Data.Attributes.Dbxrefs, "should match dbxrefs")
	assert.Equal(m.StrainProperties.SystematicName, ns.Data.Attributes.StrainProperties.SystematicName, "should match systematic_name")
	assert.Equal(m.StrainProperties.Descriptor, ns.Data.Attributes.StrainProperties.Descriptor_, "should match descriptor")
	assert.Equal(m.StrainProperties.Species, ns.Data.Attributes.StrainProperties.Species, "should match species")
	assert.Equal(m.StrainProperties.Parents, ns.Data.Attributes.StrainProperties.Parents, "should match parents")
	assert.Equal(m.StrainProperties.Names, ns.Data.Attributes.StrainProperties.Names, "should match names")
	assert.Empty(m.Genes, ns.Data.Attributes.Genes, "should have empty genes field")
	assert.Empty(m.StrainProperties.Plasmid, ns.Data.Attributes.StrainProperties.Plasmid, "should have empty plasmid field")
	assert.NotEmpty(m.Key, "should not have empty key")
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
	assert.Equal(m.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
	assert.Equal(m.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
	assert.Equal(m.Summary, ns.Data.Attributes.Summary, "should match summary")
	assert.Equal(m.EditableSummary, ns.Data.Attributes.EditableSummary, "should match editable_summary")
	assert.Equal(m.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
	assert.Equal(m.Publications, ns.Data.Attributes.Publications, "should match publications")
	assert.Equal(m.PlasmidProperties.ImageMap, ns.Data.Attributes.PlasmidProperties.ImageMap, "should match image_map")
	assert.Equal(m.PlasmidProperties.Sequence, ns.Data.Attributes.PlasmidProperties.Sequence, "should match sequence")
	assert.Empty(m.Genes, ns.Data.Attributes.Genes, "should have empty genes field")
	assert.NotEmpty(m.Key, "should not have empty key")
}

// func TestGetStock(t *testing.T) {
// 	connP := getConnectParams()
// 	collP := getCollectionParams()
// 	repo, err := NewStockRepo(connP, collP)
// 	if err != nil {
// 		t.Fatalf("error in connecting to stock repository %s", err)
// 	}
// 	defer repo.ClearStocks()
// 	ns := newTestStrain("george@costanza.com")
// 	m, err := repo.AddStrain(ns)
// 	if err != nil {
// 		t.Fatalf("error in adding strain %s", err)
// 	}
// 	g, err := repo.GetStock(m.StockID)
// 	if err != nil {
// 		t.Fatalf("error in getting stock %s with ID %s", m.StockID, err)
// 	}
// 	assert := assert.New(t)
// 	assert.Equal(g.CreatedBy, ns.Data.Attributes.CreatedBy, "should match created_by id")
// 	assert.Equal(g.UpdatedBy, ns.Data.Attributes.UpdatedBy, "should match updated_by id")
// 	assert.Equal(g.Summary, ns.Data.Attributes.Summary, "should match summary")
// 	assert.Equal(g.EditableSummary, ns.Data.Attributes.EditableSummary, "should match editable_summary")
// 	assert.Equal(g.Depositor, ns.Data.Attributes.Depositor, "should match depositor")
// 	assert.Equal(g.Dbxrefs, ns.Data.Attributes.Dbxrefs, "should match dbxrefs")
// assert.Equal(g.SystematicName, ns.Data.Attributes.StrainProperties.SystematicName, "should match systematic_name")
// assert.Equal(g.Descriptor, ns.Data.Attributes.StrainProperties.Descriptor_, "should match descriptor")
// assert.Equal(g.Species, ns.Data.Attributes.StrainProperties.Species, "should match species")
// assert.Equal(g.Parents, ns.Data.Attributes.StrainProperties.Parents, "should match parents")
// assert.Equal(g.Names, ns.Data.Attributes.StrainProperties.Names, "should match names")
// assert.Empty(g.Genes, ns.Data.Attributes.Genes, "should have empty genes field")
// assert.Empty(g.Plasmid, ns.Data.Attributes.StrainProperties.Plasmid, "should have empty plasmid field")
// assert.Equal(len(g.Dbxrefs), 6, "should match length of six dbxrefs")
// assert.NotEmpty(g.Key, "should not have empty key")
// assert.True(m.CreatedAt.Equal(g.CreatedAt), "should match created time of stock")
// assert.True(m.UpdatedAt.Equal(g.UpdatedAt), "should match updated time of stock")
//
// ne, err := repo.GetStock("1")
// if err != nil {
// 	t.Fatalf(
// 		"error in fetching stock %s with ID %s",
// 		"1",
// 		err,
// 	)
// }
// assert.True(ne.NotFound, "entry should not exist")
// }

// func TestEditStock(t *testing.T) {

// }

// func TestListStocks(t *testing.T) {

// }

// func TestListStrains(t *testing.T) {

// }

// func TestListPlasmids(t *testing.T) {

// }

// func TestRemoveStock(t *testing.T) {
// 	connP := getConnectParams()
// 	collP := getCollectionParams()
// 	repo, err := NewStockRepo(connP, collP)
// 	if err != nil {
// 		t.Fatalf("error in connecting to stock repository %s", err)
// 	}
// 	defer repo.ClearStocks()
// 	ns := newTestStrain("george@costanza.com")
// 	m, err := repo.AddStrain(ns)
// 	if err != nil {
// 		t.Fatalf("error in adding strain: %s", err)
// 	}
// 	err = repo.RemoveStock(m.Key)
// 	if err != nil {
// 		t.Fatalf("error in removing stock %s with stock id %s",
// 			m.StockID,
// 			err)
// 	}
// 	ne, err := repo.GetStock(m.Key)
// 	if err != nil {
// 		t.Fatalf(
// 			"error in fetching stock %s with ID %s",
// 			m.Key,
// 			err,
// 		)
// 	}
// 	assert := assert.New(t)
// 	assert.True(ne.NotFound, "entry should not exist")
// }
