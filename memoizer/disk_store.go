package memoizer

import (
	"context"
	"errors"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

type diskStore struct{ db *badger.DB }

func newDiskStore(path string, opts CacheOpts) (*diskStore, error) {
	if opts.MaxEntries == 0 {
		opts.MaxEntries = 1_000_000
	}
	if opts.MaxCapacity == 0 {
		opts.MaxCapacity = 1 << 30 // 1 GiB
	}

	db, err := badger.Open(badger.DefaultOptions(path).
		WithCompression(options.ZSTD).
		WithLogger(nil).
		WithDetectConflicts(false).
		WithIndexCacheSize(64 << 20).
		WithValueLogMaxEntries(uint32(opts.MaxEntries)).
		WithValueLogFileSize(opts.MaxCapacity))
	if err != nil {
		return nil, err
	}

	// Run value log GC
	db.RunValueLogGC(0.5)

	return &diskStore{db: db}, nil
}

func (s *diskStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	var out []byte
	err := s.db.View(func(txn *badger.Txn) error {
		it, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return it.Value(func(val []byte) error {
			out = append(out[:0], val...) // copy out
			return nil
		})
	})

	if err == nil {
		return out, true, nil
	}

	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, false, nil
	}

	return nil, false, err
}

func (s *diskStore) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	return s.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), value).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

func (s *diskStore) Close() error { return s.db.Close() }
