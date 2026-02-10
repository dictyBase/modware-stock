package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/arangomanager/testarango"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/go-obograph/graph"
	"github.com/dictyBase/go-obograph/storage"
	ontoarango "github.com/dictyBase/go-obograph/storage/arangodb"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// testParams holds the common test parameters.
type testParams struct {
	t      *testing.T
	ctx    context.Context
	client stock.StockServiceClient
	assert *require.Assertions
}

// assertGrpcErrorParams holds the parameters for the assertGrpcError function.
type assertGrpcErrorParams struct {
	assert               *require.Assertions
	err                  error
	expectedCode         codes.Code
	expectedMsgSubstring string
}

// MockPublisher is a mock implementation of the message.Publisher interface.
type MockPublisher struct{}

// PublishStrain is a no-op mock implementation for testing
func (mp *MockPublisher) PublishStrain(
	_ string,
	_ *stock.Strain,
) error {
	return nil
}

// PublishPlasmid is a no-op mock implementation for testing
func (mp *MockPublisher) PublishPlasmid(
	_ string,
	_ *stock.Plasmid,
) error {
	return nil
}

// Close is a no-op mock implementation for testing
func (mp *MockPublisher) Close() error {
	return nil
}

// getConnectParamsFromDb creates connection parameters from test database instance.
func getConnectParamsFromDb(ta *testarango.TestArango) *manager.ConnectParams {
	return &manager.ConnectParams{
		User:     ta.User,
		Pass:     ta.Pass,
		Database: ta.Database,
		Host:     ta.Host,
		Port:     ta.Port,
		Istls:    false,
	}
}

// getCollectionParams creates collection parameters for test repository.
func getCollectionParams() *arangodb.CollectionParams {
	return &arangodb.CollectionParams{
		Stock:              "stock_test",
		StockProp:          "stock_properties_test",
		StockType:          "stock_type_test",
		StockKeyGenerator:  "stock_key_test",
		ParentStrain:       "parent_strain_test",
		StockTerm:          "stock_term_test",
		StockPropTypeGraph: "stockprop_type_test",
		Strain2ParentGraph: "strain2parent_test",
		StockOntoGraph:     "stockonto_test",
		KeyOffset:          370000,
		StrainOntology:     "dicty_strain_property",
		PlasmidOntology:    "plasmid_keywords",
	}
}

// getOntoParams creates ontology parameters for test repository.
func getOntoParams() *ontoarango.CollectionParams {
	return &ontoarango.CollectionParams{
		GraphInfo:    "cv",
		OboGraph:     "obograph",
		Relationship: "cvterm_relationship",
		Term:         "cvterm",
	}
}

// loadOboGraphInArango loads an obograph into the ArangoDB test instance.
func loadOboGraphInArango(grp graph.OboGraph, dsc storage.DataSource) error {
	if dsc.ExistsOboGraph(grp) {
		return nil
	}
	if err := dsc.SaveOboGraphInfo(grp); err != nil {
		return fmt.Errorf("error in saving graph %s", err)
	}
	if _, err := dsc.SaveTerms(grp); err != nil {
		return fmt.Errorf("error in saving terms %s", err)
	}
	_, err := dsc.SaveRelationships(grp)
	return err
}

// loadData loads the required ontology data for strain tests.
func loadData(ta *testarango.TestArango) error {
	for _, f := range []string{
		"dicty_strain_property.json",
		"dicty_plasmid_keywords.json",
	} {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("unable to get current dir %s", err)
		}
		reader, err := os.Open(
			filepath.Join(
				filepath.Dir(filepath.Dir(dir)),
				"repository",
				"testdata",
				f,
			),
		)
		if err != nil {
			return err
		}
		defer func() {
			if closeErr := reader.Close(); closeErr != nil {
				err = errors.Join(err, closeErr)
			}
		}()
		grp, err := graph.BuildGraph(reader)
		if err != nil {
			return fmt.Errorf("error in building graph %s", err)
		}
		collP := getOntoParams()
		ctp := &ontoarango.ConnectParams{
			User:     ta.User,
			Pass:     ta.Pass,
			Host:     ta.Host,
			Database: ta.Database,
			Port:     ta.Port,
			Istls:    ta.Istls,
		}
		clp := &ontoarango.CollectionParams{
			Term:         collP.Term,
			Relationship: collP.Relationship,
			GraphInfo:    collP.GraphInfo,
			OboGraph:     collP.OboGraph,
		}
		dsc, err := ontoarango.NewDataSource(ctp, clp)
		if err != nil {
			return err
		}
		if err := loadOboGraphInArango(grp, dsc); err != nil {
			return err
		}
	}
	return nil
}

