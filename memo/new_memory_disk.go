package memo

import (
	"time"

	"github.com/vegidio/go-sak/memo/internal"
)

// NewMemoryDisk creates a new Memoizer with a two-tier memory-disk composite store. The composite store uses an
// in-memory cache as the primary tier (L1) and a disk-based cache as the secondary tier (L2). Cache misses in memory
// are checked against disk, and successful disk hits are promoted back to memory.
//
// # Parameters:
//   - path: The filesystem path where the disk cache will be stored
//   - opts: Cache configuration options applied to both memory and disk stores
//   - promoteTTL: Time-to-live duration for entries promoted from disk to memory
//
// # Returns:
//   - *Memoizer: The configured memoizer instance
//   - func() error: A cleanup function that closes both stores and should be called when done
//   - error: Any error that occurred during initialization
//
// The function initializes both stores and ensures proper cleanup if either fails. The returned cleanup function should
// be deferred or called when the memoizer is no longer needed.
//
// # Example:
//
//	memoizer, cleanup, err := NewMemoryDisk("/tmp/cache", CacheOpts{}, time.Hour)
//	if err != nil {
//	    return err
//	}
//	defer cleanup()
func NewMemoryDisk(path string, opts CacheOpts, promoteTTL time.Duration) (*Memoizer, func() error, error) {
	mem, err := internal.NewMemoryStore(opts)
	if err != nil {
		return nil, nil, err
	}
	disk, err := internal.NewDiskStore(path, opts)

	if err != nil {
		mem.Close()
		return nil, nil, err
	}

	comp := internal.NewCompositeStore(mem, disk, promoteTTL)
	m := NewMemoizer(comp)
	closeAll := func() error { return comp.Close() }

	return m, closeAll, nil
}
