package fs

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestZip(t *testing.T, files map[string]string, dirs []string) string {
	// Create a temporary zip file
	zipFile, err := os.CreateTemp("", "test*.zip")
	require.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add directories
	for _, dir := range dirs {
		header := &zip.FileHeader{
			Name: dir + "/",
		}
		header.SetMode(0755 | os.ModeDir)
		_, err := zipWriter.CreateHeader(header)
		require.NoError(t, err)
	}

	// Add files
	for filename, content := range files {
		writer, err := zipWriter.Create(filename)
		require.NoError(t, err)
		_, err = writer.Write([]byte(content))
		require.NoError(t, err)
	}

	return zipFile.Name()
}

func createMaliciousZip(t *testing.T, filename string) string {
	zipFile, err := os.CreateTemp("", "malicious*.zip")
	require.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Create a file with a path that tries to escape the target directory
	writer, err := zipWriter.Create(filename)
	require.NoError(t, err)
	_, err = writer.Write([]byte("malicious content"))
	require.NoError(t, err)

	return zipFile.Name()
}

func TestUnzip(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Create test data
		files := map[string]string{
			"file1.txt":           "content of file1",
			"subdir/file2.txt":    "content of file2",
			"subdir/nested/file3": "content of file3",
		}
		dirs := []string{"emptydir"}

		zipPath := createTestZip(t, files, dirs)
		defer os.Remove(zipPath)

		// Create target directory
		targetDir, err := os.MkdirTemp("", "unzip_test*")
		require.NoError(t, err)
		defer os.RemoveAll(targetDir)

		// Test unzip
		err = Unzip(zipPath, targetDir)
		assert.NoError(t, err)

		// Verify extracted files
		for filename, expectedContent := range files {
			fullPath := filepath.Join(targetDir, filename)
			assert.FileExists(t, fullPath)

			content, err := os.ReadFile(fullPath)
			require.NoError(t, err)
			assert.Equal(t, expectedContent, string(content))

			// Check file permissions (should be 0755 as set by the function)
			info, err := os.Stat(fullPath)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
		}

		// Verify extracted directories
		for _, dir := range dirs {
			fullPath := filepath.Join(targetDir, dir)
			assert.DirExists(t, fullPath)
		}

		// Verify nested directories were created
		assert.DirExists(t, filepath.Join(targetDir, "subdir"))
		assert.DirExists(t, filepath.Join(targetDir, "subdir", "nested"))
	})

	t.Run("NonExistentZipFile", func(t *testing.T) {
		targetDir, err := os.MkdirTemp("", "unzip_test*")
		require.NoError(t, err)
		defer os.RemoveAll(targetDir)

		err = Unzip("/path/to/nonexistent.zip", targetDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("InvalidZipFile", func(t *testing.T) {
		// Create a file that's not a valid zip
		invalidZip, err := os.CreateTemp("", "invalid*.zip")
		require.NoError(t, err)
		defer os.Remove(invalidZip.Name())

		_, err = invalidZip.WriteString("this is not a zip file")
		require.NoError(t, err)
		invalidZip.Close()

		targetDir, err := os.MkdirTemp("", "unzip_test*")
		require.NoError(t, err)
		defer os.RemoveAll(targetDir)

		err = Unzip(invalidZip.Name(), targetDir)
		assert.Error(t, err)
	})

	t.Run("TargetDirectoryCreation", func(t *testing.T) {
		files := map[string]string{
			"test.txt": "test content",
		}
		zipPath := createTestZip(t, files, nil)
		defer os.Remove(zipPath)

		// Use a nested target directory that doesn't exist
		baseDir, err := os.MkdirTemp("", "unzip_test*")
		require.NoError(t, err)
		defer os.RemoveAll(baseDir)

		targetDir := filepath.Join(baseDir, "level1", "level2", "target")

		err = Unzip(zipPath, targetDir)
		assert.NoError(t, err)

		// Verify the directory was created and file extracted
		assert.DirExists(t, targetDir)
		assert.FileExists(t, filepath.Join(targetDir, "test.txt"))
	})

	t.Run("ZipSlipProtection", func(t *testing.T) {
		testCases := []struct {
			name     string
			filename string
		}{
			{
				name:     "path traversal with ../",
				filename: "../../../etc/passwd",
			},
			{
				name:     "absolute path",
				filename: "/etc/passwd",
			},
			{
				name:     "windows path traversal",
				filename: "..\\..\\windows\\system32\\config\\sam",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				zipPath := createMaliciousZip(t, tc.filename)
				defer os.Remove(zipPath)

				targetDir, err := os.MkdirTemp("", "unzip_test*")
				require.NoError(t, err)
				defer os.RemoveAll(targetDir)

				err = Unzip(zipPath, targetDir)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "illegal file path")
			})
		}
	})

	t.Run("EmptyZip", func(t *testing.T) {
		// Create empty zip
		zipPath := createTestZip(t, map[string]string{}, nil)
		defer os.Remove(zipPath)

		targetDir, err := os.MkdirTemp("", "unzip_test*")
		require.NoError(t, err)
		defer os.RemoveAll(targetDir)

		err = Unzip(zipPath, targetDir)
		assert.NoError(t, err)

		// Verify target directory exists but is empty (except for potential hidden files)
		entries, err := os.ReadDir(targetDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("ReadOnlyTargetDirectory", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping test when running as root")
		}

		files := map[string]string{
			"test.txt": "test content",
		}
		zipPath := createTestZip(t, files, nil)
		defer os.Remove(zipPath)

		// Create a read-only target directory
		baseDir, err := os.MkdirTemp("", "unzip_test*")
		require.NoError(t, err)
		defer os.RemoveAll(baseDir)

		targetDir := filepath.Join(baseDir, "readonly")
		err = os.Mkdir(targetDir, 0444) // read-only
		require.NoError(t, err)

		err = Unzip(zipPath, targetDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
	})

	t.Run("LargeFile", func(t *testing.T) {
		// Create a zip with a larger file
		largeContent := strings.Repeat("a", 10000)
		files := map[string]string{
			"large.txt": largeContent,
		}
		zipPath := createTestZip(t, files, nil)
		defer os.Remove(zipPath)

		targetDir, err := os.MkdirTemp("", "unzip_test*")
		require.NoError(t, err)
		defer os.RemoveAll(targetDir)

		err = Unzip(zipPath, targetDir)
		assert.NoError(t, err)

		// Verify the large file was extracted correctly
		extractedContent, err := os.ReadFile(filepath.Join(targetDir, "large.txt"))
		require.NoError(t, err)
		assert.Equal(t, largeContent, string(extractedContent))
	})

	t.Run("SpecialCharactersInFilenames", func(t *testing.T) {
		files := map[string]string{
			"file with spaces.txt":      "content1",
			"file-with-dashes.txt":      "content2",
			"file_with_underscores.txt": "content3",
		}
		zipPath := createTestZip(t, files, nil)
		defer os.Remove(zipPath)

		targetDir, err := os.MkdirTemp("", "unzip_test*")
		require.NoError(t, err)
		defer os.RemoveAll(targetDir)

		err = Unzip(zipPath, targetDir)
		assert.NoError(t, err)

		// Verify all files with special characters were extracted
		for filename, expectedContent := range files {
			fullPath := filepath.Join(targetDir, filename)
			assert.FileExists(t, fullPath)

			content, err := os.ReadFile(fullPath)
			require.NoError(t, err)
			assert.Equal(t, expectedContent, string(content))
		}
	})
}
