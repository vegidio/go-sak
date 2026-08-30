package os

import (
	"fmt"
	"os"
	"strings"
	"syscall"
)

// ReExec re-executes the current program with additional environment variables.
//
// It replaces the current process with a new instance of itself, preserving the original command-line arguments while
// adding the specified environment variables.
//
// The function automatically sets APP_REEXEC=1 to prevent infinite recursion. If APP_REEXEC is already set to "1", the
// function returns immediately without re-executing.
//
// # Parameters:
//   - envVars: Zero or more environment variable strings in the format "KEY=VALUE" to be added to the new process
//     environment.
//
// # Returns:
//   - nil if the guard variable is already set, meaning this process is the re-executed one and there is nothing to do
//   - an error if the executable cannot be located, an entry of envVars is malformed, or the exec itself fails
//
// On success this function does not return at all, because the process image has been replaced. It is therefore never
// a no-op that reports success: on Windows, where syscall.Exec cannot replace a process, it always returns an error
// rather than silently continuing.
//
// Note: This function should be used as a last resort only in situations where the existing environment variables
// cannot be modified after the program starts, like LD_LIBRARY_PATH. Always try to use os.Setenv first.
//
// # Example:
//
//	if err := ReExec("DEBUG=1", "LOG_LEVEL=trace"); err != nil {
//	    log.Fatal(err)
//	}
func ReExec(envVars ...string) error {
	if os.Getenv("APP_REEXEC") == "1" {
		return nil
	}

	for _, kv := range envVars {
		if !strings.Contains(kv, "=") {
			return fmt.Errorf("malformed environment entry %q: want KEY=VALUE", kv)
		}
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot locate the current executable: %w", err)
	}

	// Start from the current environment minus any prior APP_REEXEC entry.
	// Without this, a caller that explicitly set APP_REEXEC to a non-"1" value would leak a duplicate entry into the
	// re-executed process; POSIX getenv returns the first match, so the guard above would fail to trigger and recursion
	// would not stop.
	current := os.Environ()
	env := make([]string, 0, len(current)+len(envVars)+1)
	for _, kv := range current {
		if !strings.HasPrefix(kv, "APP_REEXEC=") {
			env = append(env, kv)
		}
	}

	env = append(env, "APP_REEXEC=1")
	env = append(env, envVars...)

	// On success this call never returns. Windows has no execve, so syscall.Exec there always fails with
	// EWINDOWS - which used to be discarded, leaving ReExec a silent no-op that looked like it had worked.
	return fmt.Errorf("cannot re-execute %s: %w", exe, syscall.Exec(exe, os.Args, env))
}
