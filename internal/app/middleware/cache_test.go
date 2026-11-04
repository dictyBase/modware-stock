package middleware

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	E "github.com/IBM/fp-go/v2/either"
	F "github.com/IBM/fp-go/v2/function"
	IOE "github.com/IBM/fp-go/v2/ioeither"
	O "github.com/IBM/fp-go/v2/option"
	"github.com/alicebob/miniredis/v2"
	ioeutils "github.com/dictyBase/fp-go-loom/ioeitherutils"
	loomcheck "github.com/dictyBase/fp-go-loom/loomcheck"
	"github.com/dictyBase/fp-go-loom/pipelinecheck"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/cache"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

type (
	failingStore struct{}

	setFailingStore struct {
		inner cache.Store
	}
)

const testStockID = "DBS1"

func (failingStore) Get(
	_ context.Context,
	_ string,
) IOE.IOEither[error, O.Option[[]byte]] {
	return IOE.Left[O.Option[[]byte]](errors.New("cache get error"))
}

func (failingStore) Set(
	_ context.Context,
	_ string,
	_ []byte,
	_ time.Duration,
) IOE.IOEither[error, struct{}] {
	return IOE.Left[struct{}](errors.New("cache set error"))
}

func (failingStore) Delete(
	_ context.Context,
	_ string,
) IOE.IOEither[error, struct{}] {
	return IOE.Left[struct{}](errors.New("cache delete error"))
}

func (failingStore) Close() error {
	return nil
}

func (s setFailingStore) Get(
	ctx context.Context,
	key string,
) IOE.IOEither[error, O.Option[[]byte]] {
	return s.inner.Get(ctx, key)
}

func (s setFailingStore) Set(
	_ context.Context,
	_ string,
	_ []byte,
	_ time.Duration,
) IOE.IOEither[error, struct{}] {
	return IOE.Left[struct{}](errors.New("cache set error"))
}

func (s setFailingStore) Delete(
	ctx context.Context,
	key string,
) IOE.IOEither[error, struct{}] {
	return s.inner.Delete(ctx, key)
}

func (s setFailingStore) Close() error {
	return s.inner.Close()
}

func sampleStockCounts(totalStrains, totalPlasmids int64) *stock.StockCounts {
	return &stock.StockCounts{
		Strains: &stock.StockEntityCounts{
			Total: totalStrains,
			Subtypes: []*stock.StockSubtypeCount{
				{Name: "regular", Count: totalStrains},
			},
		},
		Plasmids: &stock.StockEntityCounts{
			Total: totalPlasmids,
			Subtypes: []*stock.StockSubtypeCount{
				{Name: "regular", Count: totalPlasmids},
			},
		},
	}
}

func TestStockCountsCacheInterceptorWithMemoryStore(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	store := cache.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	interceptor := StockCountsCacheInterceptor(store, DefaultStockCountsCacheTTL)
	var countCalls atomic.Int32

	countHandler := func(ctx context.Context, req any) (any, error) {
		n := int64(countCalls.Add(1))
		return sampleStockCounts(10*n, 5*n), nil
	}
	countInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_CountStocks_FullMethodName,
	}

	resp1, err := interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	assert.Equal(int32(1), countCalls.Load())
	counts1, ok := resp1.(*stock.StockCounts)
	assert.True(ok)
	assert.Equal(int64(10), counts1.GetStrains().GetTotal())

	counts1.Strains.Total = 999

	resp2, err := interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	assert.Equal(int32(1), countCalls.Load())
	counts2, ok := resp2.(*stock.StockCounts)
	assert.True(ok)
	assert.Equal(int64(10), counts2.GetStrains().GetTotal())

	getStrainInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_GetStrain_FullMethodName,
	}
	_, err = interceptor(
		context.Background(),
		&stock.StockId{Id: testStockID},
		getStrainInfo,
		func(ctx context.Context, req any) (any, error) {
			return &stock.Strain{}, nil
		},
	)
	assert.NoError(err)

	_, err = interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	assert.Equal(int32(1), countCalls.Load())

	createInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_CreateStrain_FullMethodName,
	}
	_, err = interceptor(
		context.Background(),
		&stock.NewStrain{},
		createInfo,
		func(ctx context.Context, req any) (any, error) {
			return nil, errors.New("validation failed")
		},
	)
	assert.Error(err)
	_, err = interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	assert.Equal(int32(1), countCalls.Load())

	_, err = interceptor(
		context.Background(),
		&stock.NewStrain{},
		createInfo,
		func(ctx context.Context, req any) (any, error) {
			return &stock.Strain{}, nil
		},
	)
	assert.NoError(err)

	resp3, err := interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	assert.Equal(int32(2), countCalls.Load())
	counts3, ok := resp3.(*stock.StockCounts)
	assert.True(ok)
	assert.Equal(int64(20), counts3.GetStrains().GetTotal())
}

