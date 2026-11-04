package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	constEmptyStruct        = F.Constant1[string](struct{}{})
	constEmptyStructFromInt = F.Constant1[int64](struct{}{})
)

func wrapRedisPingError(addr string, err error) error {
	return fmt.Errorf("error pinging redis at %s: %w", addr, err)
}

// NewRedisStore connects to Redis at addr, verifies reachability with
// Ping, and returns the initialized Store.
func NewRedisStore(addr string) IOE.IOEither[error, Store] {
	client := rds.NewClient(&rds.Options{Addr: addr})
	return F.Pipe2(
		IOE.TryCatchError(func() (string, error) {
			return client.Ping(context.Background()).Result()
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

// Set writes val at key in Redis with the given TTL.
func (r *RedisStore) Set(
	ctx context.Context,
	key string,
	val []byte,
	ttl time.Duration,
) IOE.IOEither[error, struct{}] {
	return F.Pipe1(
		IOE.TryCatchError(func() (string, error) {
			return r.client.Set(ctx, key, val, ttl).Result()
		}),
		IOE.Map[error](constEmptyStruct),
	)
}

// Delete removes key from Redis.
func (r *RedisStore) Delete(
	ctx context.Context,
	key string,
) IOE.IOEither[error, struct{}] {
	return F.Pipe1(
		IOE.TryCatchError(func() (int64, error) {
			return r.client.Del(ctx, key).Result()
		}),
		IOE.Map[error](constEmptyStructFromInt),
	)
}

// Close closes the underlying Redis client connection.
func (r *RedisStore) Close() error {
	return r.client.Close()
}
