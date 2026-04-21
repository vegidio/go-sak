package fs

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Unzip extracts all files and directories from a ZIP archive to a target directory. It creates the target directory if
// it doesn't exist and preserves the directory structure from the archive, including symbolic links.
//
// The function implements security measures to prevent Zip Slip attacks by:
//   - Rejecting absolute paths in archive entries
//   - Preventing path traversal attacks using ".." segments
//   - Normalizing path separators to handle both forward slashes and backslashes
//   - Validating that extracted files remain within the target directory
//   - Validating that symbolic link targets remain within the target directory
//
// # Parameters:
//   - zipPath: Path to the ZIP file to extract
//   - targetDirectory: Destination directory where files will be extracted
//
// # Returns an error if:
//   - The ZIP file cannot be opened or read
//   - The target directory cannot be created
//   - Any archive entry contains an illegal path (absolute or traversal)
//   - File extraction fails due to I/O errors or permission issues
//
// All extracted regular files are set to executable mode (0o755).
func Unzip(zipPath, targetDirectory string) error {
	// Open the zip file specified by zipPath
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Ensure the destination directory exists
	if err = os.MkdirAll(targetDirectory, 0o755); err != nil {
		return err
	}

	// Iterate through each file in the zip archive
	for _, f := range r.File {
		fpath, err := sanitizeArchivePath(f.Name, targetDirectory)
		if err != nil {
			return err
		}

		if f.FileInfo().IsDir() {
			// Create a directory if it doesn't exist
			if err = os.MkdirAll(fpath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		// Check if this is a symbolic link
		if f.Mode()&os.ModeSymlink != 0 {
			// Ensure the parent directory exists
			if err = os.MkdirAll(filepath.Dir(fpath), 0o755); err != nil {
				return err
			}

			// Read the symlink target from the zip file
			rc, fErr := f.Open()
			if fErr != nil {
				return fErr
			}

			linkTarget, fErr := io.ReadAll(rc)
			rc.Close()
			if fErr != nil {
				return fErr
			}

			linkTargetStr := string(linkTarget)

			if err = sanitizeArchiveSymlink(fpath, linkTargetStr, targetDirectory); err != nil {
				return fmt.Errorf("illegal symlink target: %s -> %s", f.Name, linkTargetStr)
			}

			// Create the symbolic link
			if err = os.Symlink(linkTargetStr, fpath); err != nil {
				return err
			}
			continue
		}

		// Ensure the parent directory exists
		if err = os.MkdirAll(filepath.Dir(fpath), 0o755); err != nil {
			return err
		}

		// Create the destination file
		outFile, fErr := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if fErr != nil {
			return fErr
		}

		// Make the file executable
		if err = os.Chmod(fpath, 0o755); err != nil {
			return err
		}

		// Open the file inside the zip archive
		rc, fErr := f.Open()
		if fErr != nil {
			outFile.Close()
			return fErr
		}

		// Copy file contents from the zip archive to a destination file
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// region - Private functions

func isAbsolutePath(path string) bool {
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return true
	}

	// Check for Windows drive paths (C:, D:, etc.)
	if len(path) >= 2 && path[1] == ':' {
		firstChar := path[0]
		return (firstChar >= 'A' && firstChar <= 'Z') || (firstChar >= 'a' && firstChar <= 'z')
	}

	return false
}

// endregion
