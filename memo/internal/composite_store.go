package internal

import (
	"context"
	"sync"
	"time"
)

type CompositeStore struct {
	mem    Store
	disk   Store
	hotTTL time.Duration // TTL for memory promotion

	closeOnce sync.Once
	closeErr  error
}

func NewCompositeStore(mem, disk Store, hotTTL time.Duration) *CompositeStore {
	return &CompositeStore{mem: mem, disk: disk, hotTTL: hotTTL}
}

func (s *CompositeStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if s.mem != nil {
		if b, ok, err := s.mem.Get(ctx, key); err != nil {
			return nil, false, err
		} else if ok {
			return b, true, nil
		}
	}

	if s.disk != nil {
		if b, ok, err := s.disk.Get(ctx, key); err != nil {
			return nil, false, err
		} else if ok {
			if s.mem != nil && s.hotTTL > 0 {
				_ = s.mem.Set(ctx, key, b, s.hotTTL) // promote best-effort
			}
			return b, true, nil
		}
	}

	return nil, false, nil
}

func (s *CompositeStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return firstErr([]Store{s.disk, s.mem}, func(st Store) error {
		return st.Set(ctx, key, value, ttl)
	})
}

func (s *CompositeStore) Cleanup(ctx context.Context) error {
	return firstErr([]Store{s.mem, s.disk}, func(st Store) error {
		return st.Cleanup(ctx)
	})
}

// Close releases both tiers. It is safe to call more than once; every call returns the same error.
func (s *CompositeStore) Close() error {
	s.closeOnce.Do(func() {
		s.closeErr = firstErr([]Store{s.mem, s.disk}, Store.Close)
	})

	return s.closeErr
}

// firstErr runs fn against each store in order, skipping absent tiers, and reports the first error. Every store is
// visited even after one fails, so a broken tier can't stop the others from being reached.
func firstErr(stores []Store, fn func(Store) error) error {
	var first error
	for _, st := range stores {
		if st == nil {
			continue
		}

		if err := fn(st); err != nil && first == nil {
			first = err
		}
	}

	return first
}
