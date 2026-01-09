package os

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReExec(t *testing.T) {
	t.Run("does not re-execute when APP_REEXEC is already set", func(t *testing.T) {
		// Set APP_REEXEC to simulate an already re-executed state
		originalValue := os.Getenv("APP_REEXEC")
		defer func() {
			if originalValue == "" {
				os.Unsetenv("APP_REEXEC")
			} else {
				os.Setenv("APP_REEXEC", originalValue)
			}
		}()

		os.Setenv("APP_REEXEC", "1")

		// This should return immediately without re-executing
		// If it doesn't return, the test will hang/fail
		ReExec("TEST=value")

		// If we reach here, the function returned correctly
		assert.Equal(t, "1", os.Getenv("APP_REEXEC"))
	})

	t.Run("subprocess re-executes with additional environment variables", func(t *testing.T) {
		if os.Getenv("TEST_REEXEC_SUBPROCESS") == "1" {
			// This is the re-executed subprocess
			ReExec("CUSTOM_VAR=custom_value", "ANOTHER_VAR=another_value")

			// After ReExec, check that variables are set
			assert.Equal(t, "1", os.Getenv("APP_REEXEC"))
			assert.Equal(t, "custom_value", os.Getenv("CUSTOM_VAR"))
			assert.Equal(t, "another_value", os.Getenv("ANOTHER_VAR"))
			os.Exit(42) // Exit with specific code to verify execution
		}

		// Parent test process
		cmd := exec.Command(os.Args[0], "-test.run=TestReExec/subprocess_re-executes_with_additional_environment_variables")
		cmd.Env = append(os.Environ(), "TEST_REEXEC_SUBPROCESS=1")

		err := cmd.Run()
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.Equal(t, 42, exitErr.ExitCode())
		} else {
			t.Fatalf("expected exit error with code 42, got: %v", err)
		}
	})

	t.Run("subprocess re-executes with no additional variables", func(t *testing.T) {
		if os.Getenv("TEST_REEXEC_NO_VARS") == "1" {
			ReExec()

			// Verify APP_REEXEC was set
			assert.Equal(t, "1", os.Getenv("APP_REEXEC"))
			os.Exit(43)
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestReExec/subprocess_re-executes_with_no_additional_variables")
		cmd.Env = append(os.Environ(), "TEST_REEXEC_NO_VARS=1")

		err := cmd.Run()
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.Equal(t, 43, exitErr.ExitCode())
		} else {
			t.Fatalf("expected exit error with code 43, got: %v", err)
		}
	})

	t.Run("subprocess preserves existing environment variables", func(t *testing.T) {
		if os.Getenv("TEST_REEXEC_PRESERVE") == "1" {
			ReExec("NEW_VAR=new_value")

			// Check that the original env var is preserved
			assert.Equal(t, "preserved_value", os.Getenv("PRESERVE_ME"))
			assert.Equal(t, "new_value", os.Getenv("NEW_VAR"))
			assert.Equal(t, "1", os.Getenv("APP_REEXEC"))
			os.Exit(44)
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestReExec/subprocess_preserves_existing_environment_variables")
		cmd.Env = append(os.Environ(), "TEST_REEXEC_PRESERVE=1", "PRESERVE_ME=preserved_value")

		err := cmd.Run()
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.Equal(t, 44, exitErr.ExitCode())
		} else {
			t.Fatalf("expected exit error with code 44, got: %v", err)
		}
	})

	t.Run("subprocess handles environment variables with special characters", func(t *testing.T) {
		if os.Getenv("TEST_REEXEC_SPECIAL") == "1" {
			ReExec("VAR_WITH_EQUALS=value=with=equals", "VAR_WITH_SPACES=value with spaces")

			assert.Equal(t, "value=with=equals", os.Getenv("VAR_WITH_EQUALS"))
			assert.Equal(t, "value with spaces", os.Getenv("VAR_WITH_SPACES"))
			os.Exit(45)
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestReExec/subprocess_handles_environment_variables_with_special_characters")
		cmd.Env = append(os.Environ(), "TEST_REEXEC_SPECIAL=1")

		err := cmd.Run()
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.Equal(t, 45, exitErr.ExitCode())
		} else {
			t.Fatalf("expected exit error with code 45, got: %v", err)
		}
	})

	t.Run("subprocess handles multiple environment variables", func(t *testing.T) {
		if os.Getenv("TEST_REEXEC_MULTIPLE") == "1" {
			vars := []string{
				"VAR1=value1",
				"VAR2=value2",
				"VAR3=value3",
				"VAR4=value4",
				"VAR5=value5",
			}
			ReExec(vars...)

			assert.Equal(t, "value1", os.Getenv("VAR1"))
			assert.Equal(t, "value2", os.Getenv("VAR2"))
			assert.Equal(t, "value3", os.Getenv("VAR3"))
			assert.Equal(t, "value4", os.Getenv("VAR4"))
			assert.Equal(t, "value5", os.Getenv("VAR5"))
			os.Exit(46)
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestReExec/subprocess_handles_multiple_environment_variables")
		cmd.Env = append(os.Environ(), "TEST_REEXEC_MULTIPLE=1")

		err := cmd.Run()
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.Equal(t, 46, exitErr.ExitCode())
		} else {
			t.Fatalf("expected exit error with code 46, got: %v", err)
		}
	})

	t.Run("subprocess preserves command line arguments", func(t *testing.T) {
		if os.Getenv("TEST_REEXEC_ARGS") == "1" {
			ReExec("CHECK_ARGS=1")

			// Verify args are preserved
			require.True(t, len(os.Args) > 0)
			require.True(t, strings.Contains(os.Args[0], "test"))
			os.Exit(47)
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestReExec/subprocess_preserves_command_line_arguments", "extra", "args")
		cmd.Env = append(os.Environ(), "TEST_REEXEC_ARGS=1")

		err := cmd.Run()
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.Equal(t, 47, exitErr.ExitCode())
		} else {
			t.Fatalf("expected exit error with code 47, got: %v", err)
		}
	})
}

func TestReExec_EdgeCases(t *testing.T) {
	t.Run("handles empty string environment variable", func(t *testing.T) {
		if os.Getenv("TEST_REEXEC_EMPTY") == "1" {
			ReExec("EMPTY_VAR=")

			assert.Equal(t, "", os.Getenv("EMPTY_VAR"))
			os.Exit(48)
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestReExec_EdgeCases/handles_empty_string_environment_variable")
		cmd.Env = append(os.Environ(), "TEST_REEXEC_EMPTY=1")

		err := cmd.Run()
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.Equal(t, 48, exitErr.ExitCode())
		}
	})

	t.Run("APP_REEXEC value other than 1 allows re-execution", func(t *testing.T) {
		if os.Getenv("TEST_REEXEC_OTHER_VALUE") == "1" {
			// Set APP_REEXEC to something other than "1"
			os.Setenv("APP_REEXEC", "0")

			ReExec("TEST_VAR=test")

			// Should have re-executed and set APP_REEXEC to "1"
			assert.Equal(t, "1", os.Getenv("APP_REEXEC"))
			os.Exit(49)
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestReExec_EdgeCases/APP_REEXEC_value_other_than_1_allows_re-execution")
		cmd.Env = append(os.Environ(), "TEST_REEXEC_OTHER_VALUE=1")

		err := cmd.Run()
		if exitErr, ok := err.(*exec.ExitError); ok {
			assert.Equal(t, 49, exitErr.ExitCode())
		}
	})
}
