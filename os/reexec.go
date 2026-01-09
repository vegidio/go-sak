package os

import (
	"os"
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
// $ Parameters:
//   - envVars: Zero or more environment variable strings in the format "KEY=VALUE" to be added to the new process
//     environment.
//
// Note: This function should be used as a last resort only in situations where the existing environment variables
// cannot be modified after the program starts, like LD_LIBRARY_PATH. Always try to use os.Setenv first.
//
// # Example:
//
//	ReExec("DEBUG=1", "LOG_LEVEL=trace")
func ReExec(envVars ...string) {
	if os.Getenv("APP_REEXEC") == "1" {
		return
	}

	exe, _ := os.Executable()
	env := append(os.Environ(), "APP_REEXEC=1")
	env = append(env, envVars...)

	syscall.Exec(exe, os.Args, env)
}
