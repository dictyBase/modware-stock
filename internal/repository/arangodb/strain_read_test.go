package arangodb

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
	"github.com/stretchr/testify/require"
)

// strainListRouteCase is one expectation of the route classification table.
type strainListRouteCase struct {
	name string
	// wantDeferredProp marks the route that sorts and limits before the
	// property traversal.
	wantDeferredProp bool
	filter           string
	cursor           int64
	wantStatement    string
	wantBinds        []string
}

const (
	// testStrainRouteLimit is the page size of the routing tests.
	testStrainRouteLimit int64 = 10

	// testStrainRouteCursor is a positive page cursor of the routing tests.
	testStrainRouteCursor int64 = 1600000000
)

var (
	// aqlBindName matches one bind parameter reference of an AQL statement.
	aqlBindName = regexp.MustCompile(`@@?[a-zA-Z_][a-zA-Z0-9_]*`)

	// testStrainListNames is a fixed set of collection and graph names. It lets
	// the routing tests run without a database.
	testStrainListNames = strainListNames{
		stock:            testStockColl,
		stockPropGraph:   "stockprop_type_test",
		cvterm:           "cvterm",
		cv:               "cv",
		stockCvtermGraph: "stockonto_test",
	}
)

// bindParamKeys returns the sorted keys of a bind parameter map.
func bindParamKeys(params map[string]any) []string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// statementBindParamNames returns the bind parameter names one statement
// references, with the contents of quoted string literals removed first. A
// filter value such as an email address must not count as a bind parameter.
// A name keeps its leading @ characters, because the statement spells a
// collection bind variable with @@ while the bind map key drops one @.
func statementBindParamNames(stmt string) []string {
	plain := aqlLiteral.ReplaceAllString(stmt, "''")
	names := aqlBindName.FindAllString(plain, -1)
	sort.Strings(names)
	return names
}

// statementBindParamKeys returns the bind map keys the names of a statement
// require. "@@stock_collection" needs the key "@stock_collection", while
// "@limit" needs the key "limit".
func statementBindParamKeys(names []string) []string {
	keys := make([]string, 0, len(names))
	for _, name := range names {
		keys = append(keys, strings.TrimPrefix(name, "@"))
	}
	sort.Strings(keys)
	return keys
}

// strainListRouteCases returns the routing expectations. Each case states the
// AQL template and the exact bind parameter keys of one request shape.
func strainListRouteCases() []strainListRouteCase {
	ontologyBinds := []string{
		bindCvtermCollection,
		bindCVCollection,
		nameStockCvtermGraph,
		nameStockPropGraph,
		paramLimit,
	}
	stockBinds := []string{
		bindStockCollection,
		nameStockPropGraph,
		paramLimit,
	}
	withCursor := func(keys []string) []string {
		return append(append([]string{}, keys...), paramCursor)
	}

	return []strainListRouteCase{
		{
			name:          "empty filter lists from the stock collection",
			wantStatement: statement.StrainList,
			wantBinds:     stockBinds,
		},
		{
			name:          "empty filter with a cursor",
			cursor:        testStrainRouteCursor,
			wantStatement: statement.StrainListWithCursor,
			wantBinds:     withCursor(stockBinds),
		},
		{
			name:          "depositor filter takes the stock-first route",
			filter:        filterOne,
			wantStatement: fmt.Sprintf(statement.StrainListStockFilter, filterOne),
			wantBinds:     stockBinds,
		},
		{
			name:          "stock_prop filter keeps the property traversal before the filter",
			filter:        filterFive,
			wantStatement: fmt.Sprintf(statement.StrainListStockFilter, filterFive),
			wantBinds:     stockBinds,
		},
		{
			name:   "stock_prop filter with a cursor",
			filter: filterFive,
			cursor: testStrainRouteCursor,
			wantStatement: fmt.Sprintf(
				statement.StrainListStockFilterWithCursor, filterFive,
			),
			wantBinds: withCursor(stockBinds),
		},
		{
			name:          "gene filter takes the stock-first route",
			filter:        filterGene,
			wantStatement: fmt.Sprintf(statement.StrainListStockFilter, filterGene),
			wantBinds:     stockBinds,
		},
		{
			name:             "ontology filter defers the property traversal",
			wantDeferredProp: true,
			filter:           filterRegularStrain,
			wantStatement:    fmt.Sprintf(statement.StrainListOntologyFilter, filterRegularStrain),
			wantBinds:        ontologyBinds,
		},
		{
			name:             "ontology filter with a cursor defers the property traversal",
			wantDeferredProp: true,
			filter:           filterGwdiStrain,
			cursor:           testStrainRouteCursor,
			wantStatement: fmt.Sprintf(
				statement.StrainListOntologyFilterWithCursor, filterGwdiStrain,
			),
			wantBinds: withCursor(ontologyBinds),
		},
		{
			name:          "ontology filter with a stock_prop reference keeps the join before the limit",
			filter:        filterOntologyAndProp,
			wantStatement: fmt.Sprintf(statement.StrainListFilter, filterOntologyAndProp),
			wantBinds:     ontologyBinds,
		},
		{
			name:          "quoted cv. literal stays on the stock-first route",
			filter:        filterQuotedCvLiteral,
			wantStatement: fmt.Sprintf(statement.StrainListStockFilter, filterQuotedCvLiteral),
			wantBinds:     stockBinds,
		},
	}
}

