package internal

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/dgraph-io/ristretto/v2/z"
)

type DiskStore struct {
	db *badger.DB

	// Closed by Close before it queues on mu, so a sweep in flight bails out instead of making Close wait for it.
	// Doubles as the "this store is closing" flag.
	stop chan struct{}

	// Badger's Flatten stops and restarts the background compactors, and db.Close doesn't wait for the goroutines it
	// spawns; running the two concurrently crashes. This serializes them.
	mu        sync.Mutex
	closeOnce sync.Once
	closeErr  error
}

// Badger validates ValueLogFileSize against these bounds and refuses to open outside them, so a caller's MaxCapacity
// has to be clamped rather than passed straight through.
const (
	minValueLogFileSize = 1 << 20        // 1 MiB
	maxValueLogFileSize = 2<<30 - 1      // just under 2 GiB
	maxValueLogEntries  = math.MaxUint32 // WithValueLogMaxEntries takes a uint32
)

func NewDiskStore(path string, opts CacheOpts) (*DiskStore, error) {
	if opts.MaxEntries <= 0 {
		opts.MaxEntries = 1_000_000
	}
	if opts.MaxCapacity <= 0 {
		opts.MaxCapacity = 1 << 30 // 1 GiB
	}

	// Clamp instead of forwarding. Passing these through verbatim meant the values the documentation itself
	// suggested - 4 GiB, or anything under 1 MiB - made badger.Open fail outright, and a MaxEntries above the
	// uint32 range wrapped around silently.
	valueLogFileSize := min(max(opts.MaxCapacity, minValueLogFileSize), maxValueLogFileSize)
	valueLogEntries := uint32(min(opts.MaxEntries, maxValueLogEntries))

	db, err := badger.Open(badger.DefaultOptions(path).
		WithCompression(options.ZSTD).
		WithLogger(nil).
		WithDetectConflicts(false).
		WithIndexCacheSize(64 << 20).
		WithValueLogMaxEntries(valueLogEntries).
		WithValueLogFileSize(valueLogFileSize).
		// Drain level 0 on the way out. Badger won't compact a tree that still fits in a single level, so without
		// this the whole cache can sit in level 0 forever and Cleanup has nothing it's allowed to compact.
		WithCompactL0OnClose(true))
	if err != nil {
		return nil, err
	}

	s := &DiskStore{db: db, stop: make(chan struct{})}

	// Reclaim whatever the previous run left behind. This runs in the background so opening the store stays cheap; if
	// Close arrives first it wins the mutex and the sweep turns into a no-op.
	go func() {
		_ = s.Cleanup(context.Background()) // best-effort: a failed sweep must not fail startup
	}()

	return s, nil
}

func (s *DiskStore) Get(_ context.Context, key string) ([]byte, bool, error) {
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

func (s *DiskStore) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	return s.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), value).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

// Cleanup reclaims the disk space still held by entries whose TTL has expired, in three steps: delete the expired
// entries, compact the tree so the deleted data is dropped, then rewrite the value log files the compaction freed space
// in. It runs once automatically when the store is opened. See Memoizer.Cleanup for the user-facing contract.
//
// Compacting stops Badger's background compactors for the duration, and nothing drains level 0 while they're stopped.
// A store being written to at the same time therefore sees writes slow down — they queue rather than fail — and the
// sweep itself reclaims less, because it competes with the incoming data.
//
// Returns nil once there is nothing left worth reclaiming, the error that aborted the sweep, or ctx.Err() if the context
// is cancelled. Calling it on a closed store is a no-op and returns nil.
func (s *DiskStore) Cleanup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopping() {
		return nil
	}

	// Delete the expired entries explicitly. Compaction on its own only drops them from the levels it happens to
	// rewrite, which leaves the bottom of the tree untouched; the tombstones written here are the write pressure that
	// makes a later compaction reach down there and reclaim those too.
	if err := s.purgeExpired(ctx); err != nil {
		return err
	}

	// Compact next. This is what actually reclaims the space: it drops the expired keys the purge just marked, and
	// records which value log files hold discardable bytes. One worker matches Badger's own `badger flatten` default,
	// and must be at least 1 or Flatten never terminates.
	if err := s.db.Flatten(1); err != nil {
		return err
	}

	// Finally rewrite any value log file that is now mostly garbage, one per successful call, until Badger reports
	// there's nothing left worth rewriting. 0.5 is its recommended discard ratio: rewrite once half a file is garbage.
	// Note this is usually a single no-op call — values below Badger's 1 MiB threshold are stored inline in the LSM
	// tree and never reach the value log at all, so for a typical cache Flatten above did all the work.
	for {
		if stop, err := s.halt(ctx); stop {
			return err
		}

		if err := s.db.RunValueLogGC(0.5); err != nil {
			if errors.Is(err, badger.ErrNoRewrite) {
				return nil // nothing more to reclaim
			}
			return err
		}
	}
}

// purgeExpired writes an explicit delete for every entry whose TTL has passed, so that the next compaction can reclaim
// the space they occupy.
//
// It tests ExpiresAt directly instead of using Item.IsDeletedOrExpired, which also reports entries that are already
// tombstones; deleting those would pile tombstones on tombstones on every sweep and the store would never settle.
//
// Scanning runs through Badger's Stream, which splits the key space and walks it with several goroutines. That matters
// because entries below Badger's 1 MiB threshold are stored inline in the LSM tree, so surfacing a key means reading
// and decompressing the block holding its value too — a single-threaded pass costs seconds per gigabyte.
func (s *DiskStore) purgeExpired(ctx context.Context) error {
	now := uint64(time.Now().Unix())

	wb := s.db.NewWriteBatch()
	// Each batch commits automatically once it fills, and every one in flight holds its entries in memory until it
	// lands. The default of 16 is tuned for small batches, not the large ones a full sweep produces.
	wb.SetMaxPendingTxns(2)
	defer wb.Cancel()

	stream := s.db.NewStream()
	stream.LogPrefix = "memo.purgeExpired"

	// ChooseKey is called concurrently, once per key, always on its newest version — so a key that expired but has
	// since been written again is correctly left alone. Returning false means nothing is ever streamed or buffered,
	// which makes Send below a formality.
	stream.ChooseKey = func(item *badger.Item) bool {
		if e := item.ExpiresAt(); e != 0 && e <= now {
			_ = wb.Delete(item.KeyCopy(nil)) // safe from several goroutines; the batch holds its own lock
		}
		return false
	}
	stream.Send = func(*z.Buffer) error { return nil }

	if err := stream.Orchestrate(ctx); err != nil {
		return err
	}

	if s.stopping() {
		return nil // Close is waiting on us; don't flush tombstones into a store that's about to shut down
	}

	return wb.Flush()
}

// Close releases the store. It is safe to call more than once; every call returns the same error.
func (s *DiskStore) Close() error {
	s.closeOnce.Do(func() {
		close(s.stop) // signal an in-flight sweep before queuing behind it on mu
		s.mu.Lock()
		defer s.mu.Unlock()
		s.closeErr = s.db.Close()
	})

	return s.closeErr
}

// stopping reports whether Close has been called.
func (s *DiskStore) stopping() bool {
	select {
	case <-s.stop:
		return true
	default:
		return false
	}
}

// halt reports whether a sweep should stop: because the caller cancelled (returns ctx.Err()) or because Close was
// called (returns nil — an interrupted sweep is not a failure).
func (s *DiskStore) halt(ctx context.Context) (bool, error) {
	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-s.stop:
		return true, nil
	default:
		return false, nil
	}
}
