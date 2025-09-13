package memo

import (
	"testing"

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