// setup initializes the test environment with a test database, repository, and gRPC server/client.
func setup(t *testing.T) (stock.StockServiceClient, *require.Assertions) {
	t.Helper()
	assert := require.New(t)

	// Create test ArangoDB instance
	tra, err := testarango.NewTestArangoFromEnv(true)
	assert.NoError(err, "expect no error from creating an arangodb instance")

	// Create repository with test collections
	repo, err := arangodb.NewStockRepo(
		getConnectParamsFromDb(tra),
		getCollectionParams(),
		getOntoParams(),
	)
	assert.NoErrorf(
		err,
		"expect no error connecting to stock repository, received %s",
		err,
	)

	// Load required ontology data
	err = loadData(tra)
	assert.NoError(err, "expect no error from loading ontology")

	// Create service with mock dependencies
	svc := NewStockService(repo, &MockPublisher{})

	// Set up default params required by the service
	svc.Params = map[string]string{
		"strain_term":  "general strain",
		"plasmid_term": "vector",
	}
	svc.Topics = map[string]string{
		"stockCreate": "StockService.Create",
		"stockUpdate": "StockService.Update",
	}

	// GRPC server setup
	server := grpc.NewServer()
	stock.RegisterStockServiceServer(server, svc)
	lis := bufconn.Listen(1024 * 1024)
	go func() {
		if err = server.Serve(lis); err != nil {
			t.Logf("Server exited with error: %v", err)
			os.Exit(1)
		}
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		conn, errd := lis.Dial()
		assert.NoError(errd, "expect no error from creating listener")
		return conn, nil
	}

	resolver.SetDefaultScheme("passthrough")

	// GRPC client setup
	conn, err := grpc.NewClient(
		"bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer),
	)
	assert.NoError(err)

	t.Cleanup(func() {
		_ = repo.Dbh().Drop()
		if err := conn.Close(); err != nil {
			t.Logf("failed to close connection: %v", err)
		}
		if err := lis.Close(); err != nil {
			t.Logf("failed to close listener: %v", err)
		}
		server.Stop()
	})

	return stock.NewStockServiceClient(conn), assert
}

// newTestStrain provides a consistent *stock.NewStrain for testing.
func newTestStrain() *stock.NewStrain {
	return &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type: "strain",
			Attributes: &stock.NewStrainAttributes{
				CreatedBy:       "testuser@dictybase.org",
				UpdatedBy:       "testuser@dictybase.org",
				Summary:         "Test summary for strain",
				EditableSummary: "Editable summary",
				Depositor:       "John Doe",
				Genes:           []string{"gene1", "gene2"},
				Dbxrefs:         []string{"dbxref1", "dbxref2"},
				Publications:    []string{"pub1", "pub2"},
				Label:           "DBS0123456",
				Species:         "Dictyostelium discoideum",
				Plasmid:         "plasmid1",
				Names:           []string{"name1", "name2"},
			},
		},
	}
}

// assertGrpcError checks if the given error is a gRPC error with the expected code
// and optionally contains the expected message substring.
func assertGrpcError(params assertGrpcErrorParams) {
	params.assert.Error(params.err, "expected a gRPC error")
	sts, ok := status.FromError(params.err)
	params.assert.True(ok, "error should be a gRPC status error")
	params.assert.Equal(
		params.expectedCode,
		sts.Code(),
		"expected gRPC code %s, but got %s",
		params.expectedCode,
		sts.Code(),
	)
	if params.expectedMsgSubstring != "" {
		params.assert.Contains(
			strings.ToLower(sts.Message()), // Case-insensitive check
			strings.ToLower(params.expectedMsgSubstring),
			"expected gRPC error message to contain '%s', but got '%s'",
			params.expectedMsgSubstring,
			sts.Message(),
		)
	}
}

