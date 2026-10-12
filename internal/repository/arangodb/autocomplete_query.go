package arangodb

import (
	"fmt"
	"strings"

	A "github.com/IBM/fp-go/v2/array"
	F "github.com/IBM/fp-go/v2/function"
	L "github.com/IBM/fp-go/v2/optics/lens"
	P "github.com/IBM/fp-go/v2/predicate"
	S "github.com/IBM/fp-go/v2/string"
	"github.com/dictyBase/fp-go-loom/strutils"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

// joinStatements joins the LET blocks of the branches with newlines.
var joinStatements = F.Bind2nd(strings.Join, "\n")

// joinVariables joins the AQL variables of the branches with commas.
var joinVariables = F.Bind2nd(strings.Join, ", ")

var scalarOrArray = F.Pipe1(isScalarField, P.Fold(displayArray, displayScalar))

// queryBranch is one generated LET branch of the statement. The name
// is the AQL variable of the branch, and the body is its LET block.
type queryBranch struct {
	name string
	body string
}

var (
	nameLens = L.MakeLens(
		func(b queryBranch) string { return b.name },
		func(b queryBranch, v string) queryBranch { b.name = v; return b },
	)
	bodyLens = L.MakeLens(
		func(b queryBranch) string { return b.body },
		func(b queryBranch, v string) queryBranch { b.body = v; return b },
	)
)

// displayScalar is the display expression of a scalar field: its
// value.
func displayScalar(f autocompleteField) string {
	return fmt.Sprintf(`NOT_NULL(d.%s, "")`, f.label)
}

// displayArray is the display expression of an array field: the first
// element that contains the query, or the joined list as a fallback.
func displayArray(f autocompleteField) string {
	return fmt.Sprintf(
		`NOT_NULL(FIRST(FOR item IN NOT_NULL(d.%s, []) `+
			`FILTER CONTAINS(LOWER(item), @q) RETURN item), `+
			`CONCAT_SEPARATOR(", ", NOT_NULL(d.%s, [])))`,
		f.label,
		f.label,
	)
}

// isScalarField reports whether the field displays as a scalar.
func isScalarField(f autocompleteField) bool {
	return f.display == autocompleteScalar
}

// prefixBranch fills the prefix branch of one field: the variable
// p_<label> with the prefix score. The template is the second
// parameter so the field pipelines bind it with F.Bind2nd.
func prefixBranch(f autocompleteField, tpl string) queryBranch {
	parts := A.From[any](
		strutils.JoinStrings("p_", f.label),
		autocompletePrefixExpr(f.label),
		f.label,
		scalarOrArray(f),
		autocompletePrefixScore,
	)

	return queryBranch{
		name: strutils.JoinStrings("p_", f.label),
		body: fmt.Sprintf(tpl, parts...),
	}
}

// fuzzyBranch fills the fuzzy branch of one field: the variable
// n_<label> with the fuzzy score. The template is the second
// parameter so the field pipelines bind it with F.Bind2nd.
func fuzzyBranch(f autocompleteField, tpl string) queryBranch {
	parts := A.From[any](
		strutils.JoinStrings("n_", f.label),
		autocompleteFuzzyExpr(f.label),
		f.label,
		scalarOrArray(f),
		autocompleteFuzzyScore,
	)

	return queryBranch{
		name: strutils.JoinStrings("n_", f.label),
		body: fmt.Sprintf(tpl, parts...),
	}
}

// stockPrefixes is the prefix branch of each stock field.
var stockPrefixes = F.Pipe1(
	stockFields,
	A.Map(F.Bind2nd(prefixBranch, statement.AutocompleteStockBranch)),
)

// stockFuzzies is the fuzzy branch of each stock field.
var stockFuzzies = F.Pipe1(
	stockFields,
	A.Map(F.Bind2nd(fuzzyBranch, statement.AutocompleteStockBranch)),
)

// stockBranches is the prefix branches and the fuzzy branches of the
// stock fields.
var stockBranches = F.Pipe1(stockPrefixes, A.Concat(stockFuzzies))

// propPrefixes is the prefix branch of each property field.
var propPrefixes = F.Pipe1(
	propFields,
	A.Map(F.Bind2nd(prefixBranch, statement.AutocompletePropBranch)),
)

// propFuzzies is the fuzzy branch of each property field.
var propFuzzies = F.Pipe1(
	propFields,
	A.Map(F.Bind2nd(fuzzyBranch, statement.AutocompletePropBranch)),
)

// propBranches is the prefix branches and the fuzzy branches of the
// property fields.
var propBranches = F.Pipe1(propPrefixes, A.Concat(propFuzzies))

// mergeBlock is the merge of the statement. It keeps one best row per
// stock key and returns the top rows.
var mergeBlock = F.Pipe4(
	stockBranches,
	A.Concat(propBranches),
	A.Map(nameLens.Get),
	joinVariables,
	S.Format[string](statement.AutocompleteMerge),
)

// buildAutocompleteQuery generates the full statement: the LET blocks
// of the branches and the merge block.
func buildAutocompleteQuery() string {
	return F.Pipe4(
		stockBranches,
		A.Concat(propBranches),
		A.Map(bodyLens.Get),
		F.Bind2nd(A.Append, mergeBlock),
		joinStatements,
	)
}

// autocompleteQuery is the full 16-branch statement, built once per
// process.
var autocompleteQuery = buildAutocompleteQuery()

// autocompletePrefixExpr is the SEARCH expression of a prefix branch.
// STARTS_WITH inside SEARCH needs the ANALYZER wrapper: without it the
// comparison uses the identity analyzer and matches nothing.
func autocompletePrefixExpr(field string) string {
	return fmt.Sprintf(
		"ANALYZER(STARTS_WITH(d.%s, @q), %q)",
		field,
		autocompleteNormAnalyzer,
	)
}

// autocompleteFuzzyExpr is the SEARCH expression of a fuzzy branch.
// NGRAM_MATCH takes the analyzer as its fourth argument.
func autocompleteFuzzyExpr(field string) string {
	return fmt.Sprintf(
		"NGRAM_MATCH(d.%s, @q, @th, %q)",
		field,
		autocompleteNgramAnalyzer,
	)
}
