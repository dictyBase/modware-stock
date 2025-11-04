package arangodb

import (
	"bufio"
	"strings"
	"testing"

	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/stretchr/testify/require"
)

// TestCheckStock tests the checkStock function with various scenarios
func TestCheckStock(t *testing.T) {
	t.Run("check stock that does not exist", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		repo, err := NewStockRepo(connParams, collParams, ontoParams)
		require.NoError(t, err, "Failed to create stock repository")
		defer func() { _ = repo.Dbh().Drop() }()

		// Try to check a non-existent stock
		_, err = repo.(*arangorepository).checkStock("DBS0000000")
		require.Error(t, err, "Should return error for non-existent stock")
		require.Contains(t, err.Error(), "absent in database", "Error should mention stock is absent")
	})

	t.Run("check stock with invalid ID format", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		repo, err := NewStockRepo(connParams, collParams, ontoParams)
		require.NoError(t, err, "Failed to create stock repository")
		defer func() { _ = repo.Dbh().Drop() }()

		// Try to check with invalid ID
		_, err = repo.(*arangorepository).checkStock("")
		require.Error(t, err, "Should return error for empty stock ID")
	})

	t.Run("check existing stock", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		err = loadData(testArango)
		require.NoError(t, err, "Failed to load test data")

		repo, err := NewStockRepo(connParams, collParams, ontoParams)
		require.NoError(t, err, "Failed to create stock repository")
		defer func() { _ = repo.Dbh().Drop() }()

		// Create a test strain first
		testStrain := newTestStrain("test@example.com", General)
		model, err := repo.AddStrain(testStrain)
		require.NoError(t, err, "Failed to add test strain")
		require.NotEmpty(t, model.Key, "Stock key should not be empty")

		// Check the stock
		propKey, err := repo.(*arangorepository).checkStock(model.Key)
		require.NoError(t, err, "Should successfully check existing stock")
		require.NotEmpty(t, propKey, "Property key should not be empty")
	})
}

// TestMergeBindParams tests the mergeBindParams function
//
//nolint:funlen // Test function with comprehensive test cases
func TestMergeBindParams(t *testing.T) {
	t.Run("merge empty maps", func(t *testing.T) {
		result := mergeBindParams()
		require.Empty(t, result, "Merging no maps should return empty map")
	})

	t.Run("merge single map", func(t *testing.T) {
		params := map[string]any{
			"key1": "value1",
			"key2": 42,
		}
		result := mergeBindParams(params)
		require.Equal(t, params, result, "Merging single map should return the same map")
	})

	t.Run("merge two maps without conflicts", func(t *testing.T) {
		params1 := map[string]any{
			"key1": "value1",
			"key2": 42,
		}
		params2 := map[string]any{
			"key3": "value3",
			"key4": true,
		}
		result := mergeBindParams(params1, params2)
		require.Len(t, result, 4, "Result should have all keys from both maps")
		require.Equal(t, "value1", result["key1"])
		require.Equal(t, 42, result["key2"])
		require.Equal(t, "value3", result["key3"])
		require.Equal(t, true, result["key4"])
	})

	t.Run("merge maps with conflicting keys - last wins", func(t *testing.T) {
		params1 := map[string]any{
			"key1": "original",
			"key2": 42,
		}
		params2 := map[string]any{
			"key1": "overridden",
			"key3": true,
		}
		result := mergeBindParams(params1, params2)
		require.Len(t, result, 3, "Result should have 3 unique keys")
		require.Equal(t, "overridden", result["key1"], "Last value should win for conflicting keys")
		require.Equal(t, 42, result["key2"])
		require.Equal(t, true, result["key3"])
	})

	t.Run("merge multiple maps", func(t *testing.T) {
		params1 := map[string]any{"a": 1}
		params2 := map[string]any{"b": 2}
		params3 := map[string]any{"c": 3}
		params4 := map[string]any{"d": 4}
		result := mergeBindParams(params1, params2, params3, params4)
		require.Len(t, result, 4)
		require.Equal(t, 1, result["a"])
		require.Equal(t, 2, result["b"])
		require.Equal(t, 3, result["c"])
		require.Equal(t, 4, result["d"])
	})

	t.Run("merge maps with various types", func(t *testing.T) {
		params1 := map[string]any{
			"string":  "text",
			"integer": 123,
		}
		params2 := map[string]any{
			"boolean": true,
			"float":   3.14,
		}
		params3 := map[string]any{
			"slice": []string{"a", "b", "c"},
			"map":   map[string]int{"x": 1, "y": 2},
		}
		result := mergeBindParams(params1, params2, params3)
		require.Len(t, result, 6)
		require.Equal(t, "text", result["string"])
		require.Equal(t, 123, result["integer"])
		require.Equal(t, true, result["boolean"])
		require.Equal(t, 3.14, result["float"])
		require.Equal(t, []string{"a", "b", "c"}, result["slice"])
		require.Equal(t, map[string]int{"x": 1, "y": 2}, result["map"])
	})
}

