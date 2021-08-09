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
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
	"github.com/golang/protobuf/ptypes"
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

func createGraphCollections(ar *arangorepository, collP *CollectionParams) error {
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
	sproptypeg, err := db.FindOrCreateGraph(
		collP.StockPropTypeGraph,
		[]driver.EdgeDefinition{{
			Collection: stypec.Name(),
			From:       []string{ar.stock.Name()},
			To:         []string{ar.stockProp.Name()},
		}},
	)
	if err != nil {
		return err
	}
	strain2parentg, err := db.FindOrCreateGraph(
		collP.Strain2ParentGraph,
		[]driver.EdgeDefinition{{
			Collection: parentc.Name(),
			From:       []string{ar.stock.Name()},
			To:         []string{ar.stock.Name()},
		}},
	)
	if err != nil {
		return err
	}
	ar.stockPropType = sproptypeg
	ar.strain2Parent = strain2parentg
	ar.parentStrain = parentc
	ar.stockType = stypec
	return nil
}

func createCollections(ar *arangorepository, collP *CollectionParams) error {
	db := ar.database
	stockc, err := db.FindOrCreateCollection(
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
	ar.stockProp = spropc
	ar.stockKey = stockkeyc
	ar.stock = stockc
	return nil
}

func createDbStruct(ar *arangorepository, collP *CollectionParams) error {
	if err := createCollections(ar, collP); err != nil {
		return err
	}
	if err := createGraphCollections(ar, collP); err != nil {
		return err
	}
	return createIndex(ar)
}

func createIndex(ar *arangorepository) error {
	_, _, err := ar.database.EnsurePersistentIndex(
		ar.stock.Name(),
		[]string{"stock_id"},
		&driver.EnsurePersistentIndexOptions{
			Unique:       true,
			InBackground: true,
			Name:         "stock_id_idx",
		})
	return err
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
	if err := createDbStruct(ar, collP); err != nil {
		return ar, err
	}
	return ar, nil
}

// GetStrain retrieves a strain from the database
func (ar *arangorepository) GetStrain(id string) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	bindVars := map[string]interface{}{
		"@stock_collection": ar.stock.Name(),
		"stock_collection":  ar.stock.Name(),
		"id":                id,
		"parent_graph":      ar.strain2Parent.Name(),
		"prop_graph":        ar.stockPropType.Name(),
	}
	g, err := ar.database.GetRow(
		statement.StockFindQ,
		map[string]interface{}{
			"@stock_collection": ar.stock.Name(),
			"id":                id,
		})
	if err != nil {
		return m, fmt.Errorf("error in finding strain id %s %s", id, err)
	}
	if g.IsEmpty() {
		m.NotFound = true
		return m, nil
	}
	r, err := ar.database.GetRow(statement.StockGetStrain, bindVars)
	if err != nil {
		return m, fmt.Errorf("error in finding strain id %s %s", id, err)
	}
	if err := r.Read(m); err != nil {
		return m, err
	}
	return m, nil
}

// GetPlasmid retrieves a plasmid from the database
func (ar *arangorepository) GetPlasmid(id string) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	bindVars := map[string]interface{}{
		"@stock_collection": ar.stock.Name(),
		"id":                id,
		"graph":             ar.stockPropType.Name(),
	}
	r, err := ar.database.GetRow(statement.StockGetPlasmid, bindVars)
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
func (ar *arangorepository) AddStrain(ns *stock.NewStrain) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	var stmt string
	var bindVars map[string]interface{}
	if len(ns.Data.Attributes.Parent) > 0 { // in case parent is present
		p := ns.Data.Attributes.Parent
		pVars := map[string]interface{}{
			"@stock_collection": ar.stock.Name(),
			"id":                p,
		}
		r, err := ar.database.GetRow(statement.StockFindQ, pVars)
		if err != nil {
			return m, fmt.Errorf("error in searching for parent %s %s", p, err)
		}
		if r.IsEmpty() {
			return m, fmt.Errorf("parent %s is not found", p)
		}
		var pid string
		if err := r.Read(&pid); err != nil {
			return m, fmt.Errorf("error in scanning the value %s %s", p, err)
		}
		stmt = statement.StockStrainWithParentsIns
		bindVars = addableStrainBindParams(ns.Data.Attributes)
		bindVars["pid"] = pid
		bindVars["@parent_strain_collection"] = ar.parentStrain.Name()
		m.StrainProperties = &model.StrainProperties{Parent: p}
	} else {
		bindVars = addableStrainBindParams(ns.Data.Attributes)
		stmt = statement.StockStrainIns
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
func (ar *arangorepository) AddPlasmid(ns *stock.NewPlasmid) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	attr := ns.Data.Attributes
	bindVars := addablePlasmidBindParams(attr)
	bindVars["@stock_collection"] = ar.stock.Name()
	bindVars["@stock_key_generator"] = ar.stockKey.Name()
	bindVars["@stock_type_collection"] = ar.stockType.Name()
	bindVars["@stock_properties_collection"] = ar.stockProp.Name()
	r, err := ar.database.DoRun(statement.StockPlasmidIns, bindVars)
	if err != nil {
		return m, err
	}
	if err := r.Read(m); err != nil {
		return m, err
	}
	return m, nil
}

