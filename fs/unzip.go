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
		fpath := filepath.Join(targetDirectory, f.Name)

		// Check for ZipSlip vulnerability: ensure that the file path is within the destination directory
		if !strings.HasPrefix(fpath, filepath.Clean(targetDirectory)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
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