// TestGenAQLDocExpression tests the genAQLDocExpression function
func TestGenAQLDocExpression(t *testing.T) {
	t.Run("empty bind variables", func(t *testing.T) {
		result := genAQLDocExpression(map[string]any{})
		require.Empty(t, result, "Empty map should produce empty expression")
	})

	t.Run("single bind variable", func(t *testing.T) {
		bindVars := map[string]any{
			"name": "test",
		}
		result := genAQLDocExpression(bindVars)
		require.Equal(t, "name: @name", result)
	})

	t.Run("multiple bind variables", func(t *testing.T) {
		bindVars := map[string]any{
			"name":  "test",
			"email": "test@example.com",
			"age":   30,
		}
		result := genAQLDocExpression(bindVars)
		// Result should contain all bindings separated by commas
		require.Contains(t, result, "name: @name")
		require.Contains(t, result, "email: @email")
		require.Contains(t, result, "age: @age")
		require.Contains(t, result, ",")
	})

	t.Run("bind variables with special characters in keys", func(t *testing.T) {
		bindVars := map[string]any{
			"user_id":   123,
			"item_name": "product",
		}
		result := genAQLDocExpression(bindVars)
		require.Contains(t, result, "user_id: @user_id")
		require.Contains(t, result, "item_name: @item_name")
	})
}

// TestFormatAQLBinding tests the formatAQLBinding function
func TestFormatAQLBinding(t *testing.T) {
	t.Run("format simple key", func(t *testing.T) {
		result := formatAQLBinding("name")
		require.Equal(t, "name: @name", result)
	})

	t.Run("format key with underscore", func(t *testing.T) {
		result := formatAQLBinding("user_id")
		require.Equal(t, "user_id: @user_id", result)
	})

	t.Run("format empty key", func(t *testing.T) {
		result := formatAQLBinding("")
		require.Equal(t, ": @", result)
	})

	t.Run("format key with numbers", func(t *testing.T) {
		result := formatAQLBinding("key123")
		require.Equal(t, "key123: @key123", result)
	})
}

// TestNormalizeSliceBindParam tests the normalizeSliceBindParam function
func TestNormalizeSliceBindParam(t *testing.T) {
	t.Run("normalize empty slice", func(t *testing.T) {
		result := normalizeSliceBindParam([]string{})
		require.NotNil(t, result, "Should return non-nil slice")
		require.Empty(t, result, "Should return empty slice")
	})

	t.Run("normalize nil slice", func(t *testing.T) {
		result := normalizeSliceBindParam(nil)
		require.NotNil(t, result, "Should return non-nil slice")
		require.Empty(t, result, "Should return empty slice")
	})

	t.Run("normalize non-empty slice", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := normalizeSliceBindParam(input)
		require.Equal(t, input, result, "Should return same slice")
	})

	t.Run("normalize single element slice", func(t *testing.T) {
		input := []string{"single"}
		result := normalizeSliceBindParam(input)
		require.Equal(t, input, result, "Should return same slice")
	})
}

// TestNormalizeStrBindParam tests the normalizeStrBindParam function
func TestNormalizeStrBindParam(t *testing.T) {
	t.Run("normalize empty string", func(t *testing.T) {
		result := normalizeStrBindParam("")
		require.Equal(t, "", result, "Should return empty string")
	})

	t.Run("normalize non-empty string", func(t *testing.T) {
		result := normalizeStrBindParam("test")
		require.Equal(t, "test", result, "Should return same string")
	})

	t.Run("normalize whitespace string", func(t *testing.T) {
		result := normalizeStrBindParam("   ")
		require.Equal(t, "   ", result, "Should return same whitespace string")
	})
}

