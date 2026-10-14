package arangodb

import (
	"fmt"

	A "github.com/IBM/fp-go/v2/array"
	F "github.com/IBM/fp-go/v2/function"
	L "github.com/IBM/fp-go/v2/optics/lens"
	P "github.com/IBM/fp-go/v2/predicate"
	S "github.com/IBM/fp-go/v2/string"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

type (
	// branchSpec is everything one branch of the statement needs: the AQL
	// variable name, the indexed field, the SEARCH expression and the
	// score band. The build pipeline resolves the display expression and
	// the branch template from the field and writes both back here, so
	// the render step reads one value and nothing else.
	branchSpec struct {
		name     string
		field    fullSearchField
		search   string
		band     string
		display  string
		template string
	}

	// fullSearchSpecSet pairs a group of indexed fields with the indexed
	// builder that produces one or more branch specs per field.
	fullSearchSpecSet struct {
		fields  []fullSearchField
		builder func(int, fullSearchField) []branchSpec
	}
)

var (
	// specDisplayLens focuses the resolved display expression of a spec.
	specDisplayLens = L.MakeLens(
		func(spec branchSpec) string { return spec.display },
		func(spec branchSpec, display string) branchSpec {
			spec.display = display
			return spec
		},
	)

	// specTemplateLens focuses the resolved branch template of a spec.
	specTemplateLens = L.MakeLens(
		func(spec branchSpec) string { return spec.template },
		func(spec branchSpec, template string) branchSpec {
			spec.template = template
			return spec
		},
	)

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

	// fullSearchSpecSets lists the three groups of branch specs in
	// statement order: identifier prefix+fuzzy pairs, prose token
	// branches, and the summary phrase branch.
	fullSearchSpecSets = []fullSearchSpecSet{
		{fullSearchIdentifierFields, fullSearchIdentifierSpecPair},
		{fullSearchProseFields, fullSearchTokenSpecSlice},
		{A.Of(fullSearchProseFields[0]), fullSearchPhraseSpecSlice},
	}
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

// withDisplay resolves the display expression of the field of one spec
// into that spec.
func withDisplay(spec branchSpec) branchSpec {
	display := fullSearchScalarOrArray(spec.field)
	return F.Pipe1(spec, specDisplayLens.Set(display))
}

// withTemplate resolves the branch template of the collection of the
// field of one spec into that spec.
func withTemplate(spec branchSpec) branchSpec {
	template := fullSearchBranchTemplate(spec.field)
	return F.Pipe1(spec, specTemplateLens.Set(template))
}

// renderBranch is the terminal projection of one spec: the template over
// the name, the SEARCH expression, the field label, the display
// expression and the score band, in template order.
func renderBranch(spec branchSpec) queryBranch {
	parts := A.From[any](
		spec.name,
		spec.search,
		spec.field.label,
		spec.display,
		spec.band,
	)
	return queryBranch{
		name: spec.name,
		body: fmt.Sprintf(spec.template, parts...),
	}
}

// fullSearchBranch builds one branch of the statement from its spec:
// resolve the display expression, resolve the collection template, then
// render.
func fullSearchBranch(spec branchSpec) queryBranch {
	return F.Pipe3(
		spec,
		withDisplay,
		withTemplate,
		renderBranch,
	)
}

// fullSearchPrefixSpec is the spec of the prefix branch of the
// identifier field at position i: the variable p<i> with the prefix
// band.
func fullSearchPrefixSpec(i int, f fullSearchField) branchSpec {
	return branchSpec{
		name:   fmt.Sprintf("p%d", i),
		field:  f,
		search: fullSearchPrefixExpr(f.label),
		band:   fullSearchPrefixBand,
	}
}

// fullSearchFuzzySpec is the spec of the fuzzy branch of the identifier
// field at position i: the variable n<i> with the fuzzy band.
func fullSearchFuzzySpec(i int, f fullSearchField) branchSpec {
	return branchSpec{
		name:   fmt.Sprintf("n%d", i),
		field:  f,
		search: fullSearchFuzzyExpr(f.label),
		band:   fullSearchFuzzyBand,
	}
}

// fullSearchTokenSpecSlice is the single-element spec slice of the token
// branch of the prose field at position i: the variable t<i> with the
// prose band.
func fullSearchTokenSpecSlice(i int, f fullSearchField) []branchSpec {
	return A.Of(branchSpec{
		name:   fmt.Sprintf("t%d", i),
		field:  f,
		search: fullSearchTokenExpr(f.label),
		band:   fullSearchProseBand,
	})
}

// fullSearchPhraseSpecSlice is the single-element spec slice of the
// phrase branch at position i: the variable h<i> with the phrase band.
func fullSearchPhraseSpecSlice(i int, f fullSearchField) []branchSpec {
	return A.Of(branchSpec{
		name:   fmt.Sprintf("h%d", i),
		field:  f,
		search: fullSearchPhraseExpr(f.label),
		band:   fullSearchPhraseBand,
	})
}

// fullSearchIdentifierSpecPair makes the specs of both stages of one
// identifier field in statement order: the prefix branch, then the fuzzy
// branch.
func fullSearchIdentifierSpecPair(i int, f fullSearchField) []branchSpec {
	return A.From(
		fullSearchPrefixSpec(i, f),
		fullSearchFuzzySpec(i, f),
	)
}

// specsFromSet expands one spec set into its branch specs in statement
// order.
func specsFromSet(set fullSearchSpecSet) []branchSpec {
	return F.Pipe2(
		set.fields,
		A.MapWithIndex(set.builder),
		A.Flatten[branchSpec],
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

// appendFullSearchMerge extracts the rendered LET bodies of the branches
// and appends the trailing merge block.
func appendFullSearchMerge(branches []queryBranch) []string {
	return F.Pipe2(
		branches,
		A.Map(bodyLens.Get),
		F.Bind2nd(A.Append, fullSearchMergeBlock(branches)),
	)
}

// buildFullSearchQuery generates the full statement: the LET blocks of
// the branches and the merge block.
func buildFullSearchQuery() string {
	return F.Pipe4(
		fullSearchSpecSets,
		A.Chain(specsFromSet),
		A.Map(fullSearchBranch),
		appendFullSearchMerge,
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
