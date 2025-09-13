package memo

import "github.com/vegidio/go-sak/memo/internal"

func NewMemoryOnly(opts internal.CacheOpts) (*Memoizer, error) {
	mem, err := internal.NewMemoryStore(opts)
	if err != nil {
		return nil, err
	}

	return NewMemoizer(mem), nil
}
