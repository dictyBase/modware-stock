package arangodb

import (
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	ontoarango "github.com/dictyBase/go-obograph/storage/arangodb"

	driver "github.com/arangodb/go-driver"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/stretchr/testify/assert"
)

const (
	charSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var seedRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
var gta *testarango.TestArango

func getOntoParams() *ontoarango.CollectionParams {
	return &ontoarango.CollectionParams{
		GraphInfo:    "cv",
		OboGraph:     "obograph",
		Relationship: "cvterm_relationship",
		Term:         "cvterm",
	}
}

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
		StockTerm:          "stock_term",
		StockProp:          "stockprop",
		StockKeyGenerator:  "stock_key_generator",
		StockType:          "stock_type",
		StockOntoGraph:     "stockonto",
		ParentStrain:       "parent_strain",
		StockPropTypeGraph: "stockprop_type",
		Strain2ParentGraph: "strain2parent",
		KeyOffset:          370000,
		StrainOntology:     "dicty_strain_property",
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

func setUp(t *testing.T) (*assert.Assertions, repository.StockRepository) {
	assert := assert.New(t)
	repo, err := NewStockRepo(
		getConnectParams(),
		getCollectionParams(),
		getOntoParams(),
	)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	return assert, repo
}

func tearDown(assert *assert.Assertions, repo repository.StockRepository) {
	err := repo.ClearStocks()
	assert.NoErrorf(err, "expect no error in clearing stocks, received %s", err)
}

func TestRemoveStock(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(assert, repo)
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
