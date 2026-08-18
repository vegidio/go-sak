package internal

import (
	"context"
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

type CacheOpts struct {
	// MaxEntries is the max number of entries to store.
	MaxEntries int64
	// MaxCapacity is the max capacity in bytes.
	MaxCapacity int64
}
