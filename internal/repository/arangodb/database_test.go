package arangodb

import (
	"context"
	"testing"

	driver "github.com/arangodb/go-driver"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	ontoarango "github.com/dictyBase/go-obograph/storage/arangodb"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a test repository with ontology collections
func setupTestRepo(
	t *testing.T,
	connParams *manager.ConnectParams,
	collParams *CollectionParams,
	ontoParams *ontoarango.CollectionParams,
) (*arangorepository, func()) {
	t.Helper()
	sess, db, err := manager.NewSessionDb(connParams)
	require.NoError(t, err, "Failed to create session and database")

	ontoc, err := ontoarango.CreateCollection(db, ontoParams)
	require.NoError(t, err, "Failed to create ontology collections")

	repo := &arangorepository{
		ontoc:       ontoc,
		sess:        sess,
		database:    db,
		strainOnto:  collParams.StrainOntology,
		plasmidOnto: collParams.PlasmidOntology,
	}

	cleanup := func() {
		_ = db.Drop()
	}

	return repo, cleanup
}

// verifyStockCollections verifies that all stock collections were created
func verifyStockCollections(t *testing.T, repo *arangorepository) {
	t.Helper()
	require.NotNil(t, repo.stockc, "stockc should be initialized")
	require.NotNil(t, repo.stockc.stock, "stock collection should exist")
	require.NotNil(t, repo.stockc.stockProp, "stockProp collection should exist")
	require.NotNil(t, repo.stockc.stockKey, "stockKey collection should exist")
	require.NotNil(t, repo.stockc.stockType, "stockType collection should exist")
	require.NotNil(t, repo.stockc.parentStrain, "parentStrain collection should exist")
	require.NotNil(t, repo.stockc.stockTerm, "stockTerm collection should exist")
	require.NotNil(t, repo.stockc.stockPropType, "stockPropType graph should exist")
	require.NotNil(t, repo.stockc.strain2Parent, "strain2Parent graph should exist")
	require.NotNil(t, repo.stockc.stockOnto, "stockOnto graph should exist")
}

// TestCreateDbStruct tests the database structure creation with various scenarios
func TestCreateDbStruct(t *testing.T) {
	t.Run("successful database structure creation", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		repo, cleanup := setupTestRepo(t, connParams, collParams, ontoParams)
		defer cleanup()

		err = createDbStruct(repo, collParams)
		require.NoError(t, err, "Failed to create database structure")

		verifyStockCollections(t, repo)
	})

	t.Run("error in document collections creation", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		repo, cleanup := setupTestRepo(t, connParams, collParams, ontoParams)
		defer cleanup()

		invalidCollParams := copyCollectionParamsWithOverride(collParams, func(c *CollectionParams) {
			c.Stock = ""
		})
		err = createDbStruct(repo, invalidCollParams)
		require.Error(t, err, "Should fail with invalid collection parameters")
	})
}

