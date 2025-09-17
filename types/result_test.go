package types

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CustomError is a custom error type for testing
type CustomError struct {
	Code    int
	Message string
}

func (e CustomError) Error() string {
	return e.Message
}

func TestResult_IsSuccess(t *testing.T) {
	t.Run("returns true when no error", func(t *testing.T) {
		result := Result[string]{
			Data: "test data",
			Err:  nil,
		}

		assert.True(t, result.IsSuccess())
	})

	t.Run("returns false when error exists", func(t *testing.T) {
		result := Result[string]{
			Data: "test data",
			Err:  errors.New("operation failed"),
		}

		assert.False(t, result.IsSuccess())
	})

	t.Run("returns true with zero value data and no error", func(t *testing.T) {
		result := Result[string]{
			Data: "",
			Err:  nil,
		}

		assert.True(t, result.IsSuccess())
	})

	t.Run("returns true with nil pointer data and no error", func(t *testing.T) {
		result := Result[*string]{
			Data: nil,
			Err:  nil,
		}

		assert.True(t, result.IsSuccess())
	})
}

func TestResult_WithDifferentTypes(t *testing.T) {
	t.Run("works with int type", func(t *testing.T) {
		successResult := Result[int]{
			Data: 42,
			Err:  nil,
		}
		assert.True(t, successResult.IsSuccess())
		assert.Equal(t, 42, successResult.Data)

		errorResult := Result[int]{
			Data: 0,
			Err:  errors.New("failed"),
		}
		assert.False(t, errorResult.IsSuccess())
		assert.Equal(t, 0, errorResult.Data)
		require.Error(t, errorResult.Err)
		assert.Equal(t, "failed", errorResult.Err.Error())
	})

	t.Run("works with struct type", func(t *testing.T) {
		type TestStruct struct {
			Name string
			Age  int
		}

		data := TestStruct{Name: "John", Age: 30}
		result := Result[TestStruct]{
			Data: data,
			Err:  nil,
		}

		assert.True(t, result.IsSuccess())
		assert.Equal(t, "John", result.Data.Name)
		assert.Equal(t, 30, result.Data.Age)
	})

	t.Run("works with slice type", func(t *testing.T) {
		data := []string{"apple", "banana", "cherry"}
		result := Result[[]string]{
			Data: data,
			Err:  nil,
		}

		assert.True(t, result.IsSuccess())
		assert.Equal(t, 3, len(result.Data))
		assert.Contains(t, result.Data, "apple")
	})

	t.Run("works with map type", func(t *testing.T) {
		data := map[string]int{"a": 1, "b": 2}
		result := Result[map[string]int]{
			Data: data,
			Err:  nil,
		}

		assert.True(t, result.IsSuccess())
		assert.Equal(t, 1, result.Data["a"])
		assert.Equal(t, 2, result.Data["b"])
	})

	t.Run("works with interface{} type", func(t *testing.T) {
		result := Result[interface{}]{
			Data: "can hold anything",
			Err:  nil,
		}

		assert.True(t, result.IsSuccess())
		assert.Equal(t, "can hold anything", result.Data)
	})
}

func TestResult_ErrorHandling(t *testing.T) {
	t.Run("preserves error message", func(t *testing.T) {
		expectedError := errors.New("specific error message")
		result := Result[string]{
			Data: "",
			Err:  expectedError,
		}

		assert.False(t, result.IsSuccess())
		require.Error(t, result.Err)
		assert.Equal(t, "specific error message", result.Err.Error())
		assert.Same(t, expectedError, result.Err)
	})

	t.Run("handles custom error types", func(t *testing.T) {
		customErr := CustomError{Code: 404, Message: "not found"}
		result := Result[string]{
			Data: "",
			Err:  customErr,
		}

		assert.False(t, result.IsSuccess())
		require.Error(t, result.Err)

		// Type assertion to verify we can access custom fields
		if customError, ok := result.Err.(CustomError); ok {
			assert.Equal(t, 404, customError.Code)
			assert.Equal(t, "not found", customError.Message)
		} else {
			t.Error("Expected CustomError type")
		}
	})
}

func TestResult_ZeroValues(t *testing.T) {
	t.Run("zero value result has no error", func(t *testing.T) {
		var result Result[string]

		assert.True(t, result.IsSuccess())
		assert.Empty(t, result.Data)
		assert.NoError(t, result.Err)
	})

	t.Run("zero value with different types", func(t *testing.T) {
		var intResult Result[int]
		assert.True(t, intResult.IsSuccess())
		assert.Equal(t, 0, intResult.Data)

		var boolResult Result[bool]
		assert.True(t, boolResult.IsSuccess())
		assert.False(t, boolResult.Data)

		var sliceResult Result[[]string]
		assert.True(t, sliceResult.IsSuccess())
		assert.Nil(t, sliceResult.Data)
	})
}
