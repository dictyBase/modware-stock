package arangodb

import (
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

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

// GetPlasmid retrieves a plasmid from the database
func (ar *arangorepository) GetPlasmid(id string) (*model.StockDoc, error) {
	m := &model.StockDoc{}
	bindVars := map[string]interface{}{
		"id":                id,
		"@stock_collection": ar.stockc.stock.Name(),
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