// TestDocCollections tests document collection creation with various scenarios
//
//nolint:funlen // Test function with comprehensive test cases
func TestDocCollections(t *testing.T) {
	t.Run("successful document collections creation", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		sess, db, err := manager.NewSessionDb(connParams)
		require.NoError(t, err, "Failed to create session and database")
		defer func() { _ = db.Drop() }()
		_ = sess

		ontoc, err := ontoarango.CreateCollection(db, ontoParams)
		require.NoError(t, err, "Failed to create ontology collections")

		repo := &arangorepository{
			ontoc:    ontoc,
			sess:     sess,
			database: db,
		}

		err = docCollections(repo, collParams)
		require.NoError(t, err, "Failed to create document collections")

		// Verify collections were created
		require.NotNil(t, repo.stockc, "stockc should be initialized")
		require.NotNil(t, repo.stockc.stock, "stock collection should exist")
		require.NotNil(t, repo.stockc.stockProp, "stockProp collection should exist")
		require.NotNil(t, repo.stockc.stockKey, "stockKey collection should exist")
	})

	t.Run("error creating stock collection", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		sess, db, err := manager.NewSessionDb(connParams)
		require.NoError(t, err, "Failed to create session and database")
		defer func() { _ = db.Drop() }()
		_ = sess

		ontoc, err := ontoarango.CreateCollection(db, ontoParams)
		require.NoError(t, err, "Failed to create ontology collections")

		repo := &arangorepository{
			ontoc:    ontoc,
			sess:     sess,
			database: db,
		}

		invalidCollParams := &CollectionParams{
			Stock:             "", // Invalid: empty collection name
			StockProp:         collParams.StockProp,
			StockKeyGenerator: collParams.StockKeyGenerator,
			KeyOffset:         collParams.KeyOffset,
		}

		err = docCollections(repo, invalidCollParams)
		require.Error(t, err, "Should fail with invalid stock collection name")
		require.Contains(t, err.Error(), "error in creating collection", "Error message should mention collection creation")
	})

	t.Run("error creating stock properties collection", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		sess, db, err := manager.NewSessionDb(connParams)
		require.NoError(t, err, "Failed to create session and database")
		defer func() { _ = db.Drop() }()
		_ = sess

		ontoc, err := ontoarango.CreateCollection(db, ontoParams)
		require.NoError(t, err, "Failed to create ontology collections")

		repo := &arangorepository{
			ontoc:    ontoc,
			sess:     sess,
			database: db,
		}

		invalidCollParams := &CollectionParams{
			Stock:             collParams.Stock,
			StockProp:         "", // Invalid: empty collection name
			StockKeyGenerator: collParams.StockKeyGenerator,
			KeyOffset:         collParams.KeyOffset,
		}

		err = docCollections(repo, invalidCollParams)
		require.Error(t, err, "Should fail with invalid stock properties collection name")
		require.Contains(t, err.Error(), "error in creating collection", "Error message should mention collection creation")
	})

	t.Run("error creating stock key generator collection", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		sess, db, err := manager.NewSessionDb(connParams)
		require.NoError(t, err, "Failed to create session and database")
		defer func() { _ = db.Drop() }()
		_ = sess

		ontoc, err := ontoarango.CreateCollection(db, ontoParams)
		require.NoError(t, err, "Failed to create ontology collections")

		repo := &arangorepository{
			ontoc:    ontoc,
			sess:     sess,
			database: db,
		}

		invalidCollParams := &CollectionParams{
			Stock:             collParams.Stock,
			StockProp:         collParams.StockProp,
			StockKeyGenerator: "", // Invalid: empty collection name
			KeyOffset:         collParams.KeyOffset,
		}

		err = docCollections(repo, invalidCollParams)
		require.Error(t, err, "Should fail with invalid stock key generator collection name")
		require.Contains(t, err.Error(), "error in creating collection", "Error message should mention collection creation")
	})
}

// verifyEdgeCollections verifies that all edge collections were created
func verifyEdgeCollections(t *testing.T, repo *arangorepository) {
	t.Helper()
	require.NotNil(t, repo.stockc.stockType, "stockType edge collection should exist")
	require.NotNil(t, repo.stockc.parentStrain, "parentStrain edge collection should exist")
	require.NotNil(t, repo.stockc.stockTerm, "stockTerm edge collection should exist")
}

// verifyNamedGraphs verifies that all named graphs were created
func verifyNamedGraphs(t *testing.T, repo *arangorepository) {
	t.Helper()
	require.NotNil(t, repo.stockc.stockPropType, "stockPropType graph should exist")
	require.NotNil(t, repo.stockc.strain2Parent, "strain2Parent graph should exist")
	require.NotNil(t, repo.stockc.stockOnto, "stockOnto graph should exist")
}

// setupRepoWithDocCollections sets up a repository with document collections
func setupRepoWithDocCollections(
	t *testing.T,
	connParams *manager.ConnectParams,
	collParams *CollectionParams,
	ontoParams *ontoarango.CollectionParams,
) (*arangorepository, func()) {
	t.Helper()
	sess, db, err := manager.NewSessionDb(connParams)
	require.NoError(t, err, "Failed to create session and database")

	ontoc, err := ontoarango.CreateCollection(db, ontoParams)
	require.NoError(t, err, "Failed to create ontology collections")

	repo := &arangorepository{
		ontoc:    ontoc,
		sess:     sess,
		database: db,
	}

	err = docCollections(repo, collParams)
	require.NoError(t, err, "Failed to create document collections")

	cleanup := func() {
		_ = db.Drop()
	}

	return repo, cleanup
}