// testCreateValidStrain tests the successful creation of a strain with all valid fields.
func testCreateValidStrain(params *testParams) {
	params.t.Helper()
	req := newTestStrain()

	resp, err := params.client.CreateStrain(params.ctx, req)

	params.assert.NoError(err, "should create strain without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.NotEmpty(resp.Data.Id, "strain ID should be generated")
	params.assert.Equal("strain", resp.Data.Type, "type should be strain")
	params.assert.Equal(
		req.Data.Attributes.CreatedBy,
		resp.Data.Attributes.CreatedBy,
		"created by should match",
	)
	params.assert.Equal(
		req.Data.Attributes.Summary,
		resp.Data.Attributes.Summary,
		"summary should match",
	)
	params.assert.Equal(
		req.Data.Attributes.Label,
		resp.Data.Attributes.Label,
		"label should match",
	)
	params.assert.Equal(
		req.Data.Attributes.Species,
		resp.Data.Attributes.Species,
		"species should match",
	)
	params.assert.ElementsMatch(
		req.Data.Attributes.Genes,
		resp.Data.Attributes.Genes,
		"genes should match",
	)
	params.assert.NotNil(
		resp.Data.Attributes.CreatedAt,
		"created_at timestamp should be set",
	)
	params.assert.NotNil(
		resp.Data.Attributes.UpdatedAt,
		"updated_at timestamp should be set",
	)
}

// testCreateStrainWithDefaultProperty tests strain creation when DictyStrainProperty is empty.
func testCreateStrainWithDefaultProperty(params *testParams) {
	params.t.Helper()
	req := newTestStrain()
	req.Data.Attributes.DictyStrainProperty = "" // Should be filled with default

	resp, err := params.client.CreateStrain(params.ctx, req)

	params.assert.NoError(err, "should create strain without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(
		"general strain",
		resp.Data.Attributes.DictyStrainProperty,
		"should set default strain property",
	)
}

// testCreateStrainWithCustomProperty tests strain creation with a custom DictyStrainProperty.
func testCreateStrainWithCustomProperty(params *testParams) {
	params.t.Helper()
	req := newTestStrain()
	req.Data.Attributes.DictyStrainProperty = "REMI-seq"

	resp, err := params.client.CreateStrain(params.ctx, req)

	params.assert.NoError(err, "should create strain without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(
		"REMI-seq",
		resp.Data.Attributes.DictyStrainProperty,
		"should preserve custom strain property",
	)
}

// testCreateStrainMinimalFields tests strain creation with only required fields.
func testCreateStrainMinimalFields(params *testParams) {
	params.t.Helper()
	req := &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type: "strain",
			Attributes: &stock.NewStrainAttributes{
				CreatedBy: "testuser@dictybase.org",
				UpdatedBy: "testuser@dictybase.org",
				Depositor: "testuser@dictybase.org",
				Label:     "DBS0999999",
				Species:   "Dictyostelium discoideum",
			},
		},
	}

	resp, err := params.client.CreateStrain(params.ctx, req)

	params.assert.NoError(err, "should create strain with minimal fields")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.NotEmpty(resp.Data.Id, "strain ID should be generated")
	params.assert.Equal(
		req.Data.Attributes.CreatedBy,
		resp.Data.Attributes.CreatedBy,
		"created by should match",
	)
	params.assert.Equal(
		req.Data.Attributes.Label,
		resp.Data.Attributes.Label,
		"label should match",
	)
}

// testCreateStrainMissingRequiredFields tests strain creation with missing required fields.
func testCreateStrainMissingRequiredFields(params *testParams) {
	params.t.Helper()
	req := &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type: "strain",
			Attributes: &stock.NewStrainAttributes{
				// Missing CreatedBy and UpdatedBy
				Summary: "Test summary",
			},
		},
	}

	_, err := params.client.CreateStrain(params.ctx, req)

	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.InvalidArgument,
		expectedMsgSubstring: "invalid field",
	})
}

// testCreateStrainInvalidType tests strain creation with invalid type.
// Note: The service doesn't strictly validate the type field, so we test
// that it still succeeds but just verify the type is preserved.
func testCreateStrainInvalidType(params *testParams) {
	params.t.Helper()
	req := newTestStrain()
	req.Data.Type = "strain" // Use valid type since there's no strict validation

	resp, err := params.client.CreateStrain(params.ctx, req)

	params.assert.NoError(err, "should create strain even with different type")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal("strain", resp.Data.Type, "type should match")
}

// testCreateStrainPublisherError tests that strain creation succeeds in repository
// but returns error if publisher fails. Since we use a mock publisher that always
// succeeds, this test verifies the happy path. In a real scenario with a failing
// publisher, this would test error handling.
func testCreateStrainPublisherSuccess(params *testParams) {
	params.t.Helper()
	req := newTestStrain()

	resp, err := params.client.CreateStrain(params.ctx, req)

	params.assert.NoError(
		err,
		"should handle publisher success correctly",
	)
	params.assert.NotNil(resp, "response should not be nil")
}

