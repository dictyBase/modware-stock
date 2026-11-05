package middleware

import (
	"context"
	"errors"
	"strconv"
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

const (
	testStockID   = "DBS1"
	testPlasmidID = "DBP1"
)

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

func sampleStrain(id, label string) *stock.Strain {
	return &stock.Strain{
		Data: &stock.Strain_Data{
			Type: "strain",
			Id:   id,
			Attributes: &stock.StrainAttributes{
				Label:   label,
				Species: "Dictyostelium discoideum",
				Names:   []string{"name-" + label},
			},
		},
	}
}

func samplePlasmid(id, name string) *stock.Plasmid {
	return &stock.Plasmid{
		Data: &stock.Plasmid_Data{
			Type: "plasmid",
			Id:   id,
			Attributes: &stock.PlasmidAttributes{
				Name:    name,
				Summary: "summary-" + name,
				Genes:   []string{"gene-" + name},
			},
		},
	}
}

func TestGetStrainAndGetPlasmidCacheInterceptorWithMemoryStore(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	store := cache.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	interceptor := StockCountsCacheInterceptor(store, DefaultStockCountsCacheTTL)
	var strainCalls atomic.Int32
	var plasmidCalls atomic.Int32

	strainInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_GetStrain_FullMethodName,
	}
	plasmidInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_GetPlasmid_FullMethodName,
	}

	strainHandler := func(_ context.Context, req any) (any, error) {
		n := strainCalls.Add(1)
		id := req.(*stock.StockId).GetId()
		return sampleStrain(id, "ax4-"+strconv.Itoa(int(n))), nil
	}
	plasmidHandler := func(_ context.Context, req any) (any, error) {
		n := plasmidCalls.Add(1)
		id := req.(*stock.StockId).GetId()
		return samplePlasmid(id, "pDNeo2-"+strconv.Itoa(int(n))), nil
	}

	resp1, err := interceptor(context.Background(), &stock.StockId{Id: testStockID}, strainInfo, strainHandler)
	assert.NoError(err)
	assert.Equal(int32(1), strainCalls.Load())
	st1, ok := resp1.(*stock.Strain)
	assert.True(ok)
	assert.Equal("ax4-1", st1.GetData().GetAttributes().GetLabel())

	// Mutating the returned pointer must not corrupt the cached value.
	st1.Data.Attributes.Label = "tampered"
	st1.Data.Attributes.Names[0] = "tampered-name"

	resp2, err := interceptor(context.Background(), &stock.StockId{Id: testStockID}, strainInfo, strainHandler)
	assert.NoError(err)
	assert.Equal(int32(1), strainCalls.Load())
	st2, ok := resp2.(*stock.Strain)
	assert.True(ok)
	assert.NotSame(st1, st2)
	assert.Equal("ax4-1", st2.GetData().GetAttributes().GetLabel())
	assert.Equal("name-ax4-1", st2.GetData().GetAttributes().GetNames()[0])

	// Different strain ID has its own cache key.
	_, err = interceptor(context.Background(), &stock.StockId{Id: "DBS2"}, strainInfo, strainHandler)
	assert.NoError(err)
	assert.Equal(int32(2), strainCalls.Load())

	// GetPlasmid caches independently.
	pResp1, err := interceptor(context.Background(), &stock.StockId{Id: testPlasmidID}, plasmidInfo, plasmidHandler)
	assert.NoError(err)
	assert.Equal(int32(1), plasmidCalls.Load())
	pl1, ok := pResp1.(*stock.Plasmid)
	assert.True(ok)
	assert.Equal("pDNeo2-1", pl1.GetData().GetAttributes().GetName())

	pl1.Data.Attributes.Name = "tampered-plasmid"
	pl1.Data.Attributes.Genes[0] = "tampered-gene"

	pResp2, err := interceptor(context.Background(), &stock.StockId{Id: testPlasmidID}, plasmidInfo, plasmidHandler)
	assert.NoError(err)
	assert.Equal(int32(1), plasmidCalls.Load())
	pl2, ok := pResp2.(*stock.Plasmid)
	assert.True(ok)
	assert.NotSame(pl1, pl2)
	assert.Equal("pDNeo2-1", pl2.GetData().GetAttributes().GetName())
	assert.Equal("gene-pDNeo2-1", pl2.GetData().GetAttributes().GetGenes()[0])

	// UpdateStrain for testStockID invalidates only that strain, not testPlasmidID or DBS2.
	updateStrainInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_UpdateStrain_FullMethodName,
	}
	_, err = interceptor(
		context.Background(),
		&stock.StrainUpdate{Data: &stock.StrainUpdate_Data{Id: testStockID}},
		updateStrainInfo,
		func(_ context.Context, _ any) (any, error) {
			return &stock.Strain{}, nil
		},
	)
	assert.NoError(err)

	// DBS2 is still cached.
	_, err = interceptor(context.Background(), &stock.StockId{Id: "DBS2"}, strainInfo, strainHandler)
	assert.NoError(err)
	assert.Equal(int32(2), strainCalls.Load())

	// testStockID was invalidated and re-fetches.
	resp3, err := interceptor(context.Background(), &stock.StockId{Id: testStockID}, strainInfo, strainHandler)
	assert.NoError(err)
	assert.Equal(int32(3), strainCalls.Load())
	assert.Equal("ax4-3", resp3.(*stock.Strain).GetData().GetAttributes().GetLabel())

	// testPlasmidID is still cached after UpdateStrain.
	_, err = interceptor(context.Background(), &stock.StockId{Id: testPlasmidID}, plasmidInfo, plasmidHandler)
	assert.NoError(err)
	assert.Equal(int32(1), plasmidCalls.Load())

	// UpdatePlasmid for testPlasmidID invalidates testPlasmidID.
	updatePlasmidInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_UpdatePlasmid_FullMethodName,
	}
	_, err = interceptor(
		context.Background(),
		&stock.PlasmidUpdate{Data: &stock.PlasmidUpdate_Data{Id: testPlasmidID}},
		updatePlasmidInfo,
		func(_ context.Context, _ any) (any, error) {
			return &stock.Plasmid{}, nil
		},
	)
	assert.NoError(err)

	pResp3, err := interceptor(context.Background(), &stock.StockId{Id: testPlasmidID}, plasmidInfo, plasmidHandler)
	assert.NoError(err)
	assert.Equal(int32(2), plasmidCalls.Load())
	assert.Equal("pDNeo2-2", pResp3.(*stock.Plasmid).GetData().GetAttributes().GetName())
}

