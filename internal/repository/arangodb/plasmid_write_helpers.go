package arangodb

import (
	"fmt"

	IOE "github.com/IBM/fp-go/ioeither"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

// Helper types for AddPlasmid

// addPlasmidWithTermID encapsulates new plasmid with validated term ID
type addPlasmidWithTermID struct {
	plasmid *stock.NewPlasmid
	termID  string
}

// validateAddPlasmidOntologyTerm validates the ontology term for new plasmid
func (ar *arangorepository) validateAddPlasmidOntologyTerm(
	plasmid *stock.NewPlasmid,
) IOE.IOEither[error, addPlasmidWithTermID] {
	return IOE.TryCatchError(
		func() (addPlasmidWithTermID, error) {
			tid, err := ar.termID(
				plasmid.Data.Attributes.DictyPlasmidProperty,
				ar.plasmidOnto,
			)
			if err != nil {
				return addPlasmidWithTermID{}, fmt.Errorf(
					"failed to validate ontology term: %w",
					err,
				)
			}
			return addPlasmidWithTermID{
				plasmid: plasmid,
				termID:  tid,
			}, nil
		},
	)
}

// buildAddPlasmidParams constructs bind parameters for add plasmid query
func (ar *arangorepository) buildAddPlasmidParams(
	params addPlasmidWithTermID,
) map[string]any {
	attr := params.plasmid.Data.Attributes
	return map[string]any{
		"@stock_collection":            ar.stockc.stock.Name(),
		"@stock_key_generator":         ar.stockc.stockKey.Name(),
		"@stock_type_collection":       ar.stockc.stockType.Name(),
		"@stock_properties_collection": ar.stockc.stockProp.Name(),
		"to":                           params.termID,
		"@stock_term_collection":       ar.stockc.stockTerm.Name(),
		"depositor":                    attr.Depositor,
		"created_by":                   attr.CreatedBy,
		"updated_by":                   attr.UpdatedBy,
		"summary":                      normalizeStrBindParam(attr.Summary),
		"editable_summary": normalizeStrBindParam(
			attr.EditableSummary,
		),
		"genes":   normalizeSliceBindParam(attr.Genes),
		"dbxrefs": normalizeSliceBindParam(attr.Dbxrefs),
		"publications": normalizeSliceBindParam(
			attr.Publications,
		),
		"image_map": normalizeStrBindParam(attr.ImageMap),
		"sequence":  normalizeStrBindParam(attr.Sequence),
		"name":      attr.Name,
	}
}

// executeAddPlasmidQuery executes the add plasmid query
func (ar *arangorepository) executeAddPlasmidQuery(
	bindVars map[string]any,
) IOE.IOEither[error, *model.StockDoc] {
	return IOE.TryCatchError(
		func() (*model.StockDoc, error) {
			stockDoc := &model.StockDoc{
				PlasmidProperties: &model.PlasmidProperties{},
			}
			row, err := ar.database.DoRun(
				statement.StockPlasmidIns,
				bindVars,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"database insert failed: %w",
					err,
				)
			}
			if err := row.Read(stockDoc); err != nil {
				return nil, fmt.Errorf(
					"failed to read stock document: %w",
					err,
				)
			}
			return stockDoc, nil
		},
	)
}

// Helper types for EditPlasmid

// editPlasmidWithPropKey encapsulates update with validated property key
type editPlasmidWithPropKey struct {
	update  *stock.PlasmidUpdate
	propKey string
}

// validateEditPlasmidStock validates that the stock exists and returns property key
func (ar *arangorepository) validateEditPlasmidStock(
	update *stock.PlasmidUpdate,
) IOE.IOEither[error, editPlasmidWithPropKey] {
	return IOE.TryCatchError(
		func() (editPlasmidWithPropKey, error) {
			propKey, err := ar.checkStock(update.Data.Id)
			if err != nil {
				return editPlasmidWithPropKey{}, fmt.Errorf(
					"stock validation failed: %w",
					err,
				)
			}
			return editPlasmidWithPropKey{
				update:  update,
				propKey: propKey,
			}, nil
		},
	)
}

