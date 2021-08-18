package arangodb

import (
	"context"
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
	"github.com/golang/protobuf/ptypes"
)

// GetStrain retrieves a strain from the database
func (ar *arangorepository) GetStrain(id string) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	bindVars := map[string]interface{}{
		"id":                id,
		"@stock_collection": ar.stockc.stock.Name(),
		"stock_collection":  ar.stockc.stock.Name(),
		"parent_graph":      ar.stockc.strain2Parent.Name(),
		"prop_graph":        ar.stockc.stockPropType.Name(),
	}
	g, err := ar.database.GetRow(
		statement.StockFindQ,
		map[string]interface{}{
			"id":                id,
			"@stock_collection": ar.stockc.stock.Name(),
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
	err = r.Read(m)
	return m, err
}

// AddStrain creates a new strain stock
func (ar *arangorepository) AddStrain(ns *stock.NewStrain) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	var stmt string
	var bindVars map[string]interface{}
	if len(ns.Data.Attributes.Parent) > 0 { // in case parent is present
		p := ns.Data.Attributes.Parent
		pVars := map[string]interface{}{
			"@stock_collection": ar.stockc.stock.Name(),
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
		bindVars["@parent_strain_collection"] = ar.stockc.parentStrain.Name()
		m.StrainProperties = &model.StrainProperties{Parent: p}
	} else {
		bindVars = addableStrainBindParams(ns.Data.Attributes)
		stmt = statement.StockStrainIns
	}
	bindVars["@stock_collection"] = ar.stockc.stock.Name()
	bindVars["@stock_key_generator"] = ar.stockc.stockKey.Name()
	bindVars["@stock_properties_collection"] = ar.stockc.stockProp.Name()
	bindVars["@stock_type_collection"] = ar.stockc.stockType.Name()
	r, err := ar.database.DoRun(stmt, bindVars)
	if err != nil {
		return m, err
	}
	err = r.Read(m)
	return m, err
}

// EditStrain updates an existing strain
func (ar *arangorepository) EditStrain(us *stock.StrainUpdate) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	r, err := ar.database.GetRow(statement.StockFindIdQ,
		map[string]interface{}{
			"stock_collection": ar.stockc.stock.Name(),
			"graph":            ar.stockc.stockPropType.Name(),
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
		ok, err := ar.stockc.stock.DocumentExists(context.Background(), parent)
		if err != nil {
			return m, fmt.Errorf("error in checking for parent id %s %s", parent, err)
		}
		if !ok {
			return m, fmt.Errorf("parent id %s does not exist in database", parent)
		}
		r, err := ar.database.GetRow(
			statement.StrainGetParentRel,
			map[string]interface{}{
				"parent_graph": ar.stockc.strain2Parent.Name(),
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
		cmBindVars["stock_collection"] = ar.stockc.stock.Name()
		cmBindVars["@parent_strain_collection"] = ar.stockc.parentStrain.Name()
		m.StrainProperties = &model.StrainProperties{Parent: parent}
	} else {
		stmt = statement.StrainUpd
	}
	cmBindVars["@stock_properties_collection"] = ar.stockc.stockProp.Name()
	cmBindVars["@stock_collection"] = ar.stockc.stock.Name()
	cmBindVars["propkey"] = propKey
	cmBindVars["key"] = us.Data.Id
	q := fmt.Sprintf(
		stmt,
		genAQLDocExpression(bindVars),
		genAQLDocExpression(bindStVars),
	)
	rupd, err := ar.database.DoRun(q, cmBindVars)
	if err != nil {
		return m, fmt.Errorf(
			"error in editing strain %s %s %s",
			us.Data.Id, err, q,
		)
	}
	err = rupd.Read(m)
	return m, err
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
				ar.stockc.stock.Name(),
				ar.stockc.stockPropType.Name(),
				f, l+1,
			)
		} else { // else include both filter and cursor
			stmt = fmt.Sprintf(
				statement.StrainListFilterWithCursor,
				ar.stockc.stock.Name(),
				ar.stockc.stockPropType.Name(),
				f, c, l+1,
			)
		}
	} else {
		// otherwise use query statement without filter
		if c == 0 { // no cursor so return first set of result
			stmt = fmt.Sprintf(
				statement.StrainList,
				ar.stockc.stock.Name(),
				ar.stockc.stockPropType.Name(),
				l+1,
			)
		} else { // add cursor if it exists
			stmt = fmt.Sprintf(
				statement.StrainListWithCursor,
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

func (ar *arangorepository) ListStrainsByIds(p *stock.StockIdList) ([]*model.StockDoc, error) {
	ms := make([]*model.StockDoc, 0)
	ids := make([]string, 0)
	for _, v := range p.Id {
		ids = append(ids, v)
	}
	bindVars := map[string]interface{}{
		"ids":                 ids,
		"limit":               len(ids),
		"stock_collection":    ar.stockc.stock.Name(),
		"@stock_collection":   ar.stockc.stock.Name(),
		"prop_graph":          ar.stockc.stockPropType.Name(),
		"par.stockcent_graph": ar.stockc.strain2Parent.Name(),
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

// LoadStrain will insert existing strain data into the database.
// It receives the already existing strain ID and the data to go with it.
func (ar *arangorepository) LoadStrain(id string, es *stock.ExistingStrain) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	var stmt string
	var bindVars map[string]interface{}
	if len(es.Data.Attributes.Parent) > 0 { // in case parent is present
		p := es.Data.Attributes.Parent
		pVars := map[string]interface{}{
			"id":                p,
			"@stock_collection": ar.stockc.stock.Name(),
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
		bindVars["@parent_strain_collection"] = ar.stockc.parentStrain.Name()
		m.StrainProperties = &model.StrainProperties{Parent: p}
	} else {
		bindVars = existingStrainBindParams(es.Data.Attributes)
		bindVars["stock_id"] = id
		stmt = statement.StockStrainLoad
	}
	bindVars["@stock_collection"] = ar.stockc.stock.Name()
	bindVars["@stock_properties_collection"] = ar.stockc.stockProp.Name()
	bindVars["@stock_type_collection"] = ar.stockc.stockType.Name()
	r, err := ar.database.DoRun(stmt, bindVars)
	if err != nil {
		return m, err
	}
	err = r.Read(m)
	return m, err
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