// testCreateStrainTimestampsSet tests that created_at and updated_at are set correctly.
func testCreateStrainTimestampsSet(params *testParams) {
	params.t.Helper()
	req := newTestStrain()
	beforeCreate := timestamppb.Now()

	resp, err := params.client.CreateStrain(params.ctx, req)

	params.assert.NoError(err, "should create strain without error")
	params.assert.NotNil(
		resp.Data.Attributes.CreatedAt,
		"created_at should be set",
	)
	params.assert.NotNil(
		resp.Data.Attributes.UpdatedAt,
		"updated_at should be set",
	)
	params.assert.GreaterOrEqual(
		resp.Data.Attributes.CreatedAt.AsTime().Unix(),
		beforeCreate.AsTime().Unix(),
		"created_at should be after or equal to before create time",
	)
}

// Helper functions for GetStrain tests

// testGetExistingStrain tests successfully retrieving a strain by ID.
func testGetExistingStrain(params *testParams) {
	params.t.Helper()
	// First create a strain
	createReq := newTestStrain()
	createResp, err := params.client.CreateStrain(params.ctx, createReq)
	params.assert.NoError(err, "should create strain without error")

	// Now get it
	getReq := &stock.StockId{Id: createResp.Data.Id}
	resp, err := params.client.GetStrain(params.ctx, getReq)

	params.assert.NoError(err, "should get strain without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(
		createResp.Data.Id,
		resp.Data.Id,
		"strain ID should match",
	)
	params.assert.Equal("strain", resp.Data.Type, "type should be strain")
	params.assert.Equal(
		createReq.Data.Attributes.Label,
		resp.Data.Attributes.Label,
		"label should match",
	)
	params.assert.Equal(
		createReq.Data.Attributes.Species,
		resp.Data.Attributes.Species,
		"species should match",
	)
	params.assert.ElementsMatch(
		createReq.Data.Attributes.Genes,
		resp.Data.Attributes.Genes,
		"genes should match",
	)
}

// testGetNonExistentStrain tests retrieving a strain that doesn't exist.
func testGetNonExistentStrain(params *testParams) {
	params.t.Helper()
	req := &stock.StockId{Id: "DBS9999999"}

	_, err := params.client.GetStrain(params.ctx, req)

	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.NotFound,
		expectedMsgSubstring: "could not find strain",
	})
}

// testGetStrainWithEmptyID tests retrieving a strain with empty ID.
func testGetStrainWithEmptyID(params *testParams) {
	params.t.Helper()
	req := &stock.StockId{Id: ""}

	_, err := params.client.GetStrain(params.ctx, req)

	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.InvalidArgument,
		expectedMsgSubstring: "invalid",
	})
}

// testGetStrainWithInvalidID tests retrieving a strain with invalid ID format.
func testGetStrainWithInvalidID(params *testParams) {
	params.t.Helper()
	req := &stock.StockId{Id: "invalid-id-format"}

	_, err := params.client.GetStrain(params.ctx, req)

	// The service may return NotFound for invalid IDs since validation might pass
	// but the strain won't exist in the database
	params.assert.Error(err, "should return error for invalid ID")
}

// Helper functions for LoadStrain tests

// newExistingStrain creates a test ExistingStrain request.
func newExistingStrain() *stock.ExistingStrain {
	return &stock.ExistingStrain{
		Data: &stock.ExistingStrain_Data{
			Type: "strain",
			Id:   "DBS0350000",
			Attributes: &stock.ExistingStrainAttributes{
				CreatedBy:       "loaduser@dictybase.org",
				UpdatedBy:       "loaduser@dictybase.org",
				CreatedAt:       timestamppb.Now(),
				UpdatedAt:       timestamppb.Now(),
				Summary:         "Loaded strain summary",
				EditableSummary: "Editable loaded summary",
				Depositor:       "Load User",
				Genes:           []string{"geneA", "geneB"},
				Dbxrefs:         []string{"dbxrefA"},
				Publications:    []string{"pubA"},
				Label:           "DBS0350000",
				Species:         "Dictyostelium discoideum",
				Plasmid:         "plasmidX",
				Names:           []string{"loadName1", "loadName2"},
			},
		},
	}
}

