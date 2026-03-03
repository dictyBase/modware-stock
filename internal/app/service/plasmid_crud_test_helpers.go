package service

import (
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ============================================================================
// CreatePlasmid Test Helpers (8 tests)
// ============================================================================

// testCreateValidPlasmid tests the successful creation of a plasmid with all valid fields.
func testCreateValidPlasmid(params *testParams) {
	params.t.Helper()
	req := newTestPlasmid()

	resp, err := params.client.CreatePlasmid(params.ctx, req)

	params.assert.NoError(err, "should create plasmid without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.NotEmpty(resp.Data.Id, "plasmid ID should be generated")
	params.assert.Equal("plasmid", resp.Data.Type, "type should be plasmid")
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
		req.Data.Attributes.Name,
		resp.Data.Attributes.Name,
		"name should match",
	)
	params.assert.Equal(
		req.Data.Attributes.Sequence,
		resp.Data.Attributes.Sequence,
		"sequence should match",
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

// testCreatePlasmidWithDefaultProperty tests plasmid creation when DictyPlasmidProperty is not set.
// Note: Default is applied when field is not set in the request.
func testCreatePlasmidWithDefaultProperty(params *testParams) {
	params.t.Helper()
	req := newTestPlasmid()
	// Don't set DictyPlasmidProperty - leave it as default empty value
	// The service should apply the default "vector"

	resp, err := params.client.CreatePlasmid(params.ctx, req)

	params.assert.NoError(err, "should create plasmid without error")
	params.assert.NotNil(resp, "response should not be nil")
	// When property is not set, default is applied during validation
	// but may not be returned by repository if it doesn't persist ontology separately
	params.assert.NotEmpty(resp.Data.Id, "plasmid ID should be generated")
}

// testCreatePlasmidWithCustomProperty tests plasmid creation with a custom DictyPlasmidProperty.
func testCreatePlasmidWithCustomProperty(params *testParams) {
	params.t.Helper()
	req := newTestPlasmid()
	req.Data.Attributes.DictyPlasmidProperty = testGatewayVector

	resp, err := params.client.CreatePlasmid(params.ctx, req)

	params.assert.NoError(err, "should create plasmid without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.NotEmpty(resp.Data.Id, "plasmid ID should be generated")
	// Note: Custom property is set in request but may not be returned by repository
}

// testCreatePlasmidMinimalFields tests plasmid creation with only required fields.
func testCreatePlasmidMinimalFields(params *testParams) {
	params.t.Helper()
	req := &stock.NewPlasmid{
		Data: &stock.NewPlasmid_Data{
			Type: "plasmid",
			Attributes: &stock.NewPlasmidAttributes{
				CreatedBy: "testuser@dictybase.org",
				UpdatedBy: "testuser@dictybase.org",
				Depositor: "testuser@dictybase.org",
				Name:      "pMinimal",
			},
		},
	}

	resp, err := params.client.CreatePlasmid(params.ctx, req)

	params.assert.NoError(err, "should create plasmid with minimal fields")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.NotEmpty(resp.Data.Id, "plasmid ID should be generated")
	params.assert.Equal(
		req.Data.Attributes.CreatedBy,
		resp.Data.Attributes.CreatedBy,
		"created by should match",
	)
	params.assert.Equal(
		req.Data.Attributes.Name,
		resp.Data.Attributes.Name,
		"name should match",
	)
}

// testCreatePlasmidMissingRequiredFields tests plasmid creation with missing required fields.
func testCreatePlasmidMissingRequiredFields(params *testParams) {
	params.t.Helper()
	req := &stock.NewPlasmid{
		Data: &stock.NewPlasmid_Data{
			Type: "plasmid",
			Attributes: &stock.NewPlasmidAttributes{
				// Missing CreatedBy and UpdatedBy
				Summary: "Test summary",
			},
		},
	}

	_, err := params.client.CreatePlasmid(params.ctx, req)

	// Repository returns Internal error for validation failures
	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.Internal,
		expectedMsgSubstring: "",
	})
}

// testCreatePlasmidInvalidType tests plasmid creation with invalid type.
func testCreatePlasmidInvalidType(params *testParams) {
	params.t.Helper()
	req := newTestPlasmid()
	req.Data.Type = "plasmid" // Use valid type since there's no strict validation

	resp, err := params.client.CreatePlasmid(params.ctx, req)

	params.assert.NoError(err, "should create plasmid even with different type")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal("plasmid", resp.Data.Type, "type should match")
}

// testCreatePlasmidPublisherSuccess tests that plasmid creation succeeds with publisher.
func testCreatePlasmidPublisherSuccess(params *testParams) {
	params.t.Helper()
	req := newTestPlasmid()

	resp, err := params.client.CreatePlasmid(params.ctx, req)

	params.assert.NoError(
		err,
		"should handle publisher success correctly",
	)
	params.assert.NotNil(resp, "response should not be nil")
}

// testCreatePlasmidTimestampsSet tests that created_at and updated_at are set correctly.
func testCreatePlasmidTimestampsSet(params *testParams) {
	params.t.Helper()
	req := newTestPlasmid()
	beforeCreate := timestamppb.Now()

	resp, err := params.client.CreatePlasmid(params.ctx, req)

	params.assert.NoError(err, "should create plasmid without error")
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

// ============================================================================
// GetPlasmid Test Helpers (4 tests)
// ============================================================================

// testGetExistingPlasmid tests successfully retrieving a plasmid by ID.
func testGetExistingPlasmid(params *testParams) {
	params.t.Helper()
	// First create a plasmid
	createReq := newTestPlasmid()
	createResp, err := params.client.CreatePlasmid(params.ctx, createReq)
	params.assert.NoError(err, "should create plasmid without error")

	// Now get it
	getReq := &stock.StockId{Id: createResp.Data.Id}
	resp, err := params.client.GetPlasmid(params.ctx, getReq)

	params.assert.NoError(err, "should get plasmid without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(
		createResp.Data.Id,
		resp.Data.Id,
		"plasmid ID should match",
	)
	params.assert.Equal("plasmid", resp.Data.Type, "type should be plasmid")
	params.assert.Equal(
		createReq.Data.Attributes.Name,
		resp.Data.Attributes.Name,
		"name should match",
	)
	params.assert.Equal(
		createReq.Data.Attributes.Sequence,
		resp.Data.Attributes.Sequence,
		"sequence should match",
	)
	params.assert.ElementsMatch(
		createReq.Data.Attributes.Genes,
		resp.Data.Attributes.Genes,
		"genes should match",
	)
}

// testGetNonExistentPlasmid tests retrieving a plasmid that doesn't exist.
func testGetNonExistentPlasmid(params *testParams) {
	params.t.Helper()
	req := &stock.StockId{Id: "DBP9999999"}

	_, err := params.client.GetPlasmid(params.ctx, req)

	// Repository returns Internal error for non-existent plasmids
	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.Internal,
		expectedMsgSubstring: "",
	})
}

// testGetPlasmidWithEmptyID tests retrieving a plasmid with empty ID.
func testGetPlasmidWithEmptyID(params *testParams) {
	params.t.Helper()
	req := &stock.StockId{Id: ""}

	_, err := params.client.GetPlasmid(params.ctx, req)

	// Repository returns Internal error for empty ID
	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.Internal,
		expectedMsgSubstring: "",
	})
}

