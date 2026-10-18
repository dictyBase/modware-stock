package arangodb

import (
	"regexp"

	F "github.com/IBM/fp-go/v2/function"
	O "github.com/IBM/fp-go/v2/option"
	P "github.com/IBM/fp-go/v2/predicate"
	S "github.com/IBM/fp-go/v2/string"
	MO "github.com/dictyBase/fp-go-loom/matchopt"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

type (
	// strainListNames carries the collection and graph names one strain list
	// query binds. The names travel as a value so that the route
	// classification stays pure and the unit tests can drive it without a
	// database.
	strainListNames struct {
		stock            string
		stockPropGraph   string
		cvterm           string
		cv               string
		stockCvtermGraph string
	}

	// strainListPage is the resolved AQL template and bind parameter set of one
	// strain list page.
	strainListPage struct {
		statement  string
		bindParams map[string]any
	}

	// strainListRoute is one classified strain list route: the selector of the
	// page template, and the builder of the bind parameters that template
	// needs.
	strainListRoute struct {
		// statement selects the template of a page: true asks for the template
		// of a later cursor page.
		statement  func(bool) string
		bindParams func(*stock.StockParameters, strainListNames) map[string]any
	}
)

var (
	// aqlLiteral matches one single quoted AQL string literal.
	aqlLiteral = regexp.MustCompile(`'[^']*'`)

	// filterLiteralFree blanks the string literals of a filter, so that the
	// text of a filter value cannot select a route.
	filterLiteralFree = func(filter string) string {
		return aqlLiteral.ReplaceAllString(filter, "''")
	}

	// filterCvRef reports a filter that references the cv collection.
	filterCvRef = F.Flow2(filterLiteralFree, S.Includes("cv."))

	// filterCvtermRef reports a filter that references the cvterm collection.
	filterCvtermRef = F.Flow2(filterLiteralFree, S.Includes("cvterm."))

	// filterOntologyRef reports a filter that references cv or cvterm.
	filterOntologyRef = F.Pipe1(filterCvRef, P.Or(filterCvtermRef))

	// filterStockPropRef reports a filter that references a strain property.
	filterStockPropRef = F.Flow2(filterLiteralFree, S.Includes("stock_prop."))

	// filterOntologyDeferredProp reports the ontology-first route whose filter
	// never references a strain property. Its template sorts and limits before
	// the property traversal, so the traversal runs at most @limit times.
	filterOntologyDeferredProp = F.Pipe1(
		filterOntologyRef,
		P.And(P.Not(filterStockPropRef)),
	)

	// filterOntologyJoinedProp reports the ontology-first route whose filter
	// references a strain property. Its template must traverse the property
	// graph before the filter, the sort and the limit.
	filterOntologyJoinedProp = F.Pipe1(
		filterOntologyRef,
		P.And(filterStockPropRef),
	)

	// filterText lifts a predicate of the filter string onto the request.
	filterText = P.ContraMap(
		func(param *stock.StockParameters) string { return param.Filter },
	)

	// paramOntologyDeferredProp reports a request of the deferred route.
	paramOntologyDeferredProp = filterText(filterOntologyDeferredProp)

	// paramOntologyJoinedProp reports a request of the joined route.
	paramOntologyJoinedProp = filterText(filterOntologyJoinedProp)

	// paramHasFilter reports a request that carries a filter.
	paramHasFilter = filterText(hasFilter)

	// paramCursorText lifts a predicate of the page cursor onto the request.
	paramCursorText = P.ContraMap(
		func(param *stock.StockParameters) int64 { return param.Cursor },
	)

	// paramHasCursor reports a request that carries a page cursor.
	paramHasCursor = paramCursorText(hasCursor)

	// cursorBindParam builds the cursor bind parameter of a page.
	cursorBindParam = func(param *stock.StockParameters) map[string]any {
		return map[string]any{paramCursor: param.Cursor}
	}

	// unfilteredStrainRoute lists every strain from the stock collection.
	unfilteredStrainRoute = strainListRoute{
		statement: strainPageTemplate(
			statement.StrainList,
			statement.StrainListWithCursor,
		),
		bindParams: stockFilterBindParams,
	}

	// stockFirstStrainRoute filters from the stock collection. Its templates
	// bind no ontology collection, so cv, cvterm and stock_cvterm_graph stay
	// out of the query.
	stockFirstStrainRoute = strainListRoute{
		statement: strainPageTemplate(
			statement.StrainListStockFilter,
			statement.StrainListStockFilterWithCursor,
		),
		bindParams: stockFilterBindParams,
	}

	// ontologyDeferredPropStrainRoute traverses the ontology graph first and
	// defers the property traversal until after the sort and the limit.
	ontologyDeferredPropStrainRoute = strainListRoute{
		statement: strainPageTemplate(
			statement.StrainListOntologyFilter,
			statement.StrainListOntologyFilterWithCursor,
		),
		bindParams: ontologyFilterBindParams,
	}

	// ontologyJoinedPropStrainRoute traverses the property graph before the
	// filter, because its filter compares a strain property.
	ontologyJoinedPropStrainRoute = strainListRoute{
		statement: strainPageTemplate(
			statement.StrainListFilter,
			statement.StrainListFilterWithCursor,
		),
		bindParams: ontologyFilterBindParams,
	}
)

// strainPageTemplate returns the selector of the page template of one route: the
// cursor template for a later page, the first page template otherwise.
func strainPageTemplate(firstPage, nextPage string) func(bool) string {
	fold := P.Fold(
		F.Constant1[bool](firstPage),
		F.Constant1[bool](nextPage),
	)
	return fold(F.Identity[bool])
}

// strainListPageOf resolves the statement and the bind parameters of one strain
// list page. One route decides both, so a statement can never receive a bind
// parameter of another route and ArangoDB never sees an unused bind variable.
func strainListPageOf(
	param *stock.StockParameters,
	names strainListNames,
) strainListPage {
	route := strainListRouteOf(param)
	template := F.Pipe2(param.Cursor, hasCursor, route.statement)
	return strainListPage{
		statement:  F.Pipe1(template, injectFilter(param.Filter)),
		bindParams: route.bindParams(param, names),
	}
}

// strainListRouteOf classifies one strain list request into its route.
func strainListRouteOf(param *stock.StockParameters) strainListRoute {
	classify := MO.Match(
		fallbackStrainRoute(param),
		[]O.Kleisli[*stock.StockParameters, strainListRoute]{
			MO.Const(paramOntologyDeferredProp, ontologyDeferredPropStrainRoute),
			MO.Const(paramOntologyJoinedProp, ontologyJoinedPropStrainRoute),
		},
	)
	return classify(param)
}

// fallbackStrainRoute is the route of a request that references no ontology
// collection: the unfiltered route for an empty filter, the stock-first route
// for any other filter.
func fallbackStrainRoute(param *stock.StockParameters) strainListRoute {
	choose := P.Fold(
		F.Constant1[*stock.StockParameters](unfilteredStrainRoute),
		F.Constant1[*stock.StockParameters](stockFirstStrainRoute),
	)
	pick := choose(paramHasFilter)
	return pick(param)
}

// stockFilterBindParams binds the stock collection and the property graph. The
// unfiltered route and the stock-first route reference only
// @@stock_collection and @stock_prop_graph, so no bind parameter is unused.
func stockFilterBindParams(
	param *stock.StockParameters,
	names strainListNames,
) map[string]any {
	return withPageBindParams(param, map[string]any{
		bindStockCollection: names.stock,
		nameStockPropGraph:  names.stockPropGraph,
	})
}

// ontologyFilterBindParams binds the ontology collections and the property
// graph. The two ontology-first routes reference @@cv_collection,
// @@cvterm_collection, @stock_cvterm_graph and @stock_prop_graph.
func ontologyFilterBindParams(
	param *stock.StockParameters,
	names strainListNames,
) map[string]any {
	return withPageBindParams(param, map[string]any{
		bindCvtermCollection: names.cvterm,
		bindCVCollection:     names.cv,
		nameStockCvtermGraph: names.stockCvtermGraph,
		nameStockPropGraph:   names.stockPropGraph,
	})
}

// withPageBindParams adds the @limit bind parameter and, when the page carries a
// cursor, the @cursor bind parameter.
func withPageBindParams(
	param *stock.StockParameters,
	params map[string]any,
) map[string]any {
	cursorParams := F.Pipe3(
		param,
		O.FromPredicate(paramHasCursor),
		O.Map(cursorBindParam),
		O.GetOrElse(F.Constant(map[string]any{})),
	)
	return F.Pipe2(
		params,
		mergeParams(cursorParams),
		addLimitParam(param.Limit+1),
	)
}