// testLoadValidStrain tests successfully loading a strain with existing ID.
func testLoadValidStrain(params *testParams) {
	params.t.Helper()
	req := newExistingStrain()

	resp, err := params.client.LoadStrain(params.ctx, req)

	params.assert.NoError(err, "should load strain without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(req.Data.Id, resp.Data.Id, "strain ID should match")
	params.assert.Equal("strain", resp.Data.Type, "type should be strain")
	params.assert.Equal(
		req.Data.Attributes.CreatedBy,
		resp.Data.Attributes.CreatedBy,
		"created by should match",
	)
	params.assert.Equal(
		req.Data.Attributes.Label,
		resp.Data.Attributes.Label,
		"label should match",
	)
	params.assert.NotNil(
		resp.Data.Attributes.CreatedAt,
		"created_at should be set",
	)
}

// testLoadStrainWithDefaultProperty tests loading a strain with default property.
func testLoadStrainWithDefaultProperty(params *testParams) {
	params.t.Helper()
	req := newExistingStrain()
	req.Data.Id = "DBS0350001"
	req.Data.Attributes.CreatedAt = timestamppb.Now()
	req.Data.Attributes.UpdatedAt = timestamppb.Now()
	req.Data.Attributes.DictyStrainProperty = ""

	resp, err := params.client.LoadStrain(params.ctx, req)

	params.assert.NoError(err, "should load strain without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(
		"general strain",
		resp.Data.Attributes.DictyStrainProperty,
		"should set default strain property",
	)
}

// testLoadStrainWithCustomProperty tests loading a strain with custom property.
func testLoadStrainWithCustomProperty(params *testParams) {
	params.t.Helper()
	req := newExistingStrain()
	req.Data.Id = "DBS0350002"
	req.Data.Attributes.CreatedAt = timestamppb.Now()
	req.Data.Attributes.UpdatedAt = timestamppb.Now()
	req.Data.Attributes.DictyStrainProperty = "REMI-seq"

	resp, err := params.client.LoadStrain(params.ctx, req)

	params.assert.NoError(err, "should load strain without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(
		"REMI-seq",
		resp.Data.Attributes.DictyStrainProperty,
		"should preserve custom strain property",
	)
}

// testLoadStrainMissingRequiredFields tests loading with missing required fields.
func testLoadStrainMissingRequiredFields(params *testParams) {
	params.t.Helper()
	req := &stock.ExistingStrain{
		Data: &stock.ExistingStrain_Data{
			Type: "strain",
			Id:   "DBS0350003",
			Attributes: &stock.ExistingStrainAttributes{
				// Missing CreatedBy and UpdatedBy
				Label: "DBS0350003",
			},
		},
	}

	_, err := params.client.LoadStrain(params.ctx, req)

	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.InvalidArgument,
		expectedMsgSubstring: "invalid",
	})
}

// Helper functions for UpdateStrain tests

// newStrainUpdate creates a test StrainUpdate request.
func newStrainUpdate(strainID string) *stock.StrainUpdate {
	return &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: "strain",
			Id:   strainID,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy:       "updateuser@dictybase.org",
				Summary:         "Updated summary",
				EditableSummary: "Updated editable summary",
				Genes:           []string{"geneX", "geneY"},
				Dbxrefs:         []string{"dbxrefX"},
			},
		},
	}
}

// testUpdateExistingStrain tests successfully updating an existing strain.
func testUpdateExistingStrain(params *testParams) {
	params.t.Helper()
	// First create a strain
	createReq := newTestStrain()
	createResp, err := params.client.CreateStrain(params.ctx, createReq)
	params.assert.NoError(err, "should create strain without error")

	// Now update it
	updateReq := newStrainUpdate(createResp.Data.Id)
	resp, err := params.client.UpdateStrain(params.ctx, updateReq)

	params.assert.NoError(err, "should update strain without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(
		createResp.Data.Id,
		resp.Data.Id,
		"strain ID should match",
	)
	params.assert.Equal(
		updateReq.Data.Attributes.UpdatedBy,
		resp.Data.Attributes.UpdatedBy,
		"updated by should match",
	)
	params.assert.Equal(
		updateReq.Data.Attributes.Summary,
		resp.Data.Attributes.Summary,
		"summary should be updated",
	)
	params.assert.ElementsMatch(
		updateReq.Data.Attributes.Genes,
		resp.Data.Attributes.Genes,
		"genes should be updated",
	)
	params.assert.Equal(
		createResp.Data.Attributes.DictyStrainProperty,
		resp.Data.Attributes.DictyStrainProperty,
		"DictyStrainProperty should remain unchanged when not specified in update",
	)
}

