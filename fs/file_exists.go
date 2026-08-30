package fs

import "os"

// FileExists checks if a file exists at the specified path.
// It returns true if the path exists and is a file (not a directory).
// It returns false if the path does not exist, if it is a directory, or if it cannot be stat'ed at all - for instance
// when a parent directory is unreadable, when a component of the path is not a directory, or when a symlink loops.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		// os.Stat yields a nil FileInfo for every error, not only for "not exist", so this must come before any
		// use of info.
		return false
	}

	return !info.IsDir()
}
