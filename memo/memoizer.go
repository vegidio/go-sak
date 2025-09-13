package memo

import (
	"github.com/vegidio/go-sak/memo/internal"
	"golang.org/x/sync/singleflight"
)

type Memoizer struct {
	Store internal.Store
	Sf    singleflight.Group
}

func NewMemoizer(store internal.Store) *Memoizer {
	return &Memoizer{Store: store}
}

func (m *Memoizer) Close() error {
	return m.Store.Close()
}
