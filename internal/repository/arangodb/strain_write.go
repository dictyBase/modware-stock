package arangodb

import (
	"context"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

// AddStrain creates a new strain stock
func (ar *arangorepository) AddStrain(ns *stock.NewStrain) (*model.StockDoc, error) {
	m := &model.StockDoc{
		StrainProperties: &model.StrainProperties{
			DictyStrainProperty: ns.Data.Attributes.DictyStrainProperty,
		},
	}
	tid, err := ar.termID(ns.Data.Attributes.DictyStrainProperty, ar.strainOnto)
	if err != nil {
		return m, err
	}
	stmt := statement.StockStrainIns
	bindVars := mergeBindParams(map[string]interface{}{
		"to":                           tid,
		"@stock_collection":            ar.stockc.stock.Name(),
		"@stock_key_generator":         ar.stockc.stockKey.Name(),
		"@stock_properties_collection": ar.stockc.stockProp.Name(),
		"@stock_type_collection":       ar.stockc.stockType.Name(),
		"@stock_term_collection":       ar.stockc.stockTerm.Name(),
	}, addableStrainBindParams(ns.Data.Attributes))
	if len(ns.Data.Attributes.Parent) > 0 { // in case parent is present
		p := ns.Data.Attributes.Parent
		pVars, err := ar.handleAddStrainWithParent(p)
		if err != nil {
			return m, err
		}
		bindVars = mergeBindParams(bindVars, pVars)
		m.StrainProperties.Parent = p
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
	propKey, err := ar.checkStock(us.Data.Id)
	if err != nil {
		return m, err
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
		pVars, pStmt, err := ar.handleEditStrainWithParent(parent, us.Data.Id)
		if err != nil {
			return m, err
		}
		stmt = pStmt
		cmBindVars = mergeBindParams(cmBindVars, pVars)
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
		return m, errors.Errorf(
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
	bindVars := mergeBindParams(map[string]interface{}{
		"stock_id":                     id,
		"@stock_collection":            ar.stockc.stock.Name(),
		"@stock_properties_collection": ar.stockc.stockProp.Name(),
		"@stock_type_collection":       ar.stockc.stockType.Name(),
	}, existingStrainBindParams(es.Data.Attributes))
	if len(es.Data.Attributes.Parent) > 0 { // in case parent is present
		p := es.Data.Attributes.Parent
		pVars, err := ar.handleAddStrainWithParent(p)
		if err != nil {
			return m, err
		}
		bindVars = mergeBindParams(bindVars, pVars)
		m.StrainProperties = &model.StrainProperties{Parent: p}
		stmt = statement.StockStrainWithParentLoad
	}
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

func (ar *arangorepository) handleEditStrainWithParent(parent, id string) (map[string]interface{}, string, error) {
	pVar := map[string]interface{}{
		"parent_graph": ar.stockc.strain2Parent.Name(),
		"strain_key":   id,
	}
	if err := ar.validateParent(parent); err != nil {
		return pVar, "", err
	}
	r, err := ar.database.GetRow(statement.StrainGetParentRel, pVar)
	if err != nil {
		return pVar,
			"",
			errors.Errorf("error in parent relation query %s", err)
	}
	var pKey string
	if !r.IsEmpty() {
		if err := r.Read(&pKey); err != nil {
			return pVar,
				"",
				errors.Errorf("error in reading parent relation key %s", err)
		}
	}
	stmt := statement.StrainWithNewParentUpd
	cmBindVars := map[string]interface{}{
		"parent":                    parent,
		"stock_collection":          ar.stockc.stock.Name(),
		"@parent_strain_collection": ar.stockc.parentStrain.Name(),
	}
	if len(pKey) > 0 {
		stmt = statement.StrainWithExistingParentUpd
		cmBindVars["pkey"] = pKey
	}
	return cmBindVars, stmt, nil
}

func (ar *arangorepository) validateParent(parent string) error {
	ok, err := ar.stockc.stock.DocumentExists(context.Background(), parent)
	if err != nil {
		return errors.Errorf("error in checking for parent id %s %s", parent, err)
	}
	if !ok {
		return errors.Errorf("parent id %s does not exist in database", parent)
	}
	return nil
}

func (ar *arangorepository) handleAddStrainWithParent(parent string) (map[string]interface{}, error) {
	qVar := map[string]interface{}{
		"@stock_collection": ar.stockc.stock.Name(),
		"id":                parent,
	}
	r, err := ar.database.GetRow(statement.StockFindQ, qVar)
	if err != nil {
		return qVar,
			errors.Errorf("error in searching for parent %s %s", parent, err)
	}
	if r.IsEmpty() {
		return qVar, errors.Errorf("parent %s is not found", parent)
	}
	var pid string
	if err := r.Read(&pid); err != nil {
		return qVar,
			errors.Errorf("error in scanning the value %s %s", parent, err)
	}
	return map[string]interface{}{
		"pid":                       pid,
		"@parent_strain_collection": ar.stockc.parentStrain.Name(),
	}, nil
}
