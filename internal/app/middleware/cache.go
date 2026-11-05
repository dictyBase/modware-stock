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
		ctx        context.Context
		req        any
		info       *grpc.UnaryServerInfo
		handler    grpc.UnaryHandler
		store      cache.Store
		ttl        time.Duration
		key        string
		newMessage func() proto.Message
		raw        []byte
		resp       any
	}

	unaryResult struct {
		resp any
		err  error
	}

	readCacheSpec struct {
		keyFromReq func(any) O.Option[string]
		newMessage func() proto.Message
	}
)

const (
	// StockCountsCacheKey is the cache key used for the CountStocks RPC response.
	StockCountsCacheKey = "stock:counts:v1"
	// StrainCacheKeyPrefix is the cache key prefix used for GetStrain RPC responses.
	StrainCacheKeyPrefix = "stock:strain:v1:"
	// PlasmidCacheKeyPrefix is the cache key prefix used for GetPlasmid RPC responses.
	PlasmidCacheKeyPrefix = "stock:plasmid:v1:"
	// DefaultStockCountsCacheTTL is the default 1-day (24-hour) TTL for cached stock responses.
	DefaultStockCountsCacheTTL = 24 * time.Hour
)

var (
	callKeyLens = L.MakeLens(
		func(c unaryCall) string { return c.key },
		func(c unaryCall, key string) unaryCall {
			c.key = key
			return c
		},
	)
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

	hasNonEmptyID      = predord.MinStrLen(1)
	nonEmptyIDOpt      = O.FromPredicate(hasNonEmptyID)
	stockIDFromMessage = F.Flow3(
		O.InstanceOf[*stock.StockId],
		O.Map((*stock.StockId).GetId),
		O.Chain(nonEmptyIDOpt),
	)
	existingStrainIDFromMessage = F.Flow4(
		O.InstanceOf[*stock.ExistingStrain],
		O.Map((*stock.ExistingStrain).GetData),
		O.Map((*stock.ExistingStrain_Data).GetId),
		O.Chain(nonEmptyIDOpt),
	)
	existingPlasmidIDFromMessage = F.Flow4(
		O.InstanceOf[*stock.ExistingPlasmid],
		O.Map((*stock.ExistingPlasmid).GetData),
		O.Map((*stock.ExistingPlasmid_Data).GetId),
		O.Chain(nonEmptyIDOpt),
	)
	strainUpdateIDFromMessage = F.Flow4(
		O.InstanceOf[*stock.StrainUpdate],
		O.Map((*stock.StrainUpdate).GetData),
		O.Map((*stock.StrainUpdate_Data).GetId),
		O.Chain(nonEmptyIDOpt),
	)
	plasmidUpdateIDFromMessage = F.Flow4(
		O.InstanceOf[*stock.PlasmidUpdate],
		O.Map((*stock.PlasmidUpdate).GetData),
		O.Map((*stock.PlasmidUpdate_Data).GetId),
		O.Chain(nonEmptyIDOpt),
	)

	strainKeyFromStockID = F.Flow2(
		stockIDFromMessage,
		O.Map(strainCacheKey),
	)
	plasmidKeyFromStockID = F.Flow2(
		stockIDFromMessage,
		O.Map(plasmidCacheKey),
	)

	readCacheSpecs = R.Record[string, readCacheSpec]{
		stock.StockService_CountStocks_FullMethodName: {
			keyFromReq: F.Constant1[any](O.Some(StockCountsCacheKey)),
			newMessage: func() proto.Message { return &stock.StockCounts{} },
		},
		stock.StockService_GetStrain_FullMethodName: {
			keyFromReq: strainKeyFromStockID,
			newMessage: func() proto.Message { return &stock.Strain{} },
		},
		stock.StockService_GetPlasmid_FullMethodName: {
			keyFromReq: plasmidKeyFromStockID,
			newMessage: func() proto.Message { return &stock.Plasmid{} },
		},
	}

	mutationEntityKeys = R.Record[string, func(any) []string]{
		stock.StockService_CreateStrain_FullMethodName:  F.Constant1[any]([]string(nil)),
		stock.StockService_CreatePlasmid_FullMethodName: F.Constant1[any]([]string(nil)),
		stock.StockService_LoadStrain_FullMethodName: F.Flow3(
			existingStrainIDFromMessage,
			O.Map(strainCacheKey),
			O.Fold(emptyKeySlice, singleKeySlice),
		),
		stock.StockService_LoadPlasmid_FullMethodName: F.Flow3(
			existingPlasmidIDFromMessage,
			O.Map(plasmidCacheKey),
			O.Fold(emptyKeySlice, singleKeySlice),
		),
		stock.StockService_UpdateStrain_FullMethodName: F.Flow3(
			strainUpdateIDFromMessage,
			O.Map(strainCacheKey),
			O.Fold(emptyKeySlice, singleKeySlice),
		),
		stock.StockService_UpdatePlasmid_FullMethodName: F.Flow3(
			plasmidUpdateIDFromMessage,
			O.Map(plasmidCacheKey),
			O.Fold(emptyKeySlice, singleKeySlice),
		),
		stock.StockService_RemoveStock_FullMethodName: F.Flow2(
			stockIDFromMessage,
			O.Fold(emptyKeySlice, func(id string) []string {
				return []string{strainCacheKey(id), plasmidCacheKey(id)}
			}),
		),
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
	constNoneBytes      = F.Constant1[[]struct{}](emptyBytesOption)
	emptyBytesOptionIOE = IOE.Memoize(IOE.Of[error](emptyBytesOption))
	recoverNoneBytes    = F.Constant1[error](emptyBytesOptionIOE)
	constEmptyStruct    = F.Constant1[error](IOE.Of[error](struct{}{}))

	hasCachedPayload = F.Flow2(
		callCachedRawLens.Get,
		O.IsSome[[]byte],
	)
	resolveCachedReadCall = F.Pipe1(
		hasCachedPayload,
		P.Fold(fetchAndCacheReadCall, decodeCachedReadCall),
	)

	isCachedReadMethod = F.Bind2nd(R.Has[string], readCacheSpecs)
	isCachedReadCall   = F.Pipe1(isCachedReadMethod, P.ContraMap(callFullMethod))
	isMutationMethod   = F.Bind2nd(R.Has[string], mutationEntityKeys)
	isMutationCall     = F.Pipe1(isMutationMethod, P.ContraMap(callFullMethod))
	dispatchNonRead    = F.Pipe1(
		isMutationCall,
		P.Fold(invokeUnaryHandler, handleMutationCall),
	)
)

func strainCacheKey(id string) string {
	return StrainCacheKeyPrefix + id
}

func plasmidCacheKey(id string) string {
	return PlasmidCacheKeyPrefix + id
}

func singleKeySlice(key string) []string {
	return []string{key}
}

func emptyKeySlice() []string {
	return nil
}

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
	extraKeys := F.Pipe2(
		mutationEntityKeys,
		R.Lookup[func(any) []string](c.info.FullMethod),
		O.Fold(
			emptyKeySlice,
			func(extract func(any) []string) []string {
				return extract(c.req)
			},
		),
	)
	keys := append([]string{StockCountsCacheKey}, extraKeys...)
	return F.Pipe3(
		keys,
		IOE.TraverseArray(func(key string) IOE.IOEither[error, struct{}] {
			return F.Pipe1(
				c.store.Delete(c.ctx, key),
				IOE.OrElse(constEmptyStruct),
			)
		}),
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

func lookupCachedBytes(
	c unaryCall,
) IOE.IOEither[error, O.Option[[]byte]] {
	return F.Pipe1(
		c.store.Get(c.ctx, callKeyLens.Get(c)),
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
		c.store.Set(c.ctx, callKeyLens.Get(c), raw, c.ttl),
		IOE.Map[error](F.Constant1[struct{}](O.Some(raw))),
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

func fetchAndCacheReadCall(
	c unaryCall,
) IOE.IOEither[error, unaryCall] {
	return F.Pipe2(
		IOE.Of[error](c),
		IOE.Bind(callRespLens.Set, invokeUnaryHandler),
		IOE.Bind(callCachedRawLens.Set, persistCallResp),
	)
}

func unmarshalCachedResponse(c unaryCall) IOE.IOEither[error, any] {
	msg := c.newMessage()
	return F.Pipe1(
		IOE.TryCatchError(func() (struct{}, error) {
			return struct{}{}, proto.Unmarshal(callRawLens.Get(c), msg)
		}),
		IOE.Map[error](F.Constant1[struct{}](any(msg))),
	)
}

func decodeCachedReadCall(
	c unaryCall,
) IOE.IOEither[error, unaryCall] {
	return F.Pipe2(
		IOE.Of[error](c),
		IOE.Bind(callRespLens.Set, unmarshalCachedResponse),
		IOE.OrElse(
			F.Constant1[error](fetchAndCacheReadCall(c)),
		),
	)
}

func handleCachedReadCall(c unaryCall) IOE.IOEither[error, any] {
	return F.Pipe2(
		readCacheSpecs,
		R.Lookup[readCacheSpec](c.info.FullMethod),
		O.Fold(
			func() IOE.IOEither[error, any] {
				return invokeUnaryHandler(c)
			},
			func(spec readCacheSpec) IOE.IOEither[error, any] {
				return F.Pipe2(
					c.req,
					spec.keyFromReq,
					O.Fold(
						func() IOE.IOEither[error, any] {
							return invokeUnaryHandler(c)
						},
						func(key string) IOE.IOEither[error, any] {
							c.key = key
							c.newMessage = spec.newMessage
							return F.Pipe3(
								IOE.Of[error](c),
								IOE.Bind(callCachedRawLens.Set, lookupCachedBytes),
								IOE.Chain(resolveCachedReadCall),
								IOE.Map[error](callRespLens.Get),
							)
						},
					),
				)
			},
		),
	)
}

// StockCountsCacheInterceptor returns a grpc.UnaryServerInterceptor that
// caches CountStocks, GetStrain, and GetPlasmid responses for ttl (or
// DefaultStockCountsCacheTTL when ttl <= 0) and invalidates cached entries
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
				isCachedReadCall,
				P.Fold(dispatchNonRead, handleCachedReadCall),
			),
			ioeutils.ToEither[error, any],
			E.Fold(
				func(err error) unaryResult { return unaryResult{err: err} },
				func(resp any) unaryResult { return unaryResult{resp: resp} },
			),
		)
		return res.resp, res.err
	}
}
