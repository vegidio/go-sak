package internal

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStore is a Store whose every operation can be made to fail, so the layering in CompositeStore can be tested
// without standing up a real cache.
type fakeStore struct {
	mu     sync.Mutex
	data   map[string][]byte
	setErr error
	getErr error
	clsErr error

	closes int
	sets   []string
}

func newFakeStore() *fakeStore {
	return &fakeStore{data: make(map[string][]byte)}
}

func (f *fakeStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.getErr != nil {
		return nil, false, f.getErr
	}

	v, ok := f.data[key]
	return v, ok, nil
}

func (f *fakeStore) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sets = append(f.sets, key)
	if f.setErr != nil {
		return f.setErr
	}

	f.data[key] = value
	return nil
}

func (f *fakeStore) Cleanup(context.Context) error { return nil }

func (f *fakeStore) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.closes++
	return f.clsErr
}

func (f *fakeStore) setCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.sets...)
}

// region - MemoryStore

func TestMemoryStore_GetReturnsACopy(t *testing.T) {
	store, err := NewMemoryStore(CacheOpts{})
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	require.NoError(t, store.Set(ctx, "k", []byte("original"), time.Minute))

	got, ok, err := store.Get(ctx, "k")
	require.NoError(t, err)
	require.True(t, ok)

	// Handing back the stored slice itself would let this write corrupt the entry for every other reader.
	got[0] = 'X'

	again, ok, err := store.Get(ctx, "k")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "original", string(again), "a caller writing into the returned slice must not corrupt the cache")
}

func TestMemoryStore_CloseIsIdempotent(t *testing.T) {
	store, err := NewMemoryStore(CacheOpts{})
	require.NoError(t, err)

	// The Store contract requires this; MemoryStore had no closeOnce and only survived by luck, since Ristretto's
	// own re-entrancy check is itself racy.
	assert.NoError(t, store.Close())
	assert.NotPanics(t, func() { assert.NoError(t, store.Close()) })
	assert.NotPanics(t, func() { assert.NoError(t, store.Close()) })
}

func TestMemoryStore_CloseIsSafeUnderConcurrency(t *testing.T) {
	store, err := NewMemoryStore(CacheOpts{})
	require.NoError(t, err)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { _ = store.Close() })
	}

	assert.NotPanics(t, wg.Wait)
}

func TestMemoryStore_MissIsNotAnError(t *testing.T) {
	store, err := NewMemoryStore(CacheOpts{})
	require.NoError(t, err)
	defer store.Close()

	v, ok, err := store.Get(context.Background(), "absent")
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, v)
}

// endregion

// region - CompositeStore

func TestCompositeStore_PromotesADiskHitIntoMemory(t *testing.T) {
	mem, disk := newFakeStore(), newFakeStore()
	ctx := context.Background()
	require.NoError(t, disk.Set(ctx, "k", []byte("v"), time.Minute))

	store := NewCompositeStore(mem, disk, time.Minute)

	got, ok, err := store.Get(ctx, "k")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "v", string(got))

	assert.Equal(t, []string{"k"}, mem.setCalls(), "a disk hit must be promoted into the hot tier")
}

func TestCompositeStore_DoesNotPromoteWithoutAHotTTL(t *testing.T) {
	mem, disk := newFakeStore(), newFakeStore()
	ctx := context.Background()
	require.NoError(t, disk.Set(ctx, "k", []byte("v"), time.Minute))

	store := NewCompositeStore(mem, disk, 0)

	_, ok, err := store.Get(ctx, "k")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Empty(t, mem.setCalls())
}

func TestCompositeStore_MemoryHitSkipsDisk(t *testing.T) {
	mem, disk := newFakeStore(), newFakeStore()
	ctx := context.Background()
	require.NoError(t, mem.Set(ctx, "k", []byte("hot"), time.Minute))
	disk.getErr = errors.New("disk must not be consulted")

	store := NewCompositeStore(mem, disk, time.Minute)

	got, ok, err := store.Get(ctx, "k")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "hot", string(got))
}

func TestCompositeStore_CloseReportsEveryTierNotJustTheFirst(t *testing.T) {
	memErr := errors.New("memory tier failed")
	diskErr := errors.New("disk tier failed")

	mem, disk := newFakeStore(), newFakeStore()
	mem.clsErr, disk.clsErr = memErr, diskErr

	err := NewCompositeStore(mem, disk, time.Minute).Close()
	require.Error(t, err)

	// Keeping only the first error meant a memory failure masked an unflushed disk tier, which is the one that
	// actually loses data.
	assert.ErrorIs(t, err, memErr)
	assert.ErrorIs(t, err, diskErr, "the disk error must not be masked by the memory error")
}

func TestCompositeStore_CloseVisitsEveryTierEvenAfterAFailure(t *testing.T) {
	mem, disk := newFakeStore(), newFakeStore()
	mem.clsErr = errors.New("boom")

	_ = NewCompositeStore(mem, disk, time.Minute).Close()

	assert.Equal(t, 1, disk.closes, "a failing tier must not stop the others from being closed")
}

func TestCompositeStore_CloseIsIdempotent(t *testing.T) {
	mem, disk := newFakeStore(), newFakeStore()
	store := NewCompositeStore(mem, disk, time.Minute)

	first := store.Close()
	second := store.Close()

	assert.Equal(t, first, second, "every Close must return the same error")
	assert.Equal(t, 1, mem.closes, "the underlying tiers must only be closed once")
	assert.Equal(t, 1, disk.closes)
}

func TestCompositeStore_SetWritesBothTiersAndReportsBothErrors(t *testing.T) {
	memErr := errors.New("memory set failed")
	diskErr := errors.New("disk set failed")

	mem, disk := newFakeStore(), newFakeStore()
	mem.setErr, disk.setErr = memErr, diskErr

	err := NewCompositeStore(mem, disk, time.Minute).Set(context.Background(), "k", []byte("v"), time.Minute)

	assert.ErrorIs(t, err, memErr)
	assert.ErrorIs(t, err, diskErr)
}

func TestCompositeStore_ToleratesAnAbsentTier(t *testing.T) {
	ctx := context.Background()
	disk := newFakeStore()

	store := NewCompositeStore(nil, disk, time.Minute)
	require.NoError(t, store.Set(ctx, "k", []byte("v"), time.Minute))

	got, ok, err := store.Get(ctx, "k")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "v", string(got))
	assert.NoError(t, store.Close())
}

// endregion
