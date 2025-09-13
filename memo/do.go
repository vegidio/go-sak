package memo

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"
)

// Do executes a memoized computation with the given key and TTL.
//
// This function implements a memoization pattern with caching and deduplication:
//  1. First checks if the result exists in the cache
//  2. If not cached, uses singleflight to deduplicate concurrent calls with the same key
//  3. Executes the compute function and caches the result with the specified TTL
//
// The function is generic and works with any type T that can be encoded/decoded with gob.
//
// # Parameters:
//   - m: The Memoizer instance containing the cache store and singleflight group
//   - ctx: Context for cancellation and timeouts
//   - key: Unique identifier for caching the computation result
//   - ttl: Time-to-live duration for the cached result
//   - compute: Function that performs the actual computation, called only on cache miss
//
// Returns the computed result of type T and any error that occurred during cache retrieval, computation, or
// encoding/decoding.
//
// If the cache retrieval fails, the error is returned immediately; if the compute function fails, its error is
// returned; cache write failures are ignored (best-effort caching).
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