// updatePlasmidOntologyTerm updates the ontology term if provided
func (ar *arangorepository) updatePlasmidOntologyTerm(
	params editPlasmidWithPropKey,
) IOE.IOEither[error, editPlasmidWithPropKey] {
	term := params.update.Data.Attributes.DictyPlasmidProperty
	if len(term) == 0 {
		return IOE.Of[error](params)
	}

	return IOE.TryCatchError(
		func() (editPlasmidWithPropKey, error) {
			tid, err := ar.termID(term, ar.plasmidOnto)
			if err != nil {
				return editPlasmidWithPropKey{}, fmt.Errorf(
					"failed to validate ontology term: %w",
					err,
				)
			}

			_, err = ar.database.DoRun(
				statement.PlasmidTermUpd,
				map[string]any{
					"@stock_term_collection": ar.stockc.stockTerm.Name(),
					"stock_collection":       ar.stockc.stock.Name(),
					"key":                    params.update.Data.Id,
					"to":                     tid,
				},
			)
			if err != nil {
				return editPlasmidWithPropKey{}, fmt.Errorf(
					"failed to update plasmid ontology term for %s: %w",
					params.update.Data.Id,
					err,
				)
			}

			return params, nil
		},
	)
}

// editPlasmidQueryParams holds the query statement and bind parameters
type editPlasmidQueryParams struct {
	statement  string
	bindParams map[string]any
}

// buildEditPlasmidParams constructs bind parameters for edit plasmid query
func (ar *arangorepository) buildEditPlasmidParams(
	params editPlasmidWithPropKey,
) editPlasmidQueryParams {
	bindVars := getUpdatablePlasmidBindParams(params.update.Data.Attributes)
	bindPlVars := getUpdatablePlasmidPropBindParams(
		params.update.Data.Attributes,
	)

	return editPlasmidQueryParams{
		statement: fmt.Sprintf(
			statement.PlasmidUpd,
			genAQLDocExpression(bindVars),
			genAQLDocExpression(bindPlVars),
		),
		bindParams: mergeBindParams(
			map[string]any{
				"@stock_properties_collection": ar.stockc.stockProp.Name(),
				"@stock_collection":            ar.stockc.stock.Name(),
				"key":                          params.update.Data.Id,
				"propkey":                      params.propKey,
			},
			bindVars,
			bindPlVars,
		),
	}
}

// executeEditPlasmidQuery executes the edit plasmid query
func (ar *arangorepository) executeEditPlasmidQuery(
	queryParams editPlasmidQueryParams,
) IOE.IOEither[error, *model.StockDoc] {
	return IOE.TryCatchError(
		func() (*model.StockDoc, error) {
			stockDoc := &model.StockDoc{}
			row, err := ar.database.DoRun(
				queryParams.statement,
				queryParams.bindParams,
			)
			if err != nil {
				return nil, fmt.Errorf("database update failed: %w", err)
			}
			if err := row.Read(stockDoc); err != nil {
				return nil, fmt.Errorf("failed to read stock document: %w", err)
			}
			return stockDoc, nil
		},
	)
}

// Helper types for LoadPlasmid

// loadPlasmidParams holds the initial load plasmid parameters
type loadPlasmidParams struct {
	id      string
	plasmid *stock.ExistingPlasmid
}

// loadPlasmidWithTermID encapsulates load params with validated term ID
type loadPlasmidWithTermID struct {
	id      string
	plasmid *stock.ExistingPlasmid
	termID  string
}

// validateLoadPlasmidOntologyTerm validates the ontology term for loading plasmid
func (ar *arangorepository) validateLoadPlasmidOntologyTerm(
	params loadPlasmidParams,
) IOE.IOEither[error, loadPlasmidWithTermID] {
	return IOE.TryCatchError(
		func() (loadPlasmidWithTermID, error) {
			tid, err := ar.termID(
				params.plasmid.Data.Attributes.DictyPlasmidProperty,
				ar.plasmidOnto,
			)
			if err != nil {
				return loadPlasmidWithTermID{}, fmt.Errorf(
					"failed to validate ontology term: %w",
					err,
				)
			}
			return loadPlasmidWithTermID{
				id:      params.id,
				plasmid: params.plasmid,
				termID:  tid,
			}, nil
		},
	)
}

