package internal

import (
	"context"
	"time"
)

type CompositeStore struct {
	mem    Store
	disk   Store
	hotTTL time.Duration // TTL for memory promotion
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
	var firstErr error
	if s.disk != nil {
		if err := s.disk.Set(ctx, key, value, ttl); err != nil {
			firstErr = err
		}
	}

	if s.mem != nil {
		if err := s.mem.Set(ctx, key, value, ttl); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (s *CompositeStore) Close() error {
	var firstErr error
	if s.mem != nil {
		if err := s.mem.Close(); err != nil {
			firstErr = err
		}
	}

	if s.disk != nil {
		if err := s.disk.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
