package fs

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPath(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "listpath_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test structure:
	// tempDir/
	//   ├── file1.txt
	//   ├── file2.go
	//   ├── file3.md
	//   ├── dir1/
	//   │   ├── file4.txt
	//   │   ├── file5.go
	//   │   └── subdir1/
	//   │       └── file6.txt
	//   └── dir2/
	//       └── file7.md

	// Create files and directories
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "file2.go"), []byte("content"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "file3.md"), []byte("content"), 0644))

	dir1 := filepath.Join(tempDir, "dir1")
	require.NoError(t, os.Mkdir(dir1, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "file4.txt"), []byte("content"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "file5.go"), []byte("content"), 0644))

	subdir1 := filepath.Join(dir1, "subdir1")
	require.NoError(t, os.Mkdir(subdir1, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir1, "file6.txt"), []byte("content"), 0644))

	dir2 := filepath.Join(tempDir, "dir2")
	require.NoError(t, os.Mkdir(dir2, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "file7.md"), []byte("content"), 0644))

	t.Run("list files only, non-recursive", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpFile, nil)
		require.NoError(t, err)
		sort.Strings(paths)

		expected := []string{
			filepath.Join(tempDir, "file1.txt"),
			filepath.Join(tempDir, "file2.go"),
			filepath.Join(tempDir, "file3.md"),
		}
		sort.Strings(expected)
		assert.Equal(t, expected, paths)
	})

	t.Run("list directories only, non-recursive", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpDir, nil)
		require.NoError(t, err)
		sort.Strings(paths)

		expected := []string{
			filepath.Join(tempDir, "dir1"),
			filepath.Join(tempDir, "dir2"),
		}
		sort.Strings(expected)
		assert.Equal(t, expected, paths)
	})

	t.Run("list files and directories, non-recursive", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpFile|LpDir, nil)
		require.NoError(t, err)
		sort.Strings(paths)

		expected := []string{
			filepath.Join(tempDir, "dir1"),
			filepath.Join(tempDir, "dir2"),
			filepath.Join(tempDir, "file1.txt"),
			filepath.Join(tempDir, "file2.go"),
			filepath.Join(tempDir, "file3.md"),
		}
		sort.Strings(expected)
		assert.Equal(t, expected, paths)
	})

	t.Run("list files recursively", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpFile|LpRecursive, nil)
		require.NoError(t, err)
		sort.Strings(paths)

		expected := []string{
			filepath.Join(tempDir, "file1.txt"),
			filepath.Join(tempDir, "file2.go"),
			filepath.Join(tempDir, "file3.md"),
			filepath.Join(dir1, "file4.txt"),
			filepath.Join(dir1, "file5.go"),
			filepath.Join(subdir1, "file6.txt"),
			filepath.Join(dir2, "file7.md"),
		}
		sort.Strings(expected)
		assert.Equal(t, expected, paths)
	})

	t.Run("list directories recursively", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpDir|LpRecursive, nil)
		require.NoError(t, err)
		sort.Strings(paths)

		expected := []string{
			filepath.Join(tempDir, "dir1"),
			filepath.Join(tempDir, "dir2"),
			filepath.Join(dir1, "subdir1"),
		}
		sort.Strings(expected)
		assert.Equal(t, expected, paths)
	})

	t.Run("list files and directories recursively", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpFile|LpDir|LpRecursive, nil)
		require.NoError(t, err)
		sort.Strings(paths)

		expected := []string{
			filepath.Join(tempDir, "dir1"),
			filepath.Join(tempDir, "dir2"),
			filepath.Join(tempDir, "file1.txt"),
			filepath.Join(tempDir, "file2.go"),
			filepath.Join(tempDir, "file3.md"),
			filepath.Join(dir1, "file4.txt"),
			filepath.Join(dir1, "file5.go"),
			filepath.Join(dir1, "subdir1"),
			filepath.Join(subdir1, "file6.txt"),
			filepath.Join(dir2, "file7.md"),
		}
		sort.Strings(expected)
		assert.Equal(t, expected, paths)
	})

	t.Run("filter by single extension, non-recursive", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpFile, []string{".txt"})
		require.NoError(t, err)
		sort.Strings(paths)

		expected := []string{
			filepath.Join(tempDir, "file1.txt"),
		}
		assert.Equal(t, expected, paths)
	})

	t.Run("filter by multiple extensions, non-recursive", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpFile, []string{".txt", ".go"})
		require.NoError(t, err)
		sort.Strings(paths)

		expected := []string{
			filepath.Join(tempDir, "file1.txt"),
			filepath.Join(tempDir, "file2.go"),
		}
		sort.Strings(expected)
		assert.Equal(t, expected, paths)
	})

	t.Run("filter by extension recursively", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpFile|LpRecursive, []string{".txt"})
		require.NoError(t, err)
		sort.Strings(paths)

		expected := []string{
			filepath.Join(tempDir, "file1.txt"),
			filepath.Join(dir1, "file4.txt"),
			filepath.Join(subdir1, "file6.txt"),
		}
		sort.Strings(expected)
		assert.Equal(t, expected, paths)
	})

	t.Run("case insensitive extension matching", func(t *testing.T) {
		// Create a file with uppercase extension
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "file_upper.TXT"), []byte("content"), 0644))

		paths, err := ListPath(tempDir, LpFile, []string{".txt"})
		require.NoError(t, err)

		// Should find both .txt and .TXT files
		found := false
		for _, path := range paths {
			if filepath.Base(path) == "file_upper.TXT" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find file with uppercase extension")
	})

	t.Run("no flags set returns empty result", func(t *testing.T) {
		paths, err := ListPath(tempDir, 0, nil)
		require.NoError(t, err)
		assert.Empty(t, paths)
	})

	t.Run("non-existent directory returns error", func(t *testing.T) {
		paths, err := ListPath("/non/existent/path", LpFile, nil)
		assert.Error(t, err)
		assert.NotNil(t, paths) // Should still return the slice
	})

	t.Run("empty directory", func(t *testing.T) {
		emptyDir, err := os.MkdirTemp("", "empty_test")
		require.NoError(t, err)
		defer os.RemoveAll(emptyDir)

		paths, err := ListPath(emptyDir, LpFile|LpDir|LpRecursive, nil)
		require.NoError(t, err)
		assert.Empty(t, paths)
	})

	t.Run("extension without dot", func(t *testing.T) {
		// Test that extensions without dots still work (though not recommended)
		paths, err := ListPath(tempDir, LpFile, []string{"txt"})
		require.NoError(t, err)
		// Should not match any files since extensions in filesystem have dots
		assert.Empty(t, paths)
	})
}

