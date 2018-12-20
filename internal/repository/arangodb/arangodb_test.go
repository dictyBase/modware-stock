package arangodb

import (
	"log"
	"os"
	"testing"

	driver "github.com/arangodb/go-driver"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
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
	}
}

func newTestStrain(consumer string) *stock.NewStock {
	return &stock.NewStock{
		Data: &stock.NewStock_Data{
			Type: "stock",
			Id:   "DBS0238532",
			Attributes: &stock.NewStockAttributes{
				CreatedBy:       "george@costanza.com",
				UpdatedBy:       "george@costanza.com",
				Summary:         "Radiation-sensitive mutant.",
				EditableSummary: "Radiation-sensitive mutant.",
				Depositor:       "Rob Guyer (Reg Deering)",
				Dbxrefs:         []string{"5466867", "4536935", "d2578", "d0319", "d2020/1033268", "d2580"},
				StrainProperties: &stock.StrainProperties{
					SystematicName: "yS13",
					Descriptor_:    "yS13",
					Species:        "Dictyostelium discoideum",
					Parents:        []string{"NC4 (DdB)"},
					Names:          []string{"gammaS13", "gammaS-13", "γS-13"},
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

func TestGetStock(t *testing.T) {
	connP := getConnectParams()
	collP := getCollectionParams()
	_, err := NewStockRepo(connP, collP)
	if err != nil {
		t.Fatalf("error in connecting to data source %s", err)
	}
}

// func TestAddStock(t *testing.T) {

// }

// func TestEditStock(t *testing.T) {

// }

// func TestListStocks(t *testing.T) {

// }

// func TestListStrains(t *testing.T) {

// }

// func TestListPlasmids(t *testing.T) {

// }

// func RemoveStock(t *testing.T) {

// }
