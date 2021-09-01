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
			"prop_graph":         ar.stockc.stockPropType.Name(),
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
func (ar *arangorepository) ListStrains(p *stock.StockParameters) ([]*model.StockDoc, error) {
	var om []*model.StockDoc
	stmt := ar.strainStmtNoFilter(p)
	if len(p.Filter) > 0 {
		stmt = ar.strainStmtWithFilter(p)
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
		"ids":               ids,
		"limit":             len(ids),
		"stock_collection":  ar.stockc.stock.Name(),
		"@stock_collection": ar.stockc.stock.Name(),
		"prop_graph":        ar.stockc.stockPropType.Name(),
		"parent_graph":      ar.stockc.strain2Parent.Name(),
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

func (ar *arangorepository) strainStmtWithFilter(p *stock.StockParameters) string {
	if p.Cursor == 0 { // no cursor so return first set of results with filter
		return fmt.Sprintf(
			statement.StrainListFilter,
			ar.stockc.stock.Name(),
			ar.stockc.stockPropType.Name(),
			p.Filter, p.Limit+1,
		)
	}
	// else include both filter and cursor
	return fmt.Sprintf(
		statement.StrainListFilterWithCursor,
		ar.stockc.stock.Name(),
		ar.stockc.stockPropType.Name(),
		p.Filter, p.Cursor, p.Limit+1,
	)
}

func (ar *arangorepository) strainStmtNoFilter(p *stock.StockParameters) string {
	// otherwise use query statement without filter
	if p.Cursor == 0 { // no cursor so return first set of result
		return fmt.Sprintf(
			statement.StrainList,
			ar.stockc.stock.Name(),
			ar.stockc.stockPropType.Name(),
			p.Limit+1,
		)
	}
	// add cursor if it exists
	return fmt.Sprintf(
		statement.StrainListWithCursor,
		ar.stockc.stock.Name(),
		ar.stockc.stockPropType.Name(),
		p.Cursor, p.Limit+1,
	)
}
