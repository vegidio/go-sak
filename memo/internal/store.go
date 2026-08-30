package internal

import (
	"context"
	"errors"
	"time"
)

type Store interface {
	Get(ctx context.Context, key string) (value []byte, ok bool, err error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Cleanup reclaims the storage held by entries whose TTL has expired. Stores with nothing to reclaim return nil.
	Cleanup(ctx context.Context) error

	// Close releases the store. Implementations must be safe to close more than once, returning the same error each
	// time, so that layered stores and deferred cleanups can't fail on a second call.
	Close() error
}

// CacheOpts tunes a store's sizing. Both fields are hints rather than hard limits, and both are clamped to whatever
// the underlying engine accepts, so no value here can stop a store from opening.
type CacheOpts struct {
	// MaxEntries sizes the store for roughly this many entries. Zero picks a default.
	//
	// For the memory store this is Ristretto's counter count. For the disk store it bounds the entries held in
	// one value-log file, so it shapes how often those files roll over rather than capping the cache.
	MaxEntries int64

	// MaxCapacity sizes the store for roughly this many bytes. Zero picks a default of 1 GiB.
	//
	// For the memory store this is a real ceiling: Ristretto evicts to stay under it. For the disk store it sets
	// the value-log file size, clamped to the [1 MiB, 2 GiB) range Badger accepts; the disk cache is bounded by
	// entry TTLs and Cleanup rather than by a byte budget.
	MaxCapacity int64
}

// ErrNotAdmitted is returned by a Set that the cache's admission policy declined. The value is simply not cached; the
// caller's computation is unaffected, which is why Do treats a cache write as best-effort.
var ErrNotAdmitted = errors.New("memo: value not admitted to the cache")
