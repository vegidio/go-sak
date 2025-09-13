package internal

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

type MemoryStore struct {
	c *ristretto.Cache[string, []byte]
}

func NewMemoryStore(opts CacheOpts) (*MemoryStore, error) {
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

	return &MemoryStore{c: c}, nil
}

func (m *MemoryStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := m.c.Get(key)
	if !ok {
		return nil, false, nil
	}

	return v, true, nil
}

func (m *MemoryStore) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	// Cost = byte length; adjust if you want different weighting
	_ = m.c.SetWithTTL(key, value, int64(len(value)), ttl)
	// Ensure immediate visibility (tests rely on this); Ristretto writes are async otherwise
	m.c.Wait()
	return nil
}

func (m *MemoryStore) Close() error { m.c.Close(); return nil }
