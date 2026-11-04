// Package middleware provides gRPC server interceptors for the stock
// service.
package middleware

import (
	"context"
	"errors"
	"time"

	E "github.com/IBM/fp-go/v2/either"
	F "github.com/IBM/fp-go/v2/function"
	IOE "github.com/IBM/fp-go/v2/ioeither"
	L "github.com/IBM/fp-go/v2/optics/lens"
	LO "github.com/IBM/fp-go/v2/optics/lens/option"
	O "github.com/IBM/fp-go/v2/option"
	ORD "github.com/IBM/fp-go/v2/ord"
	P "github.com/IBM/fp-go/v2/predicate"
	R "github.com/IBM/fp-go/v2/record"
	ioeutils "github.com/dictyBase/fp-go-loom/ioeitherutils"
	predbytes "github.com/dictyBase/fp-go-loom/predicate/bytes"
	predord "github.com/dictyBase/fp-go-loom/predicate/ord"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/cache"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type (
	unaryCall struct {
		ctx     context.Context
		req     any
		info    *grpc.UnaryServerInfo
		handler grpc.UnaryHandler
		store   cache.Store
		ttl     time.Duration
		raw     []byte
		resp    any
		epoch   int64
	}

	unaryResult struct {
		resp any
		err  error
	}
)

const (
	// StockCountsCacheKey is the cache key used for the CountStocks RPC response.
	StockCountsCacheKey = "stock:counts:v1"
	// DefaultStockCountsCacheTTL is the default 1-month (30-day) TTL for cached stock counts.
	DefaultStockCountsCacheTTL = 30 * 24 * time.Hour
)

var (
	callRawLens = L.MakeLens(
		func(c unaryCall) []byte { return c.raw },
		func(c unaryCall, raw []byte) unaryCall {
			c.raw = raw
			return c
		},
	)
	callCachedRawLens = F.Pipe1(
		callRawLens,
		LO.FromPredicate[unaryCall](predbytes.HasPositiveLen, nil),
	)
	callRespLens = L.MakeLens(
		func(c unaryCall) any { return c.resp },
		func(c unaryCall, resp any) unaryCall {
			c.resp = resp
			return c
		},
	)
	callEpochLens = L.MakeLens(
		func(c unaryCall) int64 { return c.epoch },
		func(c unaryCall, epoch int64) unaryCall {
			c.epoch = epoch
			return c
		},
	)

	mutationMethods = R.Record[string, bool]{
		stock.StockService_CreateStrain_FullMethodName:  true,
		stock.StockService_CreatePlasmid_FullMethodName: true,
		stock.StockService_LoadStrain_FullMethodName:    true,
		stock.StockService_LoadPlasmid_FullMethodName:   true,
		stock.StockService_UpdateStrain_FullMethodName:  true,
		stock.StockService_UpdatePlasmid_FullMethodName: true,
		stock.StockService_RemoveStock_FullMethodName:   true,
	}

	durationOrd   = ORD.FromStrictCompare[time.Duration]()
	isPositiveTTL = F.Pipe1(time.Duration(0), ORD.Gt(durationOrd))
	normalizeTTL  = F.Pipe1(
		isPositiveTTL,
		P.Fold(
			F.Constant1[time.Duration](DefaultStockCountsCacheTTL),
			F.Identity[time.Duration],
		),
	)

	errNotProtoMessage = errors.New("response does not implement proto.Message")
	toProtoMessage     = E.ToType[proto.Message](
		F.Constant1[any](errNotProtoMessage),
	)
	marshalProtoMessage = E.Eitherize1(proto.Marshal)

	emptyBytesOption    = O.None[[]byte]()
	constNoneBytes      = F.Constant1[struct{}](emptyBytesOption)
	emptyBytesOptionIOE = IOE.Memoize(IOE.Of[error](emptyBytesOption))
	recoverNoneBytes    = F.Constant1[error](emptyBytesOptionIOE)
	emptyEpochIOE       = IOE.Memoize(IOE.Of[error](int64(0)))
	recoverZeroEpoch    = F.Constant1[error](emptyEpochIOE)

	hasCachedPayload = F.Flow2(
		callCachedRawLens.Get,
		O.IsSome[[]byte],
	)
	resolveCachedStockCounts = F.Pipe1(
		hasCachedPayload,
		P.Fold(fetchAndCacheStockCounts, decodeCachedStockCounts),
	)

	isCountStocksMethod  = predord.StrEq(stock.StockService_CountStocks_FullMethodName)
	isCountStocksCall    = F.Pipe1(isCountStocksMethod, P.ContraMap(callFullMethod))
	isMutationMethod     = F.Bind2nd(R.Has[string], mutationMethods)
	isMutationCall       = F.Pipe1(isMutationMethod, P.ContraMap(callFullMethod))
	dispatchNonCountCall = F.Pipe1(
		isMutationCall,
		P.Fold(invokeUnaryHandler, handleMutationCall),
	)
)

func callFullMethod(c unaryCall) string {
	return c.info.FullMethod
}

