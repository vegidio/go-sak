package internal

import (
	"context"
	"time"
)

type Store interface {
	Get(ctx context.Context, key string) (value []byte, ok bool, err error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Close() error
}

type CacheOpts struct {
	// MaxEntries is the max number of entries to store.
	MaxEntries int64
	// MaxCapacity is the max capacity in bytes.
	MaxCapacity int64
}
