package memo

import "github.com/vegidio/go-sak/memo/internal"

// NewMemoryOnly creates a new Memoizer instance that uses only in-memory storage. This function initializes a
// memory-based cache store with the provided options and wraps it in a Memoizer for memoization functionality.
//
// The opts parameter allows customization of cache behavior:
//   - MaxEntries: maximum number of entries to store (defaults to 1,000,000 if not specified)
//   - MaxCapacity: maximum storage capacity in bytes (defaults to 1 GiB if not specified)
//
// Returns a pointer to the newly created Memoizer instance and nil error on success, or an error if the memory store
// initialization fails.
//
// # Example:
//
//	opts := CacheOpts{
//		MaxEntries:  1000,
//		MaxCapacity: 1024 * 1024, // 1MB
//	}
//	memoizer, err := NewMemoryOnly(opts)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer memoizer.Close()
func NewMemoryOnly(opts CacheOpts) (*Memoizer, error) {
	mem, err := internal.NewMemoryStore(opts)
	if err != nil {
		return nil, err
	}

	return NewMemoizer(mem), nil
}
