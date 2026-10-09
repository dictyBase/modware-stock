package arangodb

import (
	"fmt"
	"strings"

	A "github.com/IBM/fp-go/v2/array"
	F "github.com/IBM/fp-go/v2/function"
	S "github.com/IBM/fp-go/v2/string"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

// joinStatements joins the LET blocks of the branches with newlines.
var joinStatements = F.Bind2nd(strings.Join, "\n")

// joinVariables joins the AQL variables of the branches with commas.
var joinVariables = F.Bind2nd(strings.Join, ", ")

// queryBranch is one generated LET branch of the statement. The name
// is the AQL variable of the branch, and the body is its LET block.
type queryBranch struct {
	name string
	body string
}

// statement is the LET block of the branch.
func (b queryBranch) statement() string { return b.body }

// variable is the AQL variable of the branch.
func (b queryBranch) variable() string { return b.name }

// branchPair fills the branch template twice for one field: the
// prefix branch with the variable p_<label> and the fuzzy branch with
// the variable n_<label>. The label makes the variables unique and
// self-describing in the generated AQL. The template is the second
// parameter so the branch pipelines bind it with F.Bind2nd.
func branchPair(f autocompleteField, tpl string) []queryBranch {
	label := f.label
	display := autocompleteDisplayExpr(f)
	prefixName := "p_" + label
	fuzzyName := "n_" + label
	prefixSearch := autocompletePrefixExpr(label)
	fuzzySearch := autocompleteFuzzyExpr(label)

	return []queryBranch{
		{
			name: prefixName,
			body: fmt.Sprintf(
				tpl,
				prefixName,
				prefixSearch,
				label,
				display,
				autocompletePrefixScore,
			),
		},
		{
			name: fuzzyName,
			body: fmt.Sprintf(
				tpl,
				fuzzyName,
				fuzzySearch,
				label,
				display,
				autocompleteFuzzyScore,
			),
		},
	}
}

// stockBranches is the prefix branch and the fuzzy branch of each
// stock field.
var stockBranches = F.Pipe1(
	stockFields,
	A.Chain(F.Bind2nd(branchPair, statement.AutocompleteStockBranch)),
)

// propBranches is the prefix branch and the fuzzy branch of each
// property field.
var propBranches = F.Pipe1(
	propFields,
	A.Chain(F.Bind2nd(branchPair, statement.AutocompletePropBranch)),
)

// allBranches is the stock branches and the property branches.
var allBranches = F.Pipe1(stockBranches, A.Concat(propBranches))

// branchVariables is the AQL variables of all branches, joined with
// commas. The merge block merges the rows of these branches.
var branchVariables = F.Pipe2(
	allBranches,
	A.Map(queryBranch.variable),
	joinVariables,
)

// mergeBlock is the merge of the statement. It keeps one best row per
// stock key and returns the top rows.
var mergeBlock = F.Pipe1(
	branchVariables,
	S.Format[string](statement.AutocompleteMerge),
)

// buildAutocompleteQuery generates the full statement: the LET blocks
// of the branches and the merge block.
func buildAutocompleteQuery() string {
	return F.Pipe3(
		allBranches,
		A.Map(queryBranch.statement),
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

// autocompleteDisplayExpr returns the display expression of a field.
// A scalar field returns its value; an array field returns the first
// element that contains the query, or the joined list as a fallback.
// The expressions run only on the rows that survive the branch LIMIT,
// never inside the index.
func autocompleteDisplayExpr(f autocompleteField) string {
	if f.display == autocompleteScalar {
		return fmt.Sprintf(`NOT_NULL(d.%s, "")`, f.label)
	}
	return fmt.Sprintf(
		`NOT_NULL(FIRST(FOR item IN NOT_NULL(d.%s, []) `+
			`FILTER CONTAINS(LOWER(item), @q) RETURN item), `+
			`CONCAT_SEPARATOR(", ", NOT_NULL(d.%s, [])))`,
		f.label,
		f.label,
	)
}
