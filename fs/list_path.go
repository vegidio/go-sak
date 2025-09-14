package fs

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
)

type Flags uint8

const (
	LpDir Flags = 1 << iota
	LpFile
	LpRecursive
)

// ListPath traverses a directory and returns a list of paths based on the specified flags and file extensions. The
// directory parameter specifies the root directory to start the traversal from.
//
// The flags parameter controls what types of entries to include and traversal behavior:
//   - LpDir: Include directories in the results
//   - LpFile: Include files in the results
//   - LpRecursive: Perform recursive traversal of subdirectories
//
// The fileExt parameter is a slice of file extensions to filter by (case-insensitive). If empty, all files are included
// (when LpFile flag is set). Extensions should include the dot (e.g., ".txt", ".go").
//
// Returns a slice of file/directory paths and any error encountered during traversal.
// If an error occurs while reading a specific entry, that entry is skipped and traversal continues.
//
// # Example:
//
//	// List all .go files recursively
//	paths, err := ListPath("/src", LpFile|LpRecursive, []string{".go"})
//
//	// List only directories at the first level
//	paths, err := ListPath("/src", LpDir, nil)
//
//	// List both files and directories recursively, filtering for .txt and .md files
//	paths, err := ListPath("/docs", LpFile|LpDir|LpRecursive, []string{".txt", ".md"})
func ListPath(directory string, flags Flags, fileExt []string) ([]string, error) {
	entries := make([]string, 0)

	includeDir := flags&LpDir != 0
	includeFile := flags&LpFile != 0
	recursive := flags&LpRecursive != 0

	// Prepare extension set for O(1) lookup
	fileExt = lo.Map(fileExt, func(ext string, _ int) string {
		return strings.ToLower(ext)
	})

	extSet := make(map[string]struct{}, len(fileExt))
	for _, ext := range fileExt {
		extSet[ext] = struct{}{}
	}

	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// If this is the root directory, and it doesn't exist, return the error
			if path == directory {
				return err
			}
			return nil // skip on error for subdirectories/files
		}

		// Skip the root directory itself
		if path == directory {
			return nil
		}

		// Non-recursive: include the first-level directory (if requested) but don't descend
		if !recursive && d.IsDir() {
			if includeDir {
				entries = append(entries, path)
			}
			return filepath.SkipDir
		}

		// Directory handling
		if d.IsDir() {
			if includeDir {
				entries = append(entries, path)
			}
			return nil
		}

		// File handling
		if !includeFile {
			return nil
		}

		if len(extSet) == 0 {
			entries = append(entries, path)
			return nil
		}

		if _, ok := extSet[strings.ToLower(filepath.Ext(path))]; ok {
			entries = append(entries, path)
		}

		return nil
	})

	if err != nil {
		return entries, err
	}

	return entries, nil
}
