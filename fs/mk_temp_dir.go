package fs

import "os"

func MkTempDir(pattern string) (string, func(), error) {
	tempDir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", nil, err
	}

	return tempDir, func() {
		os.RemoveAll(tempDir)
	}, nil
}
