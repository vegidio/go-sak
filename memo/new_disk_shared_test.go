package memo

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vegidio/go-sak/memo/internal"
)

// openShared returns the number of shared stores this process currently holds, so a test can assert the bookkeeping
// settles rather than only that the calls returned.
func openShared() int {
	sharedMu.Lock()
	defer sharedMu.Unlock()

	return len(sharedDisks)
}

func TestNewDiskShared(t *testing.T) {
	// The reason this constructor exists. NewDiskOnly on a path this process already holds fails on Badger's own
	// directory lock, with an error indistinguishable from a different process holding it.
	t.Run("opening the same directory twice succeeds", func(t *testing.T) {
		dir := t.TempDir()

		first, err := NewDiskShared(dir, CacheOpts{})
		require.NoError(t, err)
		defer first.Close()

		blocked, err := NewDiskOnly(dir, CacheOpts{})
		require.Error(t, err, "NewDiskOnly must still fail on the lock; that is what NewDiskShared is for")
		assert.Nil(t, blocked)

		second, err := NewDiskShared(dir, CacheOpts{})
		require.NoError(t, err)
		defer second.Close()

		assert.Equal(t, 1, openShared(), "both handles must be onto one store")
	})

	t.Run("handles share the underlying store", func(t *testing.T) {
		dir := t.TempDir()
		ctx := context.Background()

		writer, err := NewDiskShared(dir, CacheOpts{})
		require.NoError(t, err)
		defer writer.Close()

		reader, err := NewDiskShared(dir, CacheOpts{})
		require.NoError(t, err)
		defer reader.Close()

		require.NoError(t, writer.Store.Set(ctx, "key", []byte("value"), time.Hour))

		got, ok, err := reader.Store.Get(ctx, "key")
		require.NoError(t, err)
		require.True(t, ok, "a write through one handle must be visible through another")
		assert.Equal(t, []byte("value"), got)
	})

	// Each handle is distinct, so each Close releases exactly one reference and the store survives until the last.
	t.Run("the store closes only with the last handle", func(t *testing.T) {
		dir := t.TempDir()
		ctx := context.Background()

		first, err := NewDiskShared(dir, CacheOpts{})
		require.NoError(t, err)

		second, err := NewDiskShared(dir, CacheOpts{})
		require.NoError(t, err)

		require.NoError(t, first.Close())
		assert.Equal(t, 1, openShared(), "the store must outlive the first handle")

		// Still usable through the handle that has not been closed.
		require.NoError(t, second.Store.Set(ctx, "key", []byte("value"), time.Hour))

		require.NoError(t, second.Close())
		assert.Equal(t, 0, openShared(), "the last Close must release the store")

		// And the directory lock is genuinely gone, which is the only proof that the store really closed.
		reopened, err := NewDiskOnly(dir, CacheOpts{})
		require.NoError(t, err, "the lock is still held, so the store never closed")
		assert.NoError(t, reopened.Close())
	})

	t.Run("closing a handle twice releases one reference", func(t *testing.T) {
		dir := t.TempDir()

		first, err := NewDiskShared(dir, CacheOpts{})
		require.NoError(t, err)

		second, err := NewDiskShared(dir, CacheOpts{})
		require.NoError(t, err)
		defer second.Close()

		require.NoError(t, first.Close())
		require.NoError(t, first.Close(), "a repeated Close must be a no-op, not an error")

		assert.Equal(t, 1, openShared(), "the second handle's reference must survive the first's extra Close")
	})

	t.Run("a relative path matches the absolute one", func(t *testing.T) {
		dir := t.TempDir()

		absolute, err := NewDiskShared(dir, CacheOpts{})
		require.NoError(t, err)
		defer absolute.Close()

		// Reached through a path that is textually different but resolves to the same directory.
		messy, err := NewDiskShared(filepath.Join(dir, "sub", ".."), CacheOpts{})
		require.NoError(t, err)
		defer messy.Close()

		assert.Equal(t, 1, openShared(), "the two paths name one directory and must share one store")
	})

	t.Run("reports the directory it opened", func(t *testing.T) {
		dir := t.TempDir()

		memoizer, err := NewDiskShared(dir, CacheOpts{})
		require.NoError(t, err)
		defer memoizer.Close()

		want, err := filepath.Abs(dir)
		require.NoError(t, err)
		assert.Equal(t, want, memoizer.Path())
	})

	// The open happens under the package lock precisely so that a race on a fresh directory cannot have two callers
	// both open it. Worth running under -race.
	t.Run("concurrent callers open one store", func(t *testing.T) {
		dir := t.TempDir()

		var (
			wg       sync.WaitGroup
			mu       sync.Mutex
			handles  []*Memoizer
			failures []error
		)

		for range 8 {
			wg.Add(1)

			go func() {
				defer wg.Done()

				memoizer, err := NewDiskShared(dir, CacheOpts{})

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					failures = append(failures, err)
					return
				}

				handles = append(handles, memoizer)
			}()
		}

		wg.Wait()

		require.Empty(t, failures, "no caller may lose a race for a lock they are supposed to share")
		require.Len(t, handles, 8)
		assert.Equal(t, 1, openShared())

		for _, h := range handles {
			assert.NoError(t, h.Close())
		}

		assert.Equal(t, 0, openShared(), "every handle was closed, so nothing may be left open")
	})

	t.Run("a directory that cannot be opened is not recorded", func(t *testing.T) {
		// A regular file where the store's directory should be: Badger cannot open it, and nothing must be left
		// behind for the next caller to be handed.
		file := filepath.Join(t.TempDir(), "not-a-directory")
		require.NoError(t, writeFile(file))

		memoizer, err := NewDiskShared(file, CacheOpts{})
		require.Error(t, err)
		assert.Nil(t, memoizer)
		assert.Equal(t, 0, openShared())
	})
}

func TestMemoizerPath(t *testing.T) {
	t.Run("a disk memoizer reports its directory", func(t *testing.T) {
		dir := t.TempDir()

		memoizer, err := NewDiskOnly(dir, CacheOpts{})
		require.NoError(t, err)
		defer memoizer.Close()

		assert.Equal(t, dir, memoizer.Path())
	})

	t.Run("a memory memoizer reports nothing", func(t *testing.T) {
		memoizer, err := NewMemoryOnly(CacheOpts{})
		require.NoError(t, err)
		defer memoizer.Close()

		assert.Empty(t, memoizer.Path(), "a memory store has no directory to report")
	})

	t.Run("a composite memoizer reports its disk tier", func(t *testing.T) {
		dir := t.TempDir()

		memoizer, closeAll, err := NewMemoryDisk(dir, CacheOpts{}, 0)
		require.NoError(t, err)
		defer closeAll()

		assert.Equal(t, dir, memoizer.Path())
	})
}

// TestErrNotAdmittedIsReachable pins that the sentinel is usable from outside the module. It used to live only in the
// internal package, so a caller could tell a dropped write from a broken store only by matching the message text.
func TestErrNotAdmittedIsReachable(t *testing.T) {
	require.NotNil(t, ErrNotAdmitted)
	assert.ErrorIs(t, internal.ErrNotAdmitted, ErrNotAdmitted, "the re-export must be the same error, not a copy")
}

// writeFile creates an empty regular file, for the case that needs a non-directory where a directory is expected.
func writeFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	return f.Close()
}
