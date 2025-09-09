package memoizer

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
	// MaxEntries ~ 10x expected items
	MaxEntries int64
	// MaxCapacity is the capacity in your chosen units; we use value bytes as cost.
	MaxCapacity int64
}
