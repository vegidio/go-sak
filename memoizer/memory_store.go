package memoizer

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

type memoryStore struct {
	c *ristretto.Cache[string, []byte]
}

type MemoryOpts struct {
	// NumCounters ~ 10x expected items
	NumCounters int64
	// MaxCost is the capacity in your chosen units; we use value bytes as cost.
	MaxCost int64
	// BufferItems size; 64 is typical.
	BufferItems int64
	// IgnoreInternalCost if true, excludes internal bookkeeping from MaxCost.
	IgnoreInternalCost bool
	// Metrics enables hit ratio, etc.
	Metrics bool
}

func newMemoryStore(opts MemoryOpts) (*memoryStore, error) {
	if opts.NumCounters == 0 {
		opts.NumCounters = 1_000_000
	}
	if opts.MaxCost == 0 {
		opts.MaxCost = 1 << 30 // 1 GiB
	}
	if opts.BufferItems == 0 {
		opts.BufferItems = 64
	}

	c, err := ristretto.NewCache(&ristretto.Config[string, []byte]{
		NumCounters:        opts.NumCounters,
		MaxCost:            opts.MaxCost,
		BufferItems:        opts.BufferItems,
		IgnoreInternalCost: opts.IgnoreInternalCost,
		Metrics:            opts.Metrics,
	})
	if err != nil {
		return nil, err
	}

	return &memoryStore{c: c}, nil
}

func (m *memoryStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := m.c.Get(key)
	if !ok {
		return nil, false, nil
	}

	return v, true, nil
}

func (m *memoryStore) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	// Cost = byte length; adjust if you want different weighting
	_ = m.c.SetWithTTL(key, value, int64(len(value)), ttl) // eventual visibility; acceptable for caches
	// Optionally: m.c.Wait() if you need immediate visibility for tests.
	return nil
}

func (m *memoryStore) Close() error { m.c.Close(); return nil }
