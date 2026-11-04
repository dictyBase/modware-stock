package cache

import (
	"bytes"
	"context"
	"sync"
	"time"

	F "github.com/IBM/fp-go/v2/function"
	IOE "github.com/IBM/fp-go/v2/ioeither"
	O "github.com/IBM/fp-go/v2/option"
	"github.com/jellydator/ttlcache/v3"
)

// MemoryStore is an in-memory TTL cache backed by ttlcache/v3 with a
// mutex-guarded invalidation-epoch map.
type MemoryStore struct {
	mu     sync.Mutex
	cache  *ttlcache.Cache[string, []byte]
	epochs map[string]int64
}

// NewMemoryStore creates an in-memory TTL cache store with touch-on-hit
// disabled so entries expire strictly after their configured TTL.
func NewMemoryStore() Store {
	c := ttlcache.New(
		ttlcache.WithDisableTouchOnHit[string, []byte](),
	)
	return &MemoryStore{cache: c, epochs: make(map[string]int64)}
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

// Epoch returns the current invalidation epoch for key.
func (m *MemoryStore) Epoch(
	_ context.Context,
	key string,
) IOE.IOEither[error, int64] {
	return IOE.TryCatchError(func() (int64, error) {
		m.mu.Lock()
		defer m.mu.Unlock()
		epoch := m.epochs[key]
		return epoch, nil
	})
}

// SetIfEpoch stores a defensive copy of val under key with the given
// TTL only when the key's invalidation epoch still equals epoch. It
// reports whether the write happened.
func (m *MemoryStore) SetIfEpoch(
	_ context.Context,
	key string,
	val []byte,
	ttl time.Duration,
	epoch int64,
) IOE.IOEither[error, bool] {
	return IOE.TryCatchError(func() (bool, error) {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.epochs[key] != epoch {
			return false, nil
		}
		m.cache.Set(key, bytes.Clone(val), ttl)
		return true, nil
	})
}

// Delete removes key and advances its invalidation epoch under one
// lock so concurrent stale writers cannot repopulate the key.
func (m *MemoryStore) Delete(
	_ context.Context,
	key string,
) IOE.IOEither[error, struct{}] {
	return IOE.TryCatchError(func() (struct{}, error) {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.epochs[key]++
		m.cache.Delete(key)
		return struct{}{}, nil
	})
}

// Close clears all entries from the in-memory cache.
func (m *MemoryStore) Close() error {
	m.cache.DeleteAll()
	return nil
}
