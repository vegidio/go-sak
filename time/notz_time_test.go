package time

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotzTime_UnmarshalJSON(t *testing.T) {
	t.Run("valid timestamp", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-12-15T14:30:45"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.NoError(t, err)
		expected := time.Date(2023, 12, 15, 14, 30, 45, 0, time.UTC)
		assert.True(t, nt.Time.Equal(expected))
	})

	t.Run("valid timestamp with different date", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2024-01-01T00:00:00"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.NoError(t, err)
		expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		assert.True(t, nt.Time.Equal(expected))
	})

	t.Run("valid timestamp with maximum time values", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-12-31T23:59:59"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.NoError(t, err)
		expected := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)
		assert.True(t, nt.Time.Equal(expected))
	})

	t.Run("empty string timestamp", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `""`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Equal(t, "invalid timestamp", err.Error())
	})

	t.Run("invalid JSON - not a string", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `123456`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		// Should be a JSON unmarshal error
		assert.Contains(t, err.Error(), "cannot unmarshal")
	})

	t.Run("invalid JSON - object", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `{"time": "2023-12-15T14:30:45"}`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot unmarshal")
	})

	t.Run("invalid JSON - array", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `["2023-12-15T14:30:45"]`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot unmarshal")
	})

	t.Run("malformed JSON", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-12-15T14:30:45`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected end")
	})

	t.Run("invalid timestamp format - with timezone", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-12-15T14:30:45Z"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})

	t.Run("invalid timestamp format - with timezone offset", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-12-15T14:30:45+02:00"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})

	t.Run("invalid timestamp format - missing seconds", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-12-15T14:30"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})

	t.Run("invalid timestamp format - missing time", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-12-15"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})

	t.Run("invalid timestamp format - wrong date format", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"15-12-2023T14:30:45"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})

	t.Run("invalid timestamp format - wrong time format", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-12-15 14:30:45"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})

	t.Run("invalid date values - month", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-13-15T14:30:45"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})

	t.Run("invalid date values - day", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-12-32T14:30:45"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})

	t.Run("invalid time values - hour", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-12-15T25:30:45"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})

	t.Run("invalid time values - minute", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-12-15T14:60:45"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})

	t.Run("invalid time values - second", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-12-15T14:30:60"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})

	t.Run("leap year date", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2024-02-29T12:00:00"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.NoError(t, err)
		expected := time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC)
		assert.True(t, nt.Time.Equal(expected))
	})

	t.Run("non-leap year invalid date", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-02-29T12:00:00"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})

	t.Run("whitespace string", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"   "`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing time")
	})
}

