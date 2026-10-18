package arangodb

import (
	"context"

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
	ctx context.Context,
	param *stock.StockParameters,
) ([]*model.StockDoc, error) {
	omd := make([]*model.StockDoc, 0)
	page := ar.strainListPage(param)
	rs, err := ar.searchRows(ctx, page.statement, page.bindParams)
	if err != nil {
		return omd, err
	}
	defer func() { _ = rs.Close() }()
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
	ctx context.Context,
	p *stock.StockIdList,
) ([]*model.StockDoc, error) {
	ms := make([]*model.StockDoc, 0)
	rs, err := ar.searchRows(
		ctx,
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
	defer func() { _ = rs.Close() }()
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

// strainListPage resolves the AQL template and the bind parameters of one
// strain list page. An empty filter and a stock-only filter share the stock
// collection bindings, but they take different templates.
func (ar *arangorepository) strainListPage(
	param *stock.StockParameters,
) strainListPage {
	return strainListPageOf(param, ar.strainListNames())
}

// strainListNames reads the collection and graph names of the repository.
func (ar *arangorepository) strainListNames() strainListNames {
	return strainListNames{
		stock:            ar.stockc.stock.Name(),
		stockPropGraph:   ar.stockc.stockPropType.Name(),
		cvterm:           ar.ontoc.Term.Name(),
		cv:               ar.ontoc.Cv.Name(),
		stockCvtermGraph: ar.stockc.stockOnto.Name(),
	}
}
