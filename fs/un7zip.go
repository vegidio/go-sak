package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bodgit/sevenzip"
)

// Un7zip extracts all files and directories from a 7z archive to a target directory. It creates the target directory if
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
//   - sevenZipPath: Path to the 7z file to extract
//   - targetDirectory: Destination directory where files will be extracted
//
// # Returns an error if:
//   - The 7z file cannot be opened or read
//   - The target directory cannot be created
//   - Any archive entry contains an illegal path (absolute or traversal)
//   - File extraction fails due to I/O errors or permission issues
//
// All extracted regular files are set to executable mode (0o755).
func Un7zip(sevenZipPath, targetDirectory string) error {
	// Open the 7z file specified by sevenZipPath
	r, err := sevenzip.OpenReader(sevenZipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Ensure the destination directory exists
	if err = os.MkdirAll(targetDirectory, 0o755); err != nil {
		return err
	}

	// Iterate through each file in the 7z archive
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

			// Read the symlink target from the 7z file
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

		// Open the file inside the 7z archive
		rc, fErr := f.Open()
		if fErr != nil {
			outFile.Close()
			return fErr
		}

		// Copy file contents from the 7z archive to a destination file
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