func TestNotzTime_Integration(t *testing.T) {
	t.Run("unmarshal and access time fields", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-07-20T09:15:30"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 2023, nt.Year())
		assert.Equal(t, time.July, nt.Month())
		assert.Equal(t, 20, nt.Day())
		assert.Equal(t, 9, nt.Hour())
		assert.Equal(t, 15, nt.Minute())
		assert.Equal(t, 30, nt.Second())
	})

	t.Run("multiple NotzTime instances", func(t *testing.T) {
		// Arrange
		var nt1, nt2 NotzTime
		jsonData1 := `"2023-01-01T00:00:00"`
		jsonData2 := `"2023-12-31T23:59:59"`

		// Act
		err1 := json.Unmarshal([]byte(jsonData1), &nt1)
		err2 := json.Unmarshal([]byte(jsonData2), &nt2)

		// Assert
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.True(t, nt1.Before(nt2.Time))
		assert.True(t, nt2.After(nt1.Time))
	})

	t.Run("struct with NotzTime field", func(t *testing.T) {
		// Arrange
		type Event struct {
			Name      string   `json:"name"`
			Timestamp NotzTime `json:"timestamp"`
		}

		var event Event
		jsonData := `{"name": "test event", "timestamp": "2023-08-15T16:45:30"}`

		// Act
		err := json.Unmarshal([]byte(jsonData), &event)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "test event", event.Name)
		expected := time.Date(2023, 8, 15, 16, 45, 30, 0, time.UTC)
		assert.True(t, event.Timestamp.Time.Equal(expected))
	})

	t.Run("struct with NotzTime field - invalid timestamp", func(t *testing.T) {
		// Arrange
		type Event struct {
			Name      string   `json:"name"`
			Timestamp NotzTime `json:"timestamp"`
		}

		var event Event
		jsonData := `{"name": "test event", "timestamp": ""}`

		// Act
		err := json.Unmarshal([]byte(jsonData), &event)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid timestamp")
	})

	t.Run("array of NotzTime", func(t *testing.T) {
		// Arrange
		var timestamps []NotzTime
		jsonData := `["2023-01-01T12:00:00", "2023-06-15T18:30:45", "2023-12-31T23:59:59"]`

		// Act
		err := json.Unmarshal([]byte(jsonData), &timestamps)

		// Assert
		require.NoError(t, err)
		assert.Len(t, timestamps, 3)

		expected1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		expected2 := time.Date(2023, 6, 15, 18, 30, 45, 0, time.UTC)
		expected3 := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)

		assert.True(t, timestamps[0].Time.Equal(expected1))
		assert.True(t, timestamps[1].Time.Equal(expected2))
		assert.True(t, timestamps[2].Time.Equal(expected3))
	})

	t.Run("time operations after unmarshal", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-06-15T12:00:00"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)
		require.NoError(t, err)

		// Test various time operations
		oneHourLater := nt.Add(time.Hour)
		formattedTime := nt.Format("2006-01-02 15:04:05")
		truncated := nt.Truncate(time.Hour)

		// Assert
		expectedOriginal := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC)
		expectedLater := time.Date(2023, 6, 15, 13, 0, 0, 0, time.UTC)
		expectedTruncated := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC)

		assert.True(t, nt.Time.Equal(expectedOriginal))
		assert.True(t, oneHourLater.Equal(expectedLater))
		assert.Equal(t, "2023-06-15 12:00:00", formattedTime)
		assert.True(t, truncated.Equal(expectedTruncated))
	})
}

func TestNotzTime_EdgeCases(t *testing.T) {
	t.Run("minimum valid date", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"0001-01-01T00:00:00"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.NoError(t, err)
		expected := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
		assert.True(t, nt.Time.Equal(expected))
	})

	t.Run("year 9999", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"9999-12-31T23:59:59"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.NoError(t, err)
		expected := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
		assert.True(t, nt.Time.Equal(expected))
	})

	t.Run("leading zeros in numbers", func(t *testing.T) {
		// Arrange
		var nt NotzTime
		jsonData := `"2023-01-01T01:01:01"`

		// Act
		err := json.Unmarshal([]byte(jsonData), &nt)

		// Assert
		require.NoError(t, err)
		expected := time.Date(2023, 1, 1, 1, 1, 1, 0, time.UTC)
		assert.True(t, nt.Time.Equal(expected))
	})

	t.Run("february in leap year vs non-leap year", func(t *testing.T) {
		tests := []struct {
			name        string
			jsonData    string
			shouldError bool
		}{
			{"leap year - Feb 29 valid", `"2024-02-29T12:00:00"`, false},
			{"non-leap year - Feb 29 invalid", `"2023-02-29T12:00:00"`, true},
			{"leap year - Feb 28 valid", `"2024-02-28T12:00:00"`, false},
			{"non-leap year - Feb 28 valid", `"2023-02-28T12:00:00"`, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var nt NotzTime
				err := json.Unmarshal([]byte(tt.jsonData), &nt)

				if tt.shouldError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

func TestNotzTime_Performance(t *testing.T) {
	t.Run("unmarshal many timestamps", func(t *testing.T) {
		// This test ensures the implementation doesn't have performance issues
		jsonData := `"2023-06-15T12:30:45"`

		for i := 0; i < 1000; i++ {
			var nt NotzTime
			err := json.Unmarshal([]byte(jsonData), &nt)
			require.NoError(t, err)
		}
	})
}
