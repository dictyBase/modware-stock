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
	stmt := ar.plasmidStmtNoFilter(p)
	// if filter string exists, it needs to be included in statement
	if len(p.Filter) > 0 {
		stmt = ar.plasmidStmtWithFilter(p)
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

func (ar *arangorepository) plasmidStmtWithFilter(p *stock.StockParameters) string {
	if p.Cursor == 0 { // no cursor so return first set of result
		return fmt.Sprintf(
			statement.PlasmidListFilter,
			ar.stockc.stock.Name(),
			ar.stockc.stockPropType.Name(),
			p.Filter, p.Limit+1,
		)
	}
	// else include both filter and cursor
	return fmt.Sprintf(
		statement.PlasmidListFilterWithCursor,
		ar.stockc.stock.Name(),
		ar.stockc.stockPropType.Name(),
		p.Filter, p.Cursor, p.Limit+1,
	)
}

func (ar *arangorepository) plasmidStmtNoFilter(p *stock.StockParameters) string {
	// otherwise use query statement without filter
	if p.Cursor == 0 { // no cursor so return first set of result
		return fmt.Sprintf(
			statement.PlasmidList,
			ar.stockc.stock.Name(),
			ar.stockc.stockPropType.Name(),
			p.Limit+1,
		)
	}
	// add cursor if it exists
	return fmt.Sprintf(
		statement.PlasmidListWithCursor,
		ar.stockc.stock.Name(),
		ar.stockc.stockPropType.Name(),
		p.Cursor, p.Limit+1,
	)
}
