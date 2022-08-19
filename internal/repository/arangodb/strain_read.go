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
		map[string]interface{}{
			"id":                 id,
			"stock_cvterm_graph": ar.stockc.stockOnto.Name(),
			"ontology":           ar.strainOnto,
			"stock_collection":   ar.stockc.stock.Name(),
			"parent_graph":       ar.stockc.strain2Parent.Name(),
			"stock_prop_graph":   ar.stockc.stockPropType.Name(),
			"@stock_collection":  ar.stockc.stock.Name(),
			"@cv_collection":     ar.ontoc.Cv.Name(),
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

func (ar *arangorepository) ListStrainsByIds(
	p *stock.StockIdList,
) ([]*model.StockDoc, error) {
	ms := make([]*model.StockDoc, 0)
	rs, err := ar.database.SearchRows(
		statement.StrainListFromIds,
		map[string]interface{}{
			"ids":                p.Id,
			"limit":              len(p.Id),
			"ontology":           ar.strainOnto,
			"stock_collection":   ar.stockc.stock.Name(),
			"stock_cvterm_graph": ar.stockc.stockOnto.Name(),
			"stock_prop_graph":   ar.stockc.stockPropType.Name(),
			"parent_graph":       ar.stockc.strain2Parent.Name(),
			"@stock_collection":  ar.stockc.stock.Name(),
			"@cv_collection":     ar.ontoc.Cv.Name(),
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
) (string, map[string]interface{}) {
	stmtMap := map[string]interface{}{
		"@cvterm_collection": ar.ontoc.Term.Name(),
		"@cv_collection":     ar.ontoc.Cv.Name(),
		"stock_cvterm_graph": ar.stockc.stockOnto.Name(),
		"stock_prop_graph":   ar.stockc.stockPropType.Name(),
		"limit":              param.Limit + 1,
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
) (string, map[string]interface{}) {
	stmt := statement.StrainList
	stmtMap := map[string]interface{}{
		"@stock_collection": ar.stockc.stock.Name(),
		"stock_prop_graph":  ar.stockc.stockPropType.Name(),
		"limit":             param.Limit + 1,
	}
	if param.Cursor != 0 {
		stmt = statement.StrainListWithCursor
		stmtMap["cursor"] = param.Cursor
	}

	return stmt, stmtMap
}
