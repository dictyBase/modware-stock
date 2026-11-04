// Package cache provides a key-value cache contract with in-memory TTL
// and Redis backends for gRPC middleware caching.
package cache

import (
	"context"
	"time"

	IOE "github.com/IBM/fp-go/v2/ioeither"
	O "github.com/IBM/fp-go/v2/option"
)

// Store defines the functional key-value cache contract for serialized
// protobuf payloads. Mutations advance a per-key invalidation epoch:
// a reader captures the Epoch before it computes a value and writes the
// value back with SetIfEpoch only when the epoch is unchanged. A slow
// reader can therefore not restore data that a mutation invalidated
// while the reader computed its value.
type Store interface {
	Get(ctx context.Context, key string) IOE.IOEither[error, O.Option[[]byte]]
	Epoch(ctx context.Context, key string) IOE.IOEither[error, int64]
	SetIfEpoch(
		ctx context.Context,
		key string,
		val []byte,
		ttl time.Duration,
		epoch int64,
	) IOE.IOEither[error, bool]
	Delete(ctx context.Context, key string) IOE.IOEither[error, struct{}]
	Close() error
}
