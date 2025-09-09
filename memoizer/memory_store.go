package memoizer

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

type memoryStore struct {
	c *ristretto.Cache[string, []byte]
}

func newMemoryStore(opts CacheOpts) (*memoryStore, error) {
	if opts.MaxEntries == 0 {
		opts.MaxEntries = 1_000_000
	}
	if opts.MaxCapacity == 0 {
		opts.MaxCapacity = 1 << 30 // 1 GiB
	}

	c, err := ristretto.NewCache(&ristretto.Config[string, []byte]{
		NumCounters: opts.MaxEntries,
		MaxCost:     opts.MaxCapacity,
		BufferItems: 64,
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