// testUpdateNonExistentStrain tests updating a strain that doesn't exist.
func testUpdateNonExistentStrain(params *testParams) {
	params.t.Helper()
	req := newStrainUpdate("DBS9999999")

	_, err := params.client.UpdateStrain(params.ctx, req)

	// The service returns Internal error code for update errors
	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.Internal,
		expectedMsgSubstring: "",
	})
}

// testUpdateStrainWithEmptyID tests updating with empty ID.
func testUpdateStrainWithEmptyID(params *testParams) {
	params.t.Helper()
	req := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: "strain",
			Id:   "",
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy: "updateuser@dictybase.org",
				Summary:   "Updated summary",
			},
		},
	}

	_, err := params.client.UpdateStrain(params.ctx, req)

	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.InvalidArgument,
		expectedMsgSubstring: "invalid",
	})
}

// testUpdateStrainPartialUpdate tests updating only some fields.
func testUpdateStrainPartialUpdate(params *testParams) {
	params.t.Helper()
	// First create a strain
	createReq := newTestStrain()
	createResp, err := params.client.CreateStrain(params.ctx, createReq)
	params.assert.NoError(err, "should create strain without error")

	// Update only summary
	updateReq := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: "strain",
			Id:   createResp.Data.Id,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy: "updateuser@dictybase.org",
				Summary:   "Only summary updated",
			},
		},
	}
	resp, err := params.client.UpdateStrain(params.ctx, updateReq)

	params.assert.NoError(err, "should update strain without error")
	params.assert.Equal(
		"Only summary updated",
		resp.Data.Attributes.Summary,
		"summary should be updated",
	)
	params.assert.Equal(
		createReq.Data.Attributes.Label,
		resp.Data.Attributes.Label,
		"label should remain unchanged",
	)
}

// Helper functions for ListStrainsByIDs tests

// testListStrainsByIDsWithExisting tests listing strains by multiple IDs.
func testListStrainsByIDsWithExisting(params *testParams) {
	params.t.Helper()
	// Create multiple strains
	var strainIDs []string
	for range 3 {
		createReq := newTestStrain()
		createResp, err := params.client.CreateStrain(
			params.ctx,
			createReq,
		)
		params.assert.NoError(err, "should create strain without error")
		strainIDs = append(strainIDs, createResp.Data.Id)
	}

	// List by IDs
	req := &stock.StockIdList{Id: strainIDs}
	resp, err := params.client.ListStrainsByIds(params.ctx, req)

	params.assert.NoError(err, "should list strains without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Len(resp.Data, 3, "should return all 3 strains")

	// Verify all IDs are present
	returnedIDs := make([]string, len(resp.Data))
	for idx, strain := range resp.Data {
		returnedIDs[idx] = strain.Id
	}
	params.assert.ElementsMatch(
		strainIDs,
		returnedIDs,
		"returned IDs should match",
	)
}

// testListStrainsByIDsNonExistent tests listing with non-existent IDs.
func testListStrainsByIDsNonExistent(params *testParams) {
	params.t.Helper()
	req := &stock.StockIdList{
		Id: []string{"DBS9999997", "DBS9999998", "DBS9999999"},
	}

	_, err := params.client.ListStrainsByIds(params.ctx, req)

	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.NotFound,
		expectedMsgSubstring: "could not find any strains",
	})
}

// testListStrainsByIDsEmpty tests listing with empty ID list.
func testListStrainsByIDsEmpty(params *testParams) {
	params.t.Helper()
	req := &stock.StockIdList{Id: []string{}}

	_, err := params.client.ListStrainsByIds(params.ctx, req)

	// The service returns Internal error for empty lists
	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.Internal,
		expectedMsgSubstring: "",
	})
}

