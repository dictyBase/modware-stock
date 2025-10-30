package arangodb

import (
	"fmt"

	fperrors "github.com/IBM/fp-go/errors"
	F "github.com/IBM/fp-go/function"
	IOE "github.com/IBM/fp-go/ioeither"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

// ListPlasmids provides a list of all plasmids returning IOEither monad
func (ar *arangorepository) ListPlasmids(
	params *stock.StockParameters,
) IOE.IOEither[error, []*model.StockDoc] {
	return F.Pipe4(
		IOE.Right[error](params),
		IOE.Map[error](ar.buildPlasmidListQueryParams),
		IOE.Chain(ar.executePlasmidListQuery),
		IOE.Chain(ar.scanPlasmidRows),
		IOE.MapLeft[[]*model.StockDoc](
			fperrors.OnError("failed to list plasmids"),
		),
	)
}

// GetPlasmid retrieves a plasmid from the database using IOEither monad
func (ar *arangorepository) GetPlasmid(
	id string,
) IOE.IOEither[error, *model.StockDoc] {
	return F.Pipe4(
		IOE.Of[error](id),
		IOE.Map[error](ar.buildPlasmidQueryParams),
		IOE.Chain(ar.executePlasmidQuery),
		IOE.Chain(ar.validatePlasmidQueryResult),
		IOE.MapLeft[*model.StockDoc](
			fperrors.OnError("failed to get plasmid"),
		),
	)
}

// buildPlasmidQueryParams creates the bind parameters for plasmid query
func (ar *arangorepository) buildPlasmidQueryParams(
	plasmidID string,
) map[string]any {
	return map[string]any{
		"id":                 plasmidID,
		"@stock_collection":  ar.stockc.stock.Name(),
		"stock_prop_graph":   ar.stockc.stockPropType.Name(),
		"stock_cvterm_graph": ar.stockc.stockOnto.Name(),
		"ontology":           ar.plasmidOnto,
		"@cv_collection":     ar.ontoc.Cv.Name(),
	}
}

// executePlasmidQuery executes the database query for plasmid retrieval using IOEither
func (ar *arangorepository) executePlasmidQuery(
	bindParams map[string]any,
) IOE.IOEither[error, *dbQueryResult] {
	return IOE.TryCatchError(
		func() (*dbQueryResult, error) {
			plasmidID, _ := bindParams["id"].(string)
			row, err := ar.database.GetRow(
				statement.StockGetPlasmid,
				bindParams,
			)
			if err != nil {
				return nil, fmt.Errorf("database query failed: %w", err)
			}
			return &dbQueryResult{
				row:       row,
				plasmidID: plasmidID,
			}, nil
		},
	)
}

// validatePlasmidQueryResult validates query result and reads stock document using IOEither
func (ar *arangorepository) validatePlasmidQueryResult(
	result *dbQueryResult,
) IOE.IOEither[error, *model.StockDoc] {
	fn := func() (*model.StockDoc, error) {
		if result.row.IsEmpty() {
			return nil, fmt.Errorf(
				"plasmid not found with ID %s",
				result.plasmidID,
			)
		}

		stockDoc := &model.StockDoc{}
		if err := result.row.Read(stockDoc); err != nil {
			return nil, fmt.Errorf("failed to read stock document: %w", err)
		}

		return stockDoc, nil
	}
	return IOE.TryCatchError(fn)
}

// dbQueryResult encapsulates database query result context
type dbQueryResult struct {
	row       dbRow
	plasmidID string
}

// dbRow represents a database row interface
type dbRow interface {
	IsEmpty() bool
	Read(interface{}) error
}

// dbRows represents a database rows interface for scanning multiple results
type dbRows interface {
	IsEmpty() bool
	Scan() bool
	Read(interface{}) error
}

// plasmidListQueryParams encapsulates parameters for plasmid list query
type plasmidListQueryParams struct {
	statement  string
	bindParams map[string]any
}

// buildPlasmidListQueryParams creates query parameters for listing plasmids
func (ar *arangorepository) buildPlasmidListQueryParams(
	params *stock.StockParameters,
) plasmidListQueryParams {
	stmt := ar.selectPlasmidStatement(params)
	bindParams := map[string]any{
		"stock_cvterm_graph": ar.stockc.stockOnto.Name(),
		"ontology":           ar.plasmidOnto,
		"@cv_collection":     ar.ontoc.Cv.Name(),
	}

	return plasmidListQueryParams{
		statement:  stmt,
		bindParams: bindParams,
	}
}

// selectPlasmidStatement selects appropriate statement based on filter presence
func (ar *arangorepository) selectPlasmidStatement(
	params *stock.StockParameters,
) string {
	if len(params.Filter) > 0 {
		return ar.plasmidStmtWithFilter(params)
	}
	return ar.plasmidStmtNoFilter(params)
}

// executePlasmidListQuery executes the query and returns rows
func (ar *arangorepository) executePlasmidListQuery(
	queryParams plasmidListQueryParams,
) IOE.IOEither[error, dbRows] {
	return IOE.TryCatchError(
		func() (dbRows, error) {
			rows, err := ar.database.SearchRows(
				queryParams.statement,
				queryParams.bindParams,
			)
			if err != nil {
				return nil, fmt.Errorf("database query failed: %w", err)
			}
			return rows, nil
		},
	)
}

// scanPlasmidRows scans all rows into slice of StockDoc
func (ar *arangorepository) scanPlasmidRows(
	rows dbRows,
) IOE.IOEither[error, []*model.StockDoc] {
	return IOE.TryCatchError(
		func() ([]*model.StockDoc, error) {
			if rows.IsEmpty() {
				return []*model.StockDoc{}, nil
			}

			plasmids := make([]*model.StockDoc, 0)
			for rows.Scan() {
				stockDoc := &model.StockDoc{}
				if err := rows.Read(stockDoc); err != nil {
					return nil, fmt.Errorf(
						"failed to read stock document: %w",
						err,
					)
				}
				plasmids = append(plasmids, stockDoc)
			}

			return plasmids, nil
		},
	)
}

func (ar *arangorepository) plasmidStmtWithFilter(
	p *stock.StockParameters,
) string {
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

func (ar *arangorepository) plasmidStmtNoFilter(
	p *stock.StockParameters,
) string {
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