func TestListPathFlags(t *testing.T) {
	t.Run("flag combinations", func(t *testing.T) {
		// Test individual flags
		assert.Equal(t, ListFlags(1), LpDir)
		assert.Equal(t, ListFlags(2), LpFile)
		assert.Equal(t, ListFlags(4), LpRecursive)

		// Test flag combinations
		combined := LpDir | LpFile
		assert.NotEqual(t, ListFlags(0), combined&LpDir)
		assert.NotEqual(t, ListFlags(0), combined&LpFile)
		assert.Equal(t, ListFlags(0), combined&LpRecursive)
	})
}

func TestListPathEdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "listpath_edge_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create file with no extension
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "no_ext"), []byte("content"), 0644))

	// Create file with multiple dots
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "file.with.multiple.dots.txt"), []byte("content"), 0644))

	t.Run("file with no extension", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpFile, []string{""})
		require.NoError(t, err)

		found := false
		for _, path := range paths {
			if filepath.Base(path) == "no_ext" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find file with no extension when filtering for empty extension")
	})

	t.Run("file with multiple dots in name", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpFile, []string{".txt"})
		require.NoError(t, err)

		found := false
		for _, path := range paths {
			if filepath.Base(path) == "file.with.multiple.dots.txt" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find file with multiple dots when filtering by extension")
	})

	t.Run("nil extension slice", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpFile, nil)
		require.NoError(t, err)
		assert.Len(t, paths, 2, "Should find all files when extension filter is nil")
	})

	t.Run("empty extension slice", func(t *testing.T) {
		paths, err := ListPath(tempDir, LpFile, []string{})
		require.NoError(t, err)
		assert.Len(t, paths, 2, "Should find all files when extension filter is empty")
	})
}
