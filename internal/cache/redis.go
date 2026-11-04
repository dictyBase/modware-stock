package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	EQ "github.com/IBM/fp-go/v2/eq"
	F "github.com/IBM/fp-go/v2/function"
	IOE "github.com/IBM/fp-go/v2/ioeither"
	O "github.com/IBM/fp-go/v2/option"
	P "github.com/IBM/fp-go/v2/predicate"
	rds "github.com/redis/go-redis/v9"
)

// RedisStore is a Redis-backed Store implementation using go-redis/v9.
type RedisStore struct {
	client *rds.Client
}

// epochKeySuffix separates a cache key from its invalidation-epoch key.
const epochKeySuffix = ":epoch"

var (
	emptyBytesOption    = O.None[[]byte]()
	emptyBytesOptionIOE = IOE.Memoize(IOE.Of[error](emptyBytesOption))
	recoverNoneBytes    = F.Constant1[error](emptyBytesOptionIOE)
	isRedisNilError     = F.Bind2nd[error, error](errors.Is, rds.Nil)
	handleRedisGetError = F.Pipe1(
		isRedisNilError,
		P.Fold(
			IOE.Left[O.Option[[]byte], error],
			recoverNoneBytes,
		),
	)
	constEmptyStructFromInt = F.Constant1[int64](struct{}{})

	int64Eq              = EQ.FromStrictEquals[int64]()
	isEpochWriteAccepted = F.Pipe1(1, EQ.Equals(int64Eq))

	emptyEpochIOE         = IOE.Memoize(IOE.Of[error](int64(0)))
	recoverZeroEpoch      = F.Constant1[error](emptyEpochIOE)
	handleRedisEpochError = F.Pipe1(
		isRedisNilError,
		P.Fold(
			IOE.Left[int64, error],
			recoverZeroEpoch,
		),
	)

	// setIfEpochScript atomically writes the cached value only when the
	// key's invalidation epoch still equals the captured epoch.
	setIfEpochScript = rds.NewScript(`
local cur = tonumber(redis.call('GET', KEYS[2])) or 0
if cur == tonumber(ARGV[3]) then
	redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
	return 1
end
return 0
`)
	// bumpEpochScript deletes the cached value and advances its
	// invalidation epoch in one atomic step.
	bumpEpochScript = rds.NewScript(`
redis.call('DEL', KEYS[1])
return redis.call('INCR', KEYS[2])
`)
)

// epochKeyFor derives the invalidation-epoch key for a cache key.
func epochKeyFor(key string) string {
	return key + epochKeySuffix
}

func wrapRedisPingError(addr string, err error) error {
	return fmt.Errorf("error pinging redis at %s: %w", addr, err)
}

// NewRedisStore connects to Redis at addr, verifies reachability with
// Ping, and returns the initialized Store.
func NewRedisStore(addr string) IOE.IOEither[error, Store] {
	client := rds.NewClient(&rds.Options{Addr: addr})
	return F.Pipe3(
		IOE.TryCatchError(func() (string, error) {
			return client.Ping(context.Background()).Result()
		}),
		IOE.OrElse(func(err error) IOE.IOEither[error, string] {
			_ = client.Close()
			return IOE.Left[string](err)
		}),
		IOE.MapLeft[string](F.Bind1st(wrapRedisPingError, addr)),
		IOE.Map[error](
			F.Constant1[string](Store(&RedisStore{client: client})),
		),
	)
}

// Get retrieves the byte slice stored at key in Redis, mapping
// redis.Nil to O.None and propagating transport errors on Left.
func (r *RedisStore) Get(
	ctx context.Context,
	key string,
) IOE.IOEither[error, O.Option[[]byte]] {
	return F.Pipe2(
		IOE.TryCatchError(func() ([]byte, error) {
			return r.client.Get(ctx, key).Bytes()
		}),
		IOE.Map[error](O.Some[[]byte]),
		IOE.OrElse(handleRedisGetError),
	)
}

// Epoch returns the current invalidation epoch for key, treating a
// missing epoch key as zero.
func (r *RedisStore) Epoch(
	ctx context.Context,
	key string,
) IOE.IOEither[error, int64] {
	return F.Pipe1(
		IOE.TryCatchError(func() (int64, error) {
			return r.client.Get(ctx, epochKeyFor(key)).Int64()
		}),
		IOE.OrElse(handleRedisEpochError),
	)
}

// SetIfEpoch writes val at key in Redis with the given TTL only when
// the key's invalidation epoch still equals epoch. The
// compare-and-write runs atomically in a server-side script so it also
// holds across service replicas that share the Redis instance.
func (r *RedisStore) SetIfEpoch(
	ctx context.Context,
	key string,
	val []byte,
	ttl time.Duration,
	epoch int64,
) IOE.IOEither[error, bool] {
	return F.Pipe1(
		IOE.TryCatchError(func() (int64, error) {
			return setIfEpochScript.Run(
				ctx,
				r.client,
				[]string{key, epochKeyFor(key)},
				val,
				ttl.Milliseconds(),
				epoch,
			).Int64()
		}),
		IOE.Map[error](isEpochWriteAccepted),
	)
}

// Delete removes key and advances its invalidation epoch atomically.
func (r *RedisStore) Delete(
	ctx context.Context,
	key string,
) IOE.IOEither[error, struct{}] {
	return F.Pipe1(
		IOE.TryCatchError(func() (int64, error) {
			return bumpEpochScript.Run(
				ctx,
				r.client,
				[]string{key, epochKeyFor(key)},
			).Int64()
		}),
		IOE.Map[error](constEmptyStructFromInt),
	)
}

// Close closes the underlying Redis client connection.
func (r *RedisStore) Close() error {
	return r.client.Close()
}