func TestStockCountsCacheInterceptorWithRedisStore(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	mr, err := miniredis.Run()
	assert.NoError(err)
	t.Cleanup(mr.Close)

	storeRes := F.Pipe1(
		cache.NewRedisStore(mr.Addr()),
		ioeutils.ToEither[error, cache.Store],
	)
	assert.True(E.IsRight(storeRes))
	store := F.Pipe1(
		storeRes,
		E.GetOrElse(F.Constant1[error](cache.Store(nil))),
	)
	t.Cleanup(func() { _ = store.Close() })

	interceptor := StockCountsCacheInterceptor(store, 0)
	var countCalls atomic.Int32
	countInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_CountStocks_FullMethodName,
	}
	countHandler := func(ctx context.Context, req any) (any, error) {
		n := int64(countCalls.Add(1))
		return sampleStockCounts(100*n, 50*n), nil
	}

	_, err = interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	assert.Equal(int32(1), countCalls.Load())
	assert.Equal(DefaultStockCountsCacheTTL, mr.TTL(StockCountsCacheKey))

	_ = mr.Set(StockCountsCacheKey, "not-valid-protobuf-\xff\xfe")
	resp, err := interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	assert.Equal(int32(2), countCalls.Load())
	counts, ok := resp.(*stock.StockCounts)
	assert.True(ok)
	assert.Equal(int64(200), counts.GetStrains().GetTotal())
}

func TestStockCountsCacheInterceptorFailsOpenOnStoreError(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	interceptor := StockCountsCacheInterceptor(failingStore{}, DefaultStockCountsCacheTTL)
	countInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_CountStocks_FullMethodName,
	}
	resp, err := interceptor(
		context.Background(),
		&empty.Empty{},
		countInfo,
		func(ctx context.Context, req any) (any, error) {
			return sampleStockCounts(7, 3), nil
		},
	)
	assert.NoError(err)
	counts, ok := resp.(*stock.StockCounts)
	assert.True(ok)
	assert.Equal(int64(7), counts.GetStrains().GetTotal())

	removeInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_RemoveStock_FullMethodName,
	}
	_, err = interceptor(
		context.Background(),
		&stock.StockId{Id: testStockID},
		removeInfo,
		func(ctx context.Context, req any) (any, error) {
			return &empty.Empty{}, nil
		},
	)
	assert.NoError(err)
}

func TestMiddlewarePipelineAndLoomStyle(t *testing.T) {
	t.Parallel()
	cfg := pipelinecheck.Config{
		Roots: []string{"."},
		IsEntrypoint: func(name string) bool {
			return strings.HasPrefix(name, "Run") || strings.HasPrefix(name, "run")
		},
		RequirePointFreeSeed:      true,
		RequirePointFreeBranching: true,
		RequireTrivialDelegate:    true,
	}
	pipelinecheck.Require(t, cfg)
	pipelinecheck.RequireBareIf(t, cfg)
	pipelinecheck.RequireNoIfErrInTryCatch(t, cfg)
	pipelinecheck.RequireNoNonRawTryCatchCallback(t, cfg)
	pipelinecheck.RequireSafePrintf(t, cfg)
	pipelinecheck.RequireNoRedundantFold(t, cfg)
	pipelinecheck.RequireNoHandRolledFileJoinInArrayMap(t, cfg)
	pipelinecheck.RequireNoImperativeLoop(t, cfg)
	pipelinecheck.RequireNoImperativeBranch(t, cfg)
	pipelinecheck.RequireNoDirectApply(t, cfg)
	loomcheck.Require(t, loomcheck.Config{
		Roots:               []string{"."},
		StrictAllPredicates: true,
	})
}

func TestStockCountsCacheInterceptorInvalidatesForAllMutationMethods(t *testing.T) {
	t.Parallel()

	mutations := []string{
		stock.StockService_CreateStrain_FullMethodName,
		stock.StockService_CreatePlasmid_FullMethodName,
		stock.StockService_LoadStrain_FullMethodName,
		stock.StockService_LoadPlasmid_FullMethodName,
		stock.StockService_UpdateStrain_FullMethodName,
		stock.StockService_UpdatePlasmid_FullMethodName,
		stock.StockService_RemoveStock_FullMethodName,
	}

	for _, method := range mutations {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			assert := require.New(t)

			store := cache.NewMemoryStore()
			t.Cleanup(func() { _ = store.Close() })

			interceptor := StockCountsCacheInterceptor(store, DefaultStockCountsCacheTTL)
			var countCalls atomic.Int32
			countHandler := func(_ context.Context, _ any) (any, error) {
				n := int64(countCalls.Add(1))
				return sampleStockCounts(10*n, 5*n), nil
			}
			countInfo := &grpc.UnaryServerInfo{
				FullMethod: stock.StockService_CountStocks_FullMethodName,
			}

			_, err := interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
			assert.NoError(err)
			assert.Equal(int32(1), countCalls.Load())

			_, err = interceptor(
				context.Background(),
				&stock.StockId{Id: testStockID},
				&grpc.UnaryServerInfo{FullMethod: method},
				func(_ context.Context, _ any) (any, error) {
					return &empty.Empty{}, nil
				},
			)
			assert.NoError(err)

			_, err = interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
			assert.NoError(err)
			assert.Equal(int32(2), countCalls.Load())
		})
	}
}