// TestGraphAndEdgeCollections tests graph and edge collection creation
func TestGraphAndEdgeCollections(t *testing.T) {
	t.Run("successful graph and edge collections creation", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		repo, cleanup := setupRepoWithDocCollections(t, connParams, collParams, ontoParams)
		defer cleanup()

		err = graphAndEdgeCollections(repo, collParams)
		require.NoError(t, err, "Failed to create graph and edge collections")

		verifyEdgeCollections(t, repo)
		verifyNamedGraphs(t, repo)
	})

	t.Run("error in edge collections creation", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		repo, cleanup := setupRepoWithDocCollections(t, connParams, collParams, ontoParams)
		defer cleanup()

		invalidCollParams := &CollectionParams{
			Stock:              collParams.Stock,
			StockProp:          collParams.StockProp,
			StockKeyGenerator:  collParams.StockKeyGenerator,
			StockType:          "", // Invalid: empty edge collection name
			ParentStrain:       collParams.ParentStrain,
			StockTerm:          collParams.StockTerm,
			StockPropTypeGraph: collParams.StockPropTypeGraph,
			Strain2ParentGraph: collParams.Strain2ParentGraph,
			StockOntoGraph:     collParams.StockOntoGraph,
			KeyOffset:          collParams.KeyOffset,
		}

		err = graphAndEdgeCollections(repo, invalidCollParams)
		require.Error(t, err, "Should fail with invalid edge collection parameters")
	})
}

// TestCreateEdgeCollections tests edge collection creation with various scenarios
//
//nolint:funlen // Test function with comprehensive test cases
func TestCreateEdgeCollections(t *testing.T) {
	t.Run("successful edge collections creation", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		sess, db, err := manager.NewSessionDb(connParams)
		require.NoError(t, err, "Failed to create session and database")
		defer func() { _ = db.Drop() }()
		_ = sess

		ontoc, err := ontoarango.CreateCollection(db, ontoParams)
		require.NoError(t, err, "Failed to create ontology collections")

		repo := &arangorepository{
			ontoc:    ontoc,
			sess:     sess,
			database: db,
		}

		// Create document collections first
		err = docCollections(repo, collParams)
		require.NoError(t, err, "Failed to create document collections")

		// Create edge collections
		err = createEdgeCollections(repo, collParams)
		require.NoError(t, err, "Failed to create edge collections")

		// Verify edge collections were created
		require.NotNil(t, repo.stockc.stockType, "stockType edge collection should exist")
		require.NotNil(t, repo.stockc.parentStrain, "parentStrain edge collection should exist")
		require.NotNil(t, repo.stockc.stockTerm, "stockTerm edge collection should exist")
	})

	t.Run("error creating stock type edge collection", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		sess, db, err := manager.NewSessionDb(connParams)
		require.NoError(t, err, "Failed to create session and database")
		defer func() { _ = db.Drop() }()
		_ = sess

		ontoc, err := ontoarango.CreateCollection(db, ontoParams)
		require.NoError(t, err, "Failed to create ontology collections")

		repo := &arangorepository{
			ontoc:    ontoc,
			sess:     sess,
			database: db,
		}

		err = docCollections(repo, collParams)
		require.NoError(t, err, "Failed to create document collections")

		invalidCollParams := &CollectionParams{
			StockType:    "", // Invalid: empty edge collection name
			ParentStrain: collParams.ParentStrain,
			StockTerm:    collParams.StockTerm,
		}

		err = createEdgeCollections(repo, invalidCollParams)
		require.Error(t, err, "Should fail with invalid stock type edge collection name")
		require.Contains(
			t,
			err.Error(),
			"error in creating edge collection",
			"Error message should mention edge collection creation",
		)
	})

	t.Run("error creating parent strain edge collection", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		sess, db, err := manager.NewSessionDb(connParams)
		require.NoError(t, err, "Failed to create session and database")
		defer func() { _ = db.Drop() }()
		_ = sess

		ontoc, err := ontoarango.CreateCollection(db, ontoParams)
		require.NoError(t, err, "Failed to create ontology collections")

		repo := &arangorepository{
			ontoc:    ontoc,
			sess:     sess,
			database: db,
		}

		err = docCollections(repo, collParams)
		require.NoError(t, err, "Failed to create document collections")

		invalidCollParams := &CollectionParams{
			StockType:    collParams.StockType,
			ParentStrain: "", // Invalid: empty edge collection name
			StockTerm:    collParams.StockTerm,
		}

		err = createEdgeCollections(repo, invalidCollParams)
		require.Error(t, err, "Should fail with invalid parent strain edge collection name")
		require.Contains(
			t,
			err.Error(),
			"error in creating edge collection",
			"Error message should mention edge collection creation",
		)
	})

	t.Run("error creating stock term edge collection", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		sess, db, err := manager.NewSessionDb(connParams)
		require.NoError(t, err, "Failed to create session and database")
		defer func() { _ = db.Drop() }()
		_ = sess

		ontoc, err := ontoarango.CreateCollection(db, ontoParams)
		require.NoError(t, err, "Failed to create ontology collections")

		repo := &arangorepository{
			ontoc:    ontoc,
			sess:     sess,
			database: db,
		}

		err = docCollections(repo, collParams)
		require.NoError(t, err, "Failed to create document collections")

		invalidCollParams := &CollectionParams{
			StockType:    collParams.StockType,
			ParentStrain: collParams.ParentStrain,
			StockTerm:    "", // Invalid: empty edge collection name
		}

		err = createEdgeCollections(repo, invalidCollParams)
		require.Error(t, err, "Should fail with invalid stock term edge collection name")
		require.Contains(
			t,
			err.Error(),
			"error in creating edge collection",
			"Error message should mention edge collection creation",
		)
	})
}