// EditStrain updates an existing strain
func (ar *arangorepository) EditStrain(us *stock.StrainUpdate) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	r, err := ar.database.GetRow(
		statement.StockFindIdQ,
		map[string]interface{}{
			"stock_collection": ar.stock.Name(),
			"graph":            ar.stockPropType.Name(),
			"stock_id":         us.Data.Id,
		})
	if err != nil {
		return m, fmt.Errorf("error in finding strain id %s %s", us.Data.Id, err)
	}
	if r.IsEmpty() {
		return m, fmt.Errorf("strain id %s is absent in database", us.Data.Id)
	}
	var propKey string
	if err := r.Read(&propKey); err != nil {
		return m, fmt.Errorf("error in reading using strain id %s %s", us.Data.Id, err)
	}
	bindVars := getUpdatableStrainBindParams(us.Data.Attributes)
	bindStVars := getUpdatableStrainPropBindParams(us.Data.Attributes)
	cmBindVars := mergeBindParams([]map[string]interface{}{bindVars, bindStVars}...)
	var stmt string
	parent := us.Data.Attributes.Parent
	if len(parent) > 0 { // in case parent is present
		// parent -> relation -> child
		//   obj  ->  pred    -> sub
		// 1. Have to make sure the parent is present
		// 2. Have to figure out if child(sub) has an existing relation
		//    a) If relation exists, get and update the relation(pred)
		//    b) If not, create the new relation(pred)
		ok, err := ar.stock.DocumentExists(context.Background(), parent)
		if err != nil {
			return m, fmt.Errorf("error in checking for parent id %s %s", parent, err)
		}
		if !ok {
			return m, fmt.Errorf("parent id %s does not exist in database", parent)
		}
		r, err := ar.database.GetRow(
			statement.StrainGetParentRel,
			map[string]interface{}{
				"parent_graph": ar.strain2Parent.Name(),
				"strain_key":   us.Data.Id,
			})
		if err != nil {
			return m, fmt.Errorf("error in parent relation query %s", err)
		}
		var pKey string
		if !r.IsEmpty() {
			if err := r.Read(&pKey); err != nil {
				return m, fmt.Errorf("error in reading parent relation key %s", err)
			}
		}
		if len(pKey) > 0 {
			stmt = statement.StrainWithExistingParentUpd
			cmBindVars["pkey"] = pKey
		} else {
			stmt = statement.StrainWithNewParentUpd
		}
		cmBindVars["parent"] = us.Data.Attributes.Parent
		cmBindVars["stock_collection"] = ar.stock.Name()
		cmBindVars["@parent_strain_collection"] = ar.parentStrain.Name()
		m.StrainProperties = &model.StrainProperties{Parent: parent}
	} else {
		stmt = statement.StrainUpd
	}
	cmBindVars["@stock_properties_collection"] = ar.stockProp.Name()
	cmBindVars["@stock_collection"] = ar.stock.Name()
	cmBindVars["propkey"] = propKey
	cmBindVars["key"] = us.Data.Id
	q := fmt.Sprintf(
		stmt,
		genAQLDocExpression(bindVars),
		genAQLDocExpression(bindStVars),
	)
	rupd, err := ar.database.DoRun(
		q,
		cmBindVars,
	)
	if err != nil {
		return m, fmt.Errorf("error in editing strain %s %s %s", us.Data.Id, err, q)
	}
	if err := rupd.Read(m); err != nil {
		return m, err
	}
	return m, nil
}

