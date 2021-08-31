package arangodb

import (
	driver "github.com/arangodb/go-driver"
)

// CollectionParams are the arangodb collections required for storing stocks
type CollectionParams struct {
	// Stock is the collection for storing all stocks
	Stock string `validate:"required"`
	// Stockprop is the collection for storing stock properties
	StockProp string `validate:"required"`
	// StockKeyGenerator is the collection for generating unique stock IDs
	StockKeyGenerator string `validate:"required"`
	// StockType is the edge collection for connecting stocks to their types
	StockType string `validate:"required"`
	// ParentStrain is the edge collection for connecting strains to their parents
	ParentStrain string `validate:"required"`
	// StockPropTypeGraph is the named graph for connecting stock properties to types
	StockPropTypeGraph string `validate:"required"`
	// Strain2ParentGraph is the named graph for connecting strains to their parents
	Strain2ParentGraph string `validate:"required"`
	// KeyOffset is the initial offset for stock id generation. It is needed to
	// maintain the previous stock identifiers.
	KeyOffset int `validate:"required"`
	// StockTerm is the edge collection stock with an ontology
	// term
	StockTerm string `validate:"required"`
	// StockOntoGraph is the named graph for connecting stock
	// with the ontology
	StockOntoGraph string `validate:"required"`
	// StrainOntology is the ontology for storing strain group
	StrainOntology string `validate:"required"`
}

type stockc struct {
	stock, stockProp, stockKey         driver.Collection
	stockType, parentStrain, stockTerm driver.Collection
	stockPropType, strain2Parent       driver.Graph
	stockOnto                          driver.Graph
}

type persistStrainParams struct {
	parent, dictyStrainProp    string
	statement, parentStatement string
	bindVars                   map[string]interface{}
}

func createDbStruct(ar *arangorepository, collP *CollectionParams) error {
	if err := docCollections(ar, collP); err != nil {
		return err
	}
	if err := graphAndEdgeCollections(ar, collP); err != nil {
		return err
	}
	return createIndex(ar)
}

func docCollections(ar *arangorepository, collP *CollectionParams) error {
	db := ar.database
	stkc, err := db.FindOrCreateCollection(
		collP.Stock,
		&driver.CreateCollectionOptions{},
	)
	if err != nil {
		return err
	}
	spropc, err := db.FindOrCreateCollection(
		collP.StockProp,
		&driver.CreateCollectionOptions{},
	)
	if err != nil {
		return err
	}
	stockkeyc, err := db.FindOrCreateCollection(
		collP.StockKeyGenerator,
		&driver.CreateCollectionOptions{
			KeyOptions: &driver.CollectionKeyOptions{
				Type:      "autoincrement",
				Increment: 1,
				Offset:    collP.KeyOffset,
			}})
	if err != nil {
		return err
	}
	ar.stockc = &stockc{
		stockProp: spropc,
		stockKey:  stockkeyc,
		stock:     stkc,
	}
	return nil
}

func graphAndEdgeCollections(ar *arangorepository, collP *CollectionParams) error {
	if err := createEdgeCollections(ar, collP); err != nil {
		return err
	}
	return createNamedGraph(ar, collP)
}

func createEdgeCollections(ar *arangorepository, collP *CollectionParams) error {
	db := ar.database
	stypec, err := db.FindOrCreateCollection(
		collP.StockType,
		&driver.CreateCollectionOptions{Type: driver.CollectionTypeEdge},
	)
	if err != nil {
		return err
	}
	parentc, err := db.FindOrCreateCollection(
		collP.ParentStrain,
		&driver.CreateCollectionOptions{Type: driver.CollectionTypeEdge},
	)
	if err != nil {
		return err
	}
	sterm, err := db.FindOrCreateCollection(
		collP.StockTerm,
		&driver.CreateCollectionOptions{Type: driver.CollectionTypeEdge},
	)
	if err != nil {
		return err
	}
	ar.stockc.parentStrain = parentc
	ar.stockc.stockType = stypec
	ar.stockc.stockTerm = sterm
	return nil
}

func createNamedGraph(ar *arangorepository, collP *CollectionParams) error {
	db := ar.database
	sproptypeg, err := db.FindOrCreateGraph(
		collP.StockPropTypeGraph,
		[]driver.EdgeDefinition{{
			Collection: ar.stockc.stockType.Name(),
			From:       []string{ar.stockc.stock.Name()},
			To:         []string{ar.stockc.stockProp.Name()},
		}},
	)
	if err != nil {
		return err
	}
	strain2parentg, err := db.FindOrCreateGraph(
		collP.Strain2ParentGraph,
		[]driver.EdgeDefinition{{
			Collection: ar.stockc.parentStrain.Name(),
			From:       []string{ar.stockc.stock.Name()},
			To:         []string{ar.stockc.stock.Name()},
		}},
	)
	if err != nil {
		return err
	}
	sonto, err := db.FindOrCreateGraph(
		collP.StockOntoGraph,
		[]driver.EdgeDefinition{{
			Collection: ar.stockc.stockTerm.Name(),
			From:       []string{ar.stockc.stock.Name()},
			To:         []string{ar.ontoc.Term.Name()},
		}},
	)
	if err != nil {
		return err
	}
	ar.stockc.stockPropType = sproptypeg
	ar.stockc.strain2Parent = strain2parentg
	ar.stockc.stockOnto = sonto
	return nil
}

func createIndex(ar *arangorepository) error {
	_, _, err := ar.database.EnsurePersistentIndex(
		ar.stockc.stock.Name(),
		[]string{"stock_id"},
		&driver.EnsurePersistentIndexOptions{
			Unique:       true,
			InBackground: true,
			Name:         "stock_id_idx",
		})
	return err
}
