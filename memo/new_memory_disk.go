package memo

import (
	"time"

	"github.com/vegidio/go-sak/memo/internal"
)

func NewMemoryDisk(path string, opts internal.CacheOpts, promoteTTL time.Duration) (*Memoizer, func() error, error) {
	mem, err := internal.NewMemoryStore(opts)
	if err != nil {
		return nil, nil, err
	}
	disk, err := internal.NewDiskStore(path, opts)

	if err != nil {
		mem.Close()
		return nil, nil, err
	}

	comp := internal.NewCompositeStore(mem, disk, promoteTTL)
	m := NewMemoizer(comp)
	closeAll := func() error { return comp.Close() }

	return m, closeAll, nil
}
