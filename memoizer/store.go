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