// EditPlasmid updates an existing plasmid
func (ar *arangorepository) EditPlasmid(us *stock.PlasmidUpdate) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	r, err := ar.database.GetRow(
		statement.StockFindIdQ,
		map[string]interface{}{
			"stock_collection": ar.stock.Name(),
			"graph":            ar.stockPropType.Name(),
			"stock_id":         us.Data.Id,
		})
	if err != nil {
		return m, fmt.Errorf("error in finding plasmid id %s %s", us.Data.Id, err)
	}
	if r.IsEmpty() {
		return m, fmt.Errorf("plasmid id %s is absent in database", us.Data.Id)
	}
	var propKey string
	if err := r.Read(&propKey); err != nil {
		return m, fmt.Errorf("error in reading using plasmid id %s %s", us.Data.Id, err)
	}
	var stmt string
	bindVars := getUpdatablePlasmidBindParams(us.Data.Attributes)
	bindPlVars := getUpdatablePlasmidPropBindParams(us.Data.Attributes)
	cmBindVars := mergeBindParams([]map[string]interface{}{bindVars, bindPlVars}...)
	stmt = fmt.Sprintf(
		statement.PlasmidUpd,
		genAQLDocExpression(bindVars),
		genAQLDocExpression(bindPlVars),
	)
	cmBindVars["@stock_properties_collection"] = ar.stockProp.Name()
	cmBindVars["propkey"] = propKey
	cmBindVars["@stock_collection"] = ar.stock.Name()
	cmBindVars["key"] = us.Data.Id
	rupd, err := ar.database.DoRun(stmt, cmBindVars)
	if err != nil {
		return m, err
	}
	if err := rupd.Read(m); err != nil {
		return m, err
	}
	return m, nil
}

func (ar *arangorepository) ListStrainsByIds(p *stock.StockIdList) ([]*model.StockDoc, error) {
	ms := make([]*model.StockDoc, 0)
	ids := make([]string, 0)
	for _, v := range p.Id {
		ids = append(ids, v)
	}
	bindVars := map[string]interface{}{
		"ids":               ids,
		"limit":             len(ids),
		"stock_collection":  ar.stock.Name(),
		"@stock_collection": ar.stock.Name(),
		"prop_graph":        ar.stockPropType.Name(),
		"parent_graph":      ar.strain2Parent.Name(),
	}
	rs, err := ar.database.SearchRows(statement.StrainListFromIds, bindVars)
	if err != nil {
		return ms, err
	}
	if rs.IsEmpty() {
		return ms, nil
	}
	for rs.Scan() {
		m := &model.StockDoc{}
		if err := rs.Read(m); err != nil {
			return ms, err
		}
		ms = append(ms, m)
	}
	return ms, nil
}

