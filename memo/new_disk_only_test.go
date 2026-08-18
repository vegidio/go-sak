package memo

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vegidio/go-sak/memo/internal"
)

func TestNewDiskOnly(t *testing.T) {
	t.Run("creates memoizer with valid directory", func(t *testing.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		opts := internal.CacheOpts{
			MaxEntries:  1000,
			MaxCapacity: 1024 * 1024, // 1MB
		}

		memoizer, err := NewDiskOnly(tmpDir, opts)

		require.NoError(t, err)
		require.NotNil(t, memoizer)
		assert.NotNil(t, memoizer.Store)

		// Clean up
		err = memoizer.Close()
		assert.NoError(t, err)
	})

	t.Run("creates memoizer with default options", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := internal.CacheOpts{} // Use default values

		memoizer, err := NewDiskOnly(tmpDir, opts)

		require.NoError(t, err)
		require.NotNil(t, memoizer)
		assert.NotNil(t, memoizer.Store)

		// Clean up
		err = memoizer.Close()
		assert.NoError(t, err)
	})

	t.Run("returns error for invalid directory path", func(t *testing.T) {
		// Use a path that would cause permission issues
		invalidPath := "/root/invalid/path/that/should/fail"

		opts := internal.CacheOpts{
			MaxEntries:  100,
			MaxCapacity: 1024,
		}

		memoizer, err := NewDiskOnly(invalidPath, opts)

		assert.Error(t, err)
		assert.Nil(t, memoizer)
	})

	t.Run("returns error for empty directory path", func(t *testing.T) {
		opts := internal.CacheOpts{
			MaxEntries:  100,
			MaxCapacity: 1024,
		}

		memoizer, err := NewDiskOnly("", opts)

		assert.Error(t, err)
		assert.Nil(t, memoizer)
	})

	t.Run("returns memoizer with correct store type", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := internal.CacheOpts{
			MaxEntries:  100,
			MaxCapacity: 1024 * 1024,
		}

		memoizer, err := NewDiskOnly(tmpDir, opts)

		require.NoError(t, err)
		require.NotNil(t, memoizer)

		// Verify the store is of the expected type
		_, ok := memoizer.Store.(*internal.DiskStore)
		assert.True(t, ok, "Expected store to be of type *internal.DiskStore")

		// Clean up
		err = memoizer.Close()
		assert.NoError(t, err)
	})

	t.Run("multiple instances can use different directories", func(t *testing.T) {
		tmpDir1 := t.TempDir()
		tmpDir2 := t.TempDir()

		opts1 := internal.CacheOpts{MaxEntries: 100}
		opts2 := internal.CacheOpts{MaxEntries: 200}

		memoizer1, err1 := NewDiskOnly(tmpDir1, opts1)
		memoizer2, err2 := NewDiskOnly(tmpDir2, opts2)

		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NotNil(t, memoizer1)
		require.NotNil(t, memoizer2)

		assert.NotEqual(t, memoizer1, memoizer2)

		// Clean up
		err1 = memoizer1.Close()
		err2 = memoizer2.Close()
		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})

	t.Run("handles custom cache options", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := internal.CacheOpts{
			MaxEntries:  5000,
			MaxCapacity: 10 * 1024 * 1024, // 10MB
		}

		memoizer, err := NewDiskOnly(tmpDir, opts)

		require.NoError(t, err)
		require.NotNil(t, memoizer)

		// Clean up
		err = memoizer.Close()
		assert.NoError(t, err)
	})
}

