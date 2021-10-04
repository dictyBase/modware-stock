package arangodb

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dictyBase/go-obograph/graph"
	"github.com/dictyBase/go-obograph/storage"
	ontoarango "github.com/dictyBase/go-obograph/storage/arangodb"

	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	charSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var seedRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func getOntoParams() *ontoarango.CollectionParams {
	return &ontoarango.CollectionParams{
		GraphInfo:    "cv",
		OboGraph:     "obograph",
		Relationship: "cvterm_relationship",
		Term:         "cvterm",
	}
}

func getConnectParamsFromDb(ta *testarango.TestArango) *manager.ConnectParams {
	return &manager.ConnectParams{
		User:     ta.User,
		Pass:     ta.Pass,
		Database: ta.Database,
		Host:     ta.Host,
		Port:     ta.Port,
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
				CreatedBy:           createdby,
				UpdatedBy:           createdby,
				Depositor:           createdby,
				Summary:             "Radiation-sensitive mutant.",
				EditableSummary:     "Radiation-sensitive mutant.",
				Genes:               []string{"DDB_G0348394", "DDB_G098058933"},
				Publications:        []string{"48428304983", "83943", "839434936743"},
				Label:               "yS13",
				Species:             "Dictyostelium discoideum",
				DictyStrainProperty: "general strain",
			},
		},
	}
}

func newTestStrain(createdby string) *stock.NewStrain {
	return &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type: "strain",
			Attributes: &stock.NewStrainAttributes{
				CreatedBy:           createdby,
				UpdatedBy:           createdby,
				Depositor:           "george@costanza.com",
				Summary:             "Radiation-sensitive mutant.",
				EditableSummary:     "Radiation-sensitive mutant.",
				Dbxrefs:             []string{"5466867", "4536935", "d2578", "d0319", "d2020/1033268", "d2580"},
				Genes:               []string{"DDB_G0348394", "DDB_G098058933"},
				Publications:        []string{"4849343943", "48394394"},
				Label:               "yS13",
				Species:             "Dictyostelium discoideum",
				Plasmid:             "DBP0000027",
				Names:               []string{"gammaS13", "gammaS-13", "γS-13"},
				DictyStrainProperty: "general strain",
			},
		},
	}
}

func newTestParentStrain(createdby string) *stock.NewStrain {
	return &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type: "strain",
			Attributes: &stock.NewStrainAttributes{
				CreatedBy:           createdby,
				UpdatedBy:           createdby,
				Depositor:           createdby,
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

func setUp(t *testing.T) (*require.Assertions, repository.StockRepository) {
	ta, err := testarango.NewTestArangoFromEnv(true)
	if err != nil {
		t.Fatalf("unable to construct new TestArango instance %s", err)
	}
	assert := require.New(t)
	repo, err := NewStockRepo(
		getConnectParamsFromDb(ta),
		getCollectionParams(),
		getOntoParams(),
	)
	assert.NoErrorf(err, "expect no error connecting to stock repository, received %s", err)
	err = loadData(ta)
	assert.NoError(err, "expect no error from loading ontology")
	return assert, repo
}

func tearDown(repo repository.StockRepository) {
	repo.Dbh().Drop()
}

func oboReader() (*os.File, error) {
	dir, err := os.Getwd()
	if err != nil {
		return &os.File{}, fmt.Errorf("unable to get current dir %s", err)
	}
	return os.Open(
		filepath.Join(
			filepath.Dir(dir), "testdata", "dicty_phenotypes.json",
		),
	)
}

func loadData(ta *testarango.TestArango) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get current dir %s", err)
	}
	r, err := os.Open(
		filepath.Join(
			filepath.Dir(dir), "testdata", "dicty_strain_property.json",
		),
	)
	if err != nil {
		return err
	}
	defer r.Close()
	g, err := graph.BuildGraph(r)
	if err != nil {
		return fmt.Errorf("error in building graph %s", err)
	}
	collP := getOntoParams()
	cp := &ontoarango.ConnectParams{
		User:     ta.User,
		Pass:     ta.Pass,
		Host:     ta.Host,
		Database: ta.Database,
		Port:     ta.Port,
		Istls:    ta.Istls,
	}
	clp := &ontoarango.CollectionParams{
		Term:         collP.Term,
		Relationship: collP.Relationship,
		GraphInfo:    collP.GraphInfo,
		OboGraph:     collP.OboGraph,
	}
	ds, err := ontoarango.NewDataSource(cp, clp)
	if err != nil {
		return err
	}
	return loadOboGraphInArango(g, ds)
}

func loadOboGraphInArango(g graph.OboGraph, ds storage.DataSource) error {
	if ds.ExistsOboGraph(g) {
		return nil
	}
	if err := ds.SaveOboGraphInfo(g); err != nil {
		return fmt.Errorf("error in saving graph %s", err)
	}
	if _, err := ds.SaveTerms(g); err != nil {
		return fmt.Errorf("error in saving terms %s", err)
	}
	_, err := ds.SaveRelationships(g)
	return err
}

func TestLoadOboJson(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
	fh, err := oboReader()
	assert.NoErrorf(err, "expect no error, received %s", err)
	defer fh.Close()
	m, err := repo.LoadOboJson(bufio.NewReader(fh))
	assert.NoErrorf(err, "expect no error, received %s", err)
	assert.True(m.IsCreated, "should match created status")
}

func TestRemoveStock(t *testing.T) {
	assert, repo := setUp(t)
	defer tearDown(repo)
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
