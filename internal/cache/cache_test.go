package cache

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	E "github.com/IBM/fp-go/v2/either"
	F "github.com/IBM/fp-go/v2/function"
	O "github.com/IBM/fp-go/v2/option"
	"github.com/alicebob/miniredis/v2"
	ioeutils "github.com/dictyBase/fp-go-loom/ioeitherutils"
	loomcheck "github.com/dictyBase/fp-go-loom/loomcheck"
	"github.com/dictyBase/fp-go-loom/pipelinecheck"
	"github.com/stretchr/testify/require"
)

const (
	testCacheKey = "stock:counts:v1"
	testCacheTTL = 5 * time.Minute
)

func runStoreGet(
	t *testing.T,
	store Store,
) O.Option[[]byte] {
	t.Helper()
	res := F.Pipe1(
		store.Get(context.Background(), testCacheKey),
		ioeutils.ToEither[error, O.Option[[]byte]],
	)
	require.True(t, E.IsRight(res))
	return F.Pipe1(
		res,
		E.GetOrElse(F.Constant1[error](O.None[[]byte]())),
	)
}

func runStoreEpoch(
	t *testing.T,
	store Store,
) int64 {
	t.Helper()
	res := F.Pipe1(
		store.Epoch(context.Background(), testCacheKey),
		ioeutils.ToEither[error, int64],
	)
	require.True(t, E.IsRight(res))
	return F.Pipe1(
		res,
		E.GetOrElse(F.Constant1[error](int64(0))),
	)
}

func runStoreSetIfEpoch(
	t *testing.T,
	store Store,
	val []byte,
	ttl time.Duration,
	epoch int64,
) bool {
	t.Helper()
	res := F.Pipe1(
		store.SetIfEpoch(context.Background(), testCacheKey, val, ttl, epoch),
		ioeutils.ToEither[error, bool],
	)
	require.True(t, E.IsRight(res))
	return F.Pipe1(
		res,
		E.GetOrElse(F.Constant1[error](false)),
	)
}

func runStoreDelete(
	t *testing.T,
	store Store,
) {
	t.Helper()
	res := F.Pipe1(
		store.Delete(context.Background(), testCacheKey),
		ioeutils.ToEither[error, struct{}],
	)
	require.True(t, E.IsRight(res))
}

func TestMemoryStore(t *testing.T) {
	t.Parallel()

	t.Run("MissReturnsNone", func(t *testing.T) {
		t.Parallel()
		store := NewMemoryStore()
		t.Cleanup(func() { _ = store.Close() })

		got := runStoreGet(t, store)
		require.True(t, O.IsNone(got))
	})

	t.Run("SetGetAndDefensiveCopy", func(t *testing.T) {
		t.Parallel()
		store := NewMemoryStore()
		t.Cleanup(func() { _ = store.Close() })

		original := []byte("counts-payload")
		require.True(t, runStoreSetIfEpoch(t, store, original, testCacheTTL, 0))
		original[0] = 'X'

		got := runStoreGet(t, store)
		require.True(t, O.IsSome(got))
		val := F.Pipe1(got, O.GetOrElse(F.Constant([]byte(nil))))
		require.Equal(t, []byte("counts-payload"), val)

		val[0] = 'Z'
		gotAgain := runStoreGet(t, store)
		valAgain := F.Pipe1(gotAgain, O.GetOrElse(F.Constant([]byte(nil))))
		require.Equal(t, []byte("counts-payload"), valAgain)
	})

	t.Run("DeleteRemovesKeyAndBumpsEpoch", func(t *testing.T) {
		t.Parallel()
		store := NewMemoryStore()
		t.Cleanup(func() { _ = store.Close() })

		require.Equal(t, int64(0), runStoreEpoch(t, store))
		require.True(t, runStoreSetIfEpoch(t, store, []byte("payload"), testCacheTTL, 0))
		runStoreDelete(t, store)

		got := runStoreGet(t, store)
		require.True(t, O.IsNone(got))
		require.Equal(t, int64(1), runStoreEpoch(t, store))
	})

	t.Run("StaleEpochWriteIsRejected", func(t *testing.T) {
		t.Parallel()
		store := NewMemoryStore()
		t.Cleanup(func() { _ = store.Close() })

		require.True(t, runStoreSetIfEpoch(t, store, []byte("stale"), testCacheTTL, 0))
		runStoreDelete(t, store)

		// A writer that captured epoch 0 before the delete must fail
		// to restore its stale value.
		require.False(t, runStoreSetIfEpoch(t, store, []byte("stale"), testCacheTTL, 0))
		require.True(t, O.IsNone(runStoreGet(t, store)))

		// A writer that captures the current epoch succeeds.
		require.True(t, runStoreSetIfEpoch(t, store, []byte("fresh"), testCacheTTL, 1))
		require.Equal(t, []byte("fresh"), F.Pipe1(
			runStoreGet(t, store),
			O.GetOrElse(F.Constant([]byte(nil))),
		))
	})

	t.Run("ExpiredEntryReturnsNone", func(t *testing.T) {
		t.Parallel()
		store := NewMemoryStore()
		t.Cleanup(func() { _ = store.Close() })

		require.True(t, runStoreSetIfEpoch(t, store, []byte("ephemeral"), 15*time.Millisecond, 0))
		time.Sleep(40 * time.Millisecond)

		got := runStoreGet(t, store)
		require.True(t, O.IsNone(got))
	})
}

