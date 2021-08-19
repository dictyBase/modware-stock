package arangodb

import (
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

// GetStrain retrieves a strain from the database
func (ar *arangorepository) GetStrain(id string) (*model.StockDoc, error) {
	m := &model.StockDoc{}
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
	bindVars := map[string]interface{}{
		"id":                id,
		"@stock_collection": ar.stockc.stock.Name(),
		"stock_collection":  ar.stockc.stock.Name(),
		"parent_graph":      ar.stockc.strain2Parent.Name(),
		"prop_graph":        ar.stockc.stockPropType.Name(),
	}
	r, err := ar.database.GetRow(statement.StockGetStrain, bindVars)
	if err != nil {
		return m, fmt.Errorf("error in finding strain id %s %s", id, err)
	}
	err = r.Read(m)
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
