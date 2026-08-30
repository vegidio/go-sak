package fs

import (
	"bufio"
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path"
	"slices"
	"strings"
	"syscall"
)

// archiveEntry is one entry of any supported archive format, normalized by the per-format adapters in unzip.go,
// un7zip.go and untar_xz.go.
//
// An entry is valid only until the iterator that produced it advances: the tar adapter wraps a streaming *tar.Reader,
// so Open must be called and the reader drained before the next iteration step.
type archiveEntry struct {
	// Name is the entry name exactly as stored in the archive. It may use either separator and is untrusted.
	Name string

	// Mode carries Go's type bits (fs.ModeDir, fs.ModeSymlink, ...) as well as the permission bits. Adapters build
	// it with FileInfo().Mode() so each format's own encoding is already decoded.
	Mode fs.FileMode

	// Size is the uncompressed size declared by the archive, or -1 when unknown. It comes from untrusted metadata
	// and is only ever used as an upper bound.
	Size int64

	// LinkTarget is set whenever Mode&fs.ModeSymlink != 0. Formats that store the target as entry content read it
	// in the adapter, so the core sees one uniform representation.
	LinkTarget string

	// Open returns the entry's content. It is nil for directories and symbolic links.
	Open func() (io.ReadCloser, error)
}

// extractArchive writes every entry produced by entries below targetDirectory.
//
// Containment is enforced by os.Root: every filesystem operation goes through a *os.Root anchored at targetDirectory,
// so neither a hostile entry name nor a chain of symbolic links planted by earlier entries can reach outside it.
func extractArchive(entries iter.Seq2[archiveEntry, error], targetDirectory string, cfg extractConfig) error {
	if err := os.MkdirAll(targetDirectory, 0o755); err != nil {
		return err
	}

	root, err := os.OpenRoot(targetDirectory)
	if err != nil {
		return err
	}
	defer root.Close()

	x := &extractor{root: root, cfg: cfg, budget: cfg.maxTotalBytes}

	for e, err := range entries {
		if err != nil {
			return err
		}

		if err = x.entry(e); err != nil {
			return err
		}
	}

	return x.restoreDirModes()
}

// extractor holds the per-call state of an extraction: the root that bounds it, the configured limits, and the
// directories whose modes still need tightening once every entry has been written.
type extractor struct {
	root   *os.Root
	cfg    extractConfig
	budget int64 // remaining total bytes; only meaningful when cfg.maxTotalBytes > 0
	count  int
	dirs   []pendingDir
}

type pendingDir struct {
	name string
	perm fs.FileMode
}

// entry dispatches a single archive entry to the handler for its type.
func (x *extractor) entry(e archiveEntry) error {
	x.count++
	if x.cfg.maxEntries > 0 && x.count > x.cfg.maxEntries {
		return fmt.Errorf("archive has more than %d entries: %w", x.cfg.maxEntries, ErrLimitExceeded)
	}

	name, err := archiveEntryPath(e.Name)
	if err != nil {
		return err
	}
	if name == "" {
		// The entry denotes the target directory itself ("", ".", "./", "/"), which tar archives commonly
		// contain. There is nothing to create.
		return nil
	}

	switch {
	case e.Mode.IsDir():
		return x.mkdir(name, e.Mode)
	case e.Mode&fs.ModeSymlink != 0:
		return x.symlink(name, e)
	case e.Mode.IsRegular():
		return x.regular(name, e)
	default:
		// Block devices, character devices, FIFOs and sockets are skipped, as UntarXz already did.
		return nil
	}
}

// mkdir creates a directory entry with the owner's rwx bits forced on, so that later entries can be written below it
// even when the archive records a read-only directory. The archive's own mode is reapplied by restoreDirModes.
func (x *extractor) mkdir(name string, mode fs.FileMode) error {
	perm := permOf(mode, 0o755)
	if err := x.root.MkdirAll(name, perm|0o700); err != nil {
		return err
	}

	if perm|0o700 != perm {
		x.dirs = append(x.dirs, pendingDir{name: name, perm: perm})
	}

	return nil
}

// restoreDirModes reapplies archive modes to directories that were created with extra owner bits. It works
// deepest-first, because tightening a parent before its children would make those children unreachable.
func (x *extractor) restoreDirModes() error {
	slices.SortStableFunc(x.dirs, func(a, b pendingDir) int {
		return cmp.Compare(strings.Count(b.name, "/"), strings.Count(a.name, "/"))
	})

	for _, d := range x.dirs {
		// Not every platform supports chmod; restoring modes there is best effort.
		if err := x.root.Chmod(d.name, d.perm); err != nil && !errors.Is(err, errors.ErrUnsupported) {
			return err
		}
	}

	return nil
}

