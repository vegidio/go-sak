package fs

import (
	"archive/tar"
	"errors"
	"io"
	"iter"
	"os"

	"github.com/ulikunitz/xz"
)

// UntarXz extracts all files and directories from a TAR.XZ archive to a target directory. It creates the target
// directory if it doesn't exist and preserves the directory structure from the archive, including symbolic links.
//
// Extraction is confined to targetDirectory by an os.Root, so neither a hostile entry name nor a chain of symbolic
// links planted by earlier entries in the archive can write outside it. Entry names that are absolute, that escape the
// root, or that name a reserved Windows device are rejected before anything is created.
//
// # Parameters:
//   - tarXzPath: Path to the TAR.XZ file to extract
//   - targetDirectory: Destination directory where files will be extracted
//   - opts: Optional limits and overrides; see ExtractOption
//
// # Returns an error if:
//   - The TAR.XZ file cannot be opened or read
//   - The target directory cannot be created
//   - Any archive entry contains an illegal path (wrapping ErrIllegalPath)
//   - Any symbolic link points outside the target directory (wrapping ErrIllegalSymlink)
//   - The archive exceeds a configured limit (wrapping ErrLimitExceeded)
//   - File extraction fails due to I/O errors or permission issues
//
// Extracted files keep the permission bits recorded in the archive, subject to the process umask; setuid, setgid and
// sticky bits are always stripped. Use WithFileMode to override the mode instead.
func UntarXz(tarXzPath, targetDirectory string, opts ...ExtractOption) error {
	f, err := os.Open(tarXzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	xzReader, err := xz.NewReader(f)
	if err != nil {
		return err
	}

	return extractArchive(tarEntries(tar.NewReader(xzReader)), targetDirectory, newExtractConfig(opts))
}

// region - Private functions

// tarEntries adapts a streaming tar archive to the shared extraction core. Each entry's body is only readable until
// the iterator advances, which is exactly the archiveEntry contract.
func tarEntries(tr *tar.Reader) iter.Seq2[archiveEntry, error] {
	return func(yield func(archiveEntry, error) bool) {
		for {
			h, err := tr.Next()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				yield(archiveEntry{}, err)
				return
			}

			// Hard links, devices, FIFOs and the GNU/PAX metadata entries are skipped. This has to be
			// decided on Typeflag: FileInfo().Mode() reports a hard link as a plain regular file, which
			// would create a bogus empty file in its place.
			switch h.Typeflag {
			case tar.TypeReg, tar.TypeDir, tar.TypeSymlink:
			default:
				continue
			}

			e := archiveEntry{
				Name: h.Name,
				// FileInfo().Mode() decodes the raw POSIX mode, where setuid is 0o4000 rather than
				// where Go's FileMode keeps it.
				Mode:       h.FileInfo().Mode(),
				Size:       h.Size,
				LinkTarget: h.Linkname,
				Open:       func() (io.ReadCloser, error) { return io.NopCloser(tr), nil },
			}

			if !yield(e, nil) {
				return
			}
		}
	}
}

// endregion