// ListStrains provides a list of all strains
func (ar *arangorepository) ListStrains(p *stock.StockParameters) ([]*model.StockDoc, error) {
	var om []*model.StockDoc
	var stmt string
	c := p.Cursor
	l := p.Limit
	f := p.Filter
	// if filter string exists, it needs to be included in statement
	if len(f) > 0 {
		if c == 0 { // no cursor so return first set of results with filter
			stmt = fmt.Sprintf(
				statement.StrainListFilter,
				ar.stock.Name(),
				ar.stockPropType.Name(),
				f,
				l+1,
			)
		} else { // else include both filter and cursor
			stmt = fmt.Sprintf(
				statement.StrainListFilterWithCursor,
				ar.stock.Name(),
				ar.stockPropType.Name(),
				f,
				c,
				l+1,
			)
		}
	} else {
		// otherwise use query statement without filter
		if c == 0 { // no cursor so return first set of result
			stmt = fmt.Sprintf(
				statement.StrainList,
				ar.stock.Name(),
				ar.stockPropType.Name(),
				l+1,
			)
		} else { // add cursor if it exists
			stmt = fmt.Sprintf(
				statement.StrainListWithCursor,
				ar.stock.Name(),
				ar.stockPropType.Name(),
				c,
				l+1,
			)
		}
	}
	rs, err := ar.database.Search(stmt)
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
func (ar *arangorepository) ListPlasmids(p *stock.StockParameters) ([]*model.StockDoc, error) {
	var om []*model.StockDoc
	var stmt string
	c := p.Cursor
	l := p.Limit
	f := p.Filter
	// if filter string exists, it needs to be included in statement
	if len(f) > 0 {
		if c == 0 { // no cursor so return first set of result
			stmt = fmt.Sprintf(
				statement.PlasmidListFilter,
				ar.stock.Name(),
				ar.stockPropType.Name(),
				f,
				l+1,
			)
		} else { // else include both filter and cursor
			stmt = fmt.Sprintf(
				statement.PlasmidListFilterWithCursor,
				ar.stock.Name(),
				ar.stockPropType.Name(),
				f,
				c,
				l+1,
			)
		}
	} else {
		// otherwise use query statement without filter
		if c == 0 { // no cursor so return first set of result
			stmt = fmt.Sprintf(
				statement.PlasmidList,
				ar.stock.Name(),
				ar.stockPropType.Name(),
				l+1,
			)
		} else { // add cursor if it exists
			stmt = fmt.Sprintf(
				statement.PlasmidListWithCursor,
				ar.stock.Name(),
				ar.stockPropType.Name(),
				c,
				l+1,
			)
		}
	}
	rs, err := ar.database.Search(stmt)
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
	found, err := ar.stock.DocumentExists(context.Background(), id)
	if err != nil {
		return fmt.Errorf("error in finding document with id %s %s", id, err)
	}
	if !found {
		return fmt.Errorf("document not found %s", id)
	}
	_, err = ar.stock.RemoveDocument(
		driver.WithSilent(context.Background()),
		id,
	)
	if err != nil {
		return fmt.Errorf("error in removing document with id %s %s", id, err)
	}
	return nil
}

// LoadStrain will insert existing strain data into the database.
// It receives the already existing strain ID and the data to go with it.
func (ar *arangorepository) LoadStrain(id string, es *stock.ExistingStrain) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	var stmt string
	var bindVars map[string]interface{}
	if len(es.Data.Attributes.Parent) > 0 { // in case parent is present
		p := es.Data.Attributes.Parent
		pVars := map[string]interface{}{
			"@stock_collection": ar.stock.Name(),
			"id":                p,
		}
		r, err := ar.database.GetRow(statement.StockFindQ, pVars)
		if err != nil {
			return m, fmt.Errorf("error in searching for parent %s %s", p, err)
		}
		if r.IsEmpty() {
			return m, fmt.Errorf("parent %s is not found", p)
		}
		var pid string
		if err := r.Read(&pid); err != nil {
			return m, fmt.Errorf("error in scanning the value %s %s", p, err)
		}
		stmt = statement.StockStrainWithParentLoad
		bindVars = existingStrainBindParams(es.Data.Attributes)
		bindVars["stock_id"] = id
		bindVars["pid"] = pid
		bindVars["@parent_strain_collection"] = ar.parentStrain.Name()
		m.StrainProperties = &model.StrainProperties{Parent: p}
	} else {
		bindVars = existingStrainBindParams(es.Data.Attributes)
		bindVars["stock_id"] = id
		stmt = statement.StockStrainLoad
	}
	bindVars["@stock_collection"] = ar.stock.Name()
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

// LoadPlasmid will insert existing plasmid data into the database.
// It receives the already existing plasmid ID and the data to go with it.
func (ar *arangorepository) LoadPlasmid(id string, ep *stock.ExistingPlasmid) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	var bindVars map[string]interface{}
	attr := ep.Data.Attributes
	bindVars = existingPlasmidBindParams(attr)
	bindVars["@stock_collection"] = ar.stock.Name()
	bindVars["@stock_type_collection"] = ar.stockType.Name()
	bindVars["@stock_properties_collection"] = ar.stockProp.Name()
	bindVars["stock_id"] = id
	r, err := ar.database.DoRun(statement.StockPlasmidLoad, bindVars)
	if err != nil {
		return m, err
	}
	if err := r.Read(m); err != nil {
		return m, err
	}
	return m, nil
}

// ClearStocks clears all stocks from the repository datasource
func (ar *arangorepository) ClearStocks() error {
	if err := ar.stock.Truncate(context.Background()); err != nil {
		return err
	}
	return nil
}

func addablePlasmidBindParams(attr *stock.NewPlasmidAttributes) map[string]interface{} {
	bindVars := map[string]interface{}{
		"depositor":        attr.Depositor,
		"created_by":       attr.CreatedBy,
		"updated_by":       attr.UpdatedBy,
		"summary":          normalizeStrBindParam(attr.Summary),
		"editable_summary": normalizeStrBindParam(attr.EditableSummary),
		"genes":            normalizeSliceBindParam(attr.Genes),
		"dbxrefs":          normalizeSliceBindParam(attr.Dbxrefs),
		"publications":     normalizeSliceBindParam(attr.Publications),
		"image_map":        normalizeStrBindParam(attr.ImageMap),
		"sequence":         normalizeStrBindParam(attr.Sequence),
		"name":             attr.Name,
	}
	return bindVars
}

