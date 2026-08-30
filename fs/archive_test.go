package fs

import (
	"archive/tar"
	"archive/zip"
	"errors"
	"io"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

// region - Ordered archive builders
//
// The helpers in unzip_test.go and untar_xz_test.go take a map, so Go randomizes entry order. The chained-symlink
// attack depends entirely on ordering - the poisoned link must be extracted before the entry that walks through it -
// so these builders take a slice instead.

type entrySpec struct {
	name     string
	typeflag byte // tar.TypeReg, tar.TypeDir or tar.TypeSymlink
	linkname string
	mode     int64
	content  string
}

func createOrderedTarXz(t *testing.T, entries []entrySpec) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "ordered-*.tar.xz")
	require.NoError(t, err)
	defer tmpFile.Close()

	xzWriter, err := xz.NewWriter(tmpFile)
	require.NoError(t, err)

	tarWriter := tar.NewWriter(xzWriter)

	for _, e := range entries {
		mode := e.mode
		if mode == 0 {
			mode = 0o644
		}

		header := &tar.Header{
			Name:     e.name,
			Typeflag: e.typeflag,
			Linkname: e.linkname,
			Mode:     mode,
		}
		if e.typeflag == tar.TypeReg {
			header.Size = int64(len(e.content))
		}

		require.NoError(t, tarWriter.WriteHeader(header))

		if e.typeflag == tar.TypeReg {
			_, err = tarWriter.Write([]byte(e.content))
			require.NoError(t, err)
		}
	}

	require.NoError(t, tarWriter.Close())
	require.NoError(t, xzWriter.Close())

	return tmpFile.Name()
}

func createOrderedZip(t *testing.T, entries []entrySpec) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "ordered-*.zip")
	require.NoError(t, err)
	defer tmpFile.Close()

	zipWriter := zip.NewWriter(tmpFile)

	for _, e := range entries {
		header := &zip.FileHeader{Name: e.name, Method: zip.Deflate}

		switch e.typeflag {
		case tar.TypeDir:
			header.Name += "/"
			header.SetMode(os.ModeDir | 0o755)
		case tar.TypeSymlink:
			// A zip archive stores the symlink target as the entry's own content.
			header.SetMode(os.ModeSymlink | 0o777)
		default:
			mode := os.FileMode(e.mode)
			if mode == 0 {
				mode = 0o644
			}
			header.SetMode(mode)
		}

		w, err := zipWriter.CreateHeader(header)
		require.NoError(t, err)

		body := e.content
		if e.typeflag == tar.TypeSymlink {
			body = e.linkname
		}
		if body != "" {
			_, err = w.Write([]byte(body))
			require.NoError(t, err)
		}
	}

	require.NoError(t, zipWriter.Close())

	return tmpFile.Name()
}

// endregion

// region - Chained-symlink escape

// chainedSymlinkEntries builds the archive that defeats a purely lexical symlink check.
//
// Every entry passes validation when only its own name and target are considered:
//
//	p         -> "."   resolves to the root itself
//	p/q       -> ".."  resolves to the root, because "p" lexically looks like a directory
//	p/q/pwned          contains no ".." and is not absolute
//
// On disk, however, "p" is a link to the root, so "p/q" really lands at "<root>/q" pointing at the root's PARENT, and
// the regular file then resolves to "<root>/../pwned".
func chainedSymlinkEntries() []entrySpec {
	return []entrySpec{
		{name: "p", typeflag: tar.TypeSymlink, linkname: "."},
		{name: "p/q", typeflag: tar.TypeSymlink, linkname: ".."},
		{name: "p/q/pwned", typeflag: tar.TypeReg, mode: 0o644, content: "owned"},
	}
}

