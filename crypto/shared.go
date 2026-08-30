package crypto

import (
	"encoding/hex"
	"io"
	"sync"
)

// copyBufferSize is the chunk size used when feeding a reader into a hasher. Neither sha256's digest nor xxh3's
// Hasher implements io.ReaderFrom, so io.Copy would otherwise allocate a fresh 32 KiB buffer per call and issue a
// read syscall every 32 KiB.
const copyBufferSize = 1024 * 1024

var copyBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, copyBufferSize)
		return &buf
	},
}

// hashReader feeds everything from reader into hash using a pooled buffer.
func hashReader(hash io.Writer, reader io.Reader) error {
	buf := copyBufferPool.Get().(*[]byte)
	defer copyBufferPool.Put(buf)

	_, err := io.CopyBuffer(hash, reader, *buf)
	return err
}

// hexEncodeArray renders a fixed-size digest as a lowercase hex string.
func hexEncodeArray(sum [16]byte) string {
	return hex.EncodeToString(sum[:])
}
