package os

import goos "os"

// AppendEnvPath appends a path to an environment variable that contains a list of paths.
//
// It takes the environment variable name and a path string, then appends the path to the existing value using the
// OS-specific path list separator (e.g., ':' on Unix, ';' on Windows). This is commonly used for PATH-like environment
// variables such as PATH, LD_LIBRARY_PATH, etc.
//
// # Parameters
//   - envvar: the name of the environment variable to modify (e.g., "PATH")
//   - path: the directory path to append
//
// # Example
//
//	AppendEnvPath("PATH", "/usr/local/bin")
func AppendEnvPath(envvar string, path string) {
	newEnvPath := goos.Getenv(envvar) + string(goos.PathListSeparator) + path
	goos.Setenv(envvar, newEnvPath)
}