// TestTermID tests the termID function with various scenarios
//
//nolint:funlen // Test function with comprehensive test cases
func TestTermID(t *testing.T) {
	t.Run("find existing term ID", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		err = loadData(testArango)
		require.NoError(t, err, "Failed to load test data")

		repo, err := NewStockRepo(connParams, collParams, ontoParams)
		require.NoError(t, err, "Failed to create stock repository")
		defer func() { _ = repo.Dbh().Drop() }()

		ar := repo.(*arangorepository)

		// Try to find a term that should exist (from dicty_strain_property.json)
		termID, err := ar.termID("general strain", "dicty_strain_property")
		require.NoError(t, err, "Should find existing term")
		require.NotEmpty(t, termID, "Term ID should not be empty")
	})

	t.Run("term does not exist", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		err = loadData(testArango)
		require.NoError(t, err, "Failed to load test data")

		repo, err := NewStockRepo(connParams, collParams, ontoParams)
		require.NoError(t, err, "Failed to create stock repository")
		defer func() { _ = repo.Dbh().Drop() }()

		ar := repo.(*arangorepository)

		// Try to find a term that doesn't exist
		_, err = ar.termID("nonexistent_term", "dicty_strain_property")
		require.Error(t, err, "Should return error for non-existent term")
		require.Contains(t, err.Error(), "does not exist", "Error should mention term does not exist")
	})

	t.Run("ontology does not exist", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		err = loadData(testArango)
		require.NoError(t, err, "Failed to load test data")

		repo, err := NewStockRepo(connParams, collParams, ontoParams)
		require.NoError(t, err, "Failed to create stock repository")
		defer func() { _ = repo.Dbh().Drop() }()

		ar := repo.(*arangorepository)

		// Try to find a term in a non-existent ontology
		_, err = ar.termID("general strain", "nonexistent_ontology")
		require.Error(t, err, "Should return error for non-existent ontology")
		require.Contains(t, err.Error(), "does not exist", "Error should mention ontology does not exist")
	})

	t.Run("empty term name", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		err = loadData(testArango)
		require.NoError(t, err, "Failed to load test data")

		repo, err := NewStockRepo(connParams, collParams, ontoParams)
		require.NoError(t, err, "Failed to create stock repository")
		defer func() { _ = repo.Dbh().Drop() }()

		ar := repo.(*arangorepository)

		// Try to find a term with empty name
		_, err = ar.termID("", "dicty_strain_property")
		require.Error(t, err, "Should return error for empty term name")
	})

	t.Run("empty ontology name", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		err = loadData(testArango)
		require.NoError(t, err, "Failed to load test data")

		repo, err := NewStockRepo(connParams, collParams, ontoParams)
		require.NoError(t, err, "Failed to create stock repository")
		defer func() { _ = repo.Dbh().Drop() }()

		ar := repo.(*arangorepository)

		// Try to find a term with empty ontology
		_, err = ar.termID("general strain", "")
		require.Error(t, err, "Should return error for empty ontology name")
	})
}

// TestLoadOboJSON tests the LoadOboJSON function with various scenarios
func TestLoadOboJSON_ErrorCases(t *testing.T) {
	t.Run("load invalid JSON", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		repo, err := NewStockRepo(connParams, collParams, ontoParams)
		require.NoError(t, err, "Failed to create stock repository")
		defer func() { _ = repo.Dbh().Drop() }()

		// Create a reader with invalid JSON
		invalidJSON := strings.NewReader(`{"invalid": "json", "missing": "closing brace"`)
		_, err = repo.LoadOboJSON(bufio.NewReader(invalidJSON))
		require.Error(t, err, "Should return error for invalid JSON")
	})

	t.Run("load malformed OBO JSON", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		repo, err := NewStockRepo(connParams, collParams, ontoParams)
		require.NoError(t, err, "Failed to create stock repository")
		defer func() { _ = repo.Dbh().Drop() }()

		// Create a reader with valid JSON but invalid OBO structure
		malformedOBO := strings.NewReader(`{"graphs": [{"nodes": "invalid"}]}`)
		_, err = repo.LoadOboJSON(bufio.NewReader(malformedOBO))
		require.Error(t, err, "Should return error for malformed OBO JSON")
	})
}

// TestDbh tests the Dbh accessor method
func TestDbh(t *testing.T) {
	t.Run("get database handle", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		repo, err := NewStockRepo(connParams, collParams, ontoParams)
		require.NoError(t, err, "Failed to create stock repository")
		defer func() { _ = repo.Dbh().Drop() }()

		dbHandle := repo.Dbh()
		require.NotNil(t, dbHandle, "Database handle should not be nil")
		// Just verify we can get the database - don't compare instances
	})
}

