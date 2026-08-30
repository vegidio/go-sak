package fs

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchiveEntryPath(t *testing.T) {
	tests := []struct {
		name  string
		entry string
		want  string
		err   bool
	}{
		{name: "SimpleName", entry: "file.txt", want: "file.txt"},
		{name: "NestedName", entry: "dir/file.txt", want: "dir/file.txt"},
		{name: "LeadingDotSlash", entry: "./file.txt", want: "file.txt"},
		{name: "InteriorDot", entry: "a/./b", want: "a/b"},
		{name: "BenignParent", entry: "a/../b", want: "b"},
		{name: "TrailingSlashes", entry: "dir///", want: "dir"},
		{name: "BackslashSeparators", entry: `dir\file.txt`, want: "dir/file.txt"},

		// Entries denoting the extraction root itself are a no-op, not an error.
		{name: "Empty", entry: "", want: ""},
		{name: "Dot", entry: ".", want: ""},
		{name: "DotSlash", entry: "./", want: ""},

		// Absolute and escaping names.
		{name: "RootSlash", entry: "/", err: true},
		{name: "AbsoluteUnix", entry: "/etc/passwd", err: true},
		{name: "AbsoluteBackslash", entry: `\windows\system32`, err: true},
		{name: "Traversal", entry: "../../../etc/passwd", err: true},
		{name: "TraversalWindows", entry: `..\..\windows\system32`, err: true},
		{name: "NetEscape", entry: "dir/../../../escape.txt", err: true},
		{name: "BareParent", entry: "..", err: true},

		// Drive letters must be rejected on every platform: filepath.IsLocal only recognises them on Windows,
		// so without an explicit check "C:/x" would become an ordinary relative filename on Linux and macOS.
		{name: "DriveLetterUpper", entry: "C:/windows/x", err: true},
		{name: "DriveLetterLower", entry: `c:\windows\x`, err: true},
		{name: "DriveLetterBare", entry: "D:", err: true},

		// A colon that is not a drive specifier is a legal filename character on Unix.
		{name: "ColonNotDrive", entry: "ab:cd", want: "ab:cd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := archiveEntryPath(tt.entry)

			if tt.err {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrIllegalPath)
				assert.Contains(t, err.Error(), "illegal file path")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestArchiveEntryPath_WindowsDeviceNames(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("reserved device names are only meaningful on Windows")
	}

	// Extracting an entry called NUL or COM1 on Windows opens the device instead of creating a file.
	for _, name := range []string{"NUL", "CON", "COM1", "LPT1", "AUX", "nul.txt"} {
		_, err := archiveEntryPath(name)
		assert.ErrorIs(t, err, ErrIllegalPath, "entry %q must be rejected", name)
	}
}

func TestCheckSymlinkTarget(t *testing.T) {
	tests := []struct {
		name   string
		link   string
		target string
		err    bool
	}{
		{name: "SiblingFile", link: "dir/link", target: "file.txt"},
		{name: "ParentThatStaysInside", link: "a/b/link", target: "../file.txt"},
		{name: "ResolvesToRoot", link: "link", target: "."},

		{name: "Empty", link: "link", target: "", err: true},
		{name: "AbsoluteUnix", link: "link", target: "/etc/passwd", err: true},
		{name: "AbsoluteBackslash", link: "link", target: `\windows\x`, err: true},
		{name: "DriveLetter", link: "link", target: "C:/windows/x", err: true},
		{name: "EscapesRoot", link: "link", target: "../outside", err: true},
		{name: "DeepEscape", link: "a/link", target: "../../../outside", err: true},

		// These two are the entries of the chained-symlink attack. Both are lexically innocent, which is
		// precisely why the extractor also probes each link through os.Root after creating it. If either of
		// these ever starts failing here, TestUntarXz_ChainedSymlinkEscape is no longer testing what it
		// claims to test.
		{name: "AttackChainFirstLink", link: "p", target: "."},
		{name: "AttackChainSecondLink", link: "p/q", target: ".."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSymlinkTarget(tt.link, tt.target)

			if tt.err {
				assert.ErrorIs(t, err, ErrIllegalSymlink)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestHasDriveLetter(t *testing.T) {
	for _, s := range []string{"C:", "c:", "Z:/x", `a:\x`} {
		assert.True(t, hasDriveLetter(s), "%q should be a drive letter", s)
	}

	for _, s := range []string{"", "C", "/C:", "1:/x", "ab:cd", "::"} {
		assert.False(t, hasDriveLetter(s), "%q should not be a drive letter", s)
	}
}

// FuzzArchiveEntryPath asserts the invariant the extractor relies on: whatever comes back is a local path with no
// parent-directory element, or else an error.
func FuzzArchiveEntryPath(f *testing.F) {
	for _, seed := range []string{
		"file.txt", "dir/file.txt", "./a", "../a", "/a", `C:\a`, "..", ".", "", "a/../b", `a\b`, "NUL",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, entry string) {
		got, err := archiveEntryPath(entry)
		if err != nil {
			return
		}
		if got == "" {
			return
		}

		assert.True(t, filepath.IsLocal(filepath.FromSlash(got)), "archiveEntryPath(%q) = %q, not local", entry, got)

		for _, seg := range strings.Split(got, "/") {
			assert.NotEqual(t, "..", seg, "archiveEntryPath(%q) = %q contains a parent element", entry, got)
		}
	})
}
