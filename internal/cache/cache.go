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
// protobuf payloads.
type Store interface {
	Get(ctx context.Context, key string) IOE.IOEither[error, O.Option[[]byte]]
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) IOE.IOEither[error, struct{}]
	Delete(ctx context.Context, key string) IOE.IOEither[error, struct{}]
	Close() error
}
