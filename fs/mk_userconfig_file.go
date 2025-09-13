package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// MkUserConfigFile creates a file in the user's configuration directory with the specified application name and path
// components. It creates all necessary parent directories if they don't exist and opens the file for read/write
// operations.
//
// The function constructs a path by joining the user's config directory (typically ~/.config on Unix systems) with the
// application name and the provided path components. The last element in parts is treated as the filename, while
// preceding elements are treated as subdirectory names.
//
// # Parameters:
//   - name: The application or service name that will be used as the top-level directory within the user's config
//     directory. Cannot be empty.
//   - parts: Variable number of path components where the last element is the filename and preceding elements are
//     subdirectory names. At least one component must be provided.
//
// # Returns:
//   - *os.File: An opened file handle with read/write permissions (0o644), or nil on error
//   - error: An error if the operation fails, including cases where name is empty, no parts are provided, user config
//     directory cannot be determined, directory creation fails, or file opening fails
//
// # Example:
//
//	// Creates ~/.config/myapp/config.json
//	file, err := MkUserConfigFile("myapp", "config.json")
//
//	// Creates ~/.config/myapp/db/settings.json
//	file, err := MkUserConfigFile("myapp", "db", "settings.json")
//
// The created directories have permissions 0o755 and the file has permissions 0o644.
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
