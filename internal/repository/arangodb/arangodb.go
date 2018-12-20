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
	validator "gopkg.in/go-playground/validator.v9"
)

// CollectionParams are the arangodb collections required for storing stocks
type CollectionParams struct {
	// Stock is the collection for storing all stocks
	Stock string `validate:"required"`
	// Strain is the collection for storing strain properties
	Strain string `validate:"required"`
	// PLasmid is the collection for storing plasmid properties
	Plasmid string `validate:"required"`
	// StockKeyGenerator is the collection for generating unique stock IDs
	StockKeyGenerator string `validate:"required"`
	// StockPlasmid is the edge collection for connecting stocks and plasmids
	StockPlasmid string `validate:"required"`
	// StockStrain is the edge collection for connecting stocks and strains
	StockStrain string `validate:"required"`
	// ParentStrain is the edge collection for connecting strains to their parents
	ParentStrain string `validate:"required"`
	// Stock2PlasmidGraph is the named graph for connecting stocks to plasmids
	Stock2PlasmidGraph string `validate:"required"`
	// Stock2StrainGraph is the named graph for connecting stocks to strains
	Stock2StrainGraph string `validate:"required"`
	// Strain2ParentGraph is the named graph for connecting strains to their parents
	Strain2ParentGraph string `validate:"required"`
}

type arangorepository struct {
	sess          *manager.Session
	database      *manager.Database
	stock         driver.Collection
	strain        driver.Collection
	plasmid       driver.Collection
	stockKey      driver.Collection
	stockPlasmid  driver.Collection
	stockStrain   driver.Collection
	parentStrain  driver.Collection
	stock2Plasmid driver.Graph
	stock2Strain  driver.Graph
	strain2Parent driver.Graph
}

// NewStockRepo acts as constructor for database
func NewStockRepo(connP *manager.ConnectParams, collP *CollectionParams) (repository.StockRepository, error) {
	ar := &arangorepository{}
	validate := validator.New()
	if err := validate.Struct(collP); err != nil {
		return ar, err
	}
	sess, db, err := manager.NewSessionDb(connP)
	if err != nil {
		return ar, err
	}
	ar.sess = sess
	ar.database = db
	stockc, err := db.FindOrCreateCollection(collP.Stock, &driver.CreateCollectionOptions{})
	// KeyOptions: {type: "autoincrement", increment: 1, offset: 370000}
	if err != nil {
		return ar, err
	}
	ar.stock = stockc
	strainc, err := db.FindOrCreateCollection(collP.Strain, &driver.CreateCollectionOptions{})
	if err != nil {
		return ar, err
	}
	ar.strain = strainc
	plasmidc, err := db.FindOrCreateCollection(collP.Plasmid, &driver.CreateCollectionOptions{})
	if err != nil {
		return ar, err
	}
	ar.plasmid = plasmidc
	stockkeyc, err := db.FindOrCreateCollection(collP.StockKeyGenerator, &driver.CreateCollectionOptions{})
	if err != nil {
		return ar, err
	}
	ar.stockKey = stockkeyc
	stockplasmidc, err := db.FindOrCreateCollection(collP.StockPlasmid, &driver.CreateCollectionOptions{Type: driver.CollectionTypeEdge})
	if err != nil {
		return ar, err
	}
	ar.stockPlasmid = stockplasmidc
	stockstrainc, err := db.FindOrCreateCollection(collP.StockStrain, &driver.CreateCollectionOptions{Type: driver.CollectionTypeEdge})
	if err != nil {
		return ar, err
	}
	ar.stockStrain = stockstrainc
	parentc, err := db.FindOrCreateCollection(collP.ParentStrain, &driver.CreateCollectionOptions{Type: driver.CollectionTypeEdge})
	if err != nil {
		return ar, err
	}
	ar.parentStrain = parentc
	stock2plasmidg, err := db.FindOrCreateGraph(
		collP.Stock2PlasmidGraph,
		[]driver.EdgeDefinition{
			driver.EdgeDefinition{
				Collection: stockplasmidc.Name(),
				From:       []string{stockc.Name()},
				To:         []string{plasmidc.Name()},
			},
		},
	)
	if err != nil {
		return ar, err
	}
	ar.stock2Plasmid = stock2plasmidg
	stock2straing, err := db.FindOrCreateGraph(
		collP.Stock2StrainGraph,
		[]driver.EdgeDefinition{
			driver.EdgeDefinition{
				Collection: stockstrainc.Name(),
				From:       []string{stockc.Name()},
				To:         []string{strainc.Name()},
			},
		},
	)
	if err != nil {
		return ar, err
	}
	ar.stock2Strain = stock2straing
	strain2parentg, err := db.FindOrCreateGraph(
		collP.Strain2ParentGraph,
		[]driver.EdgeDefinition{
			driver.EdgeDefinition{
				Collection: parentc.Name(),
				From:       []string{stockc.Name()},
				To:         []string{stockc.Name()},
			},
		},
	)
	if err != nil {
		return ar, err
	}
	ar.strain2Parent = strain2parentg
	return ar, nil
}

