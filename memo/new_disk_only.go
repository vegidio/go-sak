package memo

import "github.com/vegidio/go-sak/memo/internal"

func NewDiskOnly(directory string, opts internal.CacheOpts) (*Memoizer, error) {
	d, err := internal.NewDiskStore(directory, opts)
	if err != nil {
		return nil, err
	}

	return NewMemoizer(d), nil
}
