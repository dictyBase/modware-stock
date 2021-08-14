package arangodb

import (
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
	"github.com/golang/protobuf/ptypes"
)

// LoadPlasmid will insert existing plasmid data into the database.
// It receives the already existing plasmid ID and the data to go with it.
func (ar *arangorepository) LoadPlasmid(id string, ep *stock.ExistingPlasmid) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	var bindVars map[string]interface{}
	attr := ep.Data.Attributes
	bindVars = existingPlasmidBindParams(attr)
	bindVars["@stock_collection"] = ar.stockc.stock.Name()
	bindVars["@stock_type_collection"] = ar.stockc.stockType.Name()
	bindVars["@stock_properties_collection"] = ar.stockc.stockProp.Name()
	bindVars["stock_id"] = id
	r, err := ar.database.DoRun(statement.StockPlasmidLoad, bindVars)
	if err != nil {
		return m, err
	}
	err = r.Read(m)
	return m, err
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
				ar.stockc.stock.Name(),
				ar.stockc.stockPropType.Name(),
				f, l+1,
			)
		} else { // else include both filter and cursor
			stmt = fmt.Sprintf(
				statement.PlasmidListFilterWithCursor,
				ar.stockc.stock.Name(),
				ar.stockc.stockPropType.Name(),
				f, c, l+1,
			)
		}
	} else {
		// otherwise use query statement without filter
		if c == 0 { // no cursor so return first set of result
			stmt = fmt.Sprintf(
				statement.PlasmidList,
				ar.stockc.stock.Name(),
				ar.stockc.stockPropType.Name(),
				l+1,
			)
		} else { // add cursor if it exists
			stmt = fmt.Sprintf(
				statement.PlasmidListWithCursor,
				ar.stockc.stock.Name(),
				ar.stockc.stockPropType.Name(),
				c, l+1,
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

// EditPlasmid updates an existing plasmid
func (ar *arangorepository) EditPlasmid(us *stock.PlasmidUpdate) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	r, err := ar.database.GetRow(
		statement.StockFindIdQ,
		map[string]interface{}{
			"stock_collection": ar.stockc.stock.Name(),
			"graph":            ar.stockc.stockPropType.Name(),
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
	cmBindVars["@stock_properties_collection"] = ar.stockc.stockProp.Name()
	cmBindVars["propkey"] = propKey
	cmBindVars["@stock_collection"] = ar.stockc.stock.Name()
	cmBindVars["key"] = us.Data.Id
	rupd, err := ar.database.DoRun(stmt, cmBindVars)
	if err != nil {
		return m, err
	}
	err = rupd.Read(m)
	return m, err
}

// AddPlasmid creates a new plasmid stock
func (ar *arangorepository) AddPlasmid(ns *stock.NewPlasmid) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	attr := ns.Data.Attributes
	bindVars := addablePlasmidBindParams(attr)
	bindVars["@stock_collection"] = ar.stockc.stock.Name()
	bindVars["@stock_key_generator"] = ar.stockc.stockKey.Name()
	bindVars["@stock_type_collection"] = ar.stockc.stockType.Name()
	bindVars["@stock_properties_collection"] = ar.stockc.stockProp.Name()
	r, err := ar.database.DoRun(statement.StockPlasmidIns, bindVars)
	if err != nil {
		return m, err
	}
	err = r.Read(m)
	return m, err
}

// GetPlasmid retrieves a plasmid from the database
func (ar *arangorepository) GetPlasmid(id string) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	bindVars := map[string]interface{}{
		"@stock_collection": ar.stockc.stock.Name(),
		"id":                id,
		"graph":             ar.stockc.stockPropType.Name(),
	}
	r, err := ar.database.GetRow(statement.StockGetPlasmid, bindVars)
	if err != nil {
		return m, err
	}
	if r.IsEmpty() {
		m.NotFound = true
		return m, nil
	}
	err = r.Read(m)
	return m, err
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
