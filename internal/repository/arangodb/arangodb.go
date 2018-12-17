package arangodb

import (
	"context"
	"fmt"
	"strings"

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
	m := &model.StockDoc{}
	attr := us.Data.Attributes
	// check if order exists
	em, err := ar.GetStock(us.Data.Id)
	if err != nil {
		return m, err
	}
	if em.NotFound {
		m.NotFound = true
		return m, nil
	}
	bindVars := getUpdatableBindParams(attr)
	var bindParams []string
	for k := range bindVars {
		bindParams = append(bindParams, fmt.Sprintf("%s: @%s", k, k))
	}
	stockUpdQ := fmt.Sprintf(stockUpd, strings.Join(bindParams, ","))
	bindVars["@stocks_collection"] = ar.stock.Name()
	bindVars["key"] = us.Data.Id

	rupd, err := ar.database.DoRun(stockUpdQ, bindVars)
	if err != nil {
		return m, err
	}
	if err := rupd.Read(m); err != nil {
		return m, err
	}
	return m, nil
}

// ListStocks provides a list of all stocks
func (ar *arangorepository) ListStocks(cursor int64, limit int64) ([]*model.StockDoc, error) {
	var om []*model.StockDoc
	var stmt string
	bindVars := map[string]interface{}{
		"@stocks_collection": ar.stock.Name(),
		"limit":              limit + 1,
	}
	if cursor == 0 { // no cursor so return first set of result
		stmt = stockList
	} else {
		bindVars["next_cursor"] = cursor
		stmt = stockListWithCursor
	}
	rs, err := ar.database.SearchRows(stmt, bindVars)
	if err != nil {
		return om, err
	}
	if rs.IsEmpty() {
		return om, nil
	}
	for rs.Scan() {
		m := &model.StockDoc{}
		if err := rs.Read(m); err != nil {
			return om, err
		}
		om = append(om, m)
	}
	return om, nil
}

// ListStrains provides a list of all strains
func (ar *arangorepository) ListStrains(cursor int64, limit int64) ([]*model.StockDoc, error) {

}

// ListPlasmids provides a list of all plasmids
func (ar *arangorepository) ListPlasmids(cursor int64, limit int64) ([]*model.StockDoc, error) {

}

// RemoveStock removes a stock
func (ar *arangorepository) RemoveStock(id string) error {
	m := &model.StockDoc{}
	_, err := ar.stock.ReadDocument(context.Background(), id, m)
	if err != nil {
		if driver.IsNotFound(err) {
			return fmt.Errorf("stock record with id %s does not exist %s", id, err)
		}
		return err
	}
	bindVars := map[string]interface{}{
		"@stocks_collection": ar.stock.Name(),
		"@key":               id,
	}
	err = ar.database.Do(
		stockDelQ, bindVars,
	)
	if err != nil {
		return err
	}
	return nil
}

func getUpdatableBindParams(attr *stock.StockUpdateAttributes) map[string]interface{} {
	bindVars := make(map[string]interface{})
	if len(attr.UpdatedBy) > 0 {
		bindVars["updated_by"] = attr.UpdatedBy
	}
	if len(attr.Summary) > 0 {
		bindVars["summary"] = attr.Summary
	}
	if len(attr.EditableSummary) > 0 {
		bindVars["editable_summary"] = attr.EditableSummary
	}
	if len(attr.Depositor) > 0 {
		bindVars["depositor"] = attr.Depositor
	}
	if len(attr.Genes) > 0 {
		bindVars["genes"] = attr.Genes
	}
	if len(attr.Dbxrefs) > 0 {
		bindVars["dbxrefs"] = attr.Dbxrefs
	}
	if len(attr.Publications) > 0 {
		bindVars["publications"] = attr.Publications
	}
	if len(attr.StrainProperties.SystematicName) > 0 {
		bindVars["systematic_name"] = attr.StrainProperties.SystematicName
	}
	if len(attr.StrainProperties.Descriptor_) > 0 {
		bindVars["descriptor"] = attr.StrainProperties.Descriptor_
	}
	if len(attr.StrainProperties.Species) > 0 {
		bindVars["species"] = attr.StrainProperties.Species
	}
	if len(attr.StrainProperties.Plasmid) > 0 {
		bindVars["plasmid"] = attr.StrainProperties.Plasmid
	}
	if len(attr.StrainProperties.Parents) > 0 {
		bindVars["parents"] = attr.StrainProperties.Parents
	}
	if len(attr.StrainProperties.Names) > 0 {
		bindVars["names"] = attr.StrainProperties.Names
	}
	if len(attr.PlasmidProperties.ImageMap) > 0 {
		bindVars["image_map"] = attr.PlasmidProperties.ImageMap
	}
	if len(attr.PlasmidProperties.Sequence) > 0 {
		bindVars["sequence"] = attr.PlasmidProperties.Sequence
	}
	if len(attr.PlasmidProperties.Keywords) > 0 {
		bindVars["keywords"] = attr.PlasmidProperties.Keywords
	}
	return bindVars
}

// ClearStocks clears all stocks from the repository datasource
func (ar *arangorepository) ClearStocks() error {
	if err := ar.stock.Truncate(context.Background()); err != nil {
		return err
	}
	return nil
}