// TestStrainListStatementRouting asserts that a request takes the AQL template
// of its classified route and receives exactly that route's bind parameters.
func TestStrainListStatementRouting(t *testing.T) {
	t.Parallel()
	for _, tt := range strainListRouteCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			page := strainListPageOf(
				&stock.StockParameters{
					Filter: tt.filter,
					Cursor: tt.cursor,
					Limit:  testStrainRouteLimit,
				},
				testStrainListNames,
			)

			require.Equal(t, tt.wantStatement, page.statement)
			require.ElementsMatch(t, tt.wantBinds, bindParamKeys(page.bindParams))
			require.EqualValues(
				t,
				testStrainRouteLimit+1,
				page.bindParams[paramLimit],
				"the page probe must ask for one row more than the limit",
			)
		})
	}
}

// TestStrainListBindsCoverStatement asserts the bind parameter set and the
// statement agree in both directions. ArangoDB rejects a bind parameter that
// the statement does not use, and fails on one that the statement uses but the
// map does not provide.
func TestStrainListBindsCoverStatement(t *testing.T) {
	t.Parallel()
	for _, tt := range strainListRouteCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			page := strainListPageOf(
				&stock.StockParameters{
					Filter: tt.filter,
					Cursor: tt.cursor,
					Limit:  testStrainRouteLimit,
				},
				testStrainListNames,
			)

			referenced := statementBindParamNames(page.statement)
			for _, key := range statementBindParamKeys(referenced) {
				require.Containsf(
					t,
					page.bindParams,
					key,
					"the statement references %s but the bind map does not provide it",
					key,
				)
			}
			for key := range page.bindParams {
				require.Containsf(
					t,
					referenced,
					"@"+key,
					"the bind map provides %s but the statement does not use it",
					key,
				)
			}
		})
	}
}

// TestStrainListOntologyFilterDefersPropertyTraversal asserts the order the
// optimized ontology-first template depends on: the sort and the limit run
// before the stock_prop_type traversal.
func TestStrainListOntologyFilterDefersPropertyTraversal(t *testing.T) {
	t.Parallel()
	for _, tt := range strainListRouteCases() {
		if !tt.wantDeferredProp {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			limitAt := strings.Index(tt.wantStatement, "LIMIT @limit")
			propAt := strings.Index(
				tt.wantStatement,
				"OUTBOUND s GRAPH @stock_prop_graph",
			)
			require.NotEqual(t, -1, limitAt)
			require.NotEqual(t, -1, propAt)
			require.Less(
				t,
				limitAt,
				propAt,
				"the limit must run before the property traversal",
			)
		})
	}
}

// TestStrainListStockFilterOmitsOntology asserts that a stock-first route binds
// no ontology collection, graph or term.
func TestStrainListStockFilterOmitsOntology(t *testing.T) {
	t.Parallel()
	for _, tt := range strainListRouteCases() {
		if strings.Contains(tt.wantStatement, "@stock_cvterm_graph") {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NotContains(t, tt.wantStatement, "@@cv_collection")
			require.NotContains(t, tt.wantStatement, "@@cvterm_collection")
		})
	}
}

func Test_arangorepository_GetStrain(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		id      string
		want    *model.StockDoc
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type.
			var ar arangorepository
			got, gotErr := ar.GetStrain(tt.id)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("GetStrain() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("GetStrain() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("GetStrain() = %v, want %v", got, tt.want)
			}
		})
	}
}
