package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

func MkUserConfigDir(name string, parts ...string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name cannot be empty")
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	allParts := append([]string{configDir, name}, parts...)
	fullPath := filepath.Join(allParts...)

	if mErr := os.MkdirAll(fullPath, 0o755); mErr != nil {
		return "", mErr
	}

	return fullPath, nil
}
