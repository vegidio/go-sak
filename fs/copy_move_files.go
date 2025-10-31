package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/otiai10/copy"
	"github.com/samber/lo"
)

type CmFlags uint8

const (
	CmRecursive CmFlags = 1 << iota
	CmPreserveStructure
)

// CopyFiles copies files and/or directories to a destination directory.
//
// The flags parameter controls the copy behavior:
//   - CmRecursive: Include subdirectories when copying directories
//   - CmPreserveStructure: Preserve directory structure (if false, all files are flattened to destDir)
//   - 0 (no flags): Non-recursive copy with flattened structure (default behavior)
//
// The exts parameter filters files by extension (case-insensitive, e.g., []string{".jpg", ".png"}).
// If nil or empty, no extension filtering is applied.
//
// # Example:
//
//	// Copy recursively, preserving structure, filtering by extension
//	err := CopyFiles([]string{"src"}, "dest", CmRecursive|CmPreserveStructure, []string{".go"})
//
//	// Copy recursively, flatten all files to destination
//	err := CopyFiles([]string{"src"}, "dest", CmRecursive, []string{".jpg", ".png"})
//
//	// Copy only first-level files, preserve structure
//	err := CopyFiles([]string{"src"}, "dest", CmPreserveStructure, nil)
//
//	// Copy only first-level files, flattened (no flags)
//	err := CopyFiles([]string{"src"}, "dest", 0, nil)
func CopyFiles(
	sources []string,
	destDir string,
	flags CmFlags,
	exts []string,
) error {
	return transferFiles(sources, destDir, flags, exts, false)
}

// MoveFiles moves files and/or directories to a destination directory.
//
// The flags parameter controls the move behavior:
//   - CmRecursive: Include subdirectories when moving directories
//   - CmPreserveStructure: Preserve directory structure (if false, all files are flattened to destDir)
//   - 0 (no flags): Non-recursive move with flattened structure (default behavior)
//
// The exts parameter filters files by extension (case-insensitive, e.g., []string{".jpg", ".png"}).
// If nil or empty, no extension filtering is applied.
//
// # Example:
//
//	// Move recursively, preserving structure, filtering by extension
//	err := MoveFiles([]string{"src"}, "dest", CmRecursive|CmPreserveStructure, []string{".go"})
//
//	// Move recursively, flatten all files to destination
//	err := MoveFiles([]string{"src"}, "dest", CmRecursive, []string{".jpg", ".png"})
//
//	// Move only first-level files, preserve structure
//	err := MoveFiles([]string{"src"}, "dest", CmPreserveStructure, nil)
//
//	// Move only first-level files, flattened (no flags)
//	err := MoveFiles([]string{"src"}, "dest", 0, nil)
func MoveFiles(
	sources []string,
	destDir string,
	flags CmFlags,
	exts []string,
) error {
	return transferFiles(sources, destDir, flags, exts, true)
}

// region - Private functions

func transferFiles(
	sources []string,
	destDir string,
	flags CmFlags,
	exts []string,
	move bool,
) error {
	recursive := flags&CmRecursive != 0
	preserveStructure := flags&CmPreserveStructure != 0

	// If sources is empty, just create the destination directory
	if len(sources) == 0 {
		return os.MkdirAll(destDir, 0755)
	}

	// Normalize extensions to lowercase with leading dot
	normalizedExts := lo.Map(exts, func(ext string, _ int) string {
		ext = strings.ToLower(ext)
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		return ext
	})

	// Transfer each source
	for _, source := range sources {
		info, err := os.Stat(source)
		if err != nil {
			return fmt.Errorf("failed to stat source %s: %w", source, err)
		}

		if info.IsDir() {
			if err := transferDirectory(source, destDir, recursive, preserveStructure, normalizedExts, move); err != nil {
				return err
			}
		} else {
			if err := transferSingleFile(source, destDir, normalizedExts, move); err != nil {
				return err
			}
		}
	}

	return nil
}

func transferDirectory(source, destDir string, recursive, preserveStructure bool, normalizedExts []string, move bool) error {
	hasExtFilter := len(normalizedExts) > 0

	if preserveStructure {
		if err := copyWithStructure(source, destDir, recursive, hasExtFilter, normalizedExts); err != nil {
			return err
		}
	} else {
		if err := copyFlattened(source, destDir, recursive, hasExtFilter, normalizedExts); err != nil {
			return err
		}
	}

	// Remove source if moving
	if move {
		if hasExtFilter {
			return removeFilteredFiles(source, recursive, normalizedExts)
		}

		if recursive {
			return os.RemoveAll(source)
		}

		// Remove only first-level files
		entries, err := os.ReadDir(source)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", source, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				filePath := filepath.Join(source, entry.Name())
				if err := os.Remove(filePath); err != nil {
					return fmt.Errorf("failed to remove file %s: %w", filePath, err)
				}
			}
		}
	}

	return nil
}

func removeFilteredFiles(source string, recursive bool, normalizedExts []string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Handle directories
		if info.IsDir() {
			if !recursive && path != source {
				return filepath.SkipDir
			}
			return nil
		}

		// Remove files that match the extension filter
		ext := strings.ToLower(filepath.Ext(path))
		if slices.Contains(normalizedExts, ext) {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove file %s: %w", path, err)
			}
		}

		return nil
	})
}

func copyWithStructure(source, destDir string, recursive bool, hasExtFilter bool, normalizedExts []string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(source))

	opts := copy.Options{
		Skip: func(srcInfo os.FileInfo, src, dest string) (bool, error) {
			// Skip subdirectories if not recursive
			if !recursive && srcInfo.IsDir() && src != source {
				return true, nil
			}

			// Apply extension filter to files only
			if hasExtFilter && !srcInfo.IsDir() {
				ext := strings.ToLower(filepath.Ext(src))
				return !slices.Contains(normalizedExts, ext), nil
			}

			return false, nil
		},
	}

	if err := copy.Copy(source, destPath, opts); err != nil {
		return fmt.Errorf("failed to copy directory %s: %w", source, err)
	}
	return nil
}

func copyFlattened(source, destDir string, recursive bool, hasExtFilter bool, normalizedExts []string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Handle directories
		if info.IsDir() {
			if !recursive && path != source {
				return filepath.SkipDir
			}
			return nil
		}

		// Apply extension filter
		if hasExtFilter {
			ext := strings.ToLower(filepath.Ext(path))
			if !slices.Contains(normalizedExts, ext) {
				return nil
			}
		}

		// Copy file
		destPath := filepath.Join(destDir, filepath.Base(path))
		if err := copy.Copy(path, destPath); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", path, err)
		}

		return nil
	})
}

func transferSingleFile(source, destDir string, normalizedExts []string, move bool) error {
	if len(normalizedExts) > 0 {
		ext := strings.ToLower(filepath.Ext(source))
		if matched := slices.Contains(normalizedExts, ext); !matched {
			return nil
		}
	}

	// Ensure the destination directory exists before copying
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(source))
	if err := copy.Copy(source, destPath); err != nil {
		return fmt.Errorf("failed to copy file %s: %w", source, err)
	}

	// Remove source if moving
	if move {
		if err := os.Remove(source); err != nil {
			return fmt.Errorf("failed to remove source file %s: %w", source, err)
		}
	}

	return nil
}

// endregion
