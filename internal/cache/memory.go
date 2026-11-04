package cache

import (
	"bytes"
	"context"
	"time"

	F "github.com/IBM/fp-go/v2/function"
	IOE "github.com/IBM/fp-go/v2/ioeither"
	O "github.com/IBM/fp-go/v2/option"
	"github.com/jellydator/ttlcache/v3"
)

// MemoryStore is an in-memory TTL cache backed by ttlcache/v3.
type (
	MemoryStore struct {
		cache *ttlcache.Cache[string, []byte]
	}

	memorySetInput struct {
		store *MemoryStore
		key   string
		val   []byte
		ttl   time.Duration
	}
)

// NewMemoryStore creates an in-memory TTL cache store with touch-on-hit
// disabled so entries expire strictly after their configured TTL.
func NewMemoryStore() Store {
	c := ttlcache.New(
		ttlcache.WithDisableTouchOnHit[string, []byte](),
	)
	return &MemoryStore{cache: c}
}

// getMemoryCacheItem retrieves a raw ttlcache item for key without
// variadic options so it can be partially applied with F.Bind1st.
//
// fp-go:allow-trivial-delegate ttlcache.Cache.Get is variadic and
// cannot be passed directly to F.Bind1st.
func getMemoryCacheItem(
	m *MemoryStore,
	key string,
) *ttlcache.Item[string, []byte] {
	return m.cache.Get(key)
}

// cloneCachedBytes extracts and clones the byte slice held in a
// ttlcache item so callers cannot mutate internal cache state.
//
// fp-go:allow-trivial-delegate ttlcache.Item exposes its value only
// through the Value method.
func cloneCachedBytes(item *ttlcache.Item[string, []byte]) []byte {
	return bytes.Clone(item.Value())
}

func applyMemoryCacheSet(in memorySetInput) {
	in.store.cache.Set(in.key, bytes.Clone(in.val), in.ttl)
}

// Get retrieves the cached byte slice for key, returning O.None when
// the key is missing or expired.
func (m *MemoryStore) Get(
	_ context.Context,
	key string,
) IOE.IOEither[error, O.Option[[]byte]] {
	return F.Pipe3(
		IOE.Of[error](key),
		IOE.Map[error](F.Bind1st(getMemoryCacheItem, m)),
		IOE.Map[error](O.FromNonZero[*ttlcache.Item[string, []byte]]()),
		IOE.Map[error](O.Map(cloneCachedBytes)),
	)
}

// Set stores a defensive copy of val under key with the given TTL.
func (m *MemoryStore) Set(
	_ context.Context,
	key string,
	val []byte,
	ttl time.Duration,
) IOE.IOEither[error, struct{}] {
	return F.Pipe1(
		IOE.Of[error](memorySetInput{
			store: m,
			key:   key,
			val:   val,
			ttl:   ttl,
		}),
		IOE.ChainConsumer[error](applyMemoryCacheSet),
	)
}

// Delete removes key from the in-memory cache.
func (m *MemoryStore) Delete(
	_ context.Context,
	key string,
) IOE.IOEither[error, struct{}] {
	return F.Pipe1(
		IOE.Of[error](key),
		IOE.ChainConsumer[error](m.cache.Delete),
	)
}

// Close clears all entries from the in-memory cache.
func (m *MemoryStore) Close() error {
	m.cache.DeleteAll()
	return nil
}
