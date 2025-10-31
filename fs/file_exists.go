package fs

import "os"

// FileExists checks if a file exists at the specified path.
// It returns true if the path exists and is a file (not a directory).
// It returns false if the path does not exist or if it is a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}