func addableStrainBindParams(attr *stock.NewStrainAttributes) map[string]interface{} {
	return map[string]interface{}{
		"summary":          normalizeStrBindParam(attr.Summary),
		"editable_summary": normalizeStrBindParam(attr.EditableSummary),
		"genes":            normalizeSliceBindParam(attr.Genes),
		"dbxrefs":          normalizeSliceBindParam(attr.Dbxrefs),
		"publications":     normalizeSliceBindParam(attr.Publications),
		"plasmid":          normalizeStrBindParam(attr.Plasmid),
		"names":            normalizeSliceBindParam(attr.Names),
		"depositor":        attr.Depositor,
		"label":            attr.Label,
		"species":          attr.Species,
		"created_by":       attr.CreatedBy,
		"updated_by":       attr.UpdatedBy,
	}
}

func existingPlasmidBindParams(attr *stock.ExistingPlasmidAttributes) map[string]interface{} {
	bindVars := map[string]interface{}{
		"created_at":       ptypes.TimestampString(attr.CreatedAt),
		"updated_at":       ptypes.TimestampString(attr.UpdatedAt),
		"depositor":        attr.Depositor,
		"created_by":       attr.CreatedBy,
		"updated_by":       attr.UpdatedBy,
		"summary":          normalizeStrBindParam(attr.Summary),
		"editable_summary": normalizeStrBindParam(attr.EditableSummary),
		"genes":            normalizeSliceBindParam(attr.Genes),
		"dbxrefs":          normalizeSliceBindParam(attr.Dbxrefs),
		"publications":     normalizeSliceBindParam(attr.Publications),
		"image_map":        normalizeStrBindParam(attr.ImageMap),
		"sequence":         normalizeStrBindParam(attr.Sequence),
		"name":             attr.Name,
	}
	return bindVars
}

func existingStrainBindParams(attr *stock.ExistingStrainAttributes) map[string]interface{} {
	return map[string]interface{}{
		"summary":          normalizeStrBindParam(attr.Summary),
		"editable_summary": normalizeStrBindParam(attr.EditableSummary),
		"genes":            normalizeSliceBindParam(attr.Genes),
		"dbxrefs":          normalizeSliceBindParam(attr.Dbxrefs),
		"publications":     normalizeSliceBindParam(attr.Publications),
		"plasmid":          normalizeStrBindParam(attr.Plasmid),
		"names":            normalizeSliceBindParam(attr.Names),
		"depositor":        attr.Depositor,
		"label":            attr.Label,
		"species":          attr.Species,
		"created_by":       attr.CreatedBy,
		"updated_by":       attr.UpdatedBy,
		"created_at":       ptypes.TimestampString(attr.CreatedAt),
		"updated_at":       ptypes.TimestampString(attr.UpdatedAt),
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

func getUpdatablePlasmidBindParams(attr *stock.PlasmidUpdateAttributes) map[string]interface{} {
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

func getUpdatableStrainBindParams(attr *stock.StrainUpdateAttributes) map[string]interface{} {
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

func getUpdatableStrainPropBindParams(attr *stock.StrainUpdateAttributes) map[string]interface{} {
	bindVars := make(map[string]interface{})
	if len(attr.Label) > 0 {
		bindVars["label"] = attr.Label
	}
	if len(attr.Species) > 0 {
		bindVars["species"] = attr.Species
	}
	if len(attr.Plasmid) > 0 {
		bindVars["plasmid"] = attr.Plasmid
	}
	if len(attr.Names) > 0 {
		bindVars["names"] = attr.Names
	}
	return bindVars
}

func getUpdatablePlasmidPropBindParams(attr *stock.PlasmidUpdateAttributes) map[string]interface{} {
	bindVars := make(map[string]interface{})
	if len(attr.ImageMap) > 0 {
		bindVars["image_map"] = attr.ImageMap
	}
	if len(attr.Sequence) > 0 {
		bindVars["sequence"] = attr.Sequence
	}
	if len(attr.Name) > 0 {
		bindVars["name"] = attr.Name
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

func genAQLDocExpression(bindVars map[string]interface{}) string {
	var bindParams []string
	for k := range bindVars {
		bindParams = append(bindParams, fmt.Sprintf("%s: @%s", k, k))
	}
	return strings.Join(bindParams, ",")
}
