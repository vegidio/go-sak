package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyFiles(t *testing.T) {
	t.Run("copy single file without extension filter", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.WriteFile(srcFile, []byte("test content"), 0644))

		// Execute
		err := CopyFiles([]string{srcFile}, destDir, 0, nil)

		// Assert
		require.NoError(t, err)
		destFile := filepath.Join(destDir, "source.txt")
		assert.FileExists(t, destFile)
		content, err := os.ReadFile(destFile)
		require.NoError(t, err)
		assert.Equal(t, "test content", string(content))
	})

	t.Run("copy single file with matching extension filter", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.WriteFile(srcFile, []byte("test"), 0644))

		err := CopyFiles([]string{srcFile}, destDir, 0, []string{".txt"})

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "source.txt"))
	})

	t.Run("skip single file with non-matching extension filter", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.WriteFile(srcFile, []byte("test"), 0644))

		err := CopyFiles([]string{srcFile}, destDir, 0, []string{".jpg"})

		require.NoError(t, err)
		assert.NoDirExists(t, destDir)
	})

	t.Run("extension filter is case-insensitive", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.TXT")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.WriteFile(srcFile, []byte("test"), 0644))

		err := CopyFiles([]string{srcFile}, destDir, 0, []string{".txt"})

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "source.TXT"))
	})

	t.Run("extension filter without leading dot", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.jpg")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.WriteFile(srcFile, []byte("test"), 0644))

		err := CopyFiles([]string{srcFile}, destDir, 0, []string{"jpg"})

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "source.jpg"))
	})

	t.Run("copy directory non-recursively flattened", func(t *testing.T) {
		tempDir := t.TempDir()
		srcDir := filepath.Join(tempDir, "src")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file2.txt"), []byte("file2"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "subdir", "file3.txt"), []byte("file3"), 0644))

		err := CopyFiles([]string{srcDir}, destDir, 0, nil)

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "file1.txt"))
		assert.FileExists(t, filepath.Join(destDir, "file2.txt"))
		assert.NoFileExists(t, filepath.Join(destDir, "file3.txt"))
		assert.NoDirExists(t, filepath.Join(destDir, "subdir"))
	})

	t.Run("copy directory recursively flattened", func(t *testing.T) {
		tempDir := t.TempDir()
		srcDir := filepath.Join(tempDir, "src")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("file2"), 0644))

		err := CopyFiles([]string{srcDir}, destDir, CpRecursive, nil)

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "file1.txt"))
		assert.FileExists(t, filepath.Join(destDir, "file2.txt"))
		assert.NoDirExists(t, filepath.Join(destDir, "src"))
	})

	t.Run("copy directory non-recursively with structure preserved", func(t *testing.T) {
		tempDir := t.TempDir()
		srcDir := filepath.Join(tempDir, "src")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("file2"), 0644))

		err := CopyFiles([]string{srcDir}, destDir, CpPreserveStructure, nil)

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "src", "file1.txt"))
		assert.NoFileExists(t, filepath.Join(destDir, "src", "subdir", "file2.txt"))
	})

	t.Run("copy directory recursively with structure preserved", func(t *testing.T) {
		tempDir := t.TempDir()
		srcDir := filepath.Join(tempDir, "src")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("file2"), 0644))

		err := CopyFiles([]string{srcDir}, destDir, CpRecursive|CpPreserveStructure, nil)

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "src", "file1.txt"))
		assert.FileExists(t, filepath.Join(destDir, "src", "subdir", "file2.txt"))
	})

	t.Run("copy directory with extension filter flattened", func(t *testing.T) {
		tempDir := t.TempDir()
		srcDir := filepath.Join(tempDir, "src")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.MkdirAll(srcDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file2.jpg"), []byte("file2"), 0644))

		err := CopyFiles([]string{srcDir}, destDir, CpRecursive, []string{".txt"})

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "file1.txt"))
		assert.NoFileExists(t, filepath.Join(destDir, "file2.jpg"))
	})

	t.Run("copy directory with extension filter preserving structure", func(t *testing.T) {
		tempDir := t.TempDir()
		srcDir := filepath.Join(tempDir, "src")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file2.jpg"), []byte("file2"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "subdir", "file3.txt"), []byte("file3"), 0644))

		err := CopyFiles([]string{srcDir}, destDir, CpRecursive|CpPreserveStructure, []string{".txt"})

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "src", "file1.txt"))
		assert.NoFileExists(t, filepath.Join(destDir, "src", "file2.jpg"))
		assert.FileExists(t, filepath.Join(destDir, "src", "subdir", "file3.txt"))
	})

	t.Run("copy multiple sources", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile1 := filepath.Join(tempDir, "file1.txt")
		srcFile2 := filepath.Join(tempDir, "file2.txt")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.WriteFile(srcFile1, []byte("content1"), 0644))
		require.NoError(t, os.WriteFile(srcFile2, []byte("content2"), 0644))

		err := CopyFiles([]string{srcFile1, srcFile2}, destDir, 0, nil)

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "file1.txt"))
		assert.FileExists(t, filepath.Join(destDir, "file2.txt"))
	})

	t.Run("copy multiple directories with mixed flags", func(t *testing.T) {
		tempDir := t.TempDir()
		srcDir1 := filepath.Join(tempDir, "src1")
		srcDir2 := filepath.Join(tempDir, "src2")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.MkdirAll(srcDir1, 0755))
		require.NoError(t, os.MkdirAll(srcDir2, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir1, "file1.txt"), []byte("content1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir2, "file2.txt"), []byte("content2"), 0644))

		err := CopyFiles([]string{srcDir1, srcDir2}, destDir, CpRecursive|CpPreserveStructure, nil)

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "src1", "file1.txt"))
		assert.FileExists(t, filepath.Join(destDir, "src2", "file2.txt"))
	})

	t.Run("destination directory is created if it doesn't exist", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		destDir := filepath.Join(tempDir, "dest", "nested", "path")

		require.NoError(t, os.WriteFile(srcFile, []byte("test"), 0644))

		err := CopyFiles([]string{srcFile}, destDir, 0, nil)

		require.NoError(t, err)
		assert.DirExists(t, destDir)
		assert.FileExists(t, filepath.Join(destDir, "source.txt"))
	})

	t.Run("error when source does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "nonexistent.txt")
		destDir := filepath.Join(tempDir, "dest")

		err := CopyFiles([]string{srcFile}, destDir, 0, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to stat source")
	})

	t.Run("multiple extension filters", func(t *testing.T) {
		tempDir := t.TempDir()
		srcDir := filepath.Join(tempDir, "src")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.MkdirAll(srcDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file2.jpg"), []byte("file2"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file3.png"), []byte("file3"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file4.pdf"), []byte("file4"), 0644))

		err := CopyFiles([]string{srcDir}, destDir, CpRecursive, []string{".jpg", ".png"})

		require.NoError(t, err)
		assert.NoFileExists(t, filepath.Join(destDir, "file1.txt"))
		assert.FileExists(t, filepath.Join(destDir, "file2.jpg"))
		assert.FileExists(t, filepath.Join(destDir, "file3.png"))
		assert.NoFileExists(t, filepath.Join(destDir, "file4.pdf"))
	})

	t.Run("deep nested directory structure", func(t *testing.T) {
		tempDir := t.TempDir()
		srcDir := filepath.Join(tempDir, "src")
		deepDir := filepath.Join(srcDir, "level1", "level2", "level3")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.MkdirAll(deepDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(deepDir, "deep.txt"), []byte("deep"), 0644))

		err := CopyFiles([]string{srcDir}, destDir, CpRecursive|CpPreserveStructure, nil)

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "src", "level1", "level2", "level3", "deep.txt"))
	})

	t.Run("empty source directory", func(t *testing.T) {
		tempDir := t.TempDir()
		srcDir := filepath.Join(tempDir, "src")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.MkdirAll(srcDir, 0755))

		err := CopyFiles([]string{srcDir}, destDir, CpRecursive|CpPreserveStructure, nil)

		require.NoError(t, err)
		assert.DirExists(t, filepath.Join(destDir, "src"))
	})

	t.Run("file name collision in flattened copy", func(t *testing.T) {
		tempDir := t.TempDir()
		srcDir := filepath.Join(tempDir, "src")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "dir1"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "dir2"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "dir1", "file.txt"), []byte("content1"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "dir2", "file.txt"), []byte("content2"), 0644))

		err := CopyFiles([]string{srcDir}, destDir, CpRecursive, nil)

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(destDir, "file.txt"))
		// Note: One file will overwrite the other in flattened mode
	})

	t.Run("empty sources slice", func(t *testing.T) {
		tempDir := t.TempDir()
		destDir := filepath.Join(tempDir, "dest")

		err := CopyFiles([]string{}, destDir, 0, nil)

		require.NoError(t, err)
		assert.DirExists(t, destDir)
	})

	t.Run("file without extension with extension filter", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "README")
		destDir := filepath.Join(tempDir, "dest")

		require.NoError(t, os.WriteFile(srcFile, []byte("test"), 0644))

		err := CopyFiles([]string{srcFile}, destDir, 0, []string{".txt"})

		require.NoError(t, err)
		assert.NoDirExists(t, destDir)
	})
}
