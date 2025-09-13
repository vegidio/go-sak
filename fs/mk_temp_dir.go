package fs

import "os"

// MkTempDir creates a temporary directory with the given pattern and returns the directory path along with a cleanup
// function.
//
// The pattern parameter is used as a prefix for the temporary directory name. An empty pattern will create a directory
// with a system-generated name.
//
// # Returns:
//   - string: The absolute path to the created temporary directory
//   - func(): A cleanup function that removes the temporary directory and all its contents
//   - error: Any error that occurred during directory creation
//
// The cleanup function should be called when the temporary directory is no longer needed, typically using defer:
//
//	tempDir, cleanup, err := MkTempDir("myapp-")
//	if err != nil {
//	    return err
//	}
//	defer cleanup()
//
// Note: The cleanup function uses os.RemoveAll, which will recursively remove the directory and all its contents
// without error even if some files are missing.
func MkTempDir(pattern string) (string, func(), error) {
	tempDir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", nil, err
	}

	return tempDir, func() {
		os.RemoveAll(tempDir)
	}, nil
}
