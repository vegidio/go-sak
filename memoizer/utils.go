package memoizer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"time"
)

func KeyFrom(parts ...any) string {
	h := sha256.New()
	enc := gob.NewEncoder(h)
	for _, p := range parts {
		_ = enc.Encode(p) // best-effort
	}

	return hex.EncodeToString(h.Sum(nil))
}

func Do[T any](
	m *Memoizer,
	ctx context.Context,
	key string,
	ttl time.Duration,
	compute func(context.Context) (T, error),
) (T, error) {
	var zero T

	// Try cache
	if b, ok, err := m.store.Get(ctx, key); err != nil {
		return zero, err
	} else if ok {
		if v, dErr := decodeGob[T](b); dErr == nil {
			return v, nil
		}
	}

	// Deduplicate concurrent misses
	val, err, _ := m.sf.Do(key, func() (any, error) {
		// Recheck inside singleflight
		if b, ok, err := m.store.Get(ctx, key); err == nil && ok {
			if v, e := decodeGob[T](b); e == nil {
				return v, nil
			}
		}

		// Compute fresh
		res, err := compute(ctx)
		if err != nil {
			return zero, err
		}

		if payload, e := encodeGob(res); e == nil {
			_ = m.store.Set(ctx, key, payload, ttl) // best-effort cache write
		}

		return res, nil
	})
	if err != nil {
		return zero, err
	}

	return val.(T), nil
}

func NewMemoryOnly(opts MemoryOpts) (*Memoizer, error) {
	mem, err := newMemoryStore(opts)
	if err != nil {
		return nil, err
	}

	return newMemoizer(mem), nil
}

func NewDiskOnly(path string) (*Memoizer, error) {
	d, err := newDiskStore(path)
	if err != nil {
		return nil, err
	}

	return newMemoizer(d), nil
}

func NewMemoryDisk(path string, memOpts MemoryOpts, promoteTTL time.Duration) (*Memoizer, func() error, error) {
	mem, err := newMemoryStore(memOpts)
	if err != nil {
		return nil, nil, err
	}
	disk, err := newDiskStore(path)

	if err != nil {
		mem.Close()
		return nil, nil, err
	}

	comp := newCompositeStore(mem, disk, promoteTTL)
	m := newMemoizer(comp)
	closeAll := func() error { return comp.Close() }

	return m, closeAll, nil
}

// region - Private methods

func encodeGob[T any](v T) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeGob[T any](data []byte) (T, error) {
	var v T
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&v)
	return v, err
}

// endregion