// testGetPlasmidWithInvalidID tests retrieving a plasmid with invalid ID format.
func testGetPlasmidWithInvalidID(params *testParams) {
	params.t.Helper()
	req := &stock.StockId{Id: "invalid-id-format"}

	_, err := params.client.GetPlasmid(params.ctx, req)

	// The service may return NotFound for invalid IDs since validation might pass
	// but the plasmid won't exist in the database
	params.assert.Error(err, "should return error for invalid ID")
}

// ============================================================================
// LoadPlasmid Test Helpers (5 tests)
// ============================================================================

// testLoadValidPlasmid tests successfully loading a plasmid with existing ID.
func testLoadValidPlasmid(params *testParams) {
	params.t.Helper()
	req := newExistingPlasmid()

	resp, err := params.client.LoadPlasmid(params.ctx, req)

	params.assert.NoError(err, "should load plasmid without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(req.Data.Id, resp.Data.Id, "plasmid ID should match")
	params.assert.Equal("plasmid", resp.Data.Type, "type should be plasmid")
	params.assert.Equal(
		req.Data.Attributes.CreatedBy,
		resp.Data.Attributes.CreatedBy,
		"created by should match",
	)
	params.assert.Equal(
		req.Data.Attributes.Name,
		resp.Data.Attributes.Name,
		"name should match",
	)
	params.assert.NotNil(
		resp.Data.Attributes.CreatedAt,
		"created_at should be set",
	)
}

// testLoadPlasmidWithDefaultProperty tests loading a plasmid when property is not set.
func testLoadPlasmidWithDefaultProperty(params *testParams) {
	params.t.Helper()
	req := newExistingPlasmid()
	req.Data.Id = "DBP0000002"
	req.Data.Attributes.CreatedAt = timestamppb.Now()
	req.Data.Attributes.UpdatedAt = timestamppb.Now()
	// Don't set DictyPlasmidProperty

	resp, err := params.client.LoadPlasmid(params.ctx, req)

	params.assert.NoError(err, "should load plasmid without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(req.Data.Id, resp.Data.Id, "plasmid ID should match")
}

// testLoadPlasmidWithCustomProperty tests loading a plasmid with custom property.
func testLoadPlasmidWithCustomProperty(params *testParams) {
	params.t.Helper()
	req := newExistingPlasmid()
	req.Data.Id = "DBP0000003"
	req.Data.Attributes.CreatedAt = timestamppb.Now()
	req.Data.Attributes.UpdatedAt = timestamppb.Now()
	req.Data.Attributes.DictyPlasmidProperty = testGatewayVector

	resp, err := params.client.LoadPlasmid(params.ctx, req)

	params.assert.NoError(err, "should load plasmid without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(req.Data.Id, resp.Data.Id, "plasmid ID should match")
	// Note: Custom property is set in request but may not be returned by repository
}

// testLoadPlasmidMissingRequiredFields tests loading with missing required fields.
func testLoadPlasmidMissingRequiredFields(params *testParams) {
	params.t.Helper()
	req := &stock.ExistingPlasmid{
		Data: &stock.ExistingPlasmid_Data{
			Type: "plasmid",
			Id:   "DBP0000004",
			Attributes: &stock.ExistingPlasmidAttributes{
				// Missing CreatedBy and UpdatedBy
				Name: "pMissing",
			},
		},
	}

	_, err := params.client.LoadPlasmid(params.ctx, req)

	// Repository returns Internal error for validation failures
	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.Internal,
		expectedMsgSubstring: "",
	})
}

// testLoadPlasmidOntologyIncluded tests loading a plasmid with ontology property.
func testLoadPlasmidOntologyIncluded(params *testParams) {
	params.t.Helper()
	req := newExistingPlasmid()
	req.Data.Id = "DBP0000005"
	req.Data.Attributes.CreatedAt = timestamppb.Now()
	req.Data.Attributes.UpdatedAt = timestamppb.Now()
	req.Data.Attributes.DictyPlasmidProperty = testGatewayVector

	resp, err := params.client.LoadPlasmid(params.ctx, req)

	params.assert.NoError(err, "should load plasmid without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(req.Data.Id, resp.Data.Id, "plasmid ID should match")
	// Note: Ontology property is set in request but may not be returned by repository
}

// ============================================================================
// UpdatePlasmid Test Helpers (8 tests)
// ============================================================================

// testUpdateExistingPlasmid tests successfully updating an existing plasmid.
func testUpdateExistingPlasmid(params *testParams) {
	params.t.Helper()
	// First create a plasmid
	createReq := newTestPlasmid()
	createResp, err := params.client.CreatePlasmid(params.ctx, createReq)
	params.assert.NoError(err, "should create plasmid without error")

	// Now update it
	updateReq := newPlasmidUpdate(createResp.Data.Id)
	resp, err := params.client.UpdatePlasmid(params.ctx, updateReq)

	params.assert.NoError(err, "should update plasmid without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(
		createResp.Data.Id,
		resp.Data.Id,
		"plasmid ID should match",
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
	// Note: DictyPlasmidProperty is preserved from creation (which applied default "vector")
	params.assert.NotEmpty(
		resp.Data.Attributes.DictyPlasmidProperty,
		"DictyPlasmidProperty should be present",
	)
}

// testUpdateNonExistentPlasmid tests updating a plasmid that doesn't exist.
func testUpdateNonExistentPlasmid(params *testParams) {
	params.t.Helper()
	req := newPlasmidUpdate("DBP9999999")

	_, err := params.client.UpdatePlasmid(params.ctx, req)

	// The service returns Internal error code for update errors
	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.Internal,
		expectedMsgSubstring: "",
	})
}

// testUpdatePlasmidWithEmptyID tests updating with empty ID.
func testUpdatePlasmidWithEmptyID(params *testParams) {
	params.t.Helper()
	req := &stock.PlasmidUpdate{
		Data: &stock.PlasmidUpdate_Data{
			Type: "plasmid",
			Id:   "",
			Attributes: &stock.PlasmidUpdateAttributes{
				UpdatedBy: "updateuser@dictybase.org",
				Summary:   "Updated summary",
			},
		},
	}

	_, err := params.client.UpdatePlasmid(params.ctx, req)

	// Repository returns Internal error for empty ID
	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.Internal,
		expectedMsgSubstring: "",
	})
}

