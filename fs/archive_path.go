package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// sanitizeArchivePath validates an archive entry name against path-traversal attacks and returns the safe on-disk path
// rooted at targetDir. It rejects absolute paths (Unix/Windows), ".." segments, and any entry whose final resolved
// location escapes targetDir.
func sanitizeArchivePath(entryName, targetDir string) (string, error) {
	// Normalize separators to forward slashes for analysis
	name := strings.ReplaceAll(entryName, "\\", "/")

	// Reject absolute paths before any trimming
	if isAbsolutePath(name) {
		return "", fmt.Errorf("illegal file path: %s", entryName)
	}

	// Reject any ".." segment
	for _, seg := range strings.Split(strings.TrimPrefix(name, "/"), "/") {
		if seg == ".." {
			return "", fmt.Errorf("illegal file path: %s", entryName)
		}
	}

	// Strip leading "./" and "/" to build a clean relative path
	cleanName := strings.TrimPrefix(name, "./")
	cleanName = strings.TrimPrefix(cleanName, "/")
	cleanName = strings.TrimPrefix(cleanName, "./")

	fpath := filepath.Join(targetDir, filepath.FromSlash(cleanName))

	// Defense-in-depth: confirm fpath stays within targetDir after join
	rel, relErr := filepath.Rel(targetDir, fpath)
	if relErr != nil || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return "", fmt.Errorf("illegal file path: %s", entryName)
	}

	return fpath, nil
}

// sanitizeArchiveSymlink validates that a symlink's target, when resolved relative to the symlink's on-disk location,
// stays within targetDir. It also rejects absolute symlink targets.
func sanitizeArchiveSymlink(linkPath, linkTarget, targetDir string) error {
	if filepath.IsAbs(linkTarget) {
		return fmt.Errorf("illegal symlink target: %s", linkTarget)
	}
	resolved := filepath.Join(filepath.Dir(linkPath), linkTarget)
	rel, relErr := filepath.Rel(targetDir, resolved)
	if relErr != nil || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return fmt.Errorf("illegal symlink target: %s", linkTarget)
	}
	return nil
}
