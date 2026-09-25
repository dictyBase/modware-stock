package arangodb

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

// GetStrain retrieves a strain from the database
func (ar *arangorepository) GetStrain(id string) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	r, err := ar.database.GetRow(
		statement.StockGetStrain,
		map[string]any{
			"id":                 id,
			nameStockCvtermGraph: ar.stockc.stockOnto.Name(),
			paramOntology:        ar.strainOnto,
			nameStockCollection:  ar.stockc.stock.Name(),
			nameParentGraph:      ar.stockc.strain2Parent.Name(),
			nameStockPropGraph:   ar.stockc.stockPropType.Name(),
			bindStockCollection:  ar.stockc.stock.Name(),
			bindCVCollection:     ar.ontoc.Cv.Name(),
		})
	if err != nil {
		return m, errors.Errorf("error in finding strain id %s %s", id, err)
	}
	if r.IsEmpty() {
		m.NotFound = true
		return m, nil
	}
	err = r.Read(m)
	return m, err
}

// ListStrains provides a list of all strains
func (ar *arangorepository) ListStrains(
	param *stock.StockParameters,
) ([]*model.StockDoc, error) {
	omd := make([]*model.StockDoc, 0)
	stmt, paramsBind := ar.strainStmtNoFilter(param)
	if len(param.Filter) > 0 {
		stmt, paramsBind = ar.strainStmtWithFilter(param)
	}
	rs, err := ar.database.SearchRows(stmt, paramsBind)
	if err != nil {
		return omd, err
	}
	if rs.IsEmpty() {
		return omd, nil
	}
	for rs.Scan() {
		m := &model.StockDoc{}
		if err := rs.Read(m); err != nil {
			return omd, err
		}
		omd = append(omd, m)
	}
	return omd, nil
}

func (ar *arangorepository) ListStrainsByIDs(
	p *stock.StockIdList,
) ([]*model.StockDoc, error) {
	ms := make([]*model.StockDoc, 0)
	rs, err := ar.database.SearchRows(
		statement.StrainListFromIDs,
		map[string]any{
			"ids":                p.Id,
			paramLimit:           len(p.Id),
			paramOntology:        ar.strainOnto,
			nameStockCollection:  ar.stockc.stock.Name(),
			nameStockCvtermGraph: ar.stockc.stockOnto.Name(),
			nameStockPropGraph:   ar.stockc.stockPropType.Name(),
			nameParentGraph:      ar.stockc.strain2Parent.Name(),
			bindStockCollection:  ar.stockc.stock.Name(),
			bindCVCollection:     ar.ontoc.Cv.Name(),
		})
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

func (ar *arangorepository) strainStmtWithFilter(
	param *stock.StockParameters,
) (string, map[string]any) {
	stmtMap := map[string]any{
		bindCvtermCollection: ar.ontoc.Term.Name(),
		bindCVCollection:     ar.ontoc.Cv.Name(),
		nameStockCvtermGraph: ar.stockc.stockOnto.Name(),
		nameStockPropGraph:   ar.stockc.stockPropType.Name(),
		paramLimit:           param.Limit + 1,
	}
	if param.Cursor != 0 { // no cursor so return first set of results with filter
		stmt := fmt.Sprintf(statement.StrainListFilterWithCursor, param.Filter)
		stmtMap["cursor"] = param.Cursor
		return stmt, stmtMap
	}
	stmt := fmt.Sprintf(statement.StrainListFilter, param.Filter)
	return stmt, stmtMap
}

func (ar *arangorepository) strainStmtNoFilter(
	param *stock.StockParameters,
) (string, map[string]any) {
	stmt := statement.StrainList
	stmtMap := map[string]any{
		bindStockCollection: ar.stockc.stock.Name(),
		nameStockPropGraph:  ar.stockc.stockPropType.Name(),
		paramLimit:          param.Limit + 1,
	}
	if param.Cursor != 0 {
		stmt = statement.StrainListWithCursor
		stmtMap["cursor"] = param.Cursor
	}

	return stmt, stmtMap
}