// regular extracts a regular file entry.
func (x *extractor) regular(name string, e archiveEntry) (err error) {
	if err = x.reserve(e.Size); err != nil {
		return err
	}

	if dir := path.Dir(name); dir != "." {
		if err = x.root.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	if e.Open == nil {
		return fmt.Errorf("archive entry %s has no content reader", e.Name)
	}

	rc, err := e.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	perm := permOf(e.Mode, 0o644)
	if x.cfg.fileMode != 0 {
		perm = x.cfg.fileMode
	}

	f, err := x.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	// Close on every path, and surface a Close error when the copy itself succeeded.
	defer func() {
		if cErr := f.Close(); err == nil {
			err = cErr
		}
	}()

	bw := bufio.NewWriterSize(f, 1024*1024)
	if err = x.copyEntry(bw, rc, e.Size); err != nil {
		return err
	}

	err = bw.Flush()
	return err
}

// symlink creates a symbolic link entry, then verifies that the link it just created does not resolve outside the
// extraction root.
func (x *extractor) symlink(name string, e archiveEntry) error {
	if !x.cfg.allowSymlinks {
		return fmt.Errorf("illegal symlink target: %s -> %s: %w", e.Name, e.LinkTarget, ErrIllegalSymlink)
	}

	// Lexical pre-check. os.Root.Symlink explicitly does not validate its target, and os.Root cannot detect an
	// escape that runs through path components which do not exist yet, so this check carries real weight.
	if err := checkSymlinkTarget(name, e.LinkTarget); err != nil {
		return fmt.Errorf("illegal symlink target: %s -> %s: %w", e.Name, e.LinkTarget, err)
	}

	if dir := path.Dir(name); dir != "." {
		if err := x.root.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	if err := x.root.Symlink(e.LinkTarget, name); err != nil {
		return err
	}

	// Resolve the link we just created, through the root. This is what defeats a chained-symlink archive: the
	// lexical check above only sees entry names, while os.Root follows the links earlier entries really created on
	// disk. A dangling target is legal in an archive and a mutual symlink pair is merely useless, so both are
	// accepted; anything else - in particular os.Root's unexported "path escapes from parent" - is not.
	if _, err := x.root.Stat(name); err != nil &&
		!errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ELOOP) {
		_ = x.root.Remove(name)
		return fmt.Errorf("illegal symlink target: %s -> %s: %w", e.Name, e.LinkTarget, ErrIllegalSymlink)
	}

	return nil
}

// reserve rejects an entry before any file is created, using the size the archive declared. Declared sizes are
// untrusted, so copyEntry re-checks the bytes actually produced.
func (x *extractor) reserve(size int64) error {
	if size < 0 {
		return nil
	}

	if x.cfg.maxFileBytes > 0 && size > x.cfg.maxFileBytes {
		return fmt.Errorf("entry declares %d bytes, limit is %d: %w", size, x.cfg.maxFileBytes, ErrLimitExceeded)
	}

	if x.cfg.maxTotalBytes > 0 && size > x.budget {
		return fmt.Errorf("entry declares %d bytes, %d remaining: %w", size, x.budget, ErrLimitExceeded)
	}

	return nil
}

// copyEntry copies one entry's body, failing if it yields more bytes than it declared or than the configured limits
// allow. It reads one byte past the limit so that an over-long entry is reported rather than silently truncated.
func (x *extractor) copyEntry(dst io.Writer, src io.Reader, declared int64) error {
	limit := int64(-1)
	if declared >= 0 {
		limit = declared
	}
	if x.cfg.maxFileBytes > 0 && (limit < 0 || x.cfg.maxFileBytes < limit) {
		limit = x.cfg.maxFileBytes
	}
	if x.cfg.maxTotalBytes > 0 && (limit < 0 || x.budget < limit) {
		limit = x.budget
	}

	var n int64
	var err error

	if limit < 0 {
		n, err = io.Copy(dst, src)
	} else {
		n, err = io.Copy(dst, io.LimitReader(src, limit+1))
		if err == nil && n > limit {
			err = fmt.Errorf("entry produced more than %d bytes: %w", limit, ErrLimitExceeded)
		}
	}
	if err != nil {
		return err
	}

	if x.cfg.maxTotalBytes > 0 {
		x.budget -= n
	}

	return nil
}

// region - Private functions

// permOf returns the on-disk permission bits for an archive entry mode.
//
// Only the nine permission bits survive. Go's high type bits (fs.ModeDir, fs.ModeSymlink, ...) must go because
// os.Root.OpenFile and os.Root.MkdirAll reject any bit outside 0o777, and fs.FileMode.Perm also drops fs.ModeSetuid,
// fs.ModeSetgid and fs.ModeSticky - which live outside 0o777 - so a hostile archive cannot plant a setuid binary.
// Entries carrying no usable mode, such as zip entries written without Unix external attributes, fall back to def.
//
// The result is still subject to the process umask, matching the behaviour of tar(1) without -p.
func permOf(mode fs.FileMode, def fs.FileMode) fs.FileMode {
	if perm := mode.Perm(); perm != 0 {
		return perm
	}

	return def
}

// readLinkTarget reads a symbolic link's target from an entry's content, which is how the zip and 7z formats store it.
func readLinkTarget(open func() (io.ReadCloser, error)) (string, error) {
	rc, err := open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	// A symlink target can never exceed a path's length; the cap keeps a hostile archive from allocating freely.
	target, err := io.ReadAll(io.LimitReader(rc, 64*1024))
	if err != nil {
		return "", err
	}

	return string(target), nil
}

// endregion
