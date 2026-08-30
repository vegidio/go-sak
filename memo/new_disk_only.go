package memo

import "github.com/vegidio/go-sak/memo/internal"

// NewDiskOnly creates a new Memoizer that uses disk-based storage only. The memoizer will persist cached values to disk
// in the specified directory using Badger as the underlying database engine.
//
// The directory parameter specifies the path where cached data will be stored. If the directory doesn't exist, it will
// be created automatically.
//
// The opts parameter tunes the store's sizing. Both fields are hints rather than hard limits and are clamped to what
// Badger accepts, so neither can prevent the store from opening:
//   - MaxEntries: entries per value-log file, which shapes how often those files roll over (defaults to 1,000,000)
//   - MaxCapacity: the value-log file size, clamped to [1 MiB, 2 GiB) (defaults to 1 GiB)
//
// Neither option caps the cache on disk. Bound its growth with entry TTLs and Memoizer.Cleanup instead.
//
// Returns a pointer to the newly created Memoizer configured with disk storage, or an error if the disk store
// initialization fails (e.g., due to permission issues or invalid directory path).
//
// A background sweep starts as soon as the store opens, reclaiming the disk space held by entries that expired during
// previous runs. It doesn't delay this call, and Close interrupts it rather than waiting for it to finish. Use
// Memoizer.Cleanup to trigger the same sweep at a moment of your choosing.
//
// # Example:
//
//	opts := CacheOpts{
//		MaxEntries:  500000,
//		MaxCapacity: 512 << 20, // 512 MiB
//	}
//	memoizer, err := NewDiskOnly("/tmp/cache", opts)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer memoizer.Close()
func NewDiskOnly(directory string, opts CacheOpts) (*Memoizer, error) {
	d, err := internal.NewDiskStore(directory, opts)
	if err != nil {
		return nil, err
	}

	return NewMemoizer(d), nil
}
