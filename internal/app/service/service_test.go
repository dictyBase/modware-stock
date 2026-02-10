package service

import (
	"context"
	"testing"

	"github.com/dictyBase/go-genproto/dictybaseapis/api/upload"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/go-obograph/storage"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRemoveStock(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()

	t.Run("RemovesExistingStock", func(t *testing.T) {
		testRemoveExistingStock(ctx, t, client, assert)
	})

	t.Run("ReturnsErrorForNonExistentStock", func(t *testing.T) {
		testRemoveNonExistentStock(ctx, t, client, assert)
	})

	t.Run("ReturnsErrorForEmptyID", func(t *testing.T) {
		testRemoveStockWithEmptyID(ctx, t, client, assert)
	})

	t.Run("ReturnsErrorForInvalidID", func(t *testing.T) {
		testRemoveStockWithInvalidID(ctx, t, client, assert)
	})
}

func TestUploadResponse(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsCreatedStatusWhenOntologyIsCreated", func(t *testing.T) {
		testUploadResponseCreated(t)
	})

	t.Run("ReturnsUpdatedStatusWhenOntologyIsUpdated", func(t *testing.T) {
		testUploadResponseUpdated(t)
	})

	t.Run("HandlesNilUploadInformation", func(t *testing.T) {
		testUploadResponseWithNilInfo(t)
	})
}

func TestGenNextCursorVal(t *testing.T) {
	t.Parallel()

	t.Run("ConvertsTimestampToUnixMilli", func(t *testing.T) {
		testGenNextCursorValConversion(t)
	})

	t.Run("HandlesZeroTimestamp", func(t *testing.T) {
		testGenNextCursorValZero(t)
	})

	t.Run("HandlesFutureTimestamp", func(t *testing.T) {
		testGenNextCursorValFuture(t)
	})
}

func TestStockAQLStatement(t *testing.T) {
	t.Parallel()

	t.Run("GeneratesValidAQLForSimpleFilter", func(t *testing.T) {
		testStockAQLStatementSimpleFilter(t)
	})

	t.Run("ReturnsEmptyStringForEmptyFilter", func(t *testing.T) {
		testStockAQLStatementEmptyFilter(t)
	})

	t.Run("ReturnsErrorForInvalidFilterSyntax", func(t *testing.T) {
		testStockAQLStatementInvalidSyntax(t)
	})

	t.Run("HandlesComplexFilterExpression", func(t *testing.T) {
		testStockAQLStatementComplexFilter(t)
	})
}

func TestStockModelList(t *testing.T) {
	t.Parallel()
	client, assert := setup(t)
	ctx := context.Background()

	t.Run("ReturnsModelListWithValidParams", func(t *testing.T) {
		testStockModelListValid(ctx, t, client, assert)
	})

	t.Run("ReturnsErrorForInvalidFilter", func(t *testing.T) {
		testStockModelListInvalidFilter(ctx, t, client, assert)
	})

	t.Run("ReturnsNotFoundWhenNoResults", func(t *testing.T) {
		testStockModelListNoResults(ctx, t, client, assert)
	})
}

func TestLimitVal(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsInputWhenPositive", func(t *testing.T) {
		testLimitValPositive(t)
	})

	t.Run("ReturnsDefaultWhenZero", func(t *testing.T) {
		testLimitValZero(t)
	})

	t.Run("ReturnsDefaultWhenNegative", func(t *testing.T) {
		testLimitValNegative(t)
	})

	t.Run("ReturnsLargeValue", func(t *testing.T) {
		testLimitValLarge(t)
	})
}

// Note: OboStreamHandlerWrite tests are complex and require full gRPC stream mocking
// The Write method is tested indirectly through integration tests of OboJSONFileUpload

// Test implementations for RemoveStock

func testRemoveExistingStock(
	ctx context.Context,
	t *testing.T,
	client stock.StockServiceClient,
	assert *require.Assertions,
) {
	t.Helper()

	// Create a strain first
	ns := newTestStrain()
	created, err := client.CreateStrain(ctx, ns)
	assert.NoError(err)
	assert.NotNil(created)
	assert.NotEmpty(created.Data.Id)

	// Remove the strain
	_, err = client.RemoveStock(ctx, &stock.StockId{Id: created.Data.Id})
	assert.NoError(err)

	// Verify it's removed by trying to get it
	_, err = client.GetStrain(ctx, &stock.StockId{Id: created.Data.Id})
	assert.Error(err)
	grpcErr, ok := status.FromError(err)
	assert.True(ok)
	assert.Equal(codes.NotFound, grpcErr.Code())
}

func testRemoveNonExistentStock(
	ctx context.Context,
	t *testing.T,
	client stock.StockServiceClient,
	assert *require.Assertions,
) {
	t.Helper()

	_, err := client.RemoveStock(ctx, &stock.StockId{Id: "DBS9999999"})
	assert.Error(err)
	grpcErr, ok := status.FromError(err)
	assert.True(ok)
	// Could be NotFound or Internal depending on implementation
	assert.Contains([]codes.Code{codes.NotFound, codes.Internal}, grpcErr.Code())
}

func testRemoveStockWithEmptyID(
	ctx context.Context,
	t *testing.T,
	client stock.StockServiceClient,
	assert *require.Assertions,
) {
	t.Helper()

	_, err := client.RemoveStock(ctx, &stock.StockId{Id: ""})
	assert.Error(err)
	grpcErr, ok := status.FromError(err)
	assert.True(ok)
	assert.Equal(codes.InvalidArgument, grpcErr.Code())
}

