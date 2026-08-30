package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/zeebo/blake3"
)

// Request

type Request struct {
	Url      string
	FilePath string

	httpReq *http.Request
}

// Response

type Response struct {
	Request *Request

	// StatusCode, Size, Downloaded and Progress are updated by the download goroutine while the transfer is in
	// flight. Read them directly only once IsComplete reports true or Error has returned; while a download is
	// running use BytesDownloaded, TotalSize and ProgressRatio, which read the same state atomically.
	StatusCode int
	Size       int64
	Downloaded int64
	Progress   float64

	Hash string
	Done chan struct{} `json:"-"`

	// Atomic mirrors of Size and Downloaded. The exported fields above are kept in step for compatibility, but
	// these are what Track and the accessors read, so progress can be observed from another goroutine without a
	// data race or a torn 64-bit read.
	size       atomic.Int64
	downloaded atomic.Int64

	cancel context.CancelFunc
	err    error
}

// setSize records the total size of the download.
func (r *Response) setSize(n int64) {
	r.size.Store(n)
	r.Size = n
}

// addDownloaded records n more bytes written to disk and refreshes the progress ratio.
func (r *Response) addDownloaded(n int64) {
	downloaded := r.downloaded.Add(n)
	r.Downloaded = downloaded

	if size := r.size.Load(); size > 0 {
		r.Progress = float64(downloaded) / float64(size)
	}
}

// resetProgress rewinds the counters after a resumed download had to start over from the beginning.
func (r *Response) resetProgress() {
	r.downloaded.Store(0)
	r.Downloaded = 0
	r.Progress = 0
}

// BytesDownloaded returns how many bytes have been written so far. It is safe to call while the download is running.
func (r *Response) BytesDownloaded() int64 {
	return r.downloaded.Load()
}

// TotalSize returns the total size of the download, or 0 if the server did not report one. It is safe to call while
// the download is running.
func (r *Response) TotalSize() int64 {
	return r.size.Load()
}

// ProgressRatio returns how much of the download is complete, from 0 to 1, or 0 when the total size is unknown. It is
// safe to call while the download is running.
func (r *Response) ProgressRatio() float64 {
	size := r.size.Load()
	if size <= 0 {
		return 0
	}

	return float64(r.downloaded.Load()) / float64(size)
}

// Error waits for the download to complete and returns any error that occurred during the process.
func (r *Response) Error() error {
	<-r.Done
	return r.err
}

// IsComplete checks if the download process is complete.
func (r *Response) IsComplete() bool {
	select {
	case <-r.Done:
		return true
	default:
		return false
	}
}

// Cancel stops the download process.
func (r *Response) Cancel() {
	r.cancel()
}

// Bytes read the file specified in the Request's FilePath and return its content as a byte slice.
// It returns an error if the file cannot be read.
func (r *Response) Bytes() ([]byte, error) {
	data, err := os.ReadFile(r.Request.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// Track monitors the progress of the download and invokes the provided callback function with the current progress
// details. The callback receives the number of bytes downloaded, the total size of the file, and the progress
// percentage.
//
// # Parameters:
//   - callback: A function that takes three arguments: completed bytes (int64), total bytes (int64),
//     and progress percentage (float64).
//
// The callback always fires one last time when the download ends, so the final call reports the terminal state. That
// matters for downloads that fail: they never move Downloaded or Progress, and a caller watching only for changes
// would never hear that they are over.
//
// # Returns:
//   - An error if one occurred during the download process.
func (r *Response) Track(callback func(completed, total int64, progress float64)) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	oldValue := int64(-1)

	for {
		// select picks at random among ready cases, so a tick that becomes ready at the same moment as Done
		// would otherwise fire the callback and then let the next iteration fire it again. Checking for
		// completion first makes the terminal call happen exactly once.
		select {
		case <-r.Done:
			callback(r.BytesDownloaded(), r.TotalSize(), r.ProgressRatio())
			return r.Error()
		default:
		}

		select {
		case <-ticker.C:
			// While the download runs there is nothing to report unless it moved
			if downloaded := r.BytesDownloaded(); downloaded != oldValue {
				oldValue = downloaded
				callback(downloaded, r.TotalSize(), r.ProgressRatio())
			}

		case <-r.Done:
			callback(r.BytesDownloaded(), r.TotalSize(), r.ProgressRatio())
			return r.Error()
		}
	}
}

// ProgressWriter

type progressWriter struct {
	file     io.Writer
	hasher   *blake3.Hasher
	callback func(downloaded int64)
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.file.Write(p)
	if err != nil {
		return n, err
	}

	if pw.hasher != nil {
		m, err := pw.hasher.Write(p)
		if err != nil {
			return m, err
		}
	}

	if pw.callback != nil {
		pw.callback(int64(n))
	}

	return n, nil
}

// Cookies

// Cookie represents a key-value pair for a typical HTTP cookie.
type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
