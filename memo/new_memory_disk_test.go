package memo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vegidio/go-sak/memo/internal"
)

func TestNewMemoryDisk(t *testing.T) {
	t.Run("creates memoizer with valid directory and default options", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := internal.CacheOpts{
			MaxEntries:  1000,
			MaxCapacity: 1024 * 1024, // 1MB
		}
		promoteTTL := 5 * time.Minute

		memoizer, closeFunc, err := NewMemoryDisk(tmpDir, opts, promoteTTL)

		require.NoError(t, err)
		require.NotNil(t, memoizer)
		require.NotNil(t, closeFunc)
		assert.NotNil(t, memoizer.Store)

		// Clean up
		err = closeFunc()
		assert.NoError(t, err)
	})

	t.Run("creates memoizer with zero promoteTTL", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := internal.CacheOpts{
			MaxEntries:  500,
			MaxCapacity: 1024 * 1024, // 512KB
		}
		promoteTTL := time.Duration(0)

		memoizer, closeFunc, err := NewMemoryDisk(tmpDir, opts, promoteTTL)

		require.NoError(t, err)
		require.NotNil(t, memoizer)
		require.NotNil(t, closeFunc)

		// Clean up
		err = closeFunc()
		assert.NoError(t, err)
	})

	t.Run("creates memoizer with large promoteTTL", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := internal.CacheOpts{
			MaxEntries:  100,
			MaxCapacity: 1024 * 1024,
		}
		promoteTTL := 24 * time.Hour

		memoizer, closeFunc, err := NewMemoryDisk(tmpDir, opts, promoteTTL)

		require.NoError(t, err)
		require.NotNil(t, memoizer)
		require.NotNil(t, closeFunc)

		// Clean up
		err = closeFunc()
		assert.NoError(t, err)
	})

	t.Run("creates memoizer with default cache options", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := internal.CacheOpts{} // Use default values
		promoteTTL := time.Minute

		memoizer, closeFunc, err := NewMemoryDisk(tmpDir, opts, promoteTTL)

		require.NoError(t, err)
		require.NotNil(t, memoizer)
		require.NotNil(t, closeFunc)

		// Clean up
		err = closeFunc()
		assert.NoError(t, err)
	})

	t.Run("returns error for invalid directory path", func(t *testing.T) {
		// Use a path that would cause permission issues
		invalidPath := "/root/invalid/path/that/should/fail"

		opts := internal.CacheOpts{
			MaxEntries:  100,
			MaxCapacity: 1024,
		}
		promoteTTL := time.Minute

		memoizer, closeFunc, err := NewMemoryDisk(invalidPath, opts, promoteTTL)

		assert.Error(t, err)
		assert.Nil(t, memoizer)
		assert.Nil(t, closeFunc)
	})

	t.Run("returns error for empty directory path", func(t *testing.T) {
		opts := internal.CacheOpts{
			MaxEntries:  100,
			MaxCapacity: 1024,
		}
		promoteTTL := time.Minute

		memoizer, closeFunc, err := NewMemoryDisk("", opts, promoteTTL)

		assert.Error(t, err)
		assert.Nil(t, memoizer)
		assert.Nil(t, closeFunc)
	})

	t.Run("returns memoizer with composite store", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := internal.CacheOpts{
			MaxEntries:  100,
			MaxCapacity: 1024 * 1024,
		}
		promoteTTL := time.Minute

		memoizer, closeFunc, err := NewMemoryDisk(tmpDir, opts, promoteTTL)

		require.NoError(t, err)
		require.NotNil(t, memoizer)

		// Verify the store is of the expected composite type
		_, ok := memoizer.Store.(*internal.CompositeStore)
		assert.True(t, ok, "Expected store to be of type *internal.CompositeStore")

		// Clean up
		err = closeFunc()
		assert.NoError(t, err)
	})

	t.Run("multiple instances can use different directories", func(t *testing.T) {
		tmpDir1 := t.TempDir()
		tmpDir2 := t.TempDir()

		opts1 := internal.CacheOpts{MaxEntries: 100}
		opts2 := internal.CacheOpts{MaxEntries: 200}
		promoteTTL1 := time.Minute
		promoteTTL2 := 2 * time.Minute

		memoizer1, closeFunc1, err1 := NewMemoryDisk(tmpDir1, opts1, promoteTTL1)
		memoizer2, closeFunc2, err2 := NewMemoryDisk(tmpDir2, opts2, promoteTTL2)

		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NotNil(t, memoizer1)
		require.NotNil(t, memoizer2)

		assert.NotEqual(t, memoizer1, memoizer2)
		assert.NotEqual(t, memoizer1.Store, memoizer2.Store)

		// Clean up
		err1 = closeFunc1()
		err2 = closeFunc2()
		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})

	t.Run("handles custom cache options with large values", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := internal.CacheOpts{
			MaxEntries:  5000,
			MaxCapacity: 10 * 1024 * 1024, // 10MB
		}
		promoteTTL := 30 * time.Minute

		memoizer, closeFunc, err := NewMemoryDisk(tmpDir, opts, promoteTTL)

		require.NoError(t, err)
		require.NotNil(t, memoizer)

		// Clean up
		err = closeFunc()
		assert.NoError(t, err)
	})

	t.Run("cleanup function can be called multiple times safely", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := internal.CacheOpts{MaxEntries: 100}
		promoteTTL := time.Minute

		memoizer, closeFunc, err := NewMemoryDisk(tmpDir, opts, promoteTTL)

		require.NoError(t, err)
		require.NotNil(t, memoizer)
		require.NotNil(t, closeFunc)

		// Call close function multiple times
		err1 := closeFunc()
		err2 := closeFunc()
		err3 := closeFunc()

		// First close should succeed, subsequent calls behavior depends on implementation
		// but they shouldn't panic
		assert.NoError(t, err1)
		// err2 and err3 might be errors or nil depending on implementation
		// but the important thing is they don't panic
		_ = err2
		_ = err3
	})

	t.Run("memory store creation failure is handled gracefully", func(t *testing.T) {
		// This test is more theoretical since it's hard to make NewMemoryStore fail
		// but we can test the error path if disk store creation fails after memory store succeeds
		tmpDir := t.TempDir()

		// Use invalid cache options that might cause memory store to fail
		opts := internal.CacheOpts{
			MaxEntries:  -1, // Negative values might cause issues
			MaxCapacity: -1,
		}
		promoteTTL := time.Minute

		memoizer, closeFunc, err := NewMemoryDisk(tmpDir, opts, promoteTTL)

		// The behavior depends on how the internal stores handle negative values
		// This test documents the current behavior
		if err != nil {
			assert.Nil(t, memoizer)
			assert.Nil(t, closeFunc)
		} else {
			require.NotNil(t, memoizer)
			require.NotNil(t, closeFunc)
			err = closeFunc()
			assert.NoError(t, err)
		}
	})

	t.Run("negative promoteTTL is handled", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := internal.CacheOpts{
			MaxEntries:  100,
			MaxCapacity: 1024 * 1024,
		}
		promoteTTL := -time.Minute

		memoizer, closeFunc, err := NewMemoryDisk(tmpDir, opts, promoteTTL)

		require.NoError(t, err)
		require.NotNil(t, memoizer)
		require.NotNil(t, closeFunc)

		// Clean up
		err = closeFunc()
		assert.NoError(t, err)
	})
}