func TestGetStrainAndGetPlasmidCacheInterceptorWithRedisStore(t *testing.T) {
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
	var strainCalls atomic.Int32
	var plasmidCalls atomic.Int32

	strainInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_GetStrain_FullMethodName,
	}
	plasmidInfo := &grpc.UnaryServerInfo{
		FullMethod: stock.StockService_GetPlasmid_FullMethodName,
	}
	strainHandler := func(_ context.Context, _ any) (any, error) {
		n := strainCalls.Add(1)
		return sampleStrain(testStockID, "ax4-"+strconv.Itoa(int(n))), nil
	}
	plasmidHandler := func(_ context.Context, _ any) (any, error) {
		n := plasmidCalls.Add(1)
		return samplePlasmid(testPlasmidID, "pDNeo2-"+strconv.Itoa(int(n))), nil
	}

	_, err = interceptor(context.Background(), &stock.StockId{Id: testStockID}, strainInfo, strainHandler)
	assert.NoError(err)
	assert.Equal(int32(1), strainCalls.Load())
	strainKey := StrainCacheKeyPrefix + testStockID
	assert.Equal(DefaultStockCountsCacheTTL, mr.TTL(strainKey))

	_ = mr.Set(strainKey, "not-valid-protobuf-\xff\xfe")
	sResp, err := interceptor(context.Background(), &stock.StockId{Id: testStockID}, strainInfo, strainHandler)
	assert.NoError(err)
	assert.Equal(int32(2), strainCalls.Load())
	assert.Equal("ax4-2", sResp.(*stock.Strain).GetData().GetAttributes().GetLabel())

	_, err = interceptor(context.Background(), &stock.StockId{Id: testPlasmidID}, plasmidInfo, plasmidHandler)
	assert.NoError(err)
	assert.Equal(int32(1), plasmidCalls.Load())
	plasmidKey := PlasmidCacheKeyPrefix + testPlasmidID
	assert.Equal(DefaultStockCountsCacheTTL, mr.TTL(plasmidKey))

	_ = mr.Set(plasmidKey, "not-valid-protobuf-\xff\xfe")
	pResp, err := interceptor(context.Background(), &stock.StockId{Id: testPlasmidID}, plasmidInfo, plasmidHandler)
	assert.NoError(err)
	assert.Equal(int32(2), plasmidCalls.Load())
	assert.Equal("pDNeo2-2", pResp.(*stock.Plasmid).GetData().GetAttributes().GetName())
}

