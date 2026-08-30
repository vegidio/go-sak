package fs

import (
	"archive/zip"
	"io/fs"
	"iter"
	"math"
)

// Unzip extracts all files and directories from a ZIP archive to a target directory. It creates the target directory
// if it doesn't exist and preserves the directory structure from the archive, including symbolic links.
//
// Extraction is confined to targetDirectory by an os.Root, so neither a hostile entry name nor a chain of symbolic
// links planted by earlier entries in the archive can write outside it. Entry names that are absolute, that escape the
// root, or that name a reserved Windows device are rejected before anything is created.
//
// # Parameters:
//   - zipPath: Path to the ZIP file to extract
//   - targetDirectory: Destination directory where files will be extracted
//   - opts: Optional limits and overrides; see ExtractOption
//
// # Returns an error if:
//   - The ZIP file cannot be opened or read
//   - The target directory cannot be created
//   - Any archive entry contains an illegal path (wrapping ErrIllegalPath)
//   - Any symbolic link points outside the target directory (wrapping ErrIllegalSymlink)
//   - The archive exceeds a configured limit (wrapping ErrLimitExceeded)
//   - File extraction fails due to I/O errors or permission issues
//
// Extracted files keep the permission bits recorded in the archive, subject to the process umask; setuid, setgid and
// sticky bits are always stripped. Use WithFileMode to override the mode instead.
func Unzip(zipPath, targetDirectory string, opts ...ExtractOption) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	return extractArchive(zipEntries(r.File), targetDirectory, newExtractConfig(opts))
}

// region - Private functions

// zipEntries adapts a zip archive to the shared extraction core.
func zipEntries(files []*zip.File) iter.Seq2[archiveEntry, error] {
	return func(yield func(archiveEntry, error) bool) {
		for _, f := range files {
			size := int64(-1)
			if f.UncompressedSize64 <= math.MaxInt64 {
				size = int64(f.UncompressedSize64)
			}

			e := archiveEntry{
				Name: f.Name,
				Mode: f.FileInfo().Mode(),
				Size: size,
				Open: f.Open,
			}

			if e.Mode&fs.ModeSymlink != 0 {
				// A zip archive stores a symlink's target as the entry's content.
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