// GetStock retrieves biological stock from database
func (ar *arangorepository) GetStock(id string) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	var stmt string
	bindVars := map[string]interface{}{
		"@stock_collection":         ar.stock.Name(),
		"key":                       id,
		"@stock_strain_collection":  ar.stockStrain.Name(),
		"@parent_strain_collection": ar.parentStrain.Name(),
		"@stock_plasmid_collection": ar.stockPlasmid.Name(),
	}
	if id[:3] == "DBS" {
		stmt = stockGetStrain
	} else {
		stmt = stockGetPlasmid
	}
	r, err := ar.database.GetRow(stmt, bindVars)
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

// AddStrain creates a new strain stock
func (ar *arangorepository) AddStrain(ns *stock.NewStock) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	attr := ns.Data.Attributes
	bindVars := map[string]interface{}{
		"@stock_collection":         ar.stock.Name(),
		"@stock_key_generator":      ar.stockKey.Name(),
		"@strain_collection":        ar.strain.Name(),
		"@stock_strain_collection":  ar.stockStrain.Name(),
		"@parent_strain_collection": ar.parentStrain.Name(),
		"@created_by":               attr.CreatedBy,
		"@updated_by":               attr.UpdatedBy,
		"@summary":                  attr.Summary,
		"@editable_summary":         attr.EditableSummary,
		"@depositor":                attr.Depositor,
		"@genes":                    attr.Genes,
		"@dbxrefs":                  attr.Dbxrefs,
		"@publications":             attr.Publications,
		"@systematic_name":          attr.StrainProperties.SystematicName,
		"@descriptor":               attr.StrainProperties.Descriptor_,
		"@species":                  attr.StrainProperties.Species,
		"@plasmid":                  attr.StrainProperties.Plasmid,
		"@parents":                  attr.StrainProperties.Parents,
		"@names":                    attr.StrainProperties.Names,
	}
	r, err := ar.database.DoRun(stockStrainIns, bindVars)
	if err != nil {
		return m, err
	}
	if err := r.Read(m); err != nil {
		return m, err
	}
	return m, nil
}

// AddPlasmid creates a new plasmid stock
func (ar *arangorepository) AddPlasmid(ns *stock.NewStock) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	attr := ns.Data.Attributes
	bindVars := map[string]interface{}{
		"@stock_collection":         ar.stock.Name(),
		"@stock_key_generator":      ar.stockKey.Name(),
		"@plasmid_collection":       ar.plasmid.Name(),
		"@stock_plasmid_collection": ar.stockPlasmid.Name(),
		"@created_by":               attr.CreatedBy,
		"@updated_by":               attr.UpdatedBy,
		"@summary":                  attr.Summary,
		"@editable_summary":         attr.EditableSummary,
		"@depositor":                attr.Depositor,
		"@genes":                    attr.Genes,
		"@dbxrefs":                  attr.Dbxrefs,
		"@publications":             attr.Publications,
		"@image_map":                attr.PlasmidProperties.ImageMap,
		"@sequence":                 attr.PlasmidProperties.Sequence,
	}
	r, err := ar.database.DoRun(stockPlasmidIns, bindVars)
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
	bindVars["@stock_collection"] = ar.stock.Name()
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
		"@stock_collection": ar.stock.Name(),
		"limit":             limit + 1,
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
	var om []*model.StockDoc
	var stmt string
	bindVars := map[string]interface{}{
		"@stock_collection": ar.stock.Name(),
		"limit":             limit + 1,
		"graph":             "stock2strain",
	}
	if cursor == 0 { // no cursor so return first set of result
		stmt = strainList
	} else {
		bindVars["next_cursor"] = cursor
		stmt = strainListWithCursor
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

// ListPlasmids provides a list of all plasmids
func (ar *arangorepository) ListPlasmids(cursor int64, limit int64) ([]*model.StockDoc, error) {
	var om []*model.StockDoc
	var stmt string
	bindVars := map[string]interface{}{
		"@stock_collection": ar.stock.Name(),
		"limit":             limit + 1,
		"graph":             "stock2plasmid",
	}
	if cursor == 0 { // no cursor so return first set of result
		stmt = plasmidList
	} else {
		bindVars["next_cursor"] = cursor
		stmt = plasmidListWithCursor
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
		"@stock_collection": ar.stock.Name(),
		"@key":              id,
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
	return bindVars
}

// ClearStocks clears all stocks from the repository datasource
func (ar *arangorepository) ClearStocks() error {
	if err := ar.stock.Truncate(context.Background()); err != nil {
		return err
	}
	return nil
}