// deepChainedSymlinkEntries is the same attack one level deeper, to confirm the probe is not merely catching a
// two-link chain by accident.
func deepChainedSymlinkEntries() []entrySpec {
	return []entrySpec{
		{name: "p", typeflag: tar.TypeSymlink, linkname: "."},
		{name: "p/q", typeflag: tar.TypeSymlink, linkname: "."},
		{name: "p/q/r", typeflag: tar.TypeSymlink, linkname: ".."},
		{name: "p/q/r/pwned", typeflag: tar.TypeReg, mode: 0o644, content: "owned"},
	}
}

func assertNoEscape(t *testing.T, err error, outer, targetDir string) {
	t.Helper()

	require.Error(t, err, "a chained-symlink archive must not extract cleanly")
	assert.ErrorIs(t, err, ErrIllegalSymlink)

	// The escape itself must not have happened...
	assert.NoFileExists(t, filepath.Join(outer, "pwned"))

	// ...and the poisoned link must not be left behind for a later, non-os.Root consumer of the extracted tree.
	_, statErr := os.Lstat(filepath.Join(targetDir, "q"))
	assert.True(t, os.IsNotExist(statErr), "escaping symlink must be removed, got %v", statErr)
}

func TestUntarXz_ChainedSymlinkEscape(t *testing.T) {
	t.Run("Depth2", func(t *testing.T) {
		outer := t.TempDir()
		targetDir := filepath.Join(outer, "out")

		archive := createOrderedTarXz(t, chainedSymlinkEntries())
		defer os.Remove(archive)

		assertNoEscape(t, UntarXz(archive, targetDir), outer, targetDir)
	})

	t.Run("Depth3", func(t *testing.T) {
		outer := t.TempDir()
		targetDir := filepath.Join(outer, "out")

		archive := createOrderedTarXz(t, deepChainedSymlinkEntries())
		defer os.Remove(archive)

		require.Error(t, UntarXz(archive, targetDir))
		assert.NoFileExists(t, filepath.Join(outer, "pwned"))
	})
}

func TestUnzip_ChainedSymlinkEscape(t *testing.T) {
	outer := t.TempDir()
	targetDir := filepath.Join(outer, "out")

	archive := createOrderedZip(t, chainedSymlinkEntries())
	defer os.Remove(archive)

	assertNoEscape(t, Unzip(archive, targetDir), outer, targetDir)
}

func TestUnzip_SymlinkHandling(t *testing.T) {
	t.Run("ValidRelativeLinkRoundTrips", func(t *testing.T) {
		targetDir := t.TempDir()

		archive := createOrderedZip(t, []entrySpec{
			{name: "real.txt", typeflag: tar.TypeReg, mode: 0o644, content: "hello"},
			{name: "sub/link.txt", typeflag: tar.TypeSymlink, linkname: "../real.txt"},
		})
		defer os.Remove(archive)

		require.NoError(t, Unzip(archive, targetDir))

		target, err := os.Readlink(filepath.Join(targetDir, "sub", "link.txt"))
		require.NoError(t, err)
		assert.Equal(t, "../real.txt", target)

		content, err := os.ReadFile(filepath.Join(targetDir, "sub", "link.txt"))
		require.NoError(t, err)
		assert.Equal(t, "hello", string(content))
	})

	t.Run("AbsoluteTargetRejected", func(t *testing.T) {
		targetDir := t.TempDir()

		archive := createOrderedZip(t, []entrySpec{
			{name: "link", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"},
		})
		defer os.Remove(archive)

		err := Unzip(archive, targetDir)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIllegalSymlink)
		assert.Contains(t, err.Error(), "illegal symlink target")
	})

	t.Run("TraversalTargetRejected", func(t *testing.T) {
		targetDir := t.TempDir()

		archive := createOrderedZip(t, []entrySpec{
			{name: "link", typeflag: tar.TypeSymlink, linkname: "../../../etc/passwd"},
		})
		defer os.Remove(archive)

		err := Unzip(archive, targetDir)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIllegalSymlink)
	})

	t.Run("WithoutSymlinksRejectsAny", func(t *testing.T) {
		targetDir := t.TempDir()

		archive := createOrderedZip(t, []entrySpec{
			{name: "real.txt", typeflag: tar.TypeReg, mode: 0o644, content: "hello"},
			{name: "link", typeflag: tar.TypeSymlink, linkname: "real.txt"},
		})
		defer os.Remove(archive)

		err := Unzip(archive, targetDir, WithoutSymlinks())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIllegalSymlink)
	})
}