// buildLoadPlasmidParams constructs bind parameters for load plasmid query
func (ar *arangorepository) buildLoadPlasmidParams(
	params loadPlasmidWithTermID,
) map[string]any {
	return mergeBindParams(
		map[string]any{
			"stock_id":                     params.id,
			"@stock_collection":            ar.stockc.stock.Name(),
			"@stock_type_collection":       ar.stockc.stockType.Name(),
			"@stock_properties_collection": ar.stockc.stockProp.Name(),
			"@stock_term_collection":       ar.stockc.stockTerm.Name(),
			"to":                           params.termID,
		},
		existingPlasmidBindParams(params.plasmid.Data.Attributes),
	)
}

// executeLoadPlasmidQuery executes the load plasmid query
func (ar *arangorepository) executeLoadPlasmidQuery(
	bindVars map[string]any,
) IOE.IOEither[error, *model.StockDoc] {
	return IOE.TryCatchError(
		func() (*model.StockDoc, error) {
			stockDoc := &model.StockDoc{
				PlasmidProperties: &model.PlasmidProperties{},
			}
			row, err := ar.database.DoRun(statement.StockPlasmidLoad, bindVars)
			if err != nil {
				return nil, fmt.Errorf("database insert failed: %w", err)
			}
			if err := row.Read(stockDoc); err != nil {
				return nil, fmt.Errorf("failed to read stock document: %w", err)
			}
			return stockDoc, nil
		},
	)
}

// Parameter building helpers

// existingPlasmidBindParams converts ExistingPlasmidAttributes to ArangoDB bind
// parameters for loading pre-existing plasmids with specific IDs. Timestamps are
// converted to milliseconds for ArangoDB storage.
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

// optionalStrParam creates a map entry only if the string value is non-empty.
// Returns an empty map if the value is empty, enabling clean Semigroup composition.
func optionalStrParam(key, value string) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	return map[string]any{key: value}
}

// optionalSliceParam creates a map entry only if the slice is non-empty.
// Returns an empty map if the slice is empty, enabling clean Semigroup composition.
func optionalSliceParam(key string, value []string) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	return map[string]any{key: value}
}

// getUpdatablePlasmidBindParams extracts updatable stock fields from
// PlasmidUpdateAttributes. Only non-empty fields are included in the returned map
// to enable partial updates without overwriting existing data.
// Uses curried Semigroup composition for point-free, declarative parameter building.
func getUpdatablePlasmidBindParams(
	attr *stock.PlasmidUpdateAttributes,
) map[string]any {
	return concatOptionalParams(
		map[string]any{"updated_by": attr.UpdatedBy},
	)([]map[string]any{
		optionalStrParam("summary", attr.Summary),
		optionalStrParam("editable_summary", attr.EditableSummary),
		optionalStrParam("depositor", attr.Depositor),
		optionalSliceParam("genes", attr.Genes),
		optionalSliceParam("dbxrefs", attr.Dbxrefs),
		optionalSliceParam("publications", attr.Publications),
	})
}

// getUpdatablePlasmidPropBindParams extracts updatable plasmid property fields
// from PlasmidUpdateAttributes. Only non-empty fields are included in the returned map
// to enable partial updates of plasmid-specific properties.
// Uses curried Semigroup composition for point-free, declarative parameter building.
func getUpdatablePlasmidPropBindParams(
	attr *stock.PlasmidUpdateAttributes,
) map[string]any {
	return concatOptionalParams(
		map[string]any{},
	)([]map[string]any{
		optionalStrParam("image_map", attr.ImageMap),
		optionalStrParam("sequence", attr.Sequence),
		optionalStrParam("name", attr.Name),
	})
}
