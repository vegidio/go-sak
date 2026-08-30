package internal

import (
	"context"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

type MemoryStore struct {
	c *ristretto.Cache[string, []byte]

	closeOnce sync.Once
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

	// Hand back a copy, as DiskStore does. Returning the cached slice itself lets a caller that writes into it
	// corrupt the entry for every other goroutine holding the same key.
	out := make([]byte, len(v))
	copy(out, v)

	return out, true, nil
}

func (m *MemoryStore) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	// Cost = byte length; adjust if you want different weighting
	if !m.c.SetWithTTL(key, value, int64(len(value)), ttl) {
		// Ristretto's admission policy rejected the write. That is not an error - the cache simply decided the
		// entry wasn't worth keeping - but it must not be reported as a successful store either.
		return ErrNotAdmitted
	}

	// Ristretto applies writes asynchronously. Draining the buffer here keeps a read immediately after a write
	// from missing, which is what callers of a memoizer expect.
	m.c.Wait()

	return nil
}

// Cleanup is a no-op. Ristretto evicts expired entries on its own and holds nothing on disk to reclaim.
func (m *MemoryStore) Cleanup(_ context.Context) error { return nil }

// Close releases the cache. It is safe to call more than once.
func (m *MemoryStore) Close() error {
	m.closeOnce.Do(m.c.Close)
	return nil
}
