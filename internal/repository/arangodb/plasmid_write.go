package arangodb

import (
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

// LoadPlasmid will insert existing plasmid data into the database.
// It receives the already existing plasmid ID and the data to go with it.
func (ar *arangorepository) LoadPlasmid(
	id string,
	ep *stock.ExistingPlasmid,
) (*model.StockDoc, error) {
	stockDoc := &model.StockDoc{PlasmidProperties: &model.PlasmidProperties{}}
	tid, err := ar.termID(
		ep.Data.Attributes.DictyPlasmidProperty,
		ar.plasmidOnto,
	)
	if err != nil {
		return stockDoc, err
	}
	bindVars := mergeBindParams(map[string]any{
		"stock_id":                     id,
		"@stock_collection":            ar.stockc.stock.Name(),
		"@stock_type_collection":       ar.stockc.stockType.Name(),
		"@stock_properties_collection": ar.stockc.stockProp.Name(),
		"@stock_term_collection":       ar.stockc.stockTerm.Name(),
		"to":                           tid,
	}, existingPlasmidBindParams(ep.Data.Attributes))
	r, err := ar.database.DoRun(statement.StockPlasmidLoad, bindVars)
	if err != nil {
		return stockDoc, err
	}
	if err := r.Read(stockDoc); err != nil {
		return stockDoc, err
	}
	return stockDoc, nil
}

// EditPlasmid updates an existing plasmid
func (ar *arangorepository) EditPlasmid(
	us *stock.PlasmidUpdate,
) (*model.StockDoc, error) {
	stockDoc := &model.StockDoc{}
	propKey, err := ar.checkStock(us.Data.Id)
	if err != nil {
		return stockDoc, err
	}
	// term is the ontology term for the plasmid
	term := us.Data.Attributes.DictyPlasmidProperty
	if len(term) > 0 {
		tid, tidErr := ar.termID(term, ar.plasmidOnto)
		if tidErr != nil {
			return stockDoc, tidErr
		}

		// Run the UPSERT query to update the ontology term
		_, err = ar.database.DoRun(
			statement.PlasmidTermUpd,
			map[string]any{
				// collection bind var for @@stock_term_collection
				"@stock_term_collection": ar.stockc.stockTerm.Name(),
				// string bind var for CONCAT(@stock_collection, '/', @key)
				"stock_collection": ar.stockc.stock.Name(),
				"key":              us.Data.Id,
				"to":               tid,
			},
		)
		if err != nil {
			return stockDoc, err
		}
	}
	bindVars := getUpdatablePlasmidBindParams(us.Data.Attributes)
	bindPlVars := getUpdatablePlasmidPropBindParams(us.Data.Attributes)
	cmBindVars := mergeBindParams(
		map[string]any{
			"@stock_properties_collection": ar.stockc.stockProp.Name(),
			"@stock_collection":            ar.stockc.stock.Name(),
			"key":                          us.Data.Id,
			"propkey":                      propKey,
		},
		bindVars, bindPlVars,
	)
	rupd, err := ar.database.DoRun(
		fmt.Sprintf(
			statement.PlasmidUpd,
			genAQLDocExpression(bindVars),
			genAQLDocExpression(bindPlVars),
		), cmBindVars)
	if err != nil {
		return stockDoc, err
	}
	if err := rupd.Read(stockDoc); err != nil {
		return stockDoc, err
	}
	return stockDoc, nil
}

// AddPlasmid creates a new plasmid stock
func (ar *arangorepository) AddPlasmid(
	ns *stock.NewPlasmid,
) (*model.StockDoc, error) {
	stockDoc := &model.StockDoc{PlasmidProperties: &model.PlasmidProperties{}}
	bindVars := mergeBindParams(map[string]any{
		"@stock_collection":            ar.stockc.stock.Name(),
		"@stock_key_generator":         ar.stockc.stockKey.Name(),
		"@stock_type_collection":       ar.stockc.stockType.Name(),
		"@stock_properties_collection": ar.stockc.stockProp.Name(),
	}, addablePlasmidBindParams(ns.Data.Attributes))
	tid, err := ar.termID(
		ns.Data.Attributes.DictyPlasmidProperty,
		ar.plasmidOnto,
	)
	if err != nil {
		return stockDoc, err
	}
	bindVars = mergeBindParams(bindVars, map[string]any{
		"to":                     tid,
		"@stock_term_collection": ar.stockc.stockTerm.Name(),
	})
	r, err := ar.database.DoRun(statement.StockPlasmidIns, bindVars)
	if err != nil {
		return stockDoc, err
	}
	if err := r.Read(stockDoc); err != nil {
		return stockDoc, err
	}
	return stockDoc, nil
}

func addablePlasmidBindParams(
	attr *stock.NewPlasmidAttributes,
) map[string]any {
	return map[string]any{
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
}

func existingPlasmidBindParams(
	attr *stock.ExistingPlasmidAttributes,
) map[string]any {
	return map[string]any{
		"created_at":       attr.CreatedAt.AsTime().UnixMilli(),
		"updated_at":       attr.UpdatedAt.AsTime().UnixMilli(),
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
}

func getUpdatablePlasmidBindParams(
	attr *stock.PlasmidUpdateAttributes,
) map[string]any {
	bindVars := map[string]any{
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

func getUpdatablePlasmidPropBindParams(
	attr *stock.PlasmidUpdateAttributes,
) map[string]any {
	bindVars := make(map[string]any)
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
