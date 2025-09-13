package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

func MkUserConfigFile(name string, parts ...string) (*os.File, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("no path components provided")
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("error getting the user config dir: %w", err)
	}

	dirParts := append([]string{configDir, name}, parts[:len(parts)-1]...)
	dirPath := filepath.Join(dirParts...)

	if mErr := os.MkdirAll(dirPath, 0o755); mErr != nil {
		return nil, mErr
	}

	filePath := parts[len(parts)-1]
	fullPath := filepath.Join(dirPath, filePath)

	file, err := os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}

	return file, nil
}
