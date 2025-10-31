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

type CopyFlags uint8

const (
	CpRecursive CopyFlags = 1 << iota
	CpPreserveStructure
)

// CopyFiles copies files and/or directories to a destination directory.
//
// The flags parameter controls the copy behavior:
//   - CpRecursive: Include subdirectories when copying directories
//   - CpPreserveStructure: Preserve directory structure (if false, all files are flattened to destDir)
//   - 0 (no flags): Non-recursive copy with flattened structure (default behavior)
//
// The exts parameter filters files by extension (case-insensitive, e.g., []string{".jpg", ".png"}).
// If nil or empty, no extension filtering is applied.
//
// # Example:
//
//	// Copy recursively, preserving structure, filtering by extension
//	err := CopyFiles([]string{"src"}, "dest", CpRecursive|CpPreserveStructure, []string{".go"})
//
//	// Copy recursively, flatten all files to destination
//	err := CopyFiles([]string{"src"}, "dest", CpRecursive, []string{".jpg", ".png"})
//
//	// Copy only first-level files, preserve structure
//	err := CopyFiles([]string{"src"}, "dest", CpPreserveStructure, nil)
//
//	// Copy only first-level files, flattened (no flags)
//	err := CopyFiles([]string{"src"}, "dest", 0, nil)
func CopyFiles(
	sources []string,
	destDir string,
	flags CopyFlags,
	exts []string,
) error {
	recursive := flags&CpRecursive != 0
	preserveStructure := flags&CpPreserveStructure != 0

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

	// Copy each source
	for _, source := range sources {
		info, err := os.Stat(source)
		if err != nil {
			return fmt.Errorf("failed to stat source %s: %w", source, err)
		}

		if info.IsDir() {
			if err := copyDirectory(source, destDir, recursive, preserveStructure, normalizedExts); err != nil {
				return err
			}
		} else {
			if err := copySingleFile(source, destDir, normalizedExts); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyDirectory(source, destDir string, recursive, preserveStructure bool, normalizedExts []string) error {
	hasExtFilter := len(normalizedExts) > 0

	if preserveStructure {
		return copyWithStructure(source, destDir, recursive, hasExtFilter, normalizedExts)
	}

	return copyFlattened(source, destDir, recursive, hasExtFilter, normalizedExts)
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

func copySingleFile(source, destDir string, normalizedExts []string) error {
	if len(normalizedExts) > 0 {
		ext := strings.ToLower(filepath.Ext(source))
		if matched := slices.Contains(normalizedExts, ext); !matched {
			return nil
		}
	}

	// Ensure destination directory exists before copying
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(source))
	if err := copy.Copy(source, destPath); err != nil {
		return fmt.Errorf("failed to copy file %s: %w", source, err)
	}

	return nil
}
