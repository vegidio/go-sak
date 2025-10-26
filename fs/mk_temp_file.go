package fs

import (
	"os"
	"path/filepath"
)

// MkTempFile creates a temporary file with the given pattern and returns the *os.File object along with a cleanup
// function.
//
// The pattern parameter is used as a prefix for the temporary directory name. An empty pattern will create a directory
// with a system-generated name.
//
// # Returns:
//   - *os.File: The object to the created temporary file
//   - func(): A cleanup function that removes the temporary file
//   - error: Any error that occurred during file creation
//
// The cleanup function should be called when the temporary file is no longer needed, typically using defer:
//
//	tempDir, cleanup, err := MkTempFile("myapp-", "test.dat")
//	if err != nil {
//	    return err
//	}
//	defer cleanup()
func MkTempFile(pattern string, name string) (*os.File, func(), error) {
	tempDir, cleanup, err := MkTempDir(pattern)
	if err != nil {
		return nil, nil, err
	}

	fullPath := filepath.Join(tempDir, name)
	file, err := os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, cleanup, err
	}

	return file, cleanup, nil
}
