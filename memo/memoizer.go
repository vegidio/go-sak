package memo

import (
	"github.com/vegidio/go-sak/memo/internal"
	"golang.org/x/sync/singleflight"
)

type Memoizer struct {
	Store internal.Store
	Sf    singleflight.Group
}

// NewMemoizer creates a new Memoizer instance with the provided store. The store parameter defines the underlying
// storage mechanism for cached values.
//
// Returns a pointer to the newly created Memoizer.
func NewMemoizer(store internal.Store) *Memoizer {
	return &Memoizer{Store: store}
}

// Close closes the Memoizer and releases any resources held by the underlying store. This method should be called when
// the Memoizer is no longer needed to ensure proper cleanup.
//
// Returns an error if the underlying store's Close operation fails, nil otherwise.
func (m *Memoizer) Close() error {
	return m.Store.Close()
}
