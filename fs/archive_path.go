package fs

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// Sentinel errors returned by Unzip, Un7zip and UntarXz. Test for them with errors.Is.
var (
	// ErrIllegalPath is returned when an archive entry's name is absolute, escapes the extraction root, or is
	// otherwise unusable as a relative path.
	ErrIllegalPath = errors.New("illegal archive entry path")

	// ErrIllegalSymlink is returned when a symbolic link in the archive points outside the extraction root, either
	// directly or through links created by earlier entries.
	ErrIllegalSymlink = errors.New("illegal symlink target")

	// ErrLimitExceeded is returned when an archive exceeds one of the limits configured with WithMaxTotalBytes,
	// WithMaxFileBytes or WithMaxEntries, or when an entry produces more bytes than it declared.
	ErrLimitExceeded = errors.New("archive exceeds extraction limits")
)

// archiveEntryPath validates an untrusted archive entry name and returns the equivalent slash-separated path relative
// to the extraction root. It returns an empty name for entries that denote the root itself ("", ".", "./", "/").
//
// The returned path is only ever used with *os.Root methods, which enforce containment themselves. This function is not
// the security boundary: it exists to reject obviously hostile names early with a useful message, and to normalize the
// separators that os.Root would otherwise treat as ordinary filename characters.
func archiveEntryPath(entryName string) (string, error) {
	// Archives written on Windows may use backslashes as separators.
	name := strings.ReplaceAll(entryName, `\`, "/")

	// Reject absolute names up front. filepath.IsLocal only recognises the drive-letter and UNC forms when
	// GOOS=windows, so on Linux and macOS "C:/x" would otherwise be accepted as an ordinary relative name.
	if strings.HasPrefix(name, "/") || hasDriveLetter(name) {
		return "", fmt.Errorf("illegal file path: %s: %w", entryName, ErrIllegalPath)
	}

	if name = path.Clean(name); name == "." {
		return "", nil
	}

	// filepath.IsLocal rejects "..", the empty string and, on Windows, reserved device names such as NUL, CON and
	// COM1 - which would otherwise open a device instead of creating a file.
	if !filepath.IsLocal(filepath.FromSlash(name)) {
		return "", fmt.Errorf("illegal file path: %s: %w", entryName, ErrIllegalPath)
	}

	return name, nil
}

// checkSymlinkTarget reports whether target, interpreted relative to the directory holding the link at the
// slash-separated archive path name, stays inside the extraction root. A target resolving to the root itself is
// allowed.
//
// This check is lexical, so it cannot see escapes that run through symbolic links earlier entries already created on
// disk. Those are caught by the extractor's post-creation probe. It is still needed, because os.Root cannot detect an
// escape through path components that do not exist yet.
func checkSymlinkTarget(name, target string) error {
	if target == "" {
		return ErrIllegalSymlink
	}

	if strings.HasPrefix(target, "/") || strings.HasPrefix(target, `\`) ||
		filepath.IsAbs(target) || hasDriveLetter(target) {
		return ErrIllegalSymlink
	}

	resolved := path.Join(path.Dir(name), path.Clean(strings.ReplaceAll(target, `\`, "/")))
	if resolved != "." && !filepath.IsLocal(filepath.FromSlash(resolved)) {
		return ErrIllegalSymlink
	}

	return nil
}

// hasDriveLetter reports whether name begins with a Windows drive specifier such as "C:". filepath.IsLocal only
// recognises these when GOOS=windows, so archives crafted on Windows must be rejected explicitly on every platform.
func hasDriveLetter(name string) bool {
	if len(name) < 2 || name[1] != ':' {
		return false
	}

	c := name[0]
	return ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z')
}
