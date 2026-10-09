package arangodb

import (
	"strings"
	"unicode"

	A "github.com/IBM/fp-go/v2/array"
	F "github.com/IBM/fp-go/v2/function"
	P "github.com/IBM/fp-go/v2/predicate"
	S "github.com/IBM/fp-go/v2/string"
	"golang.org/x/text/unicode/norm"
)

// isCombiningMark reports whether r belongs to the Unicode category Mn
// of the combining diacritical marks.
func isCombiningMark(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}

// runesToString turns the filtered rune slice back into a string. A Go
// type conversion is not a function value, so the pipeline names it.
func runesToString(runes []rune) string {
	return string(runes)
}

// normalizeAutocompleteQuery trims, lowercases and strips combining
// diacritical marks from the query. The calibration probe 10 decided
// this Go-side form: a punctuation-only query keeps its tokens safe,
// where an AQL TOKENS normalization would produce an empty token list
// and a null prefix argument.
func normalizeAutocompleteQuery(q string) string {
	return F.Pipe6(
		q,
		strings.TrimSpace,
		strings.ToLower,
		norm.NFD.String,
		S.ToRunes,
		A.Filter(P.Not(isCombiningMark)),
		runesToString,
	)
}
