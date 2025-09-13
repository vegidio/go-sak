package fs

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Unzip(zipPath, targetDirectory string) error {
	// Open the zip file specified by zipPath
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Ensure the destination directory exists
	if err = os.MkdirAll(targetDirectory, 0755); err != nil {
		return err
	}

	// Iterate through each file in the zip archive
	for _, f := range r.File {
		// Normalize the archive entry name to guard against Zip Slip attacks; Zip spec uses forward slashes as
		// separators, but we must be defensive and also handle Windows-style backslashes provided by some tools.
		name := f.Name
		name = strings.ReplaceAll(name, "\\", "/")

		// Clean the path using slash-based semantics.
		// We must reject absolute paths and any path that escapes the target via ".."
		cleanName := strings.TrimPrefix(name, "./")
		cleanName = strings.TrimPrefix(cleanName, "/") // remove leading slash to test separately
		cleanName = strings.TrimPrefix(cleanName, "./")

		// After trimming, if the original had absolute or traversal intent, detect it explicitly
		if isAbsolutePath(name) {
			return fmt.Errorf("illegal file path: %s", f.Name)
		}

		// Check for path traversal using normalized segments
		segments := strings.Split(strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "/"), "/")
		for _, seg := range segments {
			if seg == ".." {
				return fmt.Errorf("illegal file path: %s", f.Name)
			}
		}

		// Construct the destination path
		fpath := filepath.Join(targetDirectory, filepath.FromSlash(cleanName))

		// Ensure the final path is within the target directory using Rel check
		rel, relErr := filepath.Rel(targetDirectory, fpath)
		if relErr != nil || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
			return fmt.Errorf("illegal file path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			// Create a directory if it doesn't exist
			if err = os.MkdirAll(fpath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		// Ensure the parent directory exists
		if err = os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		// Create the destination file
		outFile, fErr := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if fErr != nil {
			return fErr
		}

		// Make the file executable
		if err = os.Chmod(fpath, 0755); err != nil {
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
