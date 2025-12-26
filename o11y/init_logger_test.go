package o11y

import (
	"bytes"
	"io"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log/global"
)

func TestInitLogger(t *testing.T) {
	// Save the original state to restore after tests
	originalOutput := log.StandardLogger().Out
	originalLevel := log.StandardLogger().Level
	originalHooks := log.StandardLogger().Hooks

	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.SetLevel(originalLevel)
		log.StandardLogger().ReplaceHooks(originalHooks)
	})

	tests := []struct {
		name            string
		endpoint        string
		serviceName     string
		environment     OtelEnvironment
		destination     LogDestination
		expectError     bool
		checkOutput     bool
		expectedLevel   log.Level
		expectOTelSetup bool
	}{
		{
			name:            "LogToNone discards all output",
			endpoint:        "",
			serviceName:     "test-service",
			environment:     EnvDevelopment,
			destination:     LogToNone,
			expectError:     false,
			checkOutput:     true,
			expectedLevel:   originalLevel,
			expectOTelSetup: false,
		},
		{
			name:            "LogToTerminal sets debug level",
			endpoint:        "",
			serviceName:     "test-service",
			environment:     EnvDevelopment,
			destination:     LogToTerminal,
			expectError:     false,
			checkOutput:     false,
			expectedLevel:   log.DebugLevel,
			expectOTelSetup: false,
		},
		{
			name:            "LogToOTel with valid endpoint",
			endpoint:        "localhost:4318",
			serviceName:     "test-service",
			environment:     EnvDevelopment,
			destination:     LogToOTel,
			expectError:     false,
			checkOutput:     true,
			expectedLevel:   log.DebugLevel,
			expectOTelSetup: true,
		},
		{
			name:            "LogToBoth with valid endpoint",
			endpoint:        "localhost:4318",
			serviceName:     "test-service",
			environment:     EnvDevelopment,
			destination:     LogToBoth,
			expectError:     false,
			checkOutput:     false,
			expectedLevel:   log.DebugLevel,
			expectOTelSetup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the logger state before each test
			log.SetOutput(originalOutput)
			log.SetLevel(originalLevel)
			log.StandardLogger().ReplaceHooks(make(log.LevelHooks))

			cleanup, err := InitLogger(tt.endpoint, tt.serviceName, tt.environment, tt.destination)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cleanup)

			// Verify cleanup function doesn't panic
			defer func() {
				assert.NotPanics(t, cleanup)
			}()

			// Check log level
			assert.Equal(t, tt.expectedLevel, log.GetLevel())

			// Check output destination
			if tt.checkOutput {
				assert.Equal(t, io.Discard, log.StandardLogger().Out)
			}

			// Verify OpenTelemetry setup
			if tt.expectOTelSetup {
				provider := global.GetLoggerProvider()
				assert.NotNil(t, provider)

				// Verify hooks were added for OTel
				hooks := log.StandardLogger().Hooks
				assert.NotEmpty(t, hooks)
			}
		})
	}
}

func TestInitLogger_LogToNone(t *testing.T) {
	originalOutput := log.StandardLogger().Out
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
	})

	cleanup, err := InitLogger("", "test-service", EnvDevelopment, LogToNone)

	require.NoError(t, err)
	assert.NotNil(t, cleanup)

	// Verify output is discarded
	assert.Equal(t, io.Discard, log.StandardLogger().Out)

	// Cleanup should not panic
	assert.NotPanics(t, cleanup)
}

func TestInitLogger_LogToTerminal(t *testing.T) {
	originalOutput := log.StandardLogger().Out
	originalLevel := log.StandardLogger().Level
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.SetLevel(originalLevel)
	})

	cleanup, err := InitLogger("", "test-service", EnvDevelopment, LogToTerminal)

	require.NoError(t, err)
	assert.NotNil(t, cleanup)

	// Verify if the debug level is set
	assert.Equal(t, log.DebugLevel, log.GetLevel())

	// Verify output is not discarded
	assert.NotEqual(t, io.Discard, log.StandardLogger().Out)

	// Cleanup should not panic
	assert.NotPanics(t, cleanup)
}

func TestInitLogger_LogToOTel(t *testing.T) {
	originalOutput := log.StandardLogger().Out
	originalLevel := log.StandardLogger().Level
	originalHooks := log.StandardLogger().Hooks
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.SetLevel(originalLevel)
		log.StandardLogger().ReplaceHooks(originalHooks)
	})

	cleanup, err := InitLogger("localhost:4318", "test-service", EnvDevelopment, LogToOTel)

	require.NoError(t, err)
	require.NotNil(t, cleanup)
	defer cleanup()

	// Verify terminal output is suppressed
	assert.Equal(t, io.Discard, log.StandardLogger().Out)

	// Verify if the debug level is still set
	assert.Equal(t, log.DebugLevel, log.GetLevel())

	// Verify OTel hooks are registered
	hooks := log.StandardLogger().Hooks
	assert.NotEmpty(t, hooks)
}

func TestInitLogger_LogToBoth(t *testing.T) {
	originalOutput := log.StandardLogger().Out
	originalLevel := log.StandardLogger().Level
	originalHooks := log.StandardLogger().Hooks
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.SetLevel(originalLevel)
		log.StandardLogger().ReplaceHooks(originalHooks)
	})

	cleanup, err := InitLogger("localhost:4318", "test-service", EnvDevelopment, LogToBoth)

	require.NoError(t, err)
	require.NotNil(t, cleanup)
	defer cleanup()

	// Verify terminal output is NOT suppressed
	assert.NotEqual(t, io.Discard, log.StandardLogger().Out)

	// Verify if the debug level is set
	assert.Equal(t, log.DebugLevel, log.GetLevel())

	// Verify OTel hooks are registered
	hooks := log.StandardLogger().Hooks
	assert.NotEmpty(t, hooks)
}

func TestInitLogger_CleanupFunction(t *testing.T) {
	originalHooks := log.StandardLogger().Hooks
	t.Cleanup(func() {
		log.StandardLogger().ReplaceHooks(originalHooks)
	})

	cleanup, err := InitLogger("localhost:4318", "test-service", EnvDevelopment, LogToOTel)

	require.NoError(t, err)
	require.NotNil(t, cleanup)

	// Cleanup should execute without error
	assert.NotPanics(t, func() {
		cleanup()
	})

	// Calling cleanup multiple times should be safe
	assert.NotPanics(t, func() {
		cleanup()
	})
}

func TestInitLogger_TerminalLogging(t *testing.T) {
	var buf bytes.Buffer
	originalOutput := log.StandardLogger().Out
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
	})

	log.SetOutput(&buf)

	cleanup, err := InitLogger("", "test-service", EnvDevelopment, LogToTerminal)
	require.NoError(t, err)
	defer cleanup()

	// Write a test log
	log.Info("test message")

	// Verify message appears in the output
	assert.Contains(t, buf.String(), "test message")
}

func TestInitLogger_NoTerminalLogging(t *testing.T) {
	var buf bytes.Buffer
	originalOutput := log.StandardLogger().Out
	originalHooks := log.StandardLogger().Hooks
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.StandardLogger().ReplaceHooks(originalHooks)
	})

	log.SetOutput(&buf)

	cleanup, err := InitLogger("localhost:4318", "test-service", EnvDevelopment, LogToOTel)
	require.NoError(t, err)
	defer cleanup()

	// Write a test log
	log.Info("test message")

	// Verify message does NOT appear in the terminal output
	assert.NotContains(t, buf.String(), "test message")
}
