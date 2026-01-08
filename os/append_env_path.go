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
	existingValue := goos.Getenv(envvar)

	// If the variable is empty, just set it to the path without a separator
	if existingValue == "" {
		goos.Setenv(envvar, path)
		return
	}

	// Check if the existing value already ends with a separator
	separator := string(goos.PathListSeparator)
	if len(existingValue) > 0 && existingValue[len(existingValue)-1:] == separator {
		goos.Setenv(envvar, existingValue+path)
	} else {
		goos.Setenv(envvar, existingValue+separator+path)
	}
}
