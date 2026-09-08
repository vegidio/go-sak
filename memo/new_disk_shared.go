package memo

import (
	"path/filepath"
	"sync"

	"github.com/vegidio/go-sak/memo/internal"
)

// sharedDisks holds the disk stores this process has open, keyed by the absolute directory they live in. Package-level
// because the constraint it exists to satisfy is itself process-wide: Badger takes one lock per directory, so "how many
// stores does this process have open on that path" has exactly one answer per process.
var (
	sharedMu    sync.Mutex
	sharedDisks = make(map[string]*sharedDisk)
)

type sharedDisk struct {
	store *internal.DiskStore

	// refs counts the outstanding handles, not the callers: a caller that closes twice is counted once, because each
	// handle releases at most one reference.
	refs int
}

// sharedHandle is one holder's view of a shared store. Get, Set and Cleanup pass straight through to the store; only
// Close differs, releasing this handle's reference instead of closing a store other holders are still using.
type sharedHandle struct {
	internal.Store

	key       string
	closeOnce sync.Once
	closeErr  error
}

// Path reports the directory the shared store was opened under. Declared rather than promoted: Store is an interface,
// so embedding it carries only the interface's own methods and Path is not one of them.
func (h *sharedHandle) Path() string {
	return h.key
}

// Close releases this handle. The store closes when the last handle does. Safe to call more than once - the extra
// calls return the same error and release nothing - which keeps a deferred Close honest.
func (h *sharedHandle) Close() error {
	h.closeOnce.Do(func() {
		h.closeErr = releaseShared(h.key)
	})

	return h.closeErr
}

// NewDiskShared returns a handle onto the disk store for directory, opening it only if this process has not already.
//
// It exists because Badger's directory lock is per directory, is not reentrant, and cannot be waited on. A process that
// opens a path it already holds does not block - it fails, with an error that reads exactly like a different process
// holding the lock ("Another process is using this Badger database"). Anything whose setup can run twice therefore has
// a failure mode that looks like an environment problem and is not: a library re-initialized without being torn down
// first, a desktop app whose UI reloads while the backend keeps running, a test that opens a fixture per case. Every
// caller of NewDiskOnly has to solve that on its own, which means knowing to solve it. This makes it not arise.
//
// Each call returns its own handle, so each must be matched by exactly one Close; the store closes when the last one
// does. Handles share the store but not the memoization: Do deduplicates concurrent work per Memoizer, so two handles
// computing the same key at the same time may both compute it. Use one handle in one place for that to coalesce.
//
// The directory is resolved to an absolute path before it is matched, so "cache" and "/home/u/cache" from the same
// working directory are one store. It is a lexical comparison and not an inode one: two paths that reach the same
// directory through different symlinks are treated as different, and the second open fails on the lock as before.
//
// Sizing comes from whichever call opens the store; opts is ignored on a call that finds one already open, since the
// store cannot be resized underneath the holders it already has.
//
// # Example:
//
//	memoizer, err := NewDiskShared("/tmp/cache", CacheOpts{})
//	if err != nil {
//	    return err
//	}
//	defer memoizer.Close()
func NewDiskShared(directory string, opts CacheOpts) (*Memoizer, error) {
	key, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}

	sharedMu.Lock()
	defer sharedMu.Unlock()

	entry, open := sharedDisks[key]
	if !open {
		// Opened under the lock rather than beside it. Badger takes a wall-clock moment to open a store, and two
		// callers racing on the same fresh path would otherwise both find nothing, both open, and one would fail on
		// the other's lock - which is the exact failure this function exists to remove.
		store, err := internal.NewDiskStore(key, opts)
		if err != nil {
			return nil, err
		}

		entry = &sharedDisk{store: store}
		sharedDisks[key] = entry
	}

	entry.refs++

	return NewMemoizer(&sharedHandle{Store: entry.store, key: key}), nil
}

// releaseShared drops one reference to the store at key and closes it once none are left.
func releaseShared(key string) error {
	sharedMu.Lock()
	defer sharedMu.Unlock()

	entry, open := sharedDisks[key]
	if !open {
		return nil
	}

	entry.refs--
	if entry.refs > 0 {
		return nil
	}

	// Removed before the close rather than after: a failed close still releases the path, so a later NewDiskShared
	// opens a new store instead of handing out references to one that is already gone.
	delete(sharedDisks, key)

	return entry.store.Close()
}
