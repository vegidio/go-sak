package fs

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

func TestUntarXz(t *testing.T) {
	t.Run("successful extraction with files and directories", func(t *testing.T) {
		// Create a temporary tar.xz file with test content
		tarXzPath := createTestTarXz(t, map[string]testEntry{
			"file1.txt":           {content: "content1", isDir: false, mode: 0644},
			"dir1/":               {isDir: true, mode: 0755},
			"dir1/file2.txt":      {content: "content2", isDir: false, mode: 0644},
			"dir1/dir2/":          {isDir: true, mode: 0755},
			"dir1/dir2/file3.txt": {content: "content3", isDir: false, mode: 0600},
		})
		defer os.Remove(tarXzPath)

		targetDir := t.TempDir()

		err := UntarXz(tarXzPath, targetDir)
		require.NoError(t, err)

		// Verify extracted files and directories
		assertFileExists(t, filepath.Join(targetDir, "file1.txt"), "content1")
		assert.DirExists(t, filepath.Join(targetDir, "dir1"))
		assertFileExists(t, filepath.Join(targetDir, "dir1", "file2.txt"), "content2")
		assert.DirExists(t, filepath.Join(targetDir, "dir1", "dir2"))
		assertFileExists(t, filepath.Join(targetDir, "dir1", "dir2", "file3.txt"), "content3")
	})

	t.Run("successful extraction with leading ./ in paths", func(t *testing.T) {
		tarXzPath := createTestTarXz(t, map[string]testEntry{
			"./file1.txt":      {content: "content1", isDir: false, mode: 0644},
			"./dir1/":          {isDir: true, mode: 0755},
			"./dir1/file2.txt": {content: "content2", isDir: false, mode: 0644},
		})
		defer os.Remove(tarXzPath)

		targetDir := t.TempDir()

		err := UntarXz(tarXzPath, targetDir)
		require.NoError(t, err)

		assertFileExists(t, filepath.Join(targetDir, "file1.txt"), "content1")
		assert.DirExists(t, filepath.Join(targetDir, "dir1"))
		assertFileExists(t, filepath.Join(targetDir, "dir1", "file2.txt"), "content2")
	})

	t.Run("successful extraction preserves file permissions", func(t *testing.T) {
		tarXzPath := createTestTarXz(t, map[string]testEntry{
			"executable.sh": {content: "#!/bin/bash", isDir: false, mode: 0755},
			"readonly.txt":  {content: "read only", isDir: false, mode: 0444},
		})
		defer os.Remove(tarXzPath)

		targetDir := t.TempDir()

		err := UntarXz(tarXzPath, targetDir)
		require.NoError(t, err)

		execInfo, err := os.Stat(filepath.Join(targetDir, "executable.sh"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), execInfo.Mode().Perm())

		readonlyInfo, err := os.Stat(filepath.Join(targetDir, "readonly.txt"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0444), readonlyInfo.Mode().Perm())
	})

	t.Run("creates target directory if not exists", func(t *testing.T) {
		tarXzPath := createTestTarXz(t, map[string]testEntry{
			"file.txt": {content: "test", isDir: false, mode: 0644},
		})
		defer os.Remove(tarXzPath)

		targetDir := filepath.Join(t.TempDir(), "newdir", "nested")

		err := UntarXz(tarXzPath, targetDir)
		require.NoError(t, err)

		assert.DirExists(t, targetDir)
		assertFileExists(t, filepath.Join(targetDir, "file.txt"), "test")
	})

	t.Run("error when tar.xz file does not exist", func(t *testing.T) {
		targetDir := t.TempDir()

		err := UntarXz("nonexistent.tar.xz", targetDir)
		assert.Error(t, err)
	})

	t.Run("error when file is not a valid tar.xz", func(t *testing.T) {
		invalidFile := filepath.Join(t.TempDir(), "invalid.tar.xz")
		err := os.WriteFile(invalidFile, []byte("not a tar.xz file"), 0644)
		require.NoError(t, err)

		targetDir := t.TempDir()

		err = UntarXz(invalidFile, targetDir)
		assert.Error(t, err)
	})

	t.Run("error on path traversal with ..", func(t *testing.T) {
		tarXzPath := createTestTarXz(t, map[string]testEntry{
			"../escape.txt": {content: "malicious", isDir: false, mode: 0644},
		})
		defer os.Remove(tarXzPath)

		targetDir := t.TempDir()

		err := UntarXz(tarXzPath, targetDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal file path")
	})

	t.Run("error on path traversal with nested ..", func(t *testing.T) {
		tarXzPath := createTestTarXz(t, map[string]testEntry{
			"dir/../../../escape.txt": {content: "malicious", isDir: false, mode: 0644},
		})
		defer os.Remove(tarXzPath)

		targetDir := t.TempDir()

		err := UntarXz(tarXzPath, targetDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal file path")
	})

	t.Run("error on absolute path (Unix)", func(t *testing.T) {
		tarXzPath := createTestTarXz(t, map[string]testEntry{
			"/etc/passwd": {content: "malicious", isDir: false, mode: 0644},
		})
		defer os.Remove(tarXzPath)

		targetDir := t.TempDir()

		err := UntarXz(tarXzPath, targetDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal file path")
	})

	t.Run("handles empty archive", func(t *testing.T) {
		tarXzPath := createTestTarXz(t, map[string]testEntry{})
		defer os.Remove(tarXzPath)

		targetDir := t.TempDir()

		err := UntarXz(tarXzPath, targetDir)
		require.NoError(t, err)

		assert.DirExists(t, targetDir)
	})

	t.Run("handles symbolic links", func(t *testing.T) {
		tarXzPath := createTestTarXzWithSymlink(t, map[string]testEntry{
			"file.txt": {content: "target content", isDir: false, mode: 0644},
		}, "link.txt", "file.txt")
		defer os.Remove(tarXzPath)

		targetDir := t.TempDir()

		err := UntarXz(tarXzPath, targetDir)
		require.NoError(t, err)

		linkPath := filepath.Join(targetDir, "link.txt")
		linkInfo, err := os.Lstat(linkPath)
		require.NoError(t, err)
		assert.Equal(t, os.ModeSymlink, linkInfo.Mode()&os.ModeSymlink)

		target, err := os.Readlink(linkPath)
		require.NoError(t, err)
		assert.Equal(t, "file.txt", target)
	})

	t.Run("handles files with backslashes in names", func(t *testing.T) {
		tarXzPath := createTestTarXz(t, map[string]testEntry{
			"dir\\file.txt": {content: "content", isDir: false, mode: 0644},
		})
		defer os.Remove(tarXzPath)

		targetDir := t.TempDir()

		err := UntarXz(tarXzPath, targetDir)
		require.NoError(t, err)

		// Should normalize to forward slash
		expectedPath := filepath.Join(targetDir, "dir", "file.txt")
		assertFileExists(t, expectedPath, "content")
	})

	t.Run("overwrites existing files", func(t *testing.T) {
		targetDir := t.TempDir()
		existingFile := filepath.Join(targetDir, "file.txt")
		err := os.WriteFile(existingFile, []byte("old content"), 0644)
		require.NoError(t, err)

		tarXzPath := createTestTarXz(t, map[string]testEntry{
			"file.txt": {content: "new content", isDir: false, mode: 0644},
		})
		defer os.Remove(tarXzPath)

		err = UntarXz(tarXzPath, targetDir)
		require.NoError(t, err)

		assertFileExists(t, existingFile, "new content")
	})
}

// testEntry represents an entry in a test tar archive
type testEntry struct {
	content string
	isDir   bool
	mode    os.FileMode
}

// createTestTarXz creates a temporary tar.xz file with the specified entries
func createTestTarXz(t *testing.T, entries map[string]testEntry) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-*.tar.xz")
	require.NoError(t, err)
	defer tmpFile.Close()

	xzWriter, err := xz.NewWriter(tmpFile)
	require.NoError(t, err)
	defer xzWriter.Close()

	tarWriter := tar.NewWriter(xzWriter)
	defer tarWriter.Close()

	for name, entry := range entries {
		var header *tar.Header
		if entry.isDir {
			header = &tar.Header{
				Name:     name,
				Mode:     int64(entry.mode),
				Typeflag: tar.TypeDir,
			}
		} else {
			header = &tar.Header{
				Name:     name,
				Mode:     int64(entry.mode),
				Size:     int64(len(entry.content)),
				Typeflag: tar.TypeReg,
			}
		}

		err = tarWriter.WriteHeader(header)
		require.NoError(t, err)

		if !entry.isDir {
			_, err = tarWriter.Write([]byte(entry.content))
			require.NoError(t, err)
		}
	}

	return tmpFile.Name()
}

// createTestTarXzWithSymlink creates a tar.xz file with entries and a symbolic link
func createTestTarXzWithSymlink(t *testing.T, entries map[string]testEntry, linkName, linkTarget string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-*.tar.xz")
	require.NoError(t, err)
	defer tmpFile.Close()

	xzWriter, err := xz.NewWriter(tmpFile)
	require.NoError(t, err)
	defer xzWriter.Close()

	tarWriter := tar.NewWriter(xzWriter)
	defer tarWriter.Close()

	for name, entry := range entries {
		var header *tar.Header
		if entry.isDir {
			header = &tar.Header{
				Name:     name,
				Mode:     int64(entry.mode),
				Typeflag: tar.TypeDir,
			}
		} else {
			header = &tar.Header{
				Name:     name,
				Mode:     int64(entry.mode),
				Size:     int64(len(entry.content)),
				Typeflag: tar.TypeReg,
			}
		}

		err = tarWriter.WriteHeader(header)
		require.NoError(t, err)

		if !entry.isDir {
			_, err = tarWriter.Write([]byte(entry.content))
			require.NoError(t, err)
		}
	}

	// Add symbolic link
	symlinkHeader := &tar.Header{
		Name:     linkName,
		Linkname: linkTarget,
		Typeflag: tar.TypeSymlink,
	}
	err = tarWriter.WriteHeader(symlinkHeader)
	require.NoError(t, err)

	return tmpFile.Name()
}

// assertFileExists checks if a file exists and has the expected content
func assertFileExists(t *testing.T, path, expectedContent string) {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, string(content))
}