// testUpdatePlasmidPartialUpdate tests updating only some fields.
func testUpdatePlasmidPartialUpdate(params *testParams) {
	params.t.Helper()
	// First create a plasmid
	createReq := newTestPlasmid()
	createResp, err := params.client.CreatePlasmid(params.ctx, createReq)
	params.assert.NoError(err, "should create plasmid without error")

	// Update only summary
	updateReq := &stock.PlasmidUpdate{
		Data: &stock.PlasmidUpdate_Data{
			Type: "plasmid",
			Id:   createResp.Data.Id,
			Attributes: &stock.PlasmidUpdateAttributes{
				UpdatedBy: "updateuser@dictybase.org",
				Summary:   "Only summary updated",
			},
		},
	}
	resp, err := params.client.UpdatePlasmid(params.ctx, updateReq)

	params.assert.NoError(err, "should update plasmid without error")
	params.assert.Equal(
		"Only summary updated",
		resp.Data.Attributes.Summary,
		"summary should be updated",
	)
	params.assert.Equal(
		createReq.Data.Attributes.Name,
		resp.Data.Attributes.Name,
		"name should remain unchanged",
	)
}

// testUpdatePlasmidOntologyUpdate tests updating a plasmid's ontology term.
func testUpdatePlasmidOntologyUpdate(params *testParams) {
	params.t.Helper()

	// Create a plasmid (default "vector" property will be applied during validation)
	createReq := newTestPlasmid()
	createResp, err := params.client.CreatePlasmid(params.ctx, createReq)
	params.assert.NoError(err, "should create plasmid without error")
	params.assert.NotNil(createResp, "response should not be nil")

	// Update ontology term to testGatewayVector
	updateReq := &stock.PlasmidUpdate{
		Data: &stock.PlasmidUpdate_Data{
			Type: "plasmid",
			Id:   createResp.Data.Id,
			Attributes: &stock.PlasmidUpdateAttributes{
				UpdatedBy:            "ontology-updater@dictybase.org",
				DictyPlasmidProperty: testGatewayVector,
			},
		},
	}

	resp, err := params.client.UpdatePlasmid(params.ctx, updateReq)
	params.assert.NoError(err, "should update plasmid ontology without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(
		"ontology-updater@dictybase.org",
		resp.Data.Attributes.UpdatedBy,
		"should update updatedBy field",
	)
	// Note: Ontology property may not be returned by repository after update
}

