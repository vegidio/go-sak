package fs

import "io/fs"

// ExtractOption configures Unzip, Un7zip and UntarXz. Options are applied in the order they are given, and a limit of
// zero means "no limit".
//
// # Example:
//
//	err := fs.Unzip("archive.zip", "/tmp/out",
//	    fs.WithMaxTotalBytes(1<<30),
//	    fs.WithoutSymlinks(),
//	)
type ExtractOption func(*extractConfig)

// extractConfig is the resolved set of options for a single extraction call.
type extractConfig struct {
	maxTotalBytes int64
	maxFileBytes  int64
	maxEntries    int
	allowSymlinks bool
	fileMode      fs.FileMode // 0 means "take the mode from the archive"
}

// newExtractConfig resolves opts on top of the defaults: no limits, symlinks allowed, modes taken from the archive.
func newExtractConfig(opts []ExtractOption) extractConfig {
	cfg := extractConfig{allowSymlinks: true}
	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// WithMaxTotalBytes limits the total number of uncompressed bytes a single extraction call may write. Extraction stops
// with an error wrapping ErrLimitExceeded once the budget is exhausted.
func WithMaxTotalBytes(n int64) ExtractOption {
	return func(c *extractConfig) { c.maxTotalBytes = n }
}

// WithMaxFileBytes limits the uncompressed size of any single entry in the archive.
func WithMaxFileBytes(n int64) ExtractOption {
	return func(c *extractConfig) { c.maxFileBytes = n }
}

// WithMaxEntries limits how many entries the archive may contain.
func WithMaxEntries(n int) ExtractOption {
	return func(c *extractConfig) { c.maxEntries = n }
}

// WithoutSymlinks rejects archives containing symbolic links outright, instead of validating them.
func WithoutSymlinks() ExtractOption {
	return func(c *extractConfig) { c.allowSymlinks = false }
}

// WithFileMode applies mode to every extracted regular file instead of the mode recorded in the archive. Only the
// permission bits of mode are used.
//
// WithFileMode(0o755) restores the behaviour of Unzip and Un7zip in releases before 26.5.0, which forced every
// extracted file to be executable.
func WithFileMode(mode fs.FileMode) ExtractOption {
	return func(c *extractConfig) { c.fileMode = mode.Perm() }
}
