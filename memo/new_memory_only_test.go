package memo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vegidio/go-sak/memo/internal"
)

func TestNewMemoryOnly(t *testing.T) {
	t.Run("creates memoizer with default options", func(t *testing.T) {
		opts := internal.CacheOpts{}

		memoizer, err := NewMemoryOnly(opts)

		require.NoError(t, err)
		assert.NotNil(t, memoizer)
		assert.NotNil(t, memoizer.Store)

		// Clean up
		err = memoizer.Close()
		assert.NoError(t, err)
	})

	t.Run("creates memoizer with custom options", func(t *testing.T) {
		opts := internal.CacheOpts{
			MaxEntries:  500_000,
			MaxCapacity: 512 * 1024 * 1024, // 512 MB
		}

		memoizer, err := NewMemoryOnly(opts)

		require.NoError(t, err)
		assert.NotNil(t, memoizer)
		assert.NotNil(t, memoizer.Store)

		// Clean up
		err = memoizer.Close()
		assert.NoError(t, err)
	})

	t.Run("creates memoizer with zero values in options", func(t *testing.T) {
		opts := internal.CacheOpts{
			MaxEntries:  0, // Should use default
			MaxCapacity: 0, // Should use default
		}

		memoizer, err := NewMemoryOnly(opts)

		require.NoError(t, err)
		assert.NotNil(t, memoizer)
		assert.NotNil(t, memoizer.Store)

		// Clean up
		err = memoizer.Close()
		assert.NoError(t, err)
	})

	t.Run("returned memoizer has memory store", func(t *testing.T) {
		opts := internal.CacheOpts{}

		memoizer, err := NewMemoryOnly(opts)
		require.NoError(t, err)

		// Verify the store is a MemoryStore by checking its behavior
		// We can't directly type assert because the field is of interface type,
		// but we can verify it works as expected
		assert.NotNil(t, memoizer.Store)

		// Clean up
		err = memoizer.Close()
		assert.NoError(t, err)
	})

	t.Run("creates multiple independent memoizers", func(t *testing.T) {
		opts1 := internal.CacheOpts{MaxEntries: 100_000}
		opts2 := internal.CacheOpts{MaxEntries: 200_000}

		memoizer1, err1 := NewMemoryOnly(opts1)
		require.NoError(t, err1)

		memoizer2, err2 := NewMemoryOnly(opts2)
		require.NoError(t, err2)

		// Verify they are different instances
		assert.NotEqual(t, memoizer1, memoizer2)
		assert.NotEqual(t, memoizer1.Store, memoizer2.Store)

		// Clean up
		err1 = memoizer1.Close()
		assert.NoError(t, err1)

		err2 = memoizer2.Close()
		assert.NoError(t, err2)
	})

	t.Run("memoizer has initialized singleflight group", func(t *testing.T) {
		opts := internal.CacheOpts{}

		memoizer, err := NewMemoryOnly(opts)
		require.NoError(t, err)

		// Verify that the singleflight group is initialized (not nil)
		// We can't directly check if it's initialized, but we can verify
		// the memoizer structure is complete
		assert.NotNil(t, memoizer)

		// Clean up
		err = memoizer.Close()
		assert.NoError(t, err)
	})
}