// TestCreateNamedGraph tests named graph creation with various scenarios
//
//nolint:funlen // Test function with comprehensive test cases
func TestCreateNamedGraph(t *testing.T) {
	t.Run("successful named graph creation", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		sess, db, err := manager.NewSessionDb(connParams)
		require.NoError(t, err, "Failed to create session and database")
		defer func() { _ = db.Drop() }()
		_ = sess

		ontoc, err := ontoarango.CreateCollection(db, ontoParams)
		require.NoError(t, err, "Failed to create ontology collections")

		repo := &arangorepository{
			ontoc:    ontoc,
			sess:     sess,
			database: db,
		}

		// Create document and edge collections first
		err = docCollections(repo, collParams)
		require.NoError(t, err, "Failed to create document collections")

		err = createEdgeCollections(repo, collParams)
		require.NoError(t, err, "Failed to create edge collections")

		// Create named graphs
		err = createNamedGraph(repo, collParams)
		require.NoError(t, err, "Failed to create named graphs")

		// Verify graphs were created
		require.NotNil(t, repo.stockc.stockPropType, "stockPropType graph should exist")
		require.NotNil(t, repo.stockc.strain2Parent, "strain2Parent graph should exist")
		require.NotNil(t, repo.stockc.stockOnto, "stockOnto graph should exist")
	})

	t.Run("error creating stock property type graph", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		sess, db, err := manager.NewSessionDb(connParams)
		require.NoError(t, err, "Failed to create session and database")
		defer func() { _ = db.Drop() }()
		_ = sess

		ontoc, err := ontoarango.CreateCollection(db, ontoParams)
		require.NoError(t, err, "Failed to create ontology collections")

		repo := &arangorepository{
			ontoc:    ontoc,
			sess:     sess,
			database: db,
		}

		err = docCollections(repo, collParams)
		require.NoError(t, err, "Failed to create document collections")

		err = createEdgeCollections(repo, collParams)
		require.NoError(t, err, "Failed to create edge collections")

		invalidCollParams := &CollectionParams{
			StockPropTypeGraph: "", // Invalid: empty graph name
			Strain2ParentGraph: collParams.Strain2ParentGraph,
			StockOntoGraph:     collParams.StockOntoGraph,
		}
		// Need to copy stockc for named graph creation
		invalidCollParams.Stock = collParams.Stock
		invalidCollParams.StockProp = collParams.StockProp
		invalidCollParams.StockType = collParams.StockType
		invalidCollParams.ParentStrain = collParams.ParentStrain
		invalidCollParams.StockTerm = collParams.StockTerm

		err = createNamedGraph(repo, invalidCollParams)
		require.Error(t, err, "Should fail with invalid graph name")
		require.Contains(t, err.Error(), "error in creating named graph", "Error message should mention named graph creation")
	})

	t.Run("error creating strain to parent graph", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		sess, db, err := manager.NewSessionDb(connParams)
		require.NoError(t, err, "Failed to create session and database")
		defer func() { _ = db.Drop() }()
		_ = sess

		ontoc, err := ontoarango.CreateCollection(db, ontoParams)
		require.NoError(t, err, "Failed to create ontology collections")

		repo := &arangorepository{
			ontoc:    ontoc,
			sess:     sess,
			database: db,
		}

		err = docCollections(repo, collParams)
		require.NoError(t, err, "Failed to create document collections")

		err = createEdgeCollections(repo, collParams)
		require.NoError(t, err, "Failed to create edge collections")

		invalidCollParams := &CollectionParams{
			StockPropTypeGraph: collParams.StockPropTypeGraph,
			Strain2ParentGraph: "", // Invalid: empty graph name
			StockOntoGraph:     collParams.StockOntoGraph,
		}
		invalidCollParams.Stock = collParams.Stock
		invalidCollParams.StockProp = collParams.StockProp
		invalidCollParams.StockType = collParams.StockType
		invalidCollParams.ParentStrain = collParams.ParentStrain
		invalidCollParams.StockTerm = collParams.StockTerm

		err = createNamedGraph(repo, invalidCollParams)
		require.Error(t, err, "Should fail with invalid strain2parent graph name")
		require.Contains(t, err.Error(), "error in creating named graph", "Error message should mention named graph creation")
	})

	t.Run("error creating stock ontology graph", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		sess, db, err := manager.NewSessionDb(connParams)
		require.NoError(t, err, "Failed to create session and database")
		defer func() { _ = db.Drop() }()
		_ = sess

		ontoc, err := ontoarango.CreateCollection(db, ontoParams)
		require.NoError(t, err, "Failed to create ontology collections")

		repo := &arangorepository{
			ontoc:    ontoc,
			sess:     sess,
			database: db,
		}

		err = docCollections(repo, collParams)
		require.NoError(t, err, "Failed to create document collections")

		err = createEdgeCollections(repo, collParams)
		require.NoError(t, err, "Failed to create edge collections")

		invalidCollParams := &CollectionParams{
			StockPropTypeGraph: collParams.StockPropTypeGraph,
			Strain2ParentGraph: collParams.Strain2ParentGraph,
			StockOntoGraph:     "", // Invalid: empty graph name
		}
		invalidCollParams.Stock = collParams.Stock
		invalidCollParams.StockProp = collParams.StockProp
		invalidCollParams.StockType = collParams.StockType
		invalidCollParams.ParentStrain = collParams.ParentStrain
		invalidCollParams.StockTerm = collParams.StockTerm

		err = createNamedGraph(repo, invalidCollParams)
		require.Error(t, err, "Should fail with invalid stock ontology graph name")
		require.Contains(t, err.Error(), "error in creating named graph", "Error message should mention named graph creation")
	})
}