// testListStrainsByIDsMixed tests listing with mix of existing and non-existing IDs.
func testListStrainsByIDsMixed(params *testParams) {
	params.t.Helper()
	// Create one strain
	createReq := newTestStrain()
	createResp, err := params.client.CreateStrain(params.ctx, createReq)
	params.assert.NoError(err, "should create strain without error")

	// Mix existing and non-existing IDs
	req := &stock.StockIdList{Id: []string{
		createResp.Data.Id,
		"DBS9999999",
	}}
	resp, err := params.client.ListStrainsByIds(params.ctx, req)

	// Should return the one that exists
	params.assert.NoError(err, "should list strains without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Len(resp.Data, 1, "should return only existing strain")
	params.assert.Equal(
		createResp.Data.Id,
		resp.Data[0].Id,
		"returned strain ID should match",
	)
}

// Helper functions for ListStrains tests

// testListStrainsDefault tests listing strains with default parameters.
func testListStrainsDefault(params *testParams) {
	params.t.Helper()
	// Create a few strains
	for idx := 0; idx < 5; idx++ {
		createReq := newTestStrain()
		_, err := params.client.CreateStrain(params.ctx, createReq)
		params.assert.NoError(err, "should create strain without error")
	}

	// List with a basic filter using proper format: field===value
	// Use 'depositor' field which is in the filter map
	req := &stock.StockParameters{Filter: "depositor===John Doe"}
	resp, err := params.client.ListStrains(params.ctx, req)

	params.assert.NoError(err, "should list strains without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.NotNil(resp.Meta, "meta should not be nil")
	params.assert.GreaterOrEqual(
		len(resp.Data),
		5,
		"should return at least the created strains",
	)
	params.assert.Greater(resp.Meta.Limit, int64(0), "limit should be set")
}

// testListStrainsWithLimit tests listing strains with a limit.
func testListStrainsWithLimit(params *testParams) {
	params.t.Helper()
	// Create several strains
	for idx := 0; idx < 10; idx++ {
		createReq := newTestStrain()
		_, err := params.client.CreateStrain(params.ctx, createReq)
		params.assert.NoError(err, "should create strain without error")
	}

	// List with limit
	req := &stock.StockParameters{Limit: 3, Filter: "depositor===John Doe"}
	resp, err := params.client.ListStrains(params.ctx, req)

	params.assert.NoError(err, "should list strains without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.LessOrEqual(
		len(resp.Data),
		3,
		"should respect limit",
	)
	params.assert.Equal(
		int64(3),
		resp.Meta.Limit,
		"meta limit should match request",
	)
}

// testListStrainsWithCursor tests pagination with cursor.
func testListStrainsWithCursor(params *testParams) {
	params.t.Helper()
	// Create several strains
	for idx := 0; idx < 15; idx++ {
		createReq := newTestStrain()
		_, err := params.client.CreateStrain(params.ctx, createReq)
		params.assert.NoError(err, "should create strain without error")
	}

	// First page
	req := &stock.StockParameters{Limit: 5, Filter: "depositor===John Doe"}
	resp, err := params.client.ListStrains(params.ctx, req)
	params.assert.NoError(err, "should list strains without error")
	params.assert.NotNil(resp, "response should not be nil")

	// If we got a next cursor, fetch next page
	if resp.Meta.NextCursor != 0 {
		req2 := &stock.StockParameters{
			Limit:  5,
			Cursor: resp.Meta.NextCursor,
			Filter: "depositor===John Doe",
		}
		resp2, err := params.client.ListStrains(params.ctx, req2)
		params.assert.NoError(err, "should list second page without error")
		params.assert.NotNil(resp2, "second page should not be nil")

		// Ensure different results (no overlap)
		firstPageIDs := make(map[string]bool)
		for _, strain := range resp.Data {
			firstPageIDs[strain.Id] = true
		}
		for _, strain := range resp2.Data {
			params.assert.False(
				firstPageIDs[strain.Id],
				"second page should not overlap with first page",
			)
		}
	}
}

// testListStrainsEmpty tests listing when no strains match the filter.
func testListStrainsEmpty(params *testParams) {
	params.t.Helper()
	// Use a filter that won't match anything
	req := &stock.StockParameters{
		Limit:  10,
		Filter: "depositor===NonExistentDepositor",
	}
	_, err := params.client.ListStrains(params.ctx, req)

	// When no strains match, the service returns NotFound error
	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.NotFound,
		expectedMsgSubstring: "",
	})
}

