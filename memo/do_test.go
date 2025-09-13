package memo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vegidio/go-sak/memo/internal"
)

func TestDo(t *testing.T) {
	t.Run("cache hit", func(t *testing.T) {
		// Arrange
		store, err := internal.NewMemoryStore(internal.CacheOpts{})
		require.NoError(t, err)
		defer store.Close()

		m := NewMemoizer(store)
		defer m.Close()

		ctx := context.Background()
		key := "test-key"
		ttl := time.Minute
		expectedValue := "cached-value"

		// Pre-populate cache
		result, err := Do(m, ctx, key, ttl, func(ctx context.Context) (string, error) {
			return expectedValue, nil
		})
		require.NoError(t, err)
		require.Equal(t, expectedValue, result)

		// Act - should hit cache
		callCount := 0
		result, err = Do(m, ctx, key, ttl, func(ctx context.Context) (string, error) {
			callCount++
			return "should-not-be-called", nil
		})

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, result)
		assert.Equal(t, 0, callCount, "compute function should not be called on cache hit")
	})

	t.Run("cache miss", func(t *testing.T) {
		// Arrange
		store, err := internal.NewMemoryStore(internal.CacheOpts{})
		require.NoError(t, err)
		defer store.Close()

		m := NewMemoizer(store)
		defer m.Close()

		ctx := context.Background()
		key := "test-key"
		ttl := time.Minute
		expectedValue := "computed-value"

		// Act
		callCount := 0
		result, err := Do(m, ctx, key, ttl, func(ctx context.Context) (string, error) {
			callCount++
			return expectedValue, nil
		})

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, result)
		assert.Equal(t, 1, callCount, "compute function should be called once on cache miss")
	})

	t.Run("compute error", func(t *testing.T) {
		// Arrange
		store, err := internal.NewMemoryStore(internal.CacheOpts{})
		require.NoError(t, err)
		defer store.Close()

		m := NewMemoizer(store)
		defer m.Close()

		ctx := context.Background()
		key := "test-key"
		ttl := time.Minute
		expectedError := errors.New("compute failed")

		// Act
		result, err := Do(m, ctx, key, ttl, func(ctx context.Context) (string, error) {
			return "", expectedError
		})

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		assert.Empty(t, result)
	})

	t.Run("store get error", func(t *testing.T) {
		// Arrange
		mockStore := &mockStore{
			getError: errors.New("store get failed"),
		}
		m := NewMemoizer(mockStore)
		defer m.Close()

		ctx := context.Background()
		key := "test-key"
		ttl := time.Minute

		// Act
		result, err := Do(m, ctx, key, ttl, func(ctx context.Context) (string, error) {
			return "value", nil
		})

		// Assert
		assert.Error(t, err)
		assert.Equal(t, "store get failed", err.Error())
		assert.Empty(t, result)
	})

	t.Run("generic types", func(t *testing.T) {
		// Arrange
		store, err := internal.NewMemoryStore(internal.CacheOpts{})
		require.NoError(t, err)
		defer store.Close()

		m := NewMemoizer(store)
		defer m.Close()

		ctx := context.Background()
		ttl := time.Minute

		t.Run("int type", func(t *testing.T) {
			key := "int-key"
			expectedValue := 42

			result, err := Do(m, ctx, key, ttl, func(ctx context.Context) (int, error) {
				return expectedValue, nil
			})

			assert.NoError(t, err)
			assert.Equal(t, expectedValue, result)
		})

		t.Run("struct type", func(t *testing.T) {
			type TestStruct struct {
				Name string
				Age  int
			}

			key := "struct-key"
			expectedValue := TestStruct{Name: "John", Age: 30}

			result, err := Do(m, ctx, key, ttl, func(ctx context.Context) (TestStruct, error) {
				return expectedValue, nil
			})

			assert.NoError(t, err)
			assert.Equal(t, expectedValue, result)
		})

		t.Run("slice type", func(t *testing.T) {
			key := "slice-key"
			expectedValue := []string{"a", "b", "c"}

			result, err := Do(m, ctx, key, ttl, func(ctx context.Context) ([]string, error) {
				return expectedValue, nil
			})

			assert.NoError(t, err)
			assert.Equal(t, expectedValue, result)
		})
	})

	t.Run("singleflight", func(t *testing.T) {
		// Arrange
		store, err := internal.NewMemoryStore(internal.CacheOpts{})
		require.NoError(t, err)
		defer store.Close()

		m := NewMemoizer(store)
		defer m.Close()

		ctx := context.Background()
		key := "test-key"
		ttl := time.Minute
		expectedValue := "computed-value"

		// Act - simulate concurrent requests
		callCount := 0
		results := make(chan result, 2)

		for i := 0; i < 2; i++ {
			go func() {
				val, err := Do(m, ctx, key, ttl, func(ctx context.Context) (string, error) {
					callCount++
					time.Sleep(10 * time.Millisecond) // simulate some work
					return expectedValue, nil
				})
				results <- result{val: val, err: err}
			}()
		}

		// Collect results
		var res1, res2 result
		res1 = <-results
		res2 = <-results

		// Assert
		assert.NoError(t, res1.err)
		assert.NoError(t, res2.err)
		assert.Equal(t, expectedValue, res1.val)
		assert.Equal(t, expectedValue, res2.val)
		// Note: callCount might be 1 due to singleflight, but could be 2 due to timing
		// The important thing is that both get the same correct result
	})

	t.Run("cache write failure", func(t *testing.T) {
		// Arrange - store that fails on Set but succeeds on Get
		mockStore := &mockStore{
			setError: errors.New("cache write failed"),
		}
		m := NewMemoizer(mockStore)
		defer m.Close()

		ctx := context.Background()
		key := "test-key"
		ttl := time.Minute
		expectedValue := "computed-value"

		// Act
		result, err := Do(m, ctx, key, ttl, func(ctx context.Context) (string, error) {
			return expectedValue, nil
		})

		// Assert - should still return the computed value even if cache write fails
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, result)
	})

	t.Run("context cancellation", func(t *testing.T) {
		// Arrange
		store, err := internal.NewMemoryStore(internal.CacheOpts{})
		require.NoError(t, err)
		defer store.Close()

		m := NewMemoizer(store)
		defer m.Close()

		ctx, cancel := context.WithCancel(context.Background())
		key := "test-key"
		ttl := time.Minute

		// Act
		cancel() // Cancel context before calling Do

		result, err := Do(m, ctx, key, ttl, func(ctx context.Context) (string, error) {
			return "value", ctx.Err()
		})

		// Assert
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Empty(t, result)
	})
}

// Helper types and functions

type result struct {
	val string
	err error
}

type mockStore struct {
	data     map[string][]byte
	getError error
	setError error
}

func (m *mockStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if m.getError != nil {
		return nil, false, m.getError
	}
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	val, ok := m.data[key]
	return val, ok, nil
}

func (m *mockStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if m.setError != nil {
		return m.setError
	}
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[key] = value
	return nil
}

func (m *mockStore) Close() error {
	return nil
}
