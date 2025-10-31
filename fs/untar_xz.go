package fs

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

// UntarXz extracts all files and directories from a TAR.XZ archive to a target directory. It creates the target
// directory if it doesn't exist and preserves the directory structure from the archive.
//
// The function implements security measures to prevent path traversal attacks by:
//   - Rejecting absolute paths in archive entries
//   - Preventing path traversal attacks using ".." segments
//   - Normalizing path separators to handle both forward slashes and backslashes
//   - Validating that extracted files remain within the target directory
//
// # Parameters:
//   - tarXzPath: Path to the TAR.XZ file to extract
//   - targetDirectory: Destination directory where files will be extracted
//
// # Returns an error if:
//   - The TAR.XZ file cannot be opened or read
//   - The target directory cannot be created
//   - Any archive entry contains an illegal path (absolute or traversal)
//   - File extraction fails due to I/O errors or permission issues
//
// All extracted files preserve their original permissions from the archive.
func UntarXz(tarXzPath, targetDirectory string) error {
	// Open the tar.xz file
	f, err := os.Open(tarXzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Create XZ reader
	xzReader, err := xz.NewReader(f)
	if err != nil {
		return err
	}

	// Create a tar reader
	tarReader := tar.NewReader(xzReader)

	// Ensure the destination directory exists
	if err = os.MkdirAll(targetDirectory, 0755); err != nil {
		return err
	}

	// Iterate through each file in the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}

		// Normalize the archive entry name to guard against path traversal attacks
		name := header.Name
		name = strings.ReplaceAll(name, "\\", "/")

		// Clean the path
		cleanName := strings.TrimPrefix(name, "./")
		cleanName = strings.TrimPrefix(cleanName, "/")
		cleanName = strings.TrimPrefix(cleanName, "./")

		// Check for absolute paths
		if isAbsolutePath(name) {
			return fmt.Errorf("illegal file path: %s", header.Name)
		}

		// Check for path traversal using normalized segments
		segments := strings.Split(strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "/"), "/")
		for _, seg := range segments {
			if seg == ".." {
				return fmt.Errorf("illegal file path: %s", header.Name)
			}
		}

		// Construct the destination path
		fpath := filepath.Join(targetDirectory, filepath.FromSlash(cleanName))

		// Ensure the final path is within the target directory
		rel, relErr := filepath.Rel(targetDirectory, fpath)
		if relErr != nil || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
			return fmt.Errorf("illegal file path: %s", header.Name)
		}

		// Handle different file types
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err = os.MkdirAll(fpath, os.FileMode(header.Mode)); err != nil {
				return err
			}

		case tar.TypeReg:
			// Ensure the parent directory exists
			if err = os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return err
			}

			// Create the destination file
			outFile, fErr := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if fErr != nil {
				return fErr
			}

			// Use buffered writer for better performance with large files
			bufWriter := bufio.NewWriterSize(outFile, 1024*1024) // 1MB buffer

			// Copy file contents from the tar archive to destination file
			if _, err = io.Copy(bufWriter, tarReader); err != nil {
				outFile.Close()
				return err
			}

			// Flush the buffer before closing
			if err = bufWriter.Flush(); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

		case tar.TypeSymlink:
			// Create a symbolic link
			if err = os.Symlink(header.Linkname, fpath); err != nil {
				return err
			}

		default:
			// Skip other types (block devices, char devices, FIFOs, etc.)
			continue
		}
	}

	return nil
}
