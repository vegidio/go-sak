package internal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiskStore_ClampsCacheOpts(t *testing.T) {
	// These values were forwarded straight to Badger's ValueLogFileSize, which validates a [1 MiB, 2 GiB) range
	// and refused to open outside it - including for 512 MiB-style configurations the documentation suggested.
	tests := []struct {
		name string
		opts CacheOpts
	}{
		{name: "Defaults", opts: CacheOpts{}},
		{name: "AboveBadgersCeiling", opts: CacheOpts{MaxCapacity: 4 << 30}},
		{name: "BelowBadgersFloor", opts: CacheOpts{MaxCapacity: 512 * 1024}},
		{name: "ExactlyTwoGiB", opts: CacheOpts{MaxCapacity: 2 << 30}},
		{name: "Negative", opts: CacheOpts{MaxCapacity: -1, MaxEntries: -1}},
		{name: "EntriesBeyondUint32", opts: CacheOpts{MaxEntries: 1 << 40}},
		{name: "DocumentedExample", opts: CacheOpts{MaxEntries: 500_000, MaxCapacity: 512 << 20}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewDiskStore(t.TempDir(), tt.opts)
			require.NoError(t, err, "no CacheOpts value may stop the store from opening")
			defer store.Close()

			ctx := context.Background()
			require.NoError(t, store.Set(ctx, "k", []byte("v"), time.Minute))

			got, ok, err := store.Get(ctx, "k")
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, "v", string(got))
		})
	}
}

func TestDiskStore_RoundTripAndExpiry(t *testing.T) {
	store, err := NewDiskStore(t.TempDir(), CacheOpts{})
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	require.NoError(t, store.Set(ctx, "keep", []byte("value"), time.Minute))
	require.NoError(t, store.Set(ctx, "expire", []byte("value"), 50*time.Millisecond))

	got, ok, err := store.Get(ctx, "keep")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "value", string(got))

	assert.Eventually(t, func() bool {
		_, ok, err := store.Get(ctx, "expire")
		return err == nil && !ok
	}, 5*time.Second, 25*time.Millisecond, "an entry past its TTL must stop being served")

	// A key that was never written is a miss, not an error.
	_, ok, err = store.Get(ctx, "absent")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestDiskStore_GetReturnsACopy(t *testing.T) {
	store, err := NewDiskStore(t.TempDir(), CacheOpts{})
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	require.NoError(t, store.Set(ctx, "k", []byte("original"), time.Minute))

	got, ok, err := store.Get(ctx, "k")
	require.NoError(t, err)
	require.True(t, ok)
	got[0] = 'X'

	again, _, err := store.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "original", string(again))
}

func TestDiskStore_CloseIsIdempotent(t *testing.T) {
	store, err := NewDiskStore(t.TempDir(), CacheOpts{})
	require.NoError(t, err)

	first := store.Close()
	second := store.Close()

	assert.NoError(t, first)
	assert.Equal(t, first, second, "the Store contract requires every Close to return the same error")
}

func TestDiskStore_CleanupIsSafeToCallRepeatedly(t *testing.T) {
	store, err := NewDiskStore(t.TempDir(), CacheOpts{})
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	for i := range 50 {
		require.NoError(t, store.Set(ctx, string(rune('a'+i%26))+string(rune('a'+i/26)), []byte("v"), time.Millisecond))
	}

	for range 3 {
		assert.NoError(t, store.Cleanup(ctx))
	}
}

func TestDiskStore_CleanupHonoursACancelledContext(t *testing.T) {
	store, err := NewDiskStore(t.TempDir(), CacheOpts{})
	require.NoError(t, err)
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.ErrorIs(t, store.Cleanup(ctx), context.Canceled)
}

func TestDiskStore_CleanupAfterCloseDoesNotPanic(t *testing.T) {
	store, err := NewDiskStore(t.TempDir(), CacheOpts{})
	require.NoError(t, err)
	require.NoError(t, store.Close())

	// Close races the background sweep that NewDiskStore starts; neither order may panic.
	assert.NotPanics(t, func() { _ = store.Cleanup(context.Background()) })
}