func TestRedisStore(t *testing.T) {
	t.Parallel()

	t.Run("SetGetDeleteAndExpiration", func(t *testing.T) {
		t.Parallel()
		mr, err := miniredis.Run()
		require.NoError(t, err)
		t.Cleanup(mr.Close)

		storeRes := F.Pipe1(
			NewRedisStore(mr.Addr()),
			ioeutils.ToEither[error, Store],
		)
		require.True(t, E.IsRight(storeRes))
		store := F.Pipe1(
			storeRes,
			E.GetOrElse(F.Constant1[error](Store(nil))),
		)
		require.NotNil(t, store)
		t.Cleanup(func() { _ = store.Close() })

		miss := runStoreGet(t, store)
		require.True(t, O.IsNone(miss))
		require.Equal(t, int64(0), runStoreEpoch(t, store))

		require.True(t, runStoreSetIfEpoch(t, store, []byte("redis-payload"), testCacheTTL, 0))
		hit := runStoreGet(t, store)
		require.True(t, O.IsSome(hit))
		val := F.Pipe1(hit, O.GetOrElse(F.Constant([]byte(nil))))
		require.Equal(t, []byte("redis-payload"), val)

		mr.FastForward(testCacheTTL + time.Second)
		expired := runStoreGet(t, store)
		require.True(t, O.IsNone(expired))

		require.True(t, runStoreSetIfEpoch(t, store, []byte("redis-payload-2"), testCacheTTL, 0))
		runStoreDelete(t, store)
		afterDel := runStoreGet(t, store)
		require.True(t, O.IsNone(afterDel))
		require.Equal(t, int64(1), runStoreEpoch(t, store))
	})

	t.Run("StaleEpochWriteIsRejected", func(t *testing.T) {
		t.Parallel()
		mr, err := miniredis.Run()
		require.NoError(t, err)
		t.Cleanup(mr.Close)

		storeRes := F.Pipe1(
			NewRedisStore(mr.Addr()),
			ioeutils.ToEither[error, Store],
		)
		require.True(t, E.IsRight(storeRes))
		store := F.Pipe1(
			storeRes,
			E.GetOrElse(F.Constant1[error](Store(nil))),
		)
		t.Cleanup(func() { _ = store.Close() })

		require.True(t, runStoreSetIfEpoch(t, store, []byte("stale"), testCacheTTL, 0))
		runStoreDelete(t, store)

		require.False(t, runStoreSetIfEpoch(t, store, []byte("stale"), testCacheTTL, 0))
		require.True(t, O.IsNone(runStoreGet(t, store)))

		require.True(t, runStoreSetIfEpoch(t, store, []byte("fresh"), testCacheTTL, 1))
		require.Equal(t, []byte("fresh"), F.Pipe1(
			runStoreGet(t, store),
			O.GetOrElse(F.Constant([]byte(nil))),
		))
	})

	t.Run("UnreachableRedisFailsPingAndPropagatesTransportErrors", func(t *testing.T) {
		t.Parallel()
		mr, err := miniredis.Run()
		require.NoError(t, err)
		addr := mr.Addr()

		storeRes := F.Pipe1(
			NewRedisStore(addr),
			ioeutils.ToEither[error, Store],
		)
		require.True(t, E.IsRight(storeRes))
		store := F.Pipe1(
			storeRes,
			E.GetOrElse(F.Constant1[error](Store(nil))),
		)
		t.Cleanup(func() { _ = store.Close() })

		mr.Close()

		getRes := F.Pipe1(
			store.Get(context.Background(), testCacheKey),
			ioeutils.ToEither[error, O.Option[[]byte]],
		)
		require.True(t, E.IsLeft(getRes))

		pingFailRes := F.Pipe1(
			NewRedisStore(addr),
			ioeutils.ToEither[error, Store],
		)
		require.True(t, E.IsLeft(pingFailRes))
	})
}

func TestCachePipelineAndLoomStyle(t *testing.T) {
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

func TestRedisStoreClosesClientOnFailedPing(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(err)
	t.Cleanup(func() { _ = ln.Close() })

	connClosed := make(chan struct{})
	go func() {
		conn, cerr := ln.Accept()
		if cerr != nil {
			return
		}
		// Answer the client PING with a RESP error so the ping fails
		// fast, then watch for the client to close the connection.
		buf := make([]byte, 256)
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("-ERR ping failed\r\n"))
		for {
			if _, rerr := conn.Read(buf); rerr != nil {
				break
			}
		}
		_ = conn.Close()
		close(connClosed)
	}()

	res := F.Pipe1(
		NewRedisStore(ln.Addr().String()),
		ioeutils.ToEither[error, Store],
	)
	assert.True(E.IsLeft(res))

	select {
	case <-connClosed:
	case <-time.After(5 * time.Second):
		t.Fatal("redis client connection was not closed after failed ping")
	}
}