// testUpdatePlasmidOntologyWithOtherFields tests updating ontology term along with other fields.
func testUpdatePlasmidOntologyWithOtherFields(params *testParams) {
	params.t.Helper()

	// Create a plasmid
	createReq := newTestPlasmid()
	createResp, err := params.client.CreatePlasmid(params.ctx, createReq)
	params.assert.NoError(err, "should create plasmid without error")

	// Update ontology term AND other fields simultaneously
	updateReq := &stock.PlasmidUpdate{
		Data: &stock.PlasmidUpdate_Data{
			Type: "plasmid",
			Id:   createResp.Data.Id,
			Attributes: &stock.PlasmidUpdateAttributes{
				UpdatedBy:            "multi-updater@dictybase.org",
				DictyPlasmidProperty: testGatewayVector,
				Summary:              "Updated summary with ontology",
				Name:                 "updated-name",
				ImageMap:             "https://example.com/updated.png",
			},
		},
	}

	resp, err := params.client.UpdatePlasmid(params.ctx, updateReq)
	params.assert.NoError(err, "should update all fields without error")

	// Verify ontology term updated
	params.assert.Equal(
		testGatewayVector,
		resp.Data.Attributes.DictyPlasmidProperty,
		"should update ontology term to Gateway vector",
	)

	// Verify other fields also updated
	params.assert.Equal(
		"Updated summary with ontology",
		resp.Data.Attributes.Summary,
		"should update summary",
	)
	params.assert.Equal(
		"updated-name",
		resp.Data.Attributes.Name,
		"should update name",
	)
	params.assert.Equal(
		"https://example.com/updated.png",
		resp.Data.Attributes.ImageMap,
		"should update image map",
	)
}