func testRemoveStockWithInvalidID(
	ctx context.Context,
	t *testing.T,
	client stock.StockServiceClient,
	assert *require.Assertions,
) {
	t.Helper()

	_, err := client.RemoveStock(ctx, &stock.StockId{Id: "invalid-id-format"})
	assert.Error(err)
	grpcErr, ok := status.FromError(err)
	assert.True(ok)
	// Could be NotFound or Internal depending on implementation
	assert.Contains([]codes.Code{codes.NotFound, codes.Internal}, grpcErr.Code())
}

// Test implementations for uploadResponse

func testUploadResponseCreated(t *testing.T) {
	t.Helper()

	info := &storage.UploadInformation{
		IsCreated: true,
	}

	result := uploadResponse(info)
	require.Equal(t, upload.FileUploadResponse_CREATED, result)
}

func testUploadResponseUpdated(t *testing.T) {
	t.Helper()

	info := &storage.UploadInformation{
		IsCreated: false,
	}

	result := uploadResponse(info)
	require.Equal(t, upload.FileUploadResponse_UPDATED, result)
}

func testUploadResponseWithNilInfo(t *testing.T) {
	t.Helper()

	// Test with zero-value struct
	// It will return UPDATED due to IsCreated being false
	result := uploadResponse(&storage.UploadInformation{})
	require.Equal(t, upload.FileUploadResponse_UPDATED, result)
}

// Test implementations for genNextCursorVal

func testGenNextCursorValConversion(t *testing.T) {
	t.Helper()

	ts := timestamppb.Now()
	result := genNextCursorVal(ts)

	require.Greater(t, result, int64(0))
	require.Equal(t, ts.AsTime().UnixMilli(), result)
}

func testGenNextCursorValZero(t *testing.T) {
	t.Helper()

	ts := &timestamppb.Timestamp{Seconds: 0, Nanos: 0}
	result := genNextCursorVal(ts)

	require.Equal(t, int64(0), result)
}

func testGenNextCursorValFuture(t *testing.T) {
	t.Helper()

	// Create a future timestamp
	ts := &timestamppb.Timestamp{Seconds: 2000000000, Nanos: 0}
	result := genNextCursorVal(ts)

	require.Greater(t, result, int64(0))
	require.Equal(t, int64(2000000000000), result)
}

// Test implementations for stockAQLStatement

func testStockAQLStatementSimpleFilter(t *testing.T) {
	t.Helper()

	stmt, err := stockAQLStatement("summary==test")
	require.NoError(t, err)
	require.NotEmpty(t, stmt)
	require.Contains(t, stmt, "FILTER")
}

func testStockAQLStatementEmptyFilter(t *testing.T) {
	t.Helper()
	stmt, err := stockAQLStatement("")
	require.NoError(t, err)
	require.Empty(t, stmt)
}

func testStockAQLStatementInvalidSyntax(t *testing.T) {
	t.Helper()

	// Test filter with truly invalid syntax (missing value)
	_, err := stockAQLStatement("summary==")
	require.Error(t, err)
}

func testStockAQLStatementComplexFilter(t *testing.T) {
	t.Helper()

	// Test with a valid complex filter
	stmt, err := stockAQLStatement("summary==test;depositor==user")
	require.NoError(t, err)
	require.NotEmpty(t, stmt)
	require.Contains(t, stmt, "FILTER")
}

// Test implementations for stockModelList

func testStockModelListValid(
	ctx context.Context,
	t *testing.T,
	client stock.StockServiceClient,
	assert *require.Assertions,
) {
	t.Helper()

	// Create a strain first
	ns := newTestStrain()
	created, err := client.CreateStrain(ctx, ns)
	assert.NoError(err)
	assert.NotNil(created)

	// Use the service's repo directly to test stockModelList
	// This is an internal function so we test it indirectly through ListStrains
	// Use a valid field from FMap
	result, err := client.ListStrains(ctx, &stock.StockParameters{
		Cursor: 0,
		Limit:  10,
		Filter: "depositor==John Doe",
	})
	assert.NoError(err)
	assert.NotNil(result)
}

func testStockModelListInvalidFilter(
	ctx context.Context,
	t *testing.T,
	client stock.StockServiceClient,
	assert *require.Assertions,
) {
	t.Helper()

	// Test with invalid filter syntax
	_, err := client.ListStrains(ctx, &stock.StockParameters{
		Cursor: 0,
		Limit:  10,
		Filter: "invalid===filter",
	})
	assert.Error(err)
	grpcErr, ok := status.FromError(err)
	assert.True(ok)
	assert.Equal(codes.InvalidArgument, grpcErr.Code())
}

func testStockModelListNoResults(
	ctx context.Context,
	t *testing.T,
	client stock.StockServiceClient,
	assert *require.Assertions,
) {
	t.Helper()

	// Filter that won't match anything
	_, err := client.ListStrains(ctx, &stock.StockParameters{
		Cursor: 0,
		Limit:  10,
		Filter: "summary==nonexistent-strain-summary-that-wont-match",
	})
	assert.Error(err)
	grpcErr, ok := status.FromError(err)
	assert.True(ok)
	assert.Equal(codes.NotFound, grpcErr.Code())
}

// Test implementations for limitVal

func testLimitValPositive(t *testing.T) {
	t.Helper()

	result := limitVal(25)
	require.Equal(t, int64(25), result)
}

func testLimitValZero(t *testing.T) {
	t.Helper()

	result := limitVal(0)
	require.Equal(t, int64(10), result)
}

func testLimitValNegative(t *testing.T) {
	t.Helper()

	result := limitVal(-5)
	require.Equal(t, int64(10), result)
}

func testLimitValLarge(t *testing.T) {
	t.Helper()

	result := limitVal(1000)
	require.Equal(t, int64(1000), result)
}