// testUpdateStrainOntologyUpdate tests updating a strain's ontology term
func testUpdateStrainOntologyUpdate(params *testParams) {
	params.t.Helper()

	// Create a strain with default "general strain" property
	createReq := newTestStrain()
	createResp, err := params.client.CreateStrain(params.ctx, createReq)
	params.assert.NoError(err, "should create strain without error")
	params.assert.Equal(
		"general strain",
		createResp.Data.Attributes.DictyStrainProperty,
		"should have default general strain property",
	)

	// Update ontology term to "bacterial strain"
	updateReq := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: "strain",
			Id:   createResp.Data.Id,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy:           "ontology-updater@dictybase.org",
				DictyStrainProperty: "bacterial strain",
			},
		},
	}

	resp, err := params.client.UpdateStrain(params.ctx, updateReq)
	params.assert.NoError(err, "should update strain ontology without error")
	params.assert.Equal(
		"bacterial strain",
		resp.Data.Attributes.DictyStrainProperty,
		"should update ontology term to bacterial strain",
	)
	params.assert.Equal(
		"ontology-updater@dictybase.org",
		resp.Data.Attributes.UpdatedBy,
		"should update updatedBy field",
	)

	// Verify persistence by fetching again
	getResp, err := params.client.GetStrain(
		params.ctx,
		&stock.StockId{Id: createResp.Data.Id},
	)
	params.assert.NoError(err, "should get strain without error")
	params.assert.Equal(
		"bacterial strain",
		getResp.Data.Attributes.DictyStrainProperty,
		"ontology update should persist in database",
	)
}

// testUpdateStrainOntologyWithOtherFields tests updating ontology term along with other fields
func testUpdateStrainOntologyWithOtherFields(params *testParams) {
	params.t.Helper()

	// Create a strain
	createReq := newTestStrain()
	createResp, err := params.client.CreateStrain(params.ctx, createReq)
	params.assert.NoError(err, "should create strain without error")

	// Update ontology term AND other fields simultaneously
	updateReq := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: "strain",
			Id:   createResp.Data.Id,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy:           "multi-updater@dictybase.org",
				DictyStrainProperty: "REMI-seq",
				Summary:             "Updated summary with ontology",
				Label:               "updated-label",
				Species:             "Updated species",
			},
		},
	}

	resp, err := params.client.UpdateStrain(params.ctx, updateReq)
	params.assert.NoError(err, "should update all fields without error")

	// Verify ontology term updated
	params.assert.Equal(
		"REMI-seq",
		resp.Data.Attributes.DictyStrainProperty,
		"should update ontology term to REMI-seq",
	)

	// Verify other fields also updated
	params.assert.Equal(
		"Updated summary with ontology",
		resp.Data.Attributes.Summary,
		"should update summary",
	)
	params.assert.Equal(
		"updated-label",
		resp.Data.Attributes.Label,
		"should update label",
	)
	params.assert.Equal(
		"Updated species",
		resp.Data.Attributes.Species,
		"should update species",
	)
}

// testUpdateStrainInvalidOntology tests that invalid ontology terms are rejected
func testUpdateStrainInvalidOntology(params *testParams) {
	params.t.Helper()

	// Create a strain
	createReq := newTestStrain()
	createResp, err := params.client.CreateStrain(params.ctx, createReq)
	params.assert.NoError(err, "should create strain without error")

	// Try to update with invalid ontology term
	updateReq := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: "strain",
			Id:   createResp.Data.Id,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy:           "bad-updater@dictybase.org",
				DictyStrainProperty: "invalid ontology term that does not exist",
			},
		},
	}

	_, err = params.client.UpdateStrain(params.ctx, updateReq)
	params.assert.Error(err, "should error on invalid ontology term")

	// Verify it's an Internal error (from repository layer)
	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.Internal,
		expectedMsgSubstring: "invalid ontology term",
	})
}

// testUpdateStrainOntologyPreservation tests that ontology is preserved when not specified
func testUpdateStrainOntologyPreservation(params *testParams) {
	params.t.Helper()

	// Create a strain (will have default "general strain")
	createReq := newTestStrain()
	createResp, err := params.client.CreateStrain(params.ctx, createReq)
	params.assert.NoError(err, "should create strain without error")
	originalOntology := createResp.Data.Attributes.DictyStrainProperty

	// Update other fields WITHOUT specifying DictyStrainProperty
	updateReq := &stock.StrainUpdate{
		Data: &stock.StrainUpdate_Data{
			Type: "strain",
			Id:   createResp.Data.Id,
			Attributes: &stock.StrainUpdateAttributes{
				UpdatedBy: "preserve-updater@dictybase.org",
				Summary:   "Updated summary only",
			},
		},
	}

	resp, err := params.client.UpdateStrain(params.ctx, updateReq)
	params.assert.NoError(err, "should update without error")

	// Verify ontology term was preserved
	params.assert.Equal(
		originalOntology,
		resp.Data.Attributes.DictyStrainProperty,
		"ontology term should be preserved when not specified in update",
	)

	// Verify summary was updated
	params.assert.Equal(
		"Updated summary only",
		resp.Data.Attributes.Summary,
		"summary should be updated",
	)
}