func TestDiskOnlyCleanup(t *testing.T) {
	t.Run("cleanup on a fresh store succeeds", func(t *testing.T) {
		// Arrange
		memoizer, err := NewDiskOnly(t.TempDir(), internal.CacheOpts{})
		require.NoError(t, err)
		defer memoizer.Close()

		// Act - nothing to reclaim yet, which must still be reported as success
		err = memoizer.Cleanup(context.Background())

		// Assert
		assert.NoError(t, err)
	})

	t.Run("cleanup after storing expired entries succeeds", func(t *testing.T) {
		// Arrange
		memoizer, err := NewDiskOnly(t.TempDir(), internal.CacheOpts{})
		require.NoError(t, err)
		defer memoizer.Close()

		ctx := context.Background()
		for i := range 50 {
			_, err = Do(memoizer, ctx, KeyFrom("expiring", i), 10*time.Millisecond,
				func(context.Context) (string, error) { return strings.Repeat("x", 1024), nil })
			require.NoError(t, err)
		}
		time.Sleep(50 * time.Millisecond)

		// Act
		err = memoizer.Cleanup(ctx)

		// Assert
		assert.NoError(t, err)

		// The entry is gone regardless of whether its bytes were reclaimed
		_, ok, err := memoizer.Store.Get(ctx, KeyFrom("expiring", 0))
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("cleanup keeps an entry that expired and was written again", func(t *testing.T) {
		// Arrange - an entry that expires, then gets recomputed with a fresh TTL. The old expired version and the new
		// live one both sit in the tree until a compaction merges them.
		memoizer, err := NewDiskOnly(t.TempDir(), internal.CacheOpts{})
		require.NoError(t, err)
		defer memoizer.Close()

		ctx := context.Background()
		key := KeyFrom("rewritten")

		_, err = Do(memoizer, ctx, key, 10*time.Millisecond,
			func(context.Context) (string, error) { return "stale", nil })
		require.NoError(t, err)
		time.Sleep(50 * time.Millisecond)

		fresh, err := Do(memoizer, ctx, key, time.Hour,
			func(context.Context) (string, error) { return "fresh", nil })
		require.NoError(t, err)
		require.Equal(t, "fresh", fresh)

		// Act
		require.NoError(t, memoizer.Cleanup(ctx))

		// Assert - the sweep must judge the key by its newest version, not the expired one it replaced
		got, err := Do(memoizer, ctx, key, time.Hour,
			func(context.Context) (string, error) { return "recomputed", nil })
		assert.NoError(t, err)
		assert.Equal(t, "fresh", got, "cleanup deleted a live entry because an older version of it had expired")
	})

	t.Run("cleanup honours a cancelled context", func(t *testing.T) {
		// Arrange
		memoizer, err := NewDiskOnly(t.TempDir(), internal.CacheOpts{})
		require.NoError(t, err)
		defer memoizer.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Act
		err = memoizer.Cleanup(ctx)

		// Assert - either it bailed on the context or it finished before noticing; both are fine
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
		}
	})

	t.Run("close does not hang on the startup sweep", func(t *testing.T) {
		// Arrange
		memoizer, err := NewDiskOnly(t.TempDir(), internal.CacheOpts{})
		require.NoError(t, err)

		closed := make(chan error, 1)

		// Act - close immediately, while the background sweep may still be running
		go func() { closed <- memoizer.Close() }()

		// Assert
		select {
		case err = <-closed:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Close never returned; it is deadlocked against the startup cleanup")
		}
	})

	t.Run("close can be called multiple times safely", func(t *testing.T) {
		// Arrange
		memoizer, err := NewDiskOnly(t.TempDir(), internal.CacheOpts{})
		require.NoError(t, err)

		// Act & Assert
		assert.NoError(t, memoizer.Close())
		assert.NoError(t, memoizer.Close())
		assert.NoError(t, memoizer.Close())
	})

	t.Run("cleanup after close is a no-op", func(t *testing.T) {
		// Arrange
		memoizer, err := NewDiskOnly(t.TempDir(), internal.CacheOpts{})
		require.NoError(t, err)
		require.NoError(t, memoizer.Close())

		// Act & Assert - must not panic or touch the closed database
		assert.NotPanics(t, func() {
			assert.NoError(t, memoizer.Cleanup(context.Background()))
		})
	})

	t.Run("concurrent cleanups are serialized", func(t *testing.T) {
		// Arrange
		memoizer, err := NewDiskOnly(t.TempDir(), internal.CacheOpts{})
		require.NoError(t, err)
		defer memoizer.Close()

		errs := make(chan error, 4)

		// Act
		for range 4 {
			go func() { errs <- memoizer.Cleanup(context.Background()) }()
		}

		// Assert
		for range 4 {
			select {
			case err = <-errs:
				assert.NoError(t, err)
			case <-time.After(10 * time.Second):
				t.Fatal("concurrent Cleanup calls did not all return")
			}
		}
	})
}