func TestStockCountsCacheInterceptorPreservesHandlerError(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	store := cache.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	interceptor := StockCountsCacheInterceptor(store, DefaultStockCountsCacheTTL)
	var countCalls atomic.Int32
	countHandler := func(_ context.Context, _ any) (any, error) {
		n := int64(countCalls.Add(1))
		return sampleStockCounts(10*n, 5*n), nil
	}
	countInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_CountStocks_FullMethodName,
	}

	_, err := interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)

	sentinel := errors.New("validation failed")
	_, err = interceptor(
		context.Background(),
		&stock.NewStrain{},
		&grpc.UnaryServerInfo{
			FullMethod: stock.StockService_CreateStrain_FullMethodName,
		},
		func(_ context.Context, _ any) (any, error) {
			return nil, sentinel
		},
	)
	assert.ErrorIs(err, sentinel)

	_, err = interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	assert.Equal(int32(1), countCalls.Load())
}

func TestStockCountsCacheInterceptorIsolatesCacheHitPointers(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	store := cache.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	interceptor := StockCountsCacheInterceptor(store, DefaultStockCountsCacheTTL)
	var countCalls atomic.Int32
	countHandler := func(_ context.Context, _ any) (any, error) {
		countCalls.Add(1)
		return sampleStockCounts(10, 5), nil
	}
	countInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_CountStocks_FullMethodName,
	}

	resp1, err := interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	counts1, ok := resp1.(*stock.StockCounts)
	assert.True(ok)

	counts1.Strains.Total = 999
	counts1.Strains.Subtypes[0].Name = "tampered"
	counts1.Strains.Subtypes[0].Count = -1
	counts1.Plasmids = nil

	resp2, err := interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	counts2, ok := resp2.(*stock.StockCounts)
	assert.True(ok)
	assert.NotSame(counts1, counts2)
	assert.Equal(int64(10), counts2.GetStrains().GetTotal())
	assert.Equal("regular", counts2.GetStrains().GetSubtypes()[0].GetName())
	assert.Equal(int64(10), counts2.GetStrains().GetSubtypes()[0].GetCount())
	assert.NotNil(counts2.GetPlasmids())
	assert.Equal(int64(5), counts2.GetPlasmids().GetTotal())
	assert.Equal(int32(1), countCalls.Load())
}

func TestStockCountsCacheInterceptorExpiresCachedResponse(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	store := cache.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	interceptor := StockCountsCacheInterceptor(store, 20*time.Millisecond)
	var countCalls atomic.Int32
	countHandler := func(_ context.Context, _ any) (any, error) {
		n := int64(countCalls.Add(1))
		return sampleStockCounts(10*n, 5*n), nil
	}
	countInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_CountStocks_FullMethodName,
	}

	_, err := interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	assert.Equal(int32(1), countCalls.Load())

	time.Sleep(50 * time.Millisecond)

	_, err = interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	assert.Equal(int32(2), countCalls.Load())
}

func TestStockCountsCacheInterceptorFailsOpenOnSetError(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	inner := cache.NewMemoryStore()
	t.Cleanup(func() { _ = inner.Close() })

	interceptor := StockCountsCacheInterceptor(setFailingStore{inner: inner}, DefaultStockCountsCacheTTL)
	var countCalls atomic.Int32
	countHandler := func(_ context.Context, _ any) (any, error) {
		n := int64(countCalls.Add(1))
		return sampleStockCounts(7*n, 3*n), nil
	}
	countInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_CountStocks_FullMethodName,
	}

	resp1, err := interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	counts1, ok := resp1.(*stock.StockCounts)
	assert.True(ok)
	assert.Equal(int64(7), counts1.GetStrains().GetTotal())

	resp2, err := interceptor(context.Background(), &empty.Empty{}, countInfo, countHandler)
	assert.NoError(err)
	counts2, ok := resp2.(*stock.StockCounts)
	assert.True(ok)
	assert.Equal(int64(14), counts2.GetStrains().GetTotal())
	assert.Equal(int32(2), countCalls.Load())
}
