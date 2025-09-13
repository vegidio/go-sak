package memo

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"
)

func Do[T any](
	m *Memoizer,
	ctx context.Context,
	key string,
	ttl time.Duration,
	compute func(context.Context) (T, error),
) (T, error) {
	var zero T

	// Try cache
	if b, ok, err := m.Store.Get(ctx, key); err != nil {
		return zero, err
	} else if ok {
		if v, dErr := decodeGob[T](b); dErr == nil {
			return v, nil
		}
	}

	// Deduplicate concurrent misses
	val, err, _ := m.Sf.Do(key, func() (any, error) {
		// Recheck inside singleflight
		if b, ok, err := m.Store.Get(ctx, key); err == nil && ok {
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
			_ = m.Store.Set(ctx, key, payload, ttl) // best-effort cache write
		}

		return res, nil
	})
	if err != nil {
		return zero, err
	}

	return val.(T), nil
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
