package memo

import (
	"context"

	"github.com/vegidio/go-sak/memo/internal"
	"golang.org/x/sync/singleflight"
)

type Memoizer struct {
	Store internal.Store

	// sf is unexported because singleflight.Group embeds a sync.Mutex, which made a Memoizer copied by value
	// silently unsafe. It is an implementation detail of Do in any case.
	sf singleflight.Group
}

// NewMemoizer creates a new Memoizer instance with the provided store. The store parameter defines the underlying
// storage mechanism for cached values.
//
// Returns a pointer to the newly created Memoizer.
func NewMemoizer(store internal.Store) *Memoizer {
	return &Memoizer{Store: store}
}

// pathed is implemented by the stores that live in a directory. It is deliberately not part of the Store interface: a
// memory store has no directory, and forcing it to report an empty one would make "" ambiguous between "no disk" and
// "a disk store that somehow lost its path".
type pathed interface {
	Path() string
}

// Path reports the directory backing this memoizer, or an empty string for one that keeps nothing on disk.
//
// It answers the question a process has to settle before opening a store a second time. Badger's directory lock is per
// directory and is not reentrant, so a process that has already opened a path cannot open it again - it deadlocks
// against itself, and the error looks exactly like another process holding the lock. Comparing this against the path
// about to be opened is what tells those two apart. See NewDiskShared, which removes the need to ask at all.
func (m *Memoizer) Path() string {
	if p, ok := m.Store.(pathed); ok {
		return p.Path()
	}

	return ""
}

// Cleanup reclaims the storage still held by entries whose TTL has expired.
//
// Expired entries are never served, but on disk the space they occupy isn't freed automatically: the underlying
// database only discards them while compacting, and compaction is driven by write volume rather than by the clock. A
// cache that goes idle therefore keeps growing. This forces that reclamation to happen.
//
// Disk-backed memoizers already run this once when they are created, so calling it is optional. Call it to reclaim
// space at a specific moment, e.g. before shutting down or after expiring a large batch of entries. For memory-only
// memoizers it does nothing and returns nil.
//
// The sweep is best-effort: some entries are only reclaimed by a later pass. Prefer a quiet moment — while it runs,
// writes to the cache queue up behind it and get slower (they don't fail), and a store busy taking writes reclaims
// less.
//
// Returns an error if the underlying store fails to reclaim, or ctx.Err() if the context is cancelled.
//
// # Example:
//
//	if err := memoizer.Cleanup(ctx); err != nil {
//		log.Printf("cache cleanup failed: %v", err)
//	}
func (m *Memoizer) Cleanup(ctx context.Context) error {
	return m.Store.Cleanup(ctx)
}

// Close closes the Memoizer and releases any resources held by the underlying store. This method should be called when
// the Memoizer is no longer needed to ensure proper cleanup.
//
// Returns an error if the underlying store's Close operation fails, nil otherwise.
func (m *Memoizer) Close() error {
	return m.Store.Close()
}