// verifyStockIDIndexCreated verifies that stock_id index exists
func verifyStockIDIndexCreated(t *testing.T, repo *arangorepository) {
	t.Helper()
	indices, err := repo.stockc.stock.Indexes(context.TODO())
	require.NoError(t, err, "Failed to get collection indices")
	require.NotEmpty(t, indices, "Collection should have indices")

	foundStockIDIndex := false
	for _, index := range indices {
		if index.Type() == driver.PersistentIndex {
			fields := index.Fields()
			if len(fields) > 0 && fields[0] == "stock_id" {
				foundStockIDIndex = true
				break
			}
		}
	}
	require.True(t, foundStockIDIndex, "stock_id index should be created")
}

// TestCreateIndex tests index creation with various scenarios
func TestCreateIndex(t *testing.T) {
	t.Run("successful index creation", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		repo, cleanup := setupRepoWithDocCollections(t, connParams, collParams, ontoParams)
		defer cleanup()

		err = createIndex(repo)
		require.NoError(t, err, "Failed to create index")

		verifyStockIDIndexCreated(t, repo)
	})

	t.Run("index creation is idempotent", func(t *testing.T) {
		testArango, err := testarango.NewTestArangoFromEnv(true)
		require.NoError(t, err, "Failed to create test arango instance")

		connParams := getConnectParamsFromDb(testArango)
		collParams := getCollectionParams()
		ontoParams := getOntoParams()

		repo, cleanup := setupRepoWithDocCollections(t, connParams, collParams, ontoParams)
		defer cleanup()

		err = createIndex(repo)
		require.NoError(t, err, "Failed to create index first time")

		err = createIndex(repo)
		require.NoError(t, err, "Failed to create index second time - should be idempotent")
	})
}