func TestGetStrainAndGetPlasmidInvalidationOnLoadAndRemove(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	store := cache.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	interceptor := StockCountsCacheInterceptor(store, DefaultStockCountsCacheTTL)
	var strainCalls atomic.Int32
	var plasmidCalls atomic.Int32

	strainInfo := &grpc.UnaryServerInfo{FullMethod: stock.StockService_GetStrain_FullMethodName}
	plasmidInfo := &grpc.UnaryServerInfo{FullMethod: stock.StockService_GetPlasmid_FullMethodName}
	strainHandler := func(_ context.Context, _ any) (any, error) {
		strainCalls.Add(1)
		return sampleStrain(testStockID, "ax4"), nil
	}
	plasmidHandler := func(_ context.Context, _ any) (any, error) {
		plasmidCalls.Add(1)
		return samplePlasmid(testPlasmidID, "pDNeo2"), nil
	}

	_, err := interceptor(context.Background(), &stock.StockId{Id: testStockID}, strainInfo, strainHandler)
	assert.NoError(err)
	_, err = interceptor(context.Background(), &stock.StockId{Id: testPlasmidID}, plasmidInfo, plasmidHandler)
	assert.NoError(err)
	assert.Equal(int32(1), strainCalls.Load())
	assert.Equal(int32(1), plasmidCalls.Load())

	// Missing ID on GetStrain/GetPlasmid bypasses cache and forwards to handler.
	_, err = interceptor(context.Background(), &stock.StockId{Id: ""}, strainInfo, strainHandler)
	assert.NoError(err)
	assert.Equal(int32(2), strainCalls.Load())

	_, err = interceptor(context.Background(), &stock.StockId{Id: ""}, plasmidInfo, plasmidHandler)
	assert.NoError(err)
	assert.Equal(int32(2), plasmidCalls.Load())

	// LoadStrain invalidates strain key.
	_, err = interceptor(
		context.Background(),
		&stock.ExistingStrain{Data: &stock.ExistingStrain_Data{Id: testStockID}},
		&grpc.UnaryServerInfo{FullMethod: stock.StockService_LoadStrain_FullMethodName},
		func(_ context.Context, _ any) (any, error) { return &stock.Strain{}, nil },
	)
	assert.NoError(err)

	_, err = interceptor(context.Background(), &stock.StockId{Id: testStockID}, strainInfo, strainHandler)
	assert.NoError(err)
	assert.Equal(int32(3), strainCalls.Load())

	// LoadPlasmid invalidates plasmid key.
	_, err = interceptor(
		context.Background(),
		&stock.ExistingPlasmid{Data: &stock.ExistingPlasmid_Data{Id: testPlasmidID}},
		&grpc.UnaryServerInfo{FullMethod: stock.StockService_LoadPlasmid_FullMethodName},
		func(_ context.Context, _ any) (any, error) { return &stock.Plasmid{}, nil },
	)
	assert.NoError(err)

	_, err = interceptor(context.Background(), &stock.StockId{Id: testPlasmidID}, plasmidInfo, plasmidHandler)
	assert.NoError(err)
	assert.Equal(int32(3), plasmidCalls.Load())

	// RemoveStock invalidates both strain and plasmid candidate keys for the given ID.
	_, err = interceptor(
		context.Background(),
		&stock.StockId{Id: testStockID},
		&grpc.UnaryServerInfo{FullMethod: stock.StockService_RemoveStock_FullMethodName},
		func(_ context.Context, _ any) (any, error) { return &empty.Empty{}, nil },
	)
	assert.NoError(err)

	_, err = interceptor(context.Background(), &stock.StockId{Id: testStockID}, strainInfo, strainHandler)
	assert.NoError(err)
	assert.Equal(int32(4), strainCalls.Load())

	_, err = interceptor(
		context.Background(),
		&stock.StockId{Id: testPlasmidID},
		&grpc.UnaryServerInfo{FullMethod: stock.StockService_RemoveStock_FullMethodName},
		func(_ context.Context, _ any) (any, error) { return &empty.Empty{}, nil },
	)
	assert.NoError(err)

	_, err = interceptor(context.Background(), &stock.StockId{Id: testPlasmidID}, plasmidInfo, plasmidHandler)
	assert.NoError(err)
	assert.Equal(int32(4), plasmidCalls.Load())
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
