package o11y

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTelemetry(t *testing.T) {
	t.Run("creates telemetry with enabled logging", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			"1.0.0",
			EnvDevelopment,
			true,
		)
		require.NotNil(t, telemetry)
		defer telemetry.Close()

		assert.NotNil(t, telemetry.logger)
		assert.NotNil(t, telemetry.prefilled)
		assert.NotNil(t, telemetry.cleanup)
	})

	t.Run("creates telemetry with disabled logging", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			"1.0.0",
			EnvProduction,
			false,
		)
		require.NotNil(t, telemetry)
		defer telemetry.Close()

		assert.NotNil(t, telemetry.logger)
		assert.NotNil(t, telemetry.prefilled)
		assert.NotNil(t, telemetry.cleanup)
	})

	t.Run("prefilled fields contain version", func(t *testing.T) {
		version := "2.3.4"
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			version,
			EnvDevelopment,
			false,
		)
		defer telemetry.Close()

		assert.Equal(t, version, telemetry.prefilled["version"])
	})

	t.Run("prefilled fields contain session id", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			"1.0.0",
			EnvDevelopment,
			false,
		)
		defer telemetry.Close()

		sessionID, exists := telemetry.prefilled["session.id"]
		assert.True(t, exists)
		assert.NotEmpty(t, sessionID)

		// Verify it's a valid UUID format
		sessionIDStr, ok := sessionID.(string)
		assert.True(t, ok)
		assert.Equal(t, 36, len(sessionIDStr)) // UUID format: 8-4-4-4-12
	})

	t.Run("prefilled fields contain machine info", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			"1.0.0",
			EnvDevelopment,
			false,
		)
		defer telemetry.Close()

		// Machine ID should exist and be lowercase
		machineID, exists := telemetry.prefilled["machine.id"]
		assert.True(t, exists)
		assert.NotEmpty(t, machineID)
		machineIDStr, ok := machineID.(string)
		assert.True(t, ok)
		assert.Equal(t, strings.ToLower(machineIDStr), machineIDStr)

		// OS should match runtime.GOOS
		assert.Equal(t, runtime.GOOS, telemetry.prefilled["machine.os"])

		// Arch should match runtime.GOARCH
		assert.Equal(t, runtime.GOARCH, telemetry.prefilled["machine.arch"])
	})

	t.Run("handles different environments", func(t *testing.T) {
		environments := []OtelEnvironment{
			EnvDevelopment,
			EnvProduction,
		}

		for _, env := range environments {
			telemetry := NewTelemetry(
				"localhost:4318",
				"test-service",
				"1.0.0",
				env,
				false,
			)
			require.NotNil(t, telemetry)
			telemetry.Close()
		}
	})

	t.Run("handles empty service name", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"",
			"1.0.0",
			EnvDevelopment,
			false,
		)
		require.NotNil(t, telemetry)
		defer telemetry.Close()
	})

	t.Run("handles empty version", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			"",
			EnvDevelopment,
			false,
		)
		require.NotNil(t, telemetry)
		defer telemetry.Close()

		assert.Equal(t, "", telemetry.prefilled["version"])
	})

	t.Run("handles empty endpoint", func(t *testing.T) {
		telemetry := NewTelemetry(
			"",
			"test-service",
			"1.0.0",
			EnvDevelopment,
			false,
		)
		require.NotNil(t, telemetry)
		defer telemetry.Close()
	})
}