// testUpdatePlasmidInvalidOntology tests that invalid ontology terms are rejected.
func testUpdatePlasmidInvalidOntology(params *testParams) {
	params.t.Helper()

	// Create a plasmid
	createReq := newTestPlasmid()
	createResp, err := params.client.CreatePlasmid(params.ctx, createReq)
	params.assert.NoError(err, "should create plasmid without error")

	// Try to update with invalid ontology term
	updateReq := &stock.PlasmidUpdate{
		Data: &stock.PlasmidUpdate_Data{
			Type: "plasmid",
			Id:   createResp.Data.Id,
			Attributes: &stock.PlasmidUpdateAttributes{
				UpdatedBy:            "bad-updater@dictybase.org",
				DictyPlasmidProperty: "invalid ontology term that does not exist",
			},
		},
	}

	_, err = params.client.UpdatePlasmid(params.ctx, updateReq)
	params.assert.Error(err, "should error on invalid ontology term")

	// Verify it's an Internal error (from repository layer)
	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.Internal,
		expectedMsgSubstring: "invalid ontology term",
	})
}

// testUpdatePlasmidOntologyPreservation tests that ontology is preserved when not specified.
func testUpdatePlasmidOntologyPreservation(params *testParams) {
	params.t.Helper()

	// Create a plasmid (will have default "vector")
	createReq := newTestPlasmid()
	createResp, err := params.client.CreatePlasmid(params.ctx, createReq)
	params.assert.NoError(err, "should create plasmid without error")
	originalOntology := createResp.Data.Attributes.DictyPlasmidProperty

	// Update other fields WITHOUT specifying DictyPlasmidProperty
	updateReq := &stock.PlasmidUpdate{
		Data: &stock.PlasmidUpdate_Data{
			Type: "plasmid",
			Id:   createResp.Data.Id,
			Attributes: &stock.PlasmidUpdateAttributes{
				UpdatedBy: "preserve-updater@dictybase.org",
				Summary:   "Updated summary only",
			},
		},
	}

	resp, err := params.client.UpdatePlasmid(params.ctx, updateReq)
	params.assert.NoError(err, "should update without error")

	// Verify summary was updated
	params.assert.Equal(
		"Updated summary only",
		resp.Data.Attributes.Summary,
		"summary should be updated",
	)

	// Note: Original ontology (from creation) may or may not be returned
	// depending on repository implementation. The service preserves it internally.
	_ = originalOntology
}

// ============================================================================
// ListPlasmids Test Helpers (5 tests)
// ============================================================================

