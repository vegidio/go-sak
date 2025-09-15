package time

import (
	"testing"
	gotime "time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateEta(t *testing.T) {
	t.Run("Normal calculation", func(t *testing.T) {
		// 3 out of 10 tasks completed in 30 minutes
		// Should take ~70 minutes for the remaining 7 tasks
		eta := CalculateEta(10, 3, 30*gotime.Minute)
		expected := 70 * gotime.Minute
		assert.Equal(t, expected, eta)
	})

	t.Run("Half completed", func(t *testing.T) {
		// 5 out of 10 tasks completed in 1 hour
		// Should take 1 hour for the remaining 5 tasks
		eta := CalculateEta(10, 5, gotime.Hour)
		expected := gotime.Hour
		assert.Equal(t, expected, eta)
	})

	t.Run("Almost complete", func(t *testing.T) {
		// 9 out of 10 tasks completed in 90 minutes
		// Should take 10 minutes for the remaining 1 task
		eta := CalculateEta(10, 9, 90*gotime.Minute)
		expected := 10 * gotime.Minute
		assert.Equal(t, expected, eta)
	})

	t.Run("Already complete", func(t *testing.T) {
		// All tasks completed - should return 0
		eta := CalculateEta(10, 10, gotime.Hour)
		assert.Equal(t, gotime.Duration(0), eta)
	})

	t.Run("More than complete", func(t *testing.T) {
		// Completed more than total - should return 0
		eta := CalculateEta(10, 15, gotime.Hour)
		assert.Equal(t, gotime.Duration(0), eta)
	})

	t.Run("Single task remaining", func(t *testing.T) {
		// 1 task completed in 5 minutes, 1 remaining
		eta := CalculateEta(2, 1, 5*gotime.Minute)
		expected := 5 * gotime.Minute
		assert.Equal(t, expected, eta)
	})

	t.Run("Large numbers", func(t *testing.T) {
		// 1000 out of 10_000 tasks completed in 2 hours
		// Should take 18 hours for the remaining 9000 tasks
		eta := CalculateEta(10000, 1000, 2*gotime.Hour)
		expected := 18 * gotime.Hour
		assert.Equal(t, expected, eta)
	})
}

func TestCalculateEta_InvalidInputs(t *testing.T) {
	fallbackDuration := gotime.Duration(7 * 24 * gotime.Hour) // 7 days

	t.Run("Zero total", func(t *testing.T) {
		eta := CalculateEta(0, 5, gotime.Hour)
		assert.Equal(t, fallbackDuration, eta)
	})

	t.Run("Negative total", func(t *testing.T) {
		eta := CalculateEta(-10, 5, gotime.Hour)
		assert.Equal(t, fallbackDuration, eta)
	})

	t.Run("Zero completed", func(t *testing.T) {
		eta := CalculateEta(10, 0, gotime.Hour)
		assert.Equal(t, fallbackDuration, eta)
	})

	t.Run("Negative completed", func(t *testing.T) {
		eta := CalculateEta(10, -5, gotime.Hour)
		assert.Equal(t, fallbackDuration, eta)
	})

	t.Run("Zero elapsed time", func(t *testing.T) {
		eta := CalculateEta(10, 5, 0)
		assert.Equal(t, fallbackDuration, eta)
	})

	t.Run("Negative elapsed time", func(t *testing.T) {
		eta := CalculateEta(10, 5, -gotime.Hour)
		assert.Equal(t, fallbackDuration, eta)
	})

	t.Run("All invalid inputs", func(t *testing.T) {
		eta := CalculateEta(-10, -5, -gotime.Hour)
		assert.Equal(t, fallbackDuration, eta)
	})
}

func TestCalculateEta_EdgeCases(t *testing.T) {
	t.Run("Very small durations", func(t *testing.T) {
		// 1 nanosecond per task
		eta := CalculateEta(2, 1, gotime.Nanosecond)
		expected := gotime.Nanosecond
		assert.Equal(t, expected, eta)
	})

	t.Run("Very large durations", func(t *testing.T) {
		// 1 task completed in 24 hours, 1 remaining
		eta := CalculateEta(2, 1, 24*gotime.Hour)
		expected := 24 * gotime.Hour
		assert.Equal(t, expected, eta)
	})

	t.Run("Fractional average calculation", func(t *testing.T) {
		// 3 tasks completed in 10 minutes
		// Average per task = 3.33... minutes
		// 2 remaining tasks should take 6.66... minutes
		eta := CalculateEta(5, 3, 10*gotime.Minute)
		expected := (10 * gotime.Minute * 2) / 3 // 6 minutes 40 seconds
		assert.Equal(t, expected, eta)
	})
}

func TestCalculateEta_RealWorldScenarios(t *testing.T) {
	t.Run("File download simulation", func(t *testing.T) {
		// Downloaded 25MB out of 100MB in 30 seconds
		// Should take 90 seconds for the remaining 75 MB
		eta := CalculateEta(100, 25, 30*gotime.Second)
		expected := 90 * gotime.Second
		assert.Equal(t, expected, eta)
	})

	t.Run("Database migration", func(t *testing.T) {
		// Migrated 500 out of 2000 records in 5 minutes
		// Should take 15 minutes for the remaining 1500 records
		eta := CalculateEta(2000, 500, 5*gotime.Minute)
		expected := 15 * gotime.Minute
		assert.Equal(t, expected, eta)
	})

	t.Run("Backup process", func(t *testing.T) {
		// Backed up 10 out of 50 files in 2 minutes
		// Should take 8 minutes for the remaining 40 files
		eta := CalculateEta(50, 10, 2*gotime.Minute)
		expected := 8 * gotime.Minute
		assert.Equal(t, expected, eta)
	})
}
