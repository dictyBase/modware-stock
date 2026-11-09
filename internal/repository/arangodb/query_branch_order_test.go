package arangodb

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// assertPropBranchesHoistRanking walks the branch LET blocks of a built
// statement and asserts that every property branch ranks and caps the
// view rows before it resolves the owner. An ArangoSearch view branch
// does not receive the cluster-one-shard optimizer rule, so a traversal
// before the branch limit runs once per matching row over the network;
// after the limit it runs at most limit times per branch.
func assertPropBranchesHoistRanking(
	assert *require.Assertions,
	q string,
	branchRe string,
) {
	branches := regexp.MustCompile(branchRe)
	for branch := range strings.SplitSeq(q, "\nLET ") {
		if !strings.HasPrefix(branch, "LET ") {
			branch = "LET " + branch
		}
		if !branches.MatchString(branch) {
			continue
		}
		if !strings.Contains(branch, "INBOUND d GRAPH") {
			continue
		}
		limit := strings.Index(branch, "LIMIT @limit")
		own := strings.Index(branch, "LET own = FIRST(")
		assert.GreaterOrEqual(
			limit,
			0,
			"the property branch must cap its rows with LIMIT @limit",
		)
		assert.GreaterOrEqual(
			own,
			0,
			"the property branch must resolve its owner with one INBOUND traversal",
		)
		assert.Less(
			limit,
			own,
			"the owner traversal must run after the branch LIMIT",
		)
	}
}

// TestBuildFullSearchQuerySortsBeforeOwnerTraversal asserts the hoisted
// ranking of the five property branches of the full search statement.
func TestBuildFullSearchQuerySortsBeforeOwnerTraversal(t *testing.T) {
	assert := require.New(t)
	assertPropBranchesHoistRanking(
		assert,
		buildFullSearchQuery(),
		`LET [pnth]\d+ = \(`,
	)
}

// TestBuildAutocompleteQuerySortsBeforeOwnerTraversal asserts the
// hoisted ranking of the five property branches of the autocomplete
// statement.
func TestBuildAutocompleteQuerySortsBeforeOwnerTraversal(t *testing.T) {
	assert := require.New(t)
	assertPropBranchesHoistRanking(
		assert,
		buildAutocompleteQuery(),
		`LET [pn]_[a-z0-9_]+ = \(`,
	)
}
