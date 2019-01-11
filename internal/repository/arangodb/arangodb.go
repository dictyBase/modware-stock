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

type dockey struct {
	Key     string `json:"key"`
	PropKey string `json:"propkey"`
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
	var stmt string
	var bindVars map[string]interface{}
	if len(ns.Data.Attributes.StrainProperties.Parents) > 0 { // in case parents are present
		var parents []string
		pVars := map[string]interface{}{
			"@stock_collection": ar.stock.Name(),
		}
		for _, p := range ns.Data.Attributes.StrainProperties.Parents {
			pVars["id"] = p
			var pid string
			r, err := ar.database.GetRow(stockFindQ, pVars)
			if err != nil {
				return m, fmt.Errorf("error in searching for parent %s %s", p, err)
			}
			if r.IsEmpty() {
				return m, fmt.Errorf("parent %s is not found", p)
			}
			if err := r.Read(&pid); err != nil {
				return m, fmt.Errorf("error in scanning the value %s %s", p, err)
			}
			parents = append(parents, pid)
		}
		stmt = stockStrainWithParentsIns
		bindVars = addableStrainBindParams(ns.Data.Attributes)
		bindVars["parents"] = parents
		bindVars["@parent_strain_collection"] = ar.parentStrain.Name()
		sp := &model.StrainProperties{
			Parents: ns.Data.Attributes.StrainProperties.Parents,
		}
		m.StrainProperties = sp
	} else {
		bindVars = addableStrainBindParams(ns.Data.Attributes)
		stmt = stockStrainIns
	}
	bindVars["@stock_collection"] = ar.stock.Name()
	bindVars["@stock_key_generator"] = ar.stockKey.Name()
	bindVars["@stock_properties_collection"] = ar.stockProp.Name()
	bindVars["@stock_type_collection"] = ar.stockType.Name()
	r, err := ar.database.DoRun(stmt, bindVars)
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
	bindVars := addablePlasmidBindParams(attr)
	bindVars["@stock_collection"] = ar.stock.Name()
	bindVars["@stock_key_generator"] = ar.stockKey.Name()
	bindVars["@stock_type_collection"] = ar.stockType.Name()
	bindVars["@stock_properties_collection"] = ar.stockProp.Name()
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
	// check if stock exists before running any update
	r, err := ar.database.GetRow(
		stockFindIdQ,
		map[string]interface{}{
			"@stock_collection": ar.stock.Name(),
			"graph":             ar.stockPropType.Name(),
			"id":                us.Data.Id,
		})
	if err != nil {
		return m, fmt.Errorf("error in searching for id %s %s", us.Data.Id, err)
	}
	if r.IsEmpty() {
		return m, fmt.Errorf("id %s is not found", us.Data.Id)
	}
	dk := &dockey{}
	if err := r.Read(dk); err != nil {
		return m, err
	}

	var stmt string
	cmBindVars := make(map[string]interface{})
	bindVars := getUpdatableStockBindParams(attr)
	var bindParams []string
	for k := range bindVars {
		bindParams = append(bindParams, fmt.Sprintf("%s: @%s", k, k))
	}
	if us.Data.Type == "strain" {
	} else {
		bindPlvars := getUpdatablePlasmidBindParams(attr)
		var bindPlParams []string
		for k := range bindPlVars {
			bindPlParams = append(bindPlParams, fmt.Sprintf("%s: @%s", k, k))
		}
		stmt = fmt.Sprintf(
			plasmidUpd,
			strings.Join(bindParams, ","),
			strings.Join(bindPlParams, ","),
		)
		cmBindVars = mergeBindParams([]map[string]interface{}{bindVars, bindPlvars}...)
	}
	cmBindVars["@stock_collection"] = ar.stock.Name()
	cmBindVars["@stock_properties_collection"] = ar.stockProp.Name()
	cmBindVars["key"] = dk.Key
	cmBindVars["propkey"] = dk.PropKey
	rupd, err := ar.database.DoRun(stmt, cmBindVars)
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
		"graph":             "stockprop_type",
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
		"graph":             "stockprop_type",
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

// ClearStocks clears all stocks from the repository datasource
func (ar *arangorepository) ClearStocks() error {
	if err := ar.stock.Truncate(context.Background()); err != nil {
		return err
	}
	return nil
}

func addablePlasmidBindParams(attr *stock.NewStockAttributes) map[string]interface{} {
	bindVars := map[string]interface{}{
		"depositor":        attr.Depositor,
		"created_by":       attr.CreatedBy,
		"updated_by":       attr.UpdatedBy,
		"summary":          normalizeStrBindParam(attr.Summary),
		"editable_summary": normalizeStrBindParam(attr.EditableSummary),
		"genes":            normalizeSliceBindParam(attr.Genes),
		"dbxrefs":          normalizeSliceBindParam(attr.Dbxrefs),
		"publications":     normalizeSliceBindParam(attr.Publications),
	}
	if attr.PlasmidProperties != nil {
		bindVars["image_map"] = normalizeStrBindParam(attr.PlasmidProperties.ImageMap)
		bindVars["sequence"] = normalizeStrBindParam(attr.PlasmidProperties.Sequence)
	}
	return bindVars
}

func addableStrainBindParams(attr *stock.NewStockAttributes) map[string]interface{} {
	return map[string]interface{}{
		"summary":          normalizeStrBindParam(attr.Summary),
		"editable_summary": normalizeStrBindParam(attr.EditableSummary),
		"genes":            normalizeSliceBindParam(attr.Genes),
		"dbxrefs":          normalizeSliceBindParam(attr.Dbxrefs),
		"publications":     normalizeSliceBindParam(attr.Publications),
		"plasmid":          normalizeStrBindParam(attr.StrainProperties.Plasmid),
		"names":            normalizeSliceBindParam(attr.StrainProperties.Names),
		"depositor":        attr.Depositor,
		"systematic_name":  attr.StrainProperties.SystematicName,
		"label":            attr.StrainProperties.Label,
		"species":          attr.StrainProperties.Species,
		"created_by":       attr.CreatedBy,
		"updated_by":       attr.UpdatedBy,
	}
}

func normalizeSliceBindParam(s []string) []string {
	if len(s) > 0 {
		return s
	}
	return []string{}
}

func normalizeStrBindParam(str string) string {
	if len(str) > 0 {
		return str
	}
	return ""
}

func getUpdatableStockBindParams(attr *stock.StockUpdateAttributes) map[string]interface{} {
	bindVars := map[string]interface{}{
		"updated_by": attr.UpdatedBy,
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
	return bindVars
}

func getUpdatableStrainBindParams(attr *stock.StockUpdateAttributes) map[string]interface{} {
	bindVars := map[string]interface{}{
		"systematic_name": attr.StrainProperties.SystematicName,
		"label":           attr.StrainProperties.Label,
		"species":         attr.StrainProperties.Species,
	}
	if len(attr.StrainProperties.Plasmid) > 0 {
		bindVars["plasmid"] = attr.StrainProperties.Plasmid
	}
	if len(attr.StrainProperties.Names) > 0 {
		bindVars["names"] = attr.StrainProperties.Names
	}
	return bindVars
}

func getUpdatablePlasmidBindParams(attr *stock.StockUpdateAttributes) map[string]interface{} {
	bindVars := make(map[string]interface{})
	if attr.PlasmidProperties != nil {
		if len(attr.PlasmidProperties.ImageMap) > 0 {
			bindVars["image_map"] = attr.PlasmidProperties.ImageMap
		}
		if len(attr.PlasmidProperties.Sequence) > 0 {
			bindVars["sequence"] = attr.PlasmidProperties.Sequence
		}
	}
	return bindVars
}

func mergeBindParams(bm ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range bm {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
