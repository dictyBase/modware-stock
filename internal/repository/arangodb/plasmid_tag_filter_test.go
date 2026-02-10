package arangodb

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	F "github.com/IBM/fp-go/function"
	"github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
	"github.com/stretchr/testify/assert"
)

const (
	// Tag filter constants for testing
	tagFilterGatewayExact = `FILTER cvterm.label == 'Gateway vector'`
	tagFilterGatewayRegex = `FILTER cvterm.label =~ 'Gateway'`
	tagFilterVectorRegex  = `FILTER cvterm.label =~ 'vector'`
	tagFilterGFPExact     = `FILTER cvterm.label == 'GFP marker'`
	tagFilterNonExistent  = `FILTER cvterm.label == 'NonExistentOntologyTerm'`
)

func TestSelectPlasmidStatement(t *testing.T) {
	t.Parallel()
	ar := &arangorepository{}

	tests := []struct {
		name     string
		params   *stock.StockParameters
		expected string
	}{
		{
			name:     "No filter, no cursor",
			params:   &stock.StockParameters{},
			expected: statement.PlasmidList,
		},
		{
			name:     "No filter, with cursor",
			params:   &stock.StockParameters{Cursor: 123456789},
			expected: statement.PlasmidListWithCursor,
		},
		{
			name:     "With filter, no cursor",
			params:   &stock.StockParameters{Filter: "some filter"},
			expected: statement.PlasmidListFilterByOntology,
		},
		{
			name:     "With filter, with cursor",
			params:   &stock.StockParameters{Filter: "some filter", Cursor: 123456789},
			expected: statement.PlasmidListFilterByOntologyWithCursor,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := ar.selectPlasmidStatement(tc.params)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildFilterBindParams(t *testing.T) {
	// We need a connected repo to get collection names
	_, repo := setUp(t)
	defer tearDown(repo)
	ar := repo.(*arangorepository)

	tests := []struct {
		name   string
		params *stock.StockParameters
	}{
		{
			name:   "Filter without cursor",
			params: &stock.StockParameters{Filter: "filter", Limit: 10},
		},
		{
			name:   "Filter with cursor",
			params: &stock.StockParameters{Filter: "filter", Limit: 10, Cursor: 12345},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := ar.buildFilterBindParams(tc.params)

			// Assertions
			assert.Contains(t, params, "@cvterm_collection")
			assert.Contains(t, params, "@cv_collection")
			assert.Contains(t, params, "stock_prop_graph")
			assert.Contains(t, params, "stock_cvterm_graph")
			assert.Contains(t, params, "ontology")
			assert.Contains(t, params, "limit")
			assert.Equal(t, tc.params.Limit+1, params["limit"])

			// Should NOT contain stock collection (ontology first)
			assert.NotContains(t, params, "@stock_collection")

			if tc.params.Cursor > 0 {
				assert.Contains(t, params, "cursor")
				assert.Equal(t, tc.params.Cursor, params["cursor"])
			} else {
				assert.NotContains(t, params, "cursor")
			}
		})
	}
}

func TestBuildNoFilterBindParams(t *testing.T) {
	_, repo := setUp(t)
	defer tearDown(repo)
	ar := repo.(*arangorepository)

	tests := []struct {
		name   string
		params *stock.StockParameters
	}{
		{
			name:   "No filter without cursor",
			params: &stock.StockParameters{Limit: 10},
		},
		{
			name:   "No filter with cursor",
			params: &stock.StockParameters{Limit: 10, Cursor: 12345},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := ar.buildNoFilterBindParams(tc.params)

			// Assertions
			assert.Contains(t, params, "@stock_collection")
			assert.Contains(t, params, "stock_prop_graph")
			assert.Contains(t, params, "stock_cvterm_graph")
			assert.Contains(t, params, "ontology")
			assert.Contains(t, params, "@cv_collection")
			assert.Contains(t, params, "limit")
			assert.Equal(t, tc.params.Limit+1, params["limit"])

			// Should NOT contain cvterm collection (stock first)
			assert.NotContains(t, params, "@cvterm_collection")

			if tc.params.Cursor > 0 {
				assert.Contains(t, params, "cursor")
				assert.Equal(t, tc.params.Cursor, params["cursor"])
			} else {
				assert.NotContains(t, params, "cursor")
			}
		})
	}
}

func TestInjectFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filter   string
		stmt     string
		expected string
	}{
		{
			name:     "Empty filter",
			filter:   "",
			stmt:     "SELECT * FROM %s",
			expected: "SELECT * FROM %s", // Should remain unchanged if we use Identity, but let's check implementation
			// Implementation:
			// F.Ternary(..., Constant1(fmt.Sprintf(stmt, filter)), Constant1(stmt))
			// If filter is empty, hasFilter is false, so it returns stmt.
		},
		{
			name:     "Non-empty filter",
			filter:   "WHERE id=1",
			stmt:     "SELECT * FROM table %s",
			expected: "SELECT * FROM table WHERE id=1",
		},
		{
			name:     "Real template",
			filter:   tagFilterGatewayExact,
			stmt:     statement.PlasmidListFilterByOntology,
			expected: fmt.Sprintf(statement.PlasmidListFilterByOntology, tagFilterGatewayExact),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := injectFilter(tc.filter)(tc.stmt)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestListPlasmidsByTagExactMatch(t *testing.T) {
	req, repo := setUp(t)
	defer tearDown(repo)

	// Setup: Create 2 plasmids with different tags
	p1 := newTestPlasmid(fmt.Sprintf("%s@cye.com", arangomanager.RandomString(15, 25)))
	p1.Data.Attributes.DictyPlasmidProperty = OntologyTermGatewayVector
	res1 := F.Pipe2(repo.AddPlasmid(p1), ToEither, toStockDocResult)
	_, err := res1.F1, res1.F2
	req.NoError(err)

	p2 := newTestPlasmid(fmt.Sprintf("%s@cye.com", arangomanager.RandomString(15, 25)))
	p2.Data.Attributes.DictyPlasmidProperty = OntologyTermREMIVector
	res2 := F.Pipe2(repo.AddPlasmid(p2), ToEither, toStockDocResult)
	_, err = res2.F1, res2.F2
	req.NoError(err)

	// Execute: Query for Gateway vector
	result := F.Pipe2(
		repo.ListPlasmids(&stock.StockParameters{
			Limit:  10,
			Filter: tagFilterGatewayExact,
		}),
		ToEither,
		toStockDocListResult,
	)

	docs, err := result.F1, result.F2
	req.NoError(err)
	req.NotEmpty(docs)

	// Verify
	found := false
	for _, doc := range docs {
		req.Equal(OntologyTermGatewayVector, doc.PlasmidProperties.DictyPlasmidProperty)
		if doc.CreatedBy == p1.Data.Attributes.CreatedBy {
			found = true
		}
	}
	req.True(found, "Should find the created plasmid")
}

func TestListPlasmidsByTagRegexMatch(t *testing.T) {
	req, repo := setUp(t)
	defer tearDown(repo)

	// Setup: Create 3 plasmids
	terms := []string{OntologyTermGatewayVector, OntologyTermCloningVector, OntologyTermREMIVector}
	for _, term := range terms {
		p := newTestPlasmid(fmt.Sprintf("%s@cye.com", arangomanager.RandomString(15, 25)))
		p.Data.Attributes.DictyPlasmidProperty = term
		res := F.Pipe2(repo.AddPlasmid(p), ToEither, toStockDocResult)
		_, err := res.F1, res.F2
		req.NoError(err)
		time.Sleep(100 * time.Millisecond)
	}

	// Execute: Query for "vector" (should match all 3)
	result := F.Pipe2(
		repo.ListPlasmids(&stock.StockParameters{
			Limit:  10,
			Filter: tagFilterVectorRegex,
		}),
		ToEither,
		toStockDocListResult,
	)

	docs, err := result.F1, result.F2
	req.NoError(err)
	req.True(len(docs) >= 3, "Should find at least 3 plasmids")

	for _, doc := range docs {
		req.Regexp(regexp.MustCompile("vector"), doc.PlasmidProperties.DictyPlasmidProperty)
	}
}

func TestListPlasmidsByTagWithCursorPagination(t *testing.T) {
	req, repo := setUp(t)
	defer tearDown(repo)

	// Setup: Create 15 plasmids with Gateway vector tag
	for i := 0; i < 15; i++ {
		p := newTestPlasmid(fmt.Sprintf("%s@cye.com", arangomanager.RandomString(15, 25)))
		p.Data.Attributes.DictyPlasmidProperty = OntologyTermGatewayVector
		res := F.Pipe2(repo.AddPlasmid(p), ToEither, toStockDocResult)
		_, err := res.F1, res.F2
		req.NoError(err)
		time.Sleep(50 * time.Millisecond) // Smaller sleep to speed up
	}

	// Page 1
	limit := 5
	res1 := F.Pipe2(
		repo.ListPlasmids(&stock.StockParameters{
			Limit:  int64(limit),
			Filter: tagFilterGatewayExact,
		}),
		ToEither,
		toStockDocListResult,
	)
	page1, err := res1.F1, res1.F2
	req.NoError(err)
	req.Len(page1, limit+1, "Should return limit + 1 items")

	// Page 2
	lastItem := page1[len(page1)-1]
	cursor := toTimestamp(lastItem.CreatedAt)
	res2 := F.Pipe2(
		repo.ListPlasmids(&stock.StockParameters{
			Limit:  int64(limit),
			Cursor: cursor,
			Filter: tagFilterGatewayExact,
		}),
		ToEither,
		toStockDocListResult,
	)
	page2, err := res2.F1, res2.F2
	req.NoError(err)
	req.NotEmpty(page2)

	// Verify cursor overlap
	req.Equal(lastItem.StockID, page2[0].StockID, "First item of page 2 should be last item of page 1")

	// Verify no other overlap
	for i := 1; i < len(page2); i++ {
		for j := 0; j < len(page1)-1; j++ { // Exclude last item of page 1
			req.NotEqual(page1[j].StockID, page2[i].StockID)
		}
	}
}

func TestListPlasmidsByTagEmptyResults(t *testing.T) {
	req, repo := setUp(t)
	defer tearDown(repo)

	// Setup: Create 1 plasmid
	p := newTestPlasmid(fmt.Sprintf("%s@cye.com", arangomanager.RandomString(15, 25)))
	p.Data.Attributes.DictyPlasmidProperty = OntologyTermGFPMarker
	res := F.Pipe2(repo.AddPlasmid(p), ToEither, toStockDocResult)
	_, err := res.F1, res.F2
	req.NoError(err)

	// Execute: Query for non-existent tag
	result := F.Pipe2(
		repo.ListPlasmids(&stock.StockParameters{
			Limit:  10,
			Filter: tagFilterNonExistent,
		}),
		ToEither,
		toStockDocListResult,
	)

	docs, err := result.F1, result.F2
	req.NoError(err)
	req.Empty(docs)
}

func TestListPlasmidsByTagMultipleTerms(t *testing.T) {
	req, repo := setUp(t)
	defer tearDown(repo)

	// Setup: Create 3 plasmids with different terms
	p1 := newTestPlasmid("user1@example.com")
	p1.Data.Attributes.DictyPlasmidProperty = OntologyTermGatewayVector
	F.Pipe2(repo.AddPlasmid(p1), ToEither, toStockDocResult)

	p2 := newTestPlasmid("user2@example.com")
	p2.Data.Attributes.DictyPlasmidProperty = OntologyTermGFPMarker
	F.Pipe2(repo.AddPlasmid(p2), ToEither, toStockDocResult)

	p3 := newTestPlasmid("user3@example.com")
	p3.Data.Attributes.DictyPlasmidProperty = OntologyTermAct15Promoter
	F.Pipe2(repo.AddPlasmid(p3), ToEither, toStockDocResult)

	// Test Gateway
	res1 := F.Pipe2(
		repo.ListPlasmids(&stock.StockParameters{
			Limit:  10,
			Filter: tagFilterGatewayExact,
		}),
		ToEither,
		toStockDocListResult,
	)
	docs1, _ := res1.F1, res1.F2
	for _, d := range docs1 {
		req.Equal(OntologyTermGatewayVector, d.PlasmidProperties.DictyPlasmidProperty)
	}

	// Test GFP
	res2 := F.Pipe2(
		repo.ListPlasmids(&stock.StockParameters{
			Limit:  10,
			Filter: tagFilterGFPExact,
		}),
		ToEither,
		toStockDocListResult,
	)
	docs2, _ := res2.F1, res2.F2
	for _, d := range docs2 {
		req.Equal(OntologyTermGFPMarker, d.PlasmidProperties.DictyPlasmidProperty)
	}
}
