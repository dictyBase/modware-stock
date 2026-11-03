package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

func TestCountStocksHandlerMapsCounts(t *testing.T) {
	assert := require.New(t)
	stub := &fullSearchStubRepo{
		counts: &repository.StockCounts{
			Strains: repository.EntityCounts{
				Total: 9,
				Subtypes: []repository.SubtypeCount{
					{Name: subtypeRegularName, Count: 6},
					{Name: subtypeGwdiName, Count: 2},
					{Name: subtypeBacterialName, Count: 1},
				},
			},
			Plasmids: repository.EntityCounts{
				Total: 5,
				Subtypes: []repository.SubtypeCount{
					{Name: subtypeRegularName, Count: 3},
					{Name: subtypeGoldenBraidName, Count: 2},
				},
			},
		},
	}
	svc := setupTestService(stub)
	res, err := svc.CountStocks(context.Background(), &empty.Empty{})
	assert.NoError(err)
	assert.NotNil(res)
	assert.Equal(int64(9), res.GetStrains().GetTotal())
	assert.Equal(
		[]*stock.StockSubtypeCount{
			{Name: subtypeRegularName, Count: 6},
			{Name: subtypeGwdiName, Count: 2},
			{Name: subtypeBacterialName, Count: 1},
		},
		res.GetStrains().GetSubtypes(),
	)
	assert.Equal(int64(5), res.GetPlasmids().GetTotal())
	assert.Equal(
		[]*stock.StockSubtypeCount{
			{Name: subtypeRegularName, Count: 3},
			{Name: subtypeGoldenBraidName, Count: 2},
		},
		res.GetPlasmids().GetSubtypes(),
	)
}

func TestCountStocksHandlerEmptyCounts(t *testing.T) {
	assert := require.New(t)
	stub := &fullSearchStubRepo{
		counts: &repository.StockCounts{},
	}
	svc := setupTestService(stub)
	res, err := svc.CountStocks(context.Background(), &empty.Empty{})
	assert.NoError(err)
	assert.NotNil(res.GetStrains())
	assert.Equal(int64(0), res.GetStrains().GetTotal())
	assert.Empty(res.GetStrains().GetSubtypes())
	assert.NotNil(res.GetPlasmids())
	assert.Equal(int64(0), res.GetPlasmids().GetTotal())
	assert.Empty(res.GetPlasmids().GetSubtypes())
}

func TestCountStocksHandlerForwardsContextAndMapsErrors(t *testing.T) {
	assert := require.New(t)
	stub := &fullSearchStubRepo{counts: &repository.StockCounts{}}
	svc := setupTestService(stub)
	ctx := t.Context()

	_, err := svc.CountStocks(ctx, &empty.Empty{})
	assert.NoError(err)
	assert.Equal(ctx, stub.gotCtx)

	internalSvc := setupTestService(&fullSearchStubRepo{
		err: errors.New("database failure"),
	})
	_, err = internalSvc.CountStocks(ctx, &empty.Empty{})
	assert.Equal(codes.Internal, status.Code(err))

	canceledSvc := setupTestService(&fullSearchStubRepo{
		err: fmt.Errorf("wrapped: %w", context.Canceled),
	})
	_, err = canceledSvc.CountStocks(ctx, &empty.Empty{})
	assert.Equal(codes.Canceled, status.Code(err))

	deadlineSvc := setupTestService(&fullSearchStubRepo{
		err: fmt.Errorf("wrapped: %w", context.DeadlineExceeded),
	})
	_, err = deadlineSvc.CountStocks(ctx, &empty.Empty{})
	assert.Equal(codes.DeadlineExceeded, status.Code(err))
}
