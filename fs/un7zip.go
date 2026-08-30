package fs

import (
	"io/fs"
	"iter"
	"math"

	"github.com/bodgit/sevenzip"
)

// Un7zip extracts all files and directories from a 7z archive to a target directory. It creates the target directory
// if it doesn't exist and preserves the directory structure from the archive, including symbolic links.
//
// Extraction is confined to targetDirectory by an os.Root, so neither a hostile entry name nor a chain of symbolic
// links planted by earlier entries in the archive can write outside it. Entry names that are absolute, that escape the
// root, or that name a reserved Windows device are rejected before anything is created.
//
// # Parameters:
//   - sevenZipPath: Path to the 7z file to extract
//   - targetDirectory: Destination directory where files will be extracted
//   - opts: Optional limits and overrides; see ExtractOption
//
// # Returns an error if:
//   - The 7z file cannot be opened or read
//   - The target directory cannot be created
//   - Any archive entry contains an illegal path (wrapping ErrIllegalPath)
//   - Any symbolic link points outside the target directory (wrapping ErrIllegalSymlink)
//   - The archive exceeds a configured limit (wrapping ErrLimitExceeded)
//   - File extraction fails due to I/O errors or permission issues
//
// Extracted files keep the permission bits recorded in the archive, subject to the process umask; setuid, setgid and
// sticky bits are always stripped. Use WithFileMode to override the mode instead.
func Un7zip(sevenZipPath, targetDirectory string, opts ...ExtractOption) error {
	r, err := sevenzip.OpenReader(sevenZipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	return extractArchive(sevenZipEntries(r.File), targetDirectory, newExtractConfig(opts))
}

// region - Private functions

// sevenZipEntries adapts a 7z archive to the shared extraction core.
func sevenZipEntries(files []*sevenzip.File) iter.Seq2[archiveEntry, error] {
	return func(yield func(archiveEntry, error) bool) {
		for _, f := range files {
			size := int64(-1)
			if f.UncompressedSize <= math.MaxInt64 {
				size = int64(f.UncompressedSize)
			}

			e := archiveEntry{
				Name: f.Name,
				Mode: f.FileInfo().Mode(),
				Size: size,
				Open: f.Open,
			}

			if e.Mode&fs.ModeSymlink != 0 {
				// A 7z archive stores a symlink's target as the entry's content.
				target, err := readLinkTarget(f.Open)
				if err != nil {
					yield(archiveEntry{}, err)
					return
				}

				e.LinkTarget = target
			}

			if !yield(e, nil) {
				return
			}
		}
	}
}

// endregion
