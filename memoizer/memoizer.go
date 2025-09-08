package memoizer

import "golang.org/x/sync/singleflight"

type Memoizer struct {
	store Store
	sf    singleflight.Group
}

func newMemoizer(store Store) *Memoizer { return &Memoizer{store: store} }

func (m *Memoizer) Close() error { return m.store.Close() }
