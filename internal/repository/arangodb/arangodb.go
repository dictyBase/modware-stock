package arangodb

import (
	"context"
	"fmt"
	"strings"

	driver "github.com/arangodb/go-driver"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/query"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository"
	validator "gopkg.in/go-playground/validator.v9"
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
}

type arangorepository struct {
	sess          *manager.Session
	database      *manager.Database
	stock         driver.Collection
	stockProp     driver.Collection
	stockKey      driver.Collection
	stockType     driver.Collection
	parentStrain  driver.Collection
	stockPropType driver.Graph
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
	stockc, err := db.FindOrCreateCollection(
		collP.Stock,
		&driver.CreateCollectionOptions{
			KeyOptions: &driver.CollectionKeyOptions{
				Increment: 1,
				Offset:    collP.KeyOffset,
			},
		})
	if err != nil {
		return ar, err
	}
	ar.stock = stockc
	spropc, err := db.FindOrCreateCollection(collP.StockProp, &driver.CreateCollectionOptions{})
	if err != nil {
		return ar, err
	}
	ar.stockProp = spropc
	stockkeyc, err := db.FindOrCreateCollection(collP.StockKeyGenerator, &driver.CreateCollectionOptions{})
	if err != nil {
		return ar, err
	}
	ar.stockKey = stockkeyc
	stypec, err := db.FindOrCreateCollection(collP.StockType, &driver.CreateCollectionOptions{Type: driver.CollectionTypeEdge})
	if err != nil {
		return ar, err
	}
	ar.stockType = stypec
	parentc, err := db.FindOrCreateCollection(collP.ParentStrain, &driver.CreateCollectionOptions{Type: driver.CollectionTypeEdge})
	if err != nil {
		return ar, err
	}
	ar.parentStrain = parentc
	sproptypeg, err := db.FindOrCreateGraph(
		collP.StockPropTypeGraph,
		[]driver.EdgeDefinition{
			driver.EdgeDefinition{
				Collection: stypec.Name(),
				From:       []string{stockc.Name()},
				To:         []string{spropc.Name()},
			},
		},
	)
	if err != nil {
		return ar, err
	}
	ar.stockPropType = sproptypeg
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
		"@stock_collection":      ar.stock.Name(),
		"@stock_type_collection": ar.stockType.Name(),
		"id":                     id,
		"graph":                  ar.stockPropType.Name(),
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
		"@stock_collection":            ar.stock.Name(),
		"@stock_key_generator":         ar.stockKey.Name(),
		"@stock_properties_collection": ar.stockProp.Name(),
		"@stock_type_collection":       ar.stockType.Name(),
		"@parent_strain_collection":    ar.parentStrain.Name(),
		"created_by":                   attr.CreatedBy,
		"updated_by":                   attr.UpdatedBy,
		"summary":                      attr.Summary,
		"editable_summary":             attr.EditableSummary,
		"depositor":                    attr.Depositor,
		"genes":                        attr.Genes,
		"dbxrefs":                      attr.Dbxrefs,
		"publications":                 attr.Publications,
		"systematic_name":              attr.StrainProperties.SystematicName,
		"descriptor":                   attr.StrainProperties.Descriptor_,
		"species":                      attr.StrainProperties.Species,
		"plasmid":                      attr.StrainProperties.Plasmid,
		"parents":                      attr.StrainProperties.Parents,
		"names":                        attr.StrainProperties.Names,
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
		"@stock_collection":            ar.stock.Name(),
		"@stock_key_generator":         ar.stockKey.Name(),
		"@stock_type_collection":       ar.stockType.Name(),
		"@stock_properties_collection": ar.stockProp.Name(),
		"created_by":                   attr.CreatedBy,
		"updated_by":                   attr.UpdatedBy,
		"summary":                      attr.Summary,
		"editable_summary":             attr.EditableSummary,
		"depositor":                    attr.Depositor,
		"genes":                        attr.Genes,
		"dbxrefs":                      attr.Dbxrefs,
		"publications":                 attr.Publications,
		"image_map":                    attr.PlasmidProperties.ImageMap,
		"sequence":                     attr.PlasmidProperties.Sequence,
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
func (ar *arangorepository) ListStocks(p *stock.StockParameters) ([]*model.StockDoc, error) {
	var om []*model.StockDoc
	var stmt string
	c := p.Cursor
	l := p.Limit
	f := p.Filter
	bindVars := map[string]interface{}{
		"@stock_collection": ar.stock.Name(),
		"limit":             l + 1,
	}
	// if filter string exists, call searchStocks function to get proper query statement
	if len(f) > 0 {
		n, err := (*ar).searchStocks(&stock.StockParameters{Cursor: c, Limit: l, Filter: f})
		if err != nil {
			return om, err
		}
		stmt = n
	} else {
		// otherwise use query statement without filter
		if c == 0 { // no cursor so return first set of result
			stmt = stockList
		} else {
			bindVars["next_cursor"] = c
			stmt = stockListWithCursor
		}
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

// searchStocks is a private function specifically for handling filter queries
func (ar *arangorepository) searchStocks(p *stock.StockParameters) (string, error) {
	c := p.Cursor
	l := p.Limit
	f := p.Filter
	var stmt string
	s, err := query.ParseFilterString(f)
	if err != nil {
		fmt.Println("error parsing filter string", err)
	}
	n, err := query.GenAQLFilterStatement(fmap, s, "s")
	if err != nil {
		fmt.Println("error generating AQL filter statement", err)
	}
	if c == 0 {
		if strings.Contains(f, "stock_type==strain") {
			stmt = fmt.Sprintf(
				strainListFilter,
				ar.stock.Name(),
				ar.stockType.Name(),
				n,
				l+1,
			)
		} else if strings.Contains(f, "stock_type==plasmid") {
			stmt = fmt.Sprintf(
				plasmidListFilter,
				ar.stock.Name(),
				ar.stockType.Name(),
				n,
				l+1,
			)
		} else {
			stmt = fmt.Sprintf(
				stockListFilter,
				ar.stock.Name(),
				n,
				l+1,
			)
		}
	} else {
		if strings.Contains(f, "stock_type==strain") {
			stmt = fmt.Sprintf(
				strainListFilterWithCursor,
				ar.stock.Name(),
				ar.stockType.Name(),
				n,
				c,
				l+1,
			)
		} else if strings.Contains(f, "stock_type==plasmid") {
			stmt = fmt.Sprintf(
				plasmidListFilterWithCursor,
				ar.stock.Name(),
				ar.stockType.Name(),
				n,
				c,
				l+1,
			)
		} else {
			stmt = fmt.Sprintf(
				stockListFilterWithCursor,
				ar.stock.Name(),
				c,
				n,
				l+1,
			)
		}
	}
	return stmt, nil
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
		"key":               id,
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
