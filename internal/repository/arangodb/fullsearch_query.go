package arangodb

import (
	"fmt"

	A "github.com/IBM/fp-go/v2/array"
	F "github.com/IBM/fp-go/v2/function"
	P "github.com/IBM/fp-go/v2/predicate"
	S "github.com/IBM/fp-go/v2/string"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

var (
	// fullSearchScalarOrArray selects the display expression of one
	// field: a scalar field shows its value, an array field shows the
	// element that matched or the joined list.
	fullSearchScalarOrArray = F.Pipe1(
		isFullSearchScalar,
		P.Fold(fullSearchDisplayArray, fullSearchDisplayScalar),
	)

	// fullSearchBranchTemplate selects the branch template of one field
	// by the collection that holds it.
	fullSearchBranchTemplate = F.Pipe1(
		isPropCollection,
		P.Fold(stockBranchTemplate, propBranchTemplate),
	)
)

// isFullSearchScalar reports whether the field displays as a scalar.
func isFullSearchScalar(f fullSearchField) bool {
	return f.display == fullSearchScalar
}

// isPropCollection reports whether the stock property collection holds
// the field.
func isPropCollection(f fullSearchField) bool {
	return f.coll == fullSearchCollProp
}

// fullSearchDisplayScalar is the display expression of a scalar field:
// its value.
func fullSearchDisplayScalar(f fullSearchField) string {
	return fmt.Sprintf(`NOT_NULL(d.%s, "")`, f.label)
}

// fullSearchDisplayArray is the display expression of an array field:
// the first element that contains the query, or the joined list as a
// fallback. The display expressions run only on the rows that survive
// the branch LIMIT, never inside the index.
func fullSearchDisplayArray(f fullSearchField) string {
	return fmt.Sprintf(
		`NOT_NULL(FIRST(FOR item IN NOT_NULL(d.%s, []) `+
			`FILTER CONTAINS(LOWER(item), @q) RETURN item), `+
			`CONCAT_SEPARATOR(", ", NOT_NULL(d.%s, [])))`,
		f.label,
		f.label,
	)
}

// stockBranchTemplate is the branch template of a field that the stock
// collection holds.
func stockBranchTemplate(fullSearchField) string {
	return statement.FullSearchStockBranch
}

// propBranchTemplate is the branch template of a field that the stock
// property collection holds.
func propBranchTemplate(fullSearchField) string {
	return statement.FullSearchPropBranch
}

// fullSearchBranch fills one branch of the statement. The template of
// the collection takes the branch name, the SEARCH expression, the field
// label, the display expression and the score band, in that order.
func fullSearchBranch(
	name string,
	f fullSearchField,
	search string,
	band string,
) queryBranch {
	parts := A.From[any](
		name,
		search,
		f.label,
		fullSearchScalarOrArray(f),
		band,
	)
	tpl := fullSearchBranchTemplate(f)
	return queryBranch{
		name: name,
		body: fmt.Sprintf(tpl, parts...),
	}
}

// fullSearchPrefixBranch fills the prefix branch of the identifier field
// at position i: the variable p<i> with the prefix band.
func fullSearchPrefixBranch(i int, f fullSearchField) queryBranch {
	return fullSearchBranch(
		fmt.Sprintf("p%d", i),
		f,
		fullSearchPrefixExpr(f.label),
		fullSearchPrefixBand,
	)
}

// fullSearchFuzzyBranch fills the fuzzy branch of the identifier field
// at position i: the variable n<i> with the fuzzy band.
func fullSearchFuzzyBranch(i int, f fullSearchField) queryBranch {
	return fullSearchBranch(
		fmt.Sprintf("n%d", i),
		f,
		fullSearchFuzzyExpr(f.label),
		fullSearchFuzzyBand,
	)
}

// fullSearchTokenBranch fills the token branch of the prose field at
// position i: the variable t<i> with the prose band.
func fullSearchTokenBranch(i int, f fullSearchField) queryBranch {
	return fullSearchBranch(
		fmt.Sprintf("t%d", i),
		f,
		fullSearchTokenExpr(f.label),
		fullSearchProseBand,
	)
}

// fullSearchPhraseBranch fills the single phrase branch over summary:
// the variable h0 with the phrase band.
func fullSearchPhraseBranch(f fullSearchField) queryBranch {
	return fullSearchBranch(
		"h0",
		f,
		fullSearchPhraseExpr(f.label),
		fullSearchPhraseBand,
	)
}

// fullSearchIdentifierPair makes both stages of one identifier field in
// statement order: the prefix branch, then the fuzzy branch.
func fullSearchIdentifierPair(i int, f fullSearchField) []queryBranch {
	prefix := fullSearchPrefixBranch(i, f)
	fuzzy := fullSearchFuzzyBranch(i, f)
	return A.From(prefix, fuzzy)
}

// fullSearchIdentifierBranches makes the prefix branch and the fuzzy
// branch of every identifier field.
func fullSearchIdentifierBranches() []queryBranch {
	return F.Pipe2(
		fullSearchIdentifierFields,
		A.MapWithIndex(fullSearchIdentifierPair),
		A.Flatten[queryBranch],
	)
}

// fullSearchProseBranches makes the token branch of every prose field.
func fullSearchProseBranches() []queryBranch {
	return F.Pipe1(
		fullSearchProseFields,
		A.MapWithIndex(fullSearchTokenBranch),
	)
}

// fullSearchBranches makes all 19 branches of the statement in statement
// order: both stages of every identifier field, then the token branch of
// every prose field, then the single phrase branch.
func fullSearchBranches() []queryBranch {
	identifiers := fullSearchIdentifierBranches()
	prose := fullSearchProseBranches()
	phraseBranch := fullSearchPhraseBranch(fullSearchProseFields[0])
	phrase := A.Of(phraseBranch)
	return F.Pipe1(
		A.From(identifiers, prose, phrase),
		A.Flatten[queryBranch],
	)
}

// fullSearchMergeBlock makes the merge of the statement. It keeps one
// best row per stock key, reads the stock document after the tail limit
// and returns the top rows.
func fullSearchMergeBlock(branches []queryBranch) string {
	return F.Pipe3(
		branches,
		A.Map(nameLens.Get),
		S.Join(", "),
		S.Format[string](statement.FullSearchMerge),
	)
}

// buildFullSearchQuery generates the full statement: the LET blocks of
// the branches and the merge block.
func buildFullSearchQuery() string {
	branches := fullSearchBranches()

	return F.Pipe3(
		branches,
		A.Map(bodyLens.Get),
		F.Bind2nd(A.Append, fullSearchMergeBlock(branches)),
		S.Join("\n"),
	)
}

// fullSearchPrefixExpr is the SEARCH expression of a prefix branch.
// STARTS_WITH inside SEARCH needs the ANALYZER wrapper: without it the
// comparison uses the identity analyzer and matches nothing.
func fullSearchPrefixExpr(field string) string {
	return fmt.Sprintf(
		"ANALYZER(STARTS_WITH(d.%s, @q), %q)",
		field,
		fullSearchNormAnalyzer,
	)
}

// fullSearchFuzzyExpr is the SEARCH expression of a fuzzy branch.
// NGRAM_MATCH takes the analyzer as its fourth argument.
func fullSearchFuzzyExpr(field string) string {
	return fmt.Sprintf(
		"NGRAM_MATCH(d.%s, @q, @th, %q)",
		field,
		fullSearchNgramAnalyzer,
	)
}

// fullSearchTokenExpr is the SEARCH expression of a token branch. The
// IN TOKENS form needs the ANALYZER wrapper, because the comparison of
// the indexed value against the token list must run under the same
// analyzer that produced the index.
func fullSearchTokenExpr(field string) string {
	return fmt.Sprintf(
		"ANALYZER(d.%s IN TOKENS(@q, %q), %q)",
		field,
		fullSearchTextAnalyzer,
		fullSearchTextAnalyzer,
	)
}

// fullSearchPhraseExpr is the SEARCH expression of the phrase branch.
// PHRASE takes the analyzer as its last argument.
func fullSearchPhraseExpr(field string) string {
	return fmt.Sprintf(
		"PHRASE(d.%s, @q, %q)",
		field,
		fullSearchTextAnalyzer,
	)
}
