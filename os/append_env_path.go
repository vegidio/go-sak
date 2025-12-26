package os

import goos "os"

// AppendEnvPath appends a directory path to the PATH environment variable.
//
// The path is added to the end of the existing PATH using the OS-specific path separator.
func AppendEnvPath(path string) {
	newEnvPath := goos.Getenv("PATH") + string(goos.PathListSeparator) + path
	goos.Setenv("PATH", newEnvPath)
}