func TestRenewSession(t *testing.T) {
	t.Run("changes session id", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			"1.0.0",
			EnvDevelopment,
			false,
		)
		defer telemetry.Close()

		originalSessionID := telemetry.prefilled["session.id"]
		assert.NotEmpty(t, originalSessionID)

		telemetry.RenewSession()

		newSessionID := telemetry.prefilled["session.id"]
		assert.NotEmpty(t, newSessionID)
		assert.NotEqual(t, originalSessionID, newSessionID)
	})

	t.Run("generates valid uuid format", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			"1.0.0",
			EnvDevelopment,
			false,
		)
		defer telemetry.Close()

		telemetry.RenewSession()

		sessionID, ok := telemetry.prefilled["session.id"].(string)
		require.True(t, ok)
		assert.Equal(t, 36, len(sessionID))
		assert.Contains(t, sessionID, "-")
	})

	t.Run("multiple renewals generate different ids", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			"1.0.0",
			EnvDevelopment,
			false,
		)
		defer telemetry.Close()

		sessionIDs := make(map[string]bool)
		iterations := 10

		for i := 0; i < iterations; i++ {
			telemetry.RenewSession()
			sessionID := telemetry.prefilled["session.id"].(string)
			sessionIDs[sessionID] = true
		}

		// All session IDs should be unique
		assert.Equal(t, iterations, len(sessionIDs))
	})

	t.Run("preserves other prefilled fields", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			"1.0.0",
			EnvDevelopment,
			false,
		)
		defer telemetry.Close()

		version := telemetry.prefilled["version"]
		machineID := telemetry.prefilled["machine.id"]
		machineOS := telemetry.prefilled["machine.os"]

		telemetry.RenewSession()

		assert.Equal(t, version, telemetry.prefilled["version"])
		assert.Equal(t, machineID, telemetry.prefilled["machine.id"])
		assert.Equal(t, machineOS, telemetry.prefilled["machine.os"])
	})
}

func TestClose(t *testing.T) {
	t.Run("calls cleanup function when enabled", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			"1.0.0",
			EnvDevelopment,
			true,
		)
		require.NotNil(t, telemetry)

		// Close should not panic
		assert.NotPanics(t, func() {
			telemetry.Close()
		})
	})

	t.Run("calls cleanup function when disabled", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			"1.0.0",
			EnvDevelopment,
			false,
		)
		require.NotNil(t, telemetry)

		assert.NotPanics(t, func() {
			telemetry.Close()
		})
	})

	t.Run("multiple close calls do not panic", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"test-service",
			"1.0.0",
			EnvDevelopment,
			false,
		)
		require.NotNil(t, telemetry)

		assert.NotPanics(t, func() {
			telemetry.Close()
			telemetry.Close()
			telemetry.Close()
		})
	})
}

func TestInitLogger(t *testing.T) {
	t.Run("returns no-op cleanup when disabled", func(t *testing.T) {
		cleanup, err := initLogger(
			"localhost:4318",
			"test-service",
			EnvDevelopment,
			false,
		)

		assert.NoError(t, err)
		assert.NotNil(t, cleanup)
		assert.NotPanics(t, func() {
			cleanup()
		})
	})

	t.Run("handles empty endpoint when disabled", func(t *testing.T) {
		cleanup, err := initLogger(
			"",
			"test-service",
			EnvDevelopment,
			false,
		)

		assert.NoError(t, err)
		assert.NotNil(t, cleanup)
	})

	t.Run("handles empty service name when disabled", func(t *testing.T) {
		cleanup, err := initLogger(
			"localhost:4318",
			"",
			EnvDevelopment,
			false,
		)

		assert.NoError(t, err)
		assert.NotNil(t, cleanup)
	})

	t.Run("handles all environment types when disabled", func(t *testing.T) {
		environments := []OtelEnvironment{
			EnvDevelopment,
			EnvProduction,
		}

		for _, env := range environments {
			cleanup, err := initLogger(
				"localhost:4318",
				"test-service",
				env,
				false,
			)

			assert.NoError(t, err)
			assert.NotNil(t, cleanup)
		}
	})
}

func TestTelemetryIntegration(t *testing.T) {
	t.Run("full lifecycle with disabled logging", func(t *testing.T) {
		telemetry := NewTelemetry(
			"localhost:4318",
			"integration-test",
			"1.0.0",
			EnvDevelopment,
			false,
		)
		require.NotNil(t, telemetry)

		// Verify initial state
		assert.NotNil(t, telemetry.logger)
		assert.NotEmpty(t, telemetry.prefilled)

		// Renew session
		originalSession := telemetry.prefilled["session.id"]
		telemetry.RenewSession()
		assert.NotEqual(t, originalSession, telemetry.prefilled["session.id"])

		// Close
		assert.NotPanics(t, func() {
			telemetry.Close()
		})
	})
}
