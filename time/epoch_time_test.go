package time

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEpochTime_UnmarshalJSON(t *testing.T) {
	t.Run("valid epoch time", func(t *testing.T) {
		// Test with a specific epoch time (2023-01-01 00:00:00 UTC)
		epochTime := float64(1672531200)
		jsonData := `1672531200`

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.NoError(t, err)
		expected := time.Unix(int64(epochTime), 0)
		assert.Equal(t, expected, et.Time)
	})

	t.Run("zero epoch time", func(t *testing.T) {
		jsonData := `0`

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.NoError(t, err)
		expected := time.Unix(0, 0)
		assert.Equal(t, expected, et.Time)
	})

	t.Run("floating point epoch time", func(t *testing.T) {
		// Test with fractional seconds (though they'll be truncated in Unix conversion)
		jsonData := `1672531200.5`

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.NoError(t, err)
		expected := time.Unix(1672531200, 0) // Unix truncates to seconds
		assert.Equal(t, expected, et.Time)
	})

	t.Run("large epoch time", func(t *testing.T) {
		// Test with a future date (year 2038 issue boundary)
		epochTime := float64(2147483647) // 2038-01-19 03:14:07 UTC
		jsonData := `2147483647`

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.NoError(t, err)
		expected := time.Unix(int64(epochTime), 0)
		assert.Equal(t, expected, et.Time)
	})

	t.Run("negative epoch time", func(t *testing.T) {
		jsonData := `-1`

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.Error(t, err)
		assert.Equal(t, "invalid epoch time", err.Error())
	})

	t.Run("negative floating point epoch time", func(t *testing.T) {
		jsonData := `-123.456`

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.Error(t, err)
		assert.Equal(t, "invalid epoch time", err.Error())
	})

	t.Run("invalid JSON - string", func(t *testing.T) {
		jsonData := `"not a number"`

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.Error(t, err)
		// Should be a JSON unmarshaling error, not our custom error
		assert.NotEqual(t, "invalid epoch time", err.Error())
	})

	t.Run("invalid JSON - object", func(t *testing.T) {
		jsonData := `{"timestamp": 1672531200}`

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.Error(t, err)
		// Should be a JSON unmarshaling error, not our custom error
		assert.NotEqual(t, "invalid epoch time", err.Error())
	})

	t.Run("invalid JSON - array", func(t *testing.T) {
		jsonData := `[1672531200]`

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.Error(t, err)
		// Should be a JSON unmarshaling error, not our custom error
		assert.NotEqual(t, "invalid epoch time", err.Error())
	})

	t.Run("invalid JSON - malformed", func(t *testing.T) {
		jsonData := `{invalid json`

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.Error(t, err)
		// Should be a JSON unmarshaling error, not our custom error
		assert.NotEqual(t, "invalid epoch time", err.Error())
	})

	t.Run("empty JSON", func(t *testing.T) {
		jsonData := ``

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.Error(t, err)
		// Should be a JSON unmarshaling error, not our custom error
		assert.NotEqual(t, "invalid epoch time", err.Error())
	})
}

func TestEpochTime_Integration(t *testing.T) {
	t.Run("unmarshal within struct", func(t *testing.T) {
		type TestStruct struct {
			Timestamp EpochTime `json:"timestamp"`
			Message   string    `json:"message"`
		}

		jsonData := `{"timestamp": 1672531200, "message": "test"}`

		var ts TestStruct
		err := json.Unmarshal([]byte(jsonData), &ts)

		assert.NoError(t, err)
		assert.Equal(t, "test", ts.Message)
		expected := time.Unix(1672531200, 0)
		assert.Equal(t, expected, ts.Timestamp.Time)
	})

	t.Run("unmarshal within struct with invalid epoch", func(t *testing.T) {
		type TestStruct struct {
			Timestamp EpochTime `json:"timestamp"`
			Message   string    `json:"message"`
		}

		jsonData := `{"timestamp": -1, "message": "test"}`

		var ts TestStruct
		err := json.Unmarshal([]byte(jsonData), &ts)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid epoch time")
	})

	t.Run("multiple EpochTime fields", func(t *testing.T) {
		type Event struct {
			CreatedAt EpochTime `json:"created_at"`
			UpdatedAt EpochTime `json:"updated_at"`
		}

		jsonData := `{"created_at": 1672531200, "updated_at": 1672617600}`

		var event Event
		err := json.Unmarshal([]byte(jsonData), &event)

		assert.NoError(t, err)
		expectedCreated := time.Unix(1672531200, 0)
		expectedUpdated := time.Unix(1672617600, 0)
		assert.Equal(t, expectedCreated, event.CreatedAt.Time)
		assert.Equal(t, expectedUpdated, event.UpdatedAt.Time)
	})

	t.Run("array of EpochTime", func(t *testing.T) {
		jsonData := `[1672531200, 1672617600, 0]`

		var times []EpochTime
		err := json.Unmarshal([]byte(jsonData), &times)

		assert.NoError(t, err)
		assert.Len(t, times, 3)

		expected := []time.Time{
			time.Unix(1672531200, 0),
			time.Unix(1672617600, 0),
			time.Unix(0, 0),
		}

		for i, et := range times {
			assert.Equal(t, expected[i], et.Time)
		}
	})
}

func TestEpochTime_EdgeCases(t *testing.T) {
	t.Run("scientific notation", func(t *testing.T) {
		jsonData := `1.6725312e9` // 1672531200 in scientific notation

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.NoError(t, err)
		expected := time.Unix(1672531200, 0)
		assert.Equal(t, expected, et.Time)
	})

	t.Run("precision loss with float64", func(t *testing.T) {
		// Test that fractional seconds are handled (but lost in Unix conversion)
		jsonData := `1672531200.123456789`

		var et EpochTime
		err := json.Unmarshal([]byte(jsonData), &et)

		assert.NoError(t, err)
		// Unix() truncates to seconds, so fractional part should be lost
		expected := time.Unix(1672531200, 0)
		assert.Equal(t, expected, et.Time)
	})
}
