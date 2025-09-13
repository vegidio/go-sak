package memo

import "github.com/vegidio/go-sak/memo/internal"

// NewDiskOnly creates a new Memoizer that uses disk-based storage only. The memoizer will persist cached values to disk
// in the specified directory using Badger as the underlying database engine.
//
// The directory parameter specifies the path where cached data will be stored. If the directory doesn't exist, it will
// be created automatically.
//
// The opts parameter allows customization of cache behavior:
//   - MaxEntries: maximum number of entries to store (defaults to 1,000,000 if not specified)
//   - MaxCapacity: maximum storage capacity in bytes (defaults to 1 GiB if not specified)
//
// Returns a pointer to the newly created Memoizer configured with disk storage, or an error if the disk store
// initialization fails (e.g., due to permission issues or invalid directory path).
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
