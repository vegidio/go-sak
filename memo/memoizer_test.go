package memo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vegidio/go-sak/memo/internal"
)

// MockStore is a mock implementation of internal.Store interface
type MockStore struct {
	mock.Mock
}

func (m *MockStore) Get(ctx context.Context, key string) (value []byte, ok bool, err error) {
	args := m.Called(ctx, key)
	return args.Get(0).([]byte), args.Bool(1), args.Error(2)
}

func (m *MockStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewMemoizer(t *testing.T) {
	tests := []struct {
		name  string
		store internal.Store
	}{
		{
			name:  "with mock store",
			store: &MockStore{},
		},
		{
			name:  "with nil store",
			store: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memoizer := NewMemoizer(tt.store)

			assert.NotNil(t, memoizer)
			assert.Equal(t, tt.store, memoizer.Store)
			assert.NotNil(t, memoizer.Sf) // singleflight.Group should be initialized
		})
	}
}

func TestMemoizer_Close(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockStore)
		expectError bool
		expectedErr error
	}{
		{
			name: "successful close",
			setupMock: func(m *MockStore) {
				m.On("Close").Return(nil)
			},
			expectError: false,
		},
		{
			name: "close with error",
			setupMock: func(m *MockStore) {
				m.On("Close").Return(errors.New("close error"))
			},
			expectError: true,
			expectedErr: errors.New("close error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &MockStore{}
			tt.setupMock(mockStore)

			memoizer := NewMemoizer(mockStore)
			err := memoizer.Close()

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			mockStore.AssertExpectations(t)
		})
	}
}

func TestMemoizer_Fields(t *testing.T) {
	mockStore := &MockStore{}
	memoizer := NewMemoizer(mockStore)

	t.Run("Store field is accessible", func(t *testing.T) {
		assert.Equal(t, mockStore, memoizer.Store)
	})

	t.Run("Singleflight field is accessible", func(t *testing.T) {
		assert.NotNil(t, memoizer.Sf)
		// Verify it's actually a singleflight.Group by checking we can call methods on it
		// We'll do this by ensuring the field exists and is of the correct type
		_, _, _ = memoizer.Sf.Do("test", func() (interface{}, error) {
			return nil, nil
		})
	})
}

func TestMemoizer_Integration(t *testing.T) {
	t.Run("create memoizer and close successfully", func(t *testing.T) {
		mockStore := &MockStore{}
		mockStore.On("Close").Return(nil)

		memoizer := NewMemoizer(mockStore)
		assert.NotNil(t, memoizer)

		err := memoizer.Close()
		assert.NoError(t, err)

		mockStore.AssertExpectations(t)
	})

	t.Run("multiple operations on same memoizer", func(t *testing.T) {
		mockStore := &MockStore{}
		mockStore.On("Close").Return(nil)

		memoizer := NewMemoizer(mockStore)

		// Verify we can access fields multiple times
		assert.Equal(t, mockStore, memoizer.Store)
		assert.NotNil(t, memoizer.Sf)
		assert.Equal(t, mockStore, memoizer.Store) // Second access

		// Close should work
		err := memoizer.Close()
		assert.NoError(t, err)

		mockStore.AssertExpectations(t)
	})
}
