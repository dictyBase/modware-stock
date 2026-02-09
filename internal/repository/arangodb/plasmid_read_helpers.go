package arangodb

import (
	"fmt"

	F "github.com/IBM/fp-go/function"
	IOE "github.com/IBM/fp-go/ioeither"
	M "github.com/IBM/fp-go/magma"
	R "github.com/IBM/fp-go/record"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

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

// Predicates for query selection
var (
	// hasFilter checks if filter string is non-empty
	hasFilter = func(s string) bool { return len(s) > 0 }

	// hasCursor checks if cursor is positive
	hasCursor = func(cursor int64) bool { return cursor > 0 }
)

// Magma for map merging (last key wins)
var lastWins = M.MakeMagma(func(a, b any) any { return b })

// selectStatementByCursor selects statement based on cursor presence
var selectStatementByCursor = F.Curry2(
	func(withCursor, withoutCursor string) func(bool) string {
		return F.Ternary(
			F.Identity[bool],
			F.Constant1[bool](withCursor),
			F.Constant1[bool](withoutCursor),
		)
	},
)

// selectStockFirstStatement selects stock-first query statements
var selectStockFirstStatement = selectStatementByCursor(
	statement.PlasmidListWithCursor,
)(statement.PlasmidList)

// selectOntologyFirstStatement selects ontology-first filtered query statements
var selectOntologyFirstStatement = selectStatementByCursor(
	statement.PlasmidListFilterByOntologyWithCursor,
)(statement.PlasmidListFilterByOntology)

// selectPlasmidStatement selects appropriate query using functional composition
func (ar *arangorepository) selectPlasmidStatement(
	params *stock.StockParameters,
) string {
	cursorPresent := hasCursor(params.Cursor)

	return F.Pipe2(
		params.Filter,
		hasFilter,
		F.Ternary(
			F.Identity[bool],
			F.Constant1[bool](selectOntologyFirstStatement(cursorPresent)),
			F.Constant1[bool](selectStockFirstStatement(cursorPresent)),
		),
	)
}

// Base bind parameters (common to all queries)
func (ar *arangorepository) baseBindParams() map[string]any {
	return map[string]any{
		"stock_cvterm_graph": ar.stockc.stockOnto.Name(),
		"ontology":           ar.plasmidOnto,
		"@cv_collection":     ar.ontoc.Cv.Name(),
	}
}

// Stock-first bind parameters (no filter queries)
func (ar *arangorepository) stockFirstBindParams() map[string]any {
	return map[string]any{
		"@stock_collection": ar.stockc.stock.Name(),
		"stock_prop_graph":  ar.stockc.stockPropType.Name(),
	}
}

// Ontology-first bind parameters (filtered queries)
func (ar *arangorepository) ontologyFirstBindParams() map[string]any {
	return map[string]any{
		"@cvterm_collection": ar.ontoc.Term.Name(),
		"@cv_collection":     ar.ontoc.Cv.Name(),
		"stock_prop_graph":   ar.stockc.stockPropType.Name(),
	}
}

// addCursorParam adds cursor to bind parameters
var addCursorParam = F.Curry2(
	func(cursor int64, params map[string]any) map[string]any {
		return F.Pipe1(
			params,
			R.Union[string, any](lastWins)(
				map[string]any{"cursor": cursor},
			),
		)
	},
)

// addLimitParam adds limit to bind parameters
var addLimitParam = F.Curry2(
	func(limit int64, params map[string]any) map[string]any {
		return F.Pipe1(
			params,
			R.Union[string, any](lastWins)(
				map[string]any{"limit": limit},
			),
		)
	},
)

// mergeParams merges two parameter maps
var mergeParams = F.Curry2(
	func(additional, base map[string]any) map[string]any {
		return F.Pipe1(
			base,
			R.Union[string, any](lastWins)(additional),
		)
	},
)

// buildNoFilterBindParams builds bind params for no-filter queries
func (ar *arangorepository) buildNoFilterBindParams(
	params *stock.StockParameters,
) map[string]any {
	baseWithStock := F.Pipe2(
		ar.baseBindParams(),
		mergeParams(ar.stockFirstBindParams()),
		addLimitParam(params.Limit+1),
	)

	return F.Pipe2(
		params.Cursor,
		hasCursor,
		F.Ternary(
			F.Identity[bool],
			F.Constant1[bool](F.Pipe1(baseWithStock, addCursorParam(params.Cursor))),
			F.Constant1[bool](baseWithStock),
		),
	)
}

// buildFilterBindParams builds bind params for filtered queries
func (ar *arangorepository) buildFilterBindParams(
	params *stock.StockParameters,
) map[string]any {
	baseWithOntology := F.Pipe2(
		ar.baseBindParams(),
		mergeParams(ar.ontologyFirstBindParams()),
		addLimitParam(params.Limit+1),
	)

	return F.Pipe2(
		params.Cursor,
		hasCursor,
		F.Ternary(
			F.Identity[bool],
			F.Constant1[bool](F.Pipe1(baseWithOntology, addCursorParam(params.Cursor))),
			F.Constant1[bool](baseWithOntology),
		),
	)
}

// buildBindParams selects and builds bind parameters
func (ar *arangorepository) buildBindParams(
	params *stock.StockParameters,
) map[string]any {
	return F.Pipe2(
		params.Filter,
		hasFilter,
		F.Ternary(
			F.Identity[bool],
			F.Constant1[bool](ar.buildFilterBindParams(params)),
			F.Constant1[bool](ar.buildNoFilterBindParams(params)),
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

// injectFilter injects filter string into statement if present
var injectFilter = F.Curry2(
	func(filter string, stmt string) string {
		return F.Pipe2(
			filter,
			hasFilter,
			F.Ternary(
				F.Identity[bool],
				F.Constant1[bool](fmt.Sprintf(stmt, filter)),
				F.Constant1[bool](stmt),
			),
		)
	},
)

// buildPlasmidListQueryParams creates query parameters for listing plasmids
func (ar *arangorepository) buildPlasmidListQueryParams(
	params *stock.StockParameters,
) plasmidListQueryParams {
	return plasmidListQueryParams{
		statement:  injectFilter(params.Filter)(ar.selectPlasmidStatement(params)),
		bindParams: ar.buildBindParams(params),
	}
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
