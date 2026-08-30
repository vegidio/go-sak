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