func invokeUnaryHandler(c unaryCall) IOE.IOEither[error, any] {
	return IOE.TryCatchError(func() (any, error) {
		return c.handler(c.ctx, c.req)
	})
}

func clearCachedRawInStore(
	c unaryCall,
) IOE.IOEither[error, O.Option[[]byte]] {
	return F.Pipe2(
		c.store.Delete(c.ctx, StockCountsCacheKey),
		IOE.Map[error](constNoneBytes),
		IOE.OrElse(recoverNoneBytes),
	)
}

func handleMutationCall(c unaryCall) IOE.IOEither[error, any] {
	return F.Pipe3(
		IOE.Of[error](c),
		IOE.Bind(callRespLens.Set, invokeUnaryHandler),
		IOE.Bind(callCachedRawLens.Set, clearCachedRawInStore),
		IOE.Map[error](callRespLens.Get),
	)
}

func lookupCacheEpoch(
	c unaryCall,
) IOE.IOEither[error, int64] {
	return F.Pipe1(
		c.store.Epoch(c.ctx, StockCountsCacheKey),
		IOE.OrElse(recoverZeroEpoch),
	)
}

func lookupCachedBytes(
	c unaryCall,
) IOE.IOEither[error, O.Option[[]byte]] {
	return F.Pipe1(
		c.store.Get(c.ctx, StockCountsCacheKey),
		IOE.OrElse(recoverNoneBytes),
	)
}

func marshalCallResp(c unaryCall) IOE.IOEither[error, []byte] {
	return F.Pipe2(
		IOE.Of[error](callRespLens.Get(c)),
		IOE.ChainEitherK(toProtoMessage),
		IOE.ChainEitherK(marshalProtoMessage),
	)
}

func writeCallRawToStore(
	c unaryCall,
) IOE.IOEither[error, O.Option[[]byte]] {
	raw := callRawLens.Get(c)
	return F.Pipe1(
		c.store.SetIfEpoch(c.ctx, StockCountsCacheKey, raw, c.ttl, c.epoch),
		IOE.Map[error](F.Constant1[bool](O.Some(raw))),
	)
}

func persistCallResp(
	c unaryCall,
) IOE.IOEither[error, O.Option[[]byte]] {
	return F.Pipe4(
		IOE.Of[error](c),
		IOE.Bind(callRawLens.Set, marshalCallResp),
		IOE.Bind(callCachedRawLens.Set, writeCallRawToStore),
		IOE.Map[error](callCachedRawLens.Get),
		IOE.OrElse(recoverNoneBytes),
	)
}

func fetchAndCacheStockCounts(
	c unaryCall,
) IOE.IOEither[error, unaryCall] {
	return F.Pipe2(
		IOE.Of[error](c),
		IOE.Bind(callRespLens.Set, invokeUnaryHandler),
		IOE.Bind(callCachedRawLens.Set, persistCallResp),
	)
}

func unmarshalStockCounts(c unaryCall) IOE.IOEither[error, any] {
	counts := &stock.StockCounts{}
	return F.Pipe1(
		IOE.TryCatchError(func() (struct{}, error) {
			return struct{}{}, proto.Unmarshal(callRawLens.Get(c), counts)
		}),
		IOE.Map[error](F.Constant1[struct{}](any(counts))),
	)
}

func decodeCachedStockCounts(
	c unaryCall,
) IOE.IOEither[error, unaryCall] {
	return F.Pipe2(
		IOE.Of[error](c),
		IOE.Bind(callRespLens.Set, unmarshalStockCounts),
		IOE.OrElse(
			F.Constant1[error](fetchAndCacheStockCounts(c)),
		),
	)
}

func handleCountStocksCall(c unaryCall) IOE.IOEither[error, any] {
	return F.Pipe4(
		IOE.Of[error](c),
		IOE.Bind(callEpochLens.Set, lookupCacheEpoch),
		IOE.Bind(callCachedRawLens.Set, lookupCachedBytes),
		IOE.Chain(resolveCachedStockCounts),
		IOE.Map[error](callRespLens.Get),
	)
}

func failedUnaryResult(err error) unaryResult {
	return unaryResult{err: err}
}

func successfulUnaryResult(resp any) unaryResult {
	return unaryResult{resp: resp}
}

// StockCountsCacheInterceptor returns a grpc.UnaryServerInterceptor that
// caches CountStocks responses under StockCountsCacheKey for ttl (or
// DefaultStockCountsCacheTTL when ttl <= 0) and invalidates the entry
// after successful stock mutation RPCs.
func StockCountsCacheInterceptor(
	store cache.Store,
	ttl time.Duration,
) grpc.UnaryServerInterceptor {
	effectiveTTL := normalizeTTL(ttl)
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		res := F.Pipe3(
			unaryCall{
				ctx:     ctx,
				req:     req,
				info:    info,
				handler: handler,
				store:   store,
				ttl:     effectiveTTL,
			},
			F.Pipe1(
				isCountStocksCall,
				P.Fold(dispatchNonCountCall, handleCountStocksCall),
			),
			ioeutils.ToEither[error, any],
			E.Fold(failedUnaryResult, successfulUnaryResult),
		)
		return res.resp, res.err
	}
}
