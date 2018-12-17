package arangodb

import (
	"context"

	driver "github.com/arangodb/go-driver"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository"
)

type arangorepository struct {
	sess     *manager.Session
	database *manager.Database
	stock    driver.Collection
}

// NewStockRepo acts as constructor for database
func NewStockRepo(connP *manager.ConnectParams, coll string) (repository.StockRepository, error) {
	ar := &arangorepository{}
	sess, db, err := manager.NewSessionDb(connP)
	if err != nil {
		return ar, err
	}
	ar.sess = sess
	ar.database = db
	stockc, err := db.FindOrCreateCollection(coll, &driver.CreateCollectionOptions{})
	if err != nil {
		return ar, err
	}
	ar.stock = stockc
	return ar, nil
}

// GetStock retrieves biological stock from database
func (ar *arangorepository) GetStock(id string) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	bindVars := map[string]interface{}{
		"@stocks_collection": ar.stock.Name(),
		"key":                id,
	}
	r, err := ar.database.GetRow(stockGet, bindVars)
	if err != nil {
		return m, err
	}
	if r.IsEmpty() {
		m.NotFound = true
		return m, nil
	}
	if err := r.Read(m); err != nil {
		return m, err
	}
	return m, nil
}

// AddStock creates a new biological stock
func (ar *arangorepository) AddStock(ns *stock.NewStock) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	attr := ns.Data.Attributes
	bindVars := map[string]interface{}{
		"@stocks_collection": ar.stock.Name(),
		"@created_by":        attr.CreatedBy,
		"@updated_by":        attr.UpdatedBy,
		"@summary":           attr.Summary,
		"@editable_summary":  attr.EditableSummary,
		"@depositor":         attr.Depositor,
		"@genes":             attr.Genes,
		"@dbxrefs":           attr.Dbxrefs,
		"@publications":      attr.Publications,
		"@systematic_name":   attr.StrainProperties.SystematicName,
		"@descriptor":        attr.StrainProperties.Descriptor,
		"@species":           attr.StrainProperties.Species,
		"@plasmid":           attr.StrainProperties.Plasmid,
		"@parents":           attr.StrainProperties.Parents,
		"@names":             attr.StrainProperties.Names,
		"@image_map":         attr.PlasmidProperties.ImageMap,
		"@sequence":          attr.PlasmidProperties.Sequence,
		"@keywords":          attr.PlasmidProperties.Keywords,
	}
	r, err := ar.database.DoRun(stockIns, bindVars)
	if err != nil {
		return m, err
	}
	if err := r.Read(m); err != nil {
		return m, err
	}
	return m, nil
}

// EditStock updates an existing stock
func (ar *arangorepository) EditStock(us *stock.StockUpdate) (*model.StockDoc, error) {

}

// ListStocks provides a list of all stocks
func (ar *arangorepository) ListStocks(cursor int64, limit int64) ([]*model.StockDoc, error) {

}

// ListStrains provides a list of all strains
func (ar *arangorepository) ListStrains(cursor int64, limit int64) ([]*model.StockDoc, error) {

}

// ListPlasmids provides a list of all plasmids
func (ar *arangorepository) ListPlasmids(cursor int64, limit int64) ([]*model.StockDoc, error) {

}

// RemoveStock removes a stock
func (ar *arangorepository) RemoveStock(id string) error {

}

// ClearStocks clears all stocks from the repository datasource
func (ar *arangorepository) ClearStocks() error {
	if err := ar.stock.Truncate(context.Background()); err != nil {
		return err
	}
	return nil
}
