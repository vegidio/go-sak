package internal

import (
	"context"
	"errors"
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

// Path reports the directory of the disk tier, so a composite store answers the question as usefully as a plain disk
// one does. It is empty when there is no disk tier, or when that tier has no directory of its own.
func (s *CompositeStore) Path() string {
	if p, ok := s.disk.(interface{ Path() string }); ok {
		return p.Path()
	}

	return ""
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
	return joinErrs([]Store{s.disk, s.mem}, func(st Store) error {
		return st.Set(ctx, key, value, ttl)
	})
}

func (s *CompositeStore) Cleanup(ctx context.Context) error {
	return joinErrs([]Store{s.mem, s.disk}, func(st Store) error {
		return st.Cleanup(ctx)
	})
}

// Close releases both tiers. It is safe to call more than once; every call returns the same error.
func (s *CompositeStore) Close() error {
	s.closeOnce.Do(func() {
		s.closeErr = joinErrs([]Store{s.mem, s.disk}, Store.Close)
	})

	return s.closeErr
}

// joinErrs runs fn against each store in order, skipping absent tiers, and reports every failure. Every store is
// visited even after one fails, so a broken tier can't stop the others from being reached.
//
// All errors are reported rather than just the first: keeping only the first meant a memory-tier failure masked an
// unflushed disk tier, which is the one that actually loses data.
func joinErrs(stores []Store, fn func(Store) error) error {
	var errs []error
	for _, st := range stores {
		if st == nil {
			continue
		}

		errs = append(errs, fn(st))
	}

	return errors.Join(errs...)
}