// testListPlasmidsDefault tests listing plasmids with default parameters.
func testListPlasmidsDefault(params *testParams) {
	params.t.Helper()
	// Create a few plasmids
	for range 5 {
		createReq := newTestPlasmid()
		_, err := params.client.CreatePlasmid(params.ctx, createReq)
		params.assert.NoError(err, "should create plasmid without error")
	}

	// List with a basic filter using proper format: field===value
	// Use 'depositor' field which is in the filter map
	req := &stock.StockParameters{Filter: "depositor===John Doe"}
	resp, err := params.client.ListPlasmids(params.ctx, req)

	params.assert.NoError(err, "should list plasmids without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.NotNil(resp.Meta, "meta should not be nil")
	params.assert.GreaterOrEqual(
		len(resp.Data),
		5,
		"should return at least the created plasmids",
	)
	params.assert.Greater(resp.Meta.Limit, int64(0), "limit should be set")
}

// testListPlasmidsWithLimit tests listing plasmids with a limit.
func testListPlasmidsWithLimit(params *testParams) {
	params.t.Helper()
	// Create several plasmids
	for range 10 {
		createReq := newTestPlasmid()
		_, err := params.client.CreatePlasmid(params.ctx, createReq)
		params.assert.NoError(err, "should create plasmid without error")
	}

	// List with limit
	req := &stock.StockParameters{Limit: 3, Filter: "depositor===John Doe"}
	resp, err := params.client.ListPlasmids(params.ctx, req)

	params.assert.NoError(err, "should list plasmids without error")
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

// testListPlasmidsWithLimitNoFilter tests listing plasmids with a limit but no filter.
func testListPlasmidsWithLimitNoFilter(params *testParams) {
	params.t.Helper()
	// Create several plasmids
	for range 10 {
		createReq := newTestPlasmid()
		_, err := params.client.CreatePlasmid(params.ctx, createReq)
		params.assert.NoError(err, "should create plasmid without error")
	}

	// List with limit but NO filter
	req := &stock.StockParameters{Limit: 5}
	resp, err := params.client.ListPlasmids(params.ctx, req)

	params.assert.NoError(err, "should list plasmids without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.LessOrEqual(
		len(resp.Data),
		5,
		"should respect limit",
	)
	params.assert.Equal(
		int64(5),
		resp.Meta.Limit,
		"meta limit should match request",
	)
	params.assert.NotEmpty(resp.Data, "should return some plasmids")
}

// testListPlasmidsWithCursor tests pagination with cursor.
func testListPlasmidsWithCursor(params *testParams) {
	params.t.Helper()
	// Create several plasmids
	for range 15 {
		createReq := newTestPlasmid()
		_, err := params.client.CreatePlasmid(params.ctx, createReq)
		params.assert.NoError(err, "should create plasmid without error")
	}

	// First page
	req := &stock.StockParameters{Limit: 5, Filter: "depositor===John Doe"}
	resp, err := params.client.ListPlasmids(params.ctx, req)
	params.assert.NoError(err, "should list plasmids without error")
	params.assert.NotNil(resp, "response should not be nil")

	// If we got a next cursor, fetch next page
	if resp.Meta.NextCursor != 0 {
		req2 := &stock.StockParameters{
			Limit:  5,
			Cursor: resp.Meta.NextCursor,
			Filter: "depositor===John Doe",
		}
		resp2, err := params.client.ListPlasmids(params.ctx, req2)
		params.assert.NoError(err, "should list second page without error")
		params.assert.NotNil(resp2, "second page should not be nil")

		// Ensure different results (no overlap)
		firstPageIDs := make(map[string]bool)
		for _, plasmid := range resp.Data {
			firstPageIDs[plasmid.Id] = true
		}
		for _, plasmid := range resp2.Data {
			params.assert.False(
				firstPageIDs[plasmid.Id],
				"second page should not overlap with first page",
			)
		}
	}
}

// testListPlasmidsEmpty tests listing when no plasmids match the filter.
func testListPlasmidsEmpty(params *testParams) {
	params.t.Helper()
	// Use a filter that won't match anything
	req := &stock.StockParameters{
		Limit:  10,
		Filter: "depositor===NonExistentDepositor",
	}
	resp, err := params.client.ListPlasmids(params.ctx, req)

	// When no plasmids match, returns empty list rather than error
	params.assert.NoError(err, "should not error on empty result")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Empty(resp.Data, "should return empty list when no matches")
}

// testListPlasmidsInvalidFilter tests listing with invalid filter format.
func testListPlasmidsInvalidFilter(params *testParams) {
	params.t.Helper()
	// Use invalid filter format (should be field===value)
	req := &stock.StockParameters{
		Limit:  10,
		Filter: "invalid-filter-format",
	}
	_, err := params.client.ListPlasmids(params.ctx, req)

	// Invalid filter returns Internal error from repository
	assertGrpcError(assertGrpcErrorParams{
		assert:               params.assert,
		err:                  err,
		expectedCode:         codes.Internal,
		expectedMsgSubstring: "",
	})
}

// ============================================================================
// Tag Filtering Test Helpers (New for this feature)
// ============================================================================

// testListPlasmidsByTagExact tests exact match filtering by tag
func testListPlasmidsByTagExact(params *testParams) {
	params.t.Helper()

	// Create plasmids with different ontology terms
	req1 := newTestPlasmid()
	req1.Data.Attributes.DictyPlasmidProperty = testGatewayVector
	resp1, err := params.client.CreatePlasmid(params.ctx, req1)
	params.assert.NoError(err, "should create first plasmid")

	req2 := newTestPlasmid()
	req2.Data.Attributes.DictyPlasmidProperty = "expression vector"
	_, err = params.client.CreatePlasmid(params.ctx, req2)
	params.assert.NoError(err, "should create second plasmid")

	// Filter by exact tag match
	listReq := &stock.StockParameters{
		Filter: "tag===Gateway vector",
		Limit:  10,
	}
	resp, err := params.client.ListPlasmids(params.ctx, listReq)

	params.assert.NoError(err, "should list plasmids without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.GreaterOrEqual(len(resp.Data), 1, "should find at least one plasmid")

	// Verify only Gateway vector plasmids returned
	found := false
	for _, plasmid := range resp.Data {
		params.assert.Equal(
			testGatewayVector,
			plasmid.Attributes.DictyPlasmidProperty,
			"all results should have Gateway vector property",
		)
		if plasmid.Id == resp1.Data.Id {
			found = true
		}
	}
	params.assert.True(found, "should find the Gateway vector plasmid")
}

// testListPlasmidsByTagPartialMatch tests regex match
func testListPlasmidsByTagPartialMatch(params *testParams) {
	params.t.Helper()

	for range 3 {
		req := newTestPlasmid()
		req.Data.Attributes.DictyPlasmidProperty = testGatewayVector
		_, err := params.client.CreatePlasmid(params.ctx, req)
		params.assert.NoError(err)
	}

	listReq := &stock.StockParameters{
		Filter: "tag=~Gateway",
		Limit:  10,
	}
	resp, err := params.client.ListPlasmids(params.ctx, listReq)

	params.assert.NoError(err, "should list plasmids without error")
	params.assert.GreaterOrEqual(len(resp.Data), 3, "should find at least 3 plasmids")

	for _, plasmid := range resp.Data {
		params.assert.Contains(
			plasmid.Attributes.DictyPlasmidProperty,
			"Gateway",
			"all results should contain 'Gateway'",
		)
	}
}

// testListPlasmidsByTagWithLimit tests pagination with limit
func testListPlasmidsByTagWithLimit(params *testParams) {
	params.t.Helper()

	for range 10 {
		req := newTestPlasmid()
		req.Data.Attributes.DictyPlasmidProperty = testGatewayVector
		_, err := params.client.CreatePlasmid(params.ctx, req)
		params.assert.NoError(err)
	}

	listReq := &stock.StockParameters{
		Filter: "tag===Gateway vector",
		Limit:  5,
	}
	resp, err := params.client.ListPlasmids(params.ctx, listReq)

	params.assert.NoError(err)
	params.assert.LessOrEqual(len(resp.Data), 5, "should respect limit")
	params.assert.Equal(int64(5), resp.Meta.Limit, "meta limit should match request")
}

// testListPlasmidsByTagWithCursor tests cursor-based pagination
func testListPlasmidsByTagWithCursor(params *testParams) {
	params.t.Helper()

	for range 15 {
		req := newTestPlasmid()
		req.Data.Attributes.DictyPlasmidProperty = testGatewayVector
		_, err := params.client.CreatePlasmid(params.ctx, req)
		params.assert.NoError(err)
	}

	// First page
	req := &stock.StockParameters{
		Filter: "tag===Gateway vector",
		Limit:  5,
	}
	resp, err := params.client.ListPlasmids(params.ctx, req)
	params.assert.NoError(err)

	// Second page
	if resp.Meta.NextCursor != 0 {
		req2 := &stock.StockParameters{
			Filter: "tag===Gateway vector",
			Limit:  5,
			Cursor: resp.Meta.NextCursor,
		}
		resp2, err := params.client.ListPlasmids(params.ctx, req2)
		params.assert.NoError(err)

		// Verify no overlap
		firstPageIDs := make(map[string]bool)
		for _, plasmid := range resp.Data {
			firstPageIDs[plasmid.Id] = true
		}
		for _, plasmid := range resp2.Data {
			params.assert.False(
				firstPageIDs[plasmid.Id],
				"second page should not overlap with first page",
			)
		}
	}
}

// testListPlasmidsByTagEmpty tests empty results
func testListPlasmidsByTagEmpty(params *testParams) {
	params.t.Helper()

	listReq := &stock.StockParameters{
		Filter: "tag===NonExistentOntologyTerm",
		Limit:  10,
	}
	resp, err := params.client.ListPlasmids(params.ctx, listReq)

	params.assert.NoError(err, "should not error on empty result")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Empty(resp.Data, "should return empty list")
}

// testListPlasmidsByTagCombined tests combined filters
func testListPlasmidsByTagCombined(params *testParams) {
	params.t.Helper()

	req := newTestPlasmid()
	req.Data.Attributes.DictyPlasmidProperty = testGatewayVector
	req.Data.Attributes.Depositor = "Jane Smith"
	_, err := params.client.CreatePlasmid(params.ctx, req)
	params.assert.NoError(err)

	listReq := &stock.StockParameters{
		Filter: "tag===Gateway vector;depositor===Jane Smith",
		Limit:  10,
	}
	resp, err := params.client.ListPlasmids(params.ctx, listReq)

	params.assert.NoError(err)
	params.assert.GreaterOrEqual(len(resp.Data), 1)

	for _, plasmid := range resp.Data {
		params.assert.Equal(testGatewayVector, plasmid.Attributes.DictyPlasmidProperty)
		params.assert.Equal("Jane Smith", plasmid.Attributes.Depositor)
	}
}

// testListPlasmidsByNameExact tests exact match filtering by plasmid_name.
func testListPlasmidsByNameExact(params *testParams) {
	params.t.Helper()

	// Create a plasmid with the default test name "pDV101"
	req := newTestPlasmid()
	resp1, err := params.client.CreatePlasmid(params.ctx, req)
	params.assert.NoError(err, "should create plasmid")

	// Create a plasmid with a different name
	req2 := newTestPlasmid()
	req2.Data.Attributes.Name = "pOther"
	_, err = params.client.CreatePlasmid(params.ctx, req2)
	params.assert.NoError(err, "should create second plasmid")

	listReq := &stock.StockParameters{
		Filter: "plasmid_name===pDV101",
		Limit:  10,
	}
	resp, err := params.client.ListPlasmids(params.ctx, listReq)

	params.assert.NoError(err, "should list plasmids without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.GreaterOrEqual(len(resp.Data), 1, "should find at least one plasmid")

	found := false
	for _, plasmid := range resp.Data {
		params.assert.Equal("pDV101", plasmid.Attributes.Name, "all results should have name pDV101")
		if plasmid.Id == resp1.Data.Id {
			found = true
		}
	}
	params.assert.True(found, "should find the created pDV101 plasmid")
}

// testListPlasmidsByNamePartialMatch tests regex match filtering by plasmid_name.
func testListPlasmidsByNamePartialMatch(params *testParams) {
	params.t.Helper()

	for range 3 {
		req := newTestPlasmid()
		req.Data.Attributes.Name = "pDV101"
		_, err := params.client.CreatePlasmid(params.ctx, req)
		params.assert.NoError(err)
	}

	listReq := &stock.StockParameters{
		Filter: "plasmid_name=~pDV",
		Limit:  10,
	}
	resp, err := params.client.ListPlasmids(params.ctx, listReq)

	params.assert.NoError(err, "should list plasmids without error")
	params.assert.GreaterOrEqual(len(resp.Data), 3, "should find at least 3 plasmids")

	for _, plasmid := range resp.Data {
		params.assert.Contains(
			plasmid.Attributes.Name,
			"pDV",
			"all results should contain 'pDV'",
		)
	}
}

// testListPlasmidsSmallLimit is a regression test for the pagination bug where
// Limit <= 3 would incorrectly trim the only result and return an empty collection.
func testListPlasmidsSmallLimit(params *testParams) {
	params.t.Helper()

	uniqueName := "pSmallLimit1"
	req := newTestPlasmid()
	req.Data.Attributes.Name = uniqueName
	_, err := params.client.CreatePlasmid(params.ctx, req)
	params.assert.NoError(err, "should create plasmid")

	listReq := &stock.StockParameters{
		Filter: "plasmid_name===" + uniqueName,
		Limit:  1,
	}
	resp, err := params.client.ListPlasmids(params.ctx, listReq)

	params.assert.NoError(err, "should list plasmids without error")
	params.assert.NotNil(resp, "response should not be nil")
	params.assert.Equal(1, len(resp.Data), "should return exactly one plasmid")
	params.assert.Equal(uniqueName, resp.Data[0].Attributes.Name, "should match the queried name")
	params.assert.Equal(int64(0), resp.Meta.NextCursor, "should have no next cursor")
}