// TestNewStockRepo tests the NewStockRepo constructor with various scenarios
func TestNewStockRepo_ValidationErrors(t *testing.T) {
	testArango, err := testarango.NewTestArangoFromEnv(true)
	require.NoError(t, err, "Failed to create test arango instance")

	connParams := getConnectParamsFromDb(testArango)
	collParams := getCollectionParams()
	ontoParams := getOntoParams()

	t.Run("invalid collection params - missing stock collection", func(t *testing.T) {
		invalidCollParams := &CollectionParams{
			Stock:              "", // Invalid: required field
			StockProp:          collParams.StockProp,
			StockType:          collParams.StockType,
			StockKeyGenerator:  collParams.StockKeyGenerator,
			ParentStrain:       collParams.ParentStrain,
			StockTerm:          collParams.StockTerm,
			StockPropTypeGraph: collParams.StockPropTypeGraph,
			Strain2ParentGraph: collParams.Strain2ParentGraph,
			StockOntoGraph:     collParams.StockOntoGraph,
			KeyOffset:          collParams.KeyOffset,
			StrainOntology:     collParams.StrainOntology,
			PlasmidOntology:    collParams.PlasmidOntology,
		}

		_, err := NewStockRepo(connParams, invalidCollParams, ontoParams)
		require.Error(t, err, "Should fail validation with missing stock collection")
	})

	t.Run("invalid collection params - missing stock prop collection", func(t *testing.T) {
		invalidCollParams := &CollectionParams{
			Stock:              collParams.Stock,
			StockProp:          "", // Invalid: required field
			StockType:          collParams.StockType,
			StockKeyGenerator:  collParams.StockKeyGenerator,
			ParentStrain:       collParams.ParentStrain,
			StockTerm:          collParams.StockTerm,
			StockPropTypeGraph: collParams.StockPropTypeGraph,
			Strain2ParentGraph: collParams.Strain2ParentGraph,
			StockOntoGraph:     collParams.StockOntoGraph,
			KeyOffset:          collParams.KeyOffset,
			StrainOntology:     collParams.StrainOntology,
			PlasmidOntology:    collParams.PlasmidOntology,
		}

		_, err := NewStockRepo(connParams, invalidCollParams, ontoParams)
		require.Error(t, err, "Should fail validation with missing stock prop collection")
	})

	t.Run("invalid connection params", func(t *testing.T) {
		invalidConnParams := &manager.ConnectParams{
			User:     "",
			Pass:     "",
			Database: "",
			Host:     "invalid-host-12345",
			Port:     9999,
			Istls:    false,
		}

		_, err := NewStockRepo(invalidConnParams, collParams, ontoParams)
		require.Error(t, err, "Should fail with invalid connection params")
		require.Contains(
			t,
			err.Error(),
			"error in creating database session",
			"Error should mention database session creation",
		)
	})
}

// TestConcatOptionalParams tests the concatOptionalParams curried function
func TestConcatOptionalParams(t *testing.T) {
	t.Run("concat with empty optional params", func(t *testing.T) {
		baseParams := map[string]any{
			"id":   123,
			"name": "test",
		}
		optionalParams := []map[string]any{}

		result := concatOptionalParams(baseParams)(optionalParams)
		require.Equal(t, baseParams, result, "Should return base params when optional is empty")
	})

	t.Run("concat with one optional param", func(t *testing.T) {
		baseParams := map[string]any{
			"id": 123,
		}
		optionalParams := []map[string]any{
			{"name": "test"},
		}

		result := concatOptionalParams(baseParams)(optionalParams)
		require.Len(t, result, 2, "Result should have both base and optional params")
		require.Equal(t, 123, result["id"])
		require.Equal(t, "test", result["name"])
	})

	t.Run("concat with multiple optional params", func(t *testing.T) {
		baseParams := map[string]any{
			"id": 123,
		}
		optionalParams := []map[string]any{
			{"name": "test"},
			{"email": "test@example.com"},
			{"age": 30},
		}

		result := concatOptionalParams(baseParams)(optionalParams)
		require.Len(t, result, 4, "Result should have all params")
		require.Equal(t, 123, result["id"])
		require.Equal(t, "test", result["name"])
		require.Equal(t, "test@example.com", result["email"])
		require.Equal(t, 30, result["age"])
	})

	t.Run("concat with overlapping keys - last wins", func(t *testing.T) {
		baseParams := map[string]any{
			"id":   123,
			"name": "original",
		}
		optionalParams := []map[string]any{
			{"name": "updated"},
			{"name": "final"},
		}

		result := concatOptionalParams(baseParams)(optionalParams)
		require.Equal(t, "final", result["name"], "Last value should win for overlapping keys")
	})
}
