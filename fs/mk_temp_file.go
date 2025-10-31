package fs

import "os"

// MkTempFile creates a temporary file in a directory with the given pattern and returns the *os.File object along with
// a cleanup function.
//
// The pattern parameter is used as a prefix for the temporary file name. An empty pattern will create a file with a
// system-generated name.
//
// # Returns:
//   - *os.File: The object to the created temporary file
//   - func(): A cleanup function that removes the temporary file
//   - error: Any error that occurred during file creation
//
// The cleanup function should be called when the temporary file is no longer needed, typically using defer:
//
//	tempDir, cleanup, err := MkTempFile("myapp", "test-*.dat")
//	if err != nil {
//	    return err
//	}
//	defer cleanup()
func MkTempFile(directory string, patten string) (*os.File, func(), error) {
	tmpFile, err := os.CreateTemp(directory, patten)
	if err != nil {
		return nil, nil, err
	}

	return tmpFile, func() {
		os.Remove(tmpFile.Name())
	}, nil
}
