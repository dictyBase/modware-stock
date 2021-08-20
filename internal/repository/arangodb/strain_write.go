package arangodb

import (
	"context"
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

// AddStrain creates a new strain stock
func (ar *arangorepository) AddStrain(ns *stock.NewStrain) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	stmt := statement.StockStrainIns
	bindVars := addableStrainBindParams(ns.Data.Attributes)
	bindVars["@stock_collection"] = ar.stockc.stock.Name()
	bindVars["@stock_key_generator"] = ar.stockc.stockKey.Name()
	bindVars["@stock_properties_collection"] = ar.stockc.stockProp.Name()
	bindVars["@stock_type_collection"] = ar.stockc.stockType.Name()
	if len(ns.Data.Attributes.Parent) > 0 { // in case parent is present
		p := ns.Data.Attributes.Parent
		pVars, err := ar.processStrainWithParent(p)
		if err != nil {
			return m, err
		}
		bindVars = mergeBindParams(bindVars, pVars)
		m.StrainProperties = &model.StrainProperties{Parent: p}
		stmt = statement.StockStrainWithParentsIns
	}
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
	r, err := ar.database.GetRow(
		statement.StockFindIdQ,
		map[string]interface{}{
			"stock_collection": ar.stockc.stock.Name(),
			"graph":            ar.stockc.stockPropType.Name(),
			"stock_id":         us.Data.Id,
		})
	if err != nil {
		return m,
			fmt.Errorf("error in finding strain id %s %s", us.Data.Id, err)
	}
	if r.IsEmpty() {
		return m,
			fmt.Errorf("strain id %s is absent in database", us.Data.Id)
	}
	var propKey string
	if err := r.Read(&propKey); err != nil {
		return m,
			fmt.Errorf("error in reading using strain id %s %s", us.Data.Id, err)
	}
	bindVars := getUpdatableStrainBindParams(us.Data.Attributes)
	bindStVars := getUpdatableStrainPropBindParams(us.Data.Attributes)
	cmBindVars := mergeBindParams(
		map[string]interface{}{
			"@stock_properties_collection": ar.stockc.stockProp.Name(),
			"@stock_collection":            ar.stockc.stock.Name(),
			"key":                          us.Data.Id,
			"propkey":                      propKey,
		},
		bindVars, bindStVars,
	)
	parent := us.Data.Attributes.Parent
	stmt := statement.StrainUpd
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
	}
	rupd, err := ar.database.DoRun(
		fmt.Sprintf(
			stmt,
			genAQLDocExpression(bindVars),
			genAQLDocExpression(bindStVars),
		),
		cmBindVars,
	)
	if err != nil {
		return m, fmt.Errorf(
			"error in editing strain %s %s",
			us.Data.Id, err,
		)
	}
	err = rupd.Read(m)
	return m, err
}

// LoadStrain will insert existing strain data into the database.
// It receives the already existing strain ID and the data to go with it.
func (ar *arangorepository) LoadStrain(id string, es *stock.ExistingStrain) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	stmt := statement.StockStrainLoad
	bindVars := existingStrainBindParams(es.Data.Attributes)
	if len(es.Data.Attributes.Parent) > 0 { // in case parent is present
		p := es.Data.Attributes.Parent
		r, err := ar.database.GetRow(
			statement.StockFindQ,
			map[string]interface{}{
				"@stock_collection": ar.stockc.stock.Name(),
				"id":                p,
			})
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
		bindVars["pid"] = pid
		bindVars["@parent_strain_collection"] = ar.stockc.parentStrain.Name()
		m.StrainProperties = &model.StrainProperties{Parent: p}
	}
	bindVars["stock_id"] = id
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
		"created_at":       attr.CreatedAt.AsTime().UnixMilli(),
		"updated_at":       attr.UpdatedAt.AsTime().UnixMilli(),
		"depositor":        attr.Depositor,
		"label":            attr.Label,
		"species":          attr.Species,
		"created_by":       attr.CreatedBy,
		"updated_by":       attr.UpdatedBy,
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

func (ar *arangorepository) processStrainWithParent(parent string) (map[string]interface{}, error) {
	qVar := map[string]interface{}{
		"@stock_collection": ar.stockc.stock.Name(),
		"id":                parent,
	}
	r, err := ar.database.GetRow(statement.StockFindQ, qVar)
	if err != nil {
		return qVar,
			fmt.Errorf("error in searching for parent %s %s", parent, err)
	}
	if r.IsEmpty() {
		return qVar, fmt.Errorf("parent %s is not found", parent)
	}
	var pid string
	if err := r.Read(&pid); err != nil {
		return qVar,
			fmt.Errorf("error in scanning the value %s %s", parent, err)
	}
	return map[string]interface{}{
		"pid":                       pid,
		"@parent_strain_collection": ar.stockc.parentStrain.Name(),
	}, nil
}
