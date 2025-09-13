package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// MkUserConfigDir creates a directory within the user's configuration directory. It takes a required name parameter and
// optional additional path parts to create nested subdirectories.
//
// The function uses os.UserConfigDir() to get the platform-specific user configuration directory and creates the full
// directory path by joining the name and any additional parts. The directory is created with permissions 0o755
// (rwxr-xr-x).
//
// # Parameters:
//   - name: The primary directory name (cannot be empty)
//   - parts: Optional additional path segments to create nested subdirectories
//
// # Returns:
//   - string: The full path to the created directory
//   - error: Any error that occurred during directory creation or if name is empty
//
// # Example:
//
//	dir, err := MkUserConfigDir("myapp")
//	// Creates: ~/.config/myapp (on Linux) or similar platform-specific path
//
//	dir, err := MkUserConfigDir("myapp", "settings", "cache")
//	// Creates: ~/.config/myapp/settings/cache
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