// endregion

// region - Extraction core
//
// These tests drive extractArchive directly with a synthetic iterator. That keeps them independent of any archive
// writer, which matters because the sevenzip library is read-only and cannot produce fixtures programmatically.

func staticEntries(entries []archiveEntry) iter.Seq2[archiveEntry, error] {
	return func(yield func(archiveEntry, error) bool) {
		for _, e := range entries {
			if !yield(e, nil) {
				return
			}
		}
	}
}

func fileEntry(name string, mode os.FileMode, content string) archiveEntry {
	return archiveEntry{
		Name: name,
		Mode: mode,
		Size: int64(len(content)),
		Open: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(content)), nil },
	}
}

func extract(t *testing.T, entries []archiveEntry, opts ...ExtractOption) (string, error) {
	t.Helper()

	targetDir := t.TempDir()
	return targetDir, extractArchive(staticEntries(entries), targetDir, newExtractConfig(opts))
}

func TestExtractArchive_Modes(t *testing.T) {
	t.Run("StripsSetuidSetgidAndSticky", func(t *testing.T) {
		targetDir, err := extract(t, []archiveEntry{
			fileEntry("evil", 0o755|os.ModeSetuid|os.ModeSetgid|os.ModeSticky, "x"),
		})
		require.NoError(t, err)

		info, err := os.Stat(filepath.Join(targetDir, "evil"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0), info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky))
	})

	t.Run("FallsBackWhenArchiveRecordsNoMode", func(t *testing.T) {
		targetDir, err := extract(t, []archiveEntry{fileEntry("plain", 0, "x")})
		require.NoError(t, err)

		info, err := os.Stat(filepath.Join(targetDir, "plain"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	})

	t.Run("WithFileModeOverridesArchive", func(t *testing.T) {
		targetDir, err := extract(t, []archiveEntry{fileEntry("bin", 0o600, "x")}, WithFileMode(0o700))
		require.NoError(t, err)

		info, err := os.Stat(filepath.Join(targetDir, "bin"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	})

	t.Run("ReadOnlyDirectoryStillReceivesChildren", func(t *testing.T) {
		// A 0o555 directory must be created writable and only tightened once its children exist, otherwise the
		// very next entry fails with EACCES.
		targetDir, err := extract(t, []archiveEntry{
			{Name: "ro", Mode: os.ModeDir | 0o555},
			fileEntry("ro/child.txt", 0o600, "inside"),
		})
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(targetDir, "ro", "child.txt"))
		require.NoError(t, err)
		assert.Equal(t, "inside", string(content))

		info, err := os.Stat(filepath.Join(targetDir, "ro"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o555), info.Mode().Perm(), "directory mode must be restored afterwards")

		// The restored mode is real, which means t.TempDir's own RemoveAll cannot unlink the child.
		t.Cleanup(func() { _ = os.Chmod(filepath.Join(targetDir, "ro"), 0o755) })
	})
}

func TestExtractArchive_EntrySelection(t *testing.T) {
	t.Run("SkipsEntriesDenotingTheRoot", func(t *testing.T) {
		// tar archives routinely contain an entry for the archive root itself. It must be a no-op rather
		// than an error. A leading "/" stays rejected, as it was before this refactor.
		for _, name := range []string{"", ".", "./"} {
			_, err := extract(t, []archiveEntry{{Name: name, Mode: os.ModeDir | 0o755}})
			assert.NoError(t, err, "entry %q should be skipped, not rejected", name)
		}
	})

	t.Run("SkipsNonRegularTypes", func(t *testing.T) {
		targetDir, err := extract(t, []archiveEntry{
			{Name: "fifo", Mode: os.ModeNamedPipe | 0o644},
			{Name: "dev", Mode: os.ModeDevice | 0o644},
			fileEntry("real.txt", 0o600, "kept"),
		})
		require.NoError(t, err)

		assert.NoFileExists(t, filepath.Join(targetDir, "fifo"))
		assert.NoFileExists(t, filepath.Join(targetDir, "dev"))
		assert.FileExists(t, filepath.Join(targetDir, "real.txt"))
	})

	t.Run("RejectsIllegalPaths", func(t *testing.T) {
		for _, name := range []string{"/", "/etc/passwd", "../escape", "a/../../escape", `C:\windows\x`} {
			_, err := extract(t, []archiveEntry{fileEntry(name, 0o644, "x")})
			require.Error(t, err, "entry %q must be rejected", name)
			assert.ErrorIs(t, err, ErrIllegalPath)
			assert.Contains(t, err.Error(), "illegal file path")
		}
	})
}

func TestExtractArchive_Limits(t *testing.T) {
	t.Run("MaxEntries", func(t *testing.T) {
		_, err := extract(t, []archiveEntry{
			fileEntry("a", 0o644, "a"),
			fileEntry("b", 0o644, "b"),
			fileEntry("c", 0o644, "c"),
		}, WithMaxEntries(2))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrLimitExceeded)
	})

	t.Run("MaxFileBytes", func(t *testing.T) {
		_, err := extract(t, []archiveEntry{fileEntry("big", 0o644, "0123456789")}, WithMaxFileBytes(4))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrLimitExceeded)
	})

	t.Run("MaxTotalBytes", func(t *testing.T) {
		_, err := extract(t, []archiveEntry{
			fileEntry("a", 0o644, "12345"),
			fileEntry("b", 0o644, "12345"),
			fileEntry("c", 0o644, "12345"),
		}, WithMaxTotalBytes(12))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrLimitExceeded)
	})

	t.Run("EntryLyingAboutItsSize", func(t *testing.T) {
		// A declared size is untrusted metadata: an entry that under-reports must still be caught while it is
		// being copied, not merely truncated in silence.
		body := strings.Repeat("A", 1024)
		_, err := extract(t, []archiveEntry{{
			Name: "liar",
			Mode: 0o644,
			Size: 1, // claims one byte, delivers 1024
			Open: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(body)), nil },
		}})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrLimitExceeded)
	})

	t.Run("EntryOverReportingItsSizeIsAccepted", func(t *testing.T) {
		// Over-reporting is not an attack, just sloppy metadata, and must not fail a legitimate archive.
		targetDir, err := extract(t, []archiveEntry{{
			Name: "sloppy",
			Mode: 0o644,
			Size: 4096,
			Open: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("short")), nil },
		}})
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(targetDir, "sloppy"))
		require.NoError(t, err)
		assert.Equal(t, "short", string(content))
	})

	t.Run("UnlimitedByDefault", func(t *testing.T) {
		targetDir, err := extract(t, []archiveEntry{fileEntry("big", 0o644, strings.Repeat("x", 1<<20))})
		require.NoError(t, err)

		info, err := os.Stat(filepath.Join(targetDir, "big"))
		require.NoError(t, err)
		assert.Equal(t, int64(1<<20), info.Size())
	})
}

func TestExtractArchive_PropagatesIteratorError(t *testing.T) {
	boom := errors.New("boom")
	entries := func(yield func(archiveEntry, error) bool) {
		yield(archiveEntry{}, boom)
	}

	err := extractArchive(entries, t.TempDir(), newExtractConfig(nil))
	assert.ErrorIs(t, err, boom)
}

// endregion
