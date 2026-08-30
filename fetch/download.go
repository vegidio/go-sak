package fetch

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zeebo/blake3"
)

// maxBackoff caps the wait between download attempts; the raw fibonacci sequence reaches 89s on the
// 10th retry, which stalls a queue for minutes on a URL that will never succeed.
const maxBackoff = 30 * time.Second

// NewRequest creates a new download request with the specified URL and file path.
//
// Parameters:
//   - url: The URL to download the file from.
//   - filePath: The path where the downloaded file will be saved.
//   - headers: Optional headers to set on the request.
//
// Returns:
//   - A Request object containing the URL and file path.
//   - An error if the request creation fails.
func (f *Fetch) NewRequest(url string, filePath string, headers map[string]string) (*Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err = validateDownloadUrl(req.URL); err != nil {
		return nil, err
	}

	for key, value := range f.headers {
		req.Header.Set(key, value)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Default the User-Agent rather than forcing it, matching New. Overwriting it here meant neither the client's
	// headers nor this call's headers could change the agent used for downloads.
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", userAgent)
	}

	// Create a new Request object
	return &Request{
		Url:      url,
		FilePath: filePath,
		httpReq:  req,
	}, nil
}

// DownloadFile downloads a single file based on the provided request.
//
// Parameters:
//   - request: a Request object containing the details of the file to download.
//
// Returns:
//   - A Response object that contains the status and details of the download process.
func (f *Fetch) DownloadFile(request *Request) *Response {
	ctx, cancel := context.WithCancel(context.Background())

	// Tie the http.Request to the context.
	// It means that if the context is canceled, the request will be canceled too.
	request.httpReq = request.httpReq.WithContext(ctx)

	response := &Response{
		Request: request,
		Done:    make(chan struct{}),
		cancel:  cancel,
	}

	go func() {
		defer close(response.Done)

		// How many bytes are already on the disk?
		var offset int64
		if info, err := os.Stat(request.FilePath); err == nil {
			offset = info.Size()
		}

		// Open (or create) a file for appending and reading (needed to hash existing bytes)
		file, err := os.OpenFile(request.FilePath, os.O_CREATE|os.O_RDWR, 0o644)
		if err != nil {
			response.err = fmt.Errorf("could not open file: %w", err)
			return
		}

		defer file.Close()
		hasher := blake3.New()

		if offset > 0 {
			if _, hashErr := io.CopyN(hasher, file, offset); hashErr != nil {
				response.err = fmt.Errorf("could not hash existing data: %w", hashErr)
				return
			}

			// Seek to the end of existing data
			if _, fErr := file.Seek(offset, io.SeekStart); fErr != nil {
				response.err = fmt.Errorf("could not seek: %w", fErr)
				return
			}
		}

		// Set up the progress callback
		pw := &progressWriter{
			file:     file,
			hasher:   hasher,
			callback: response.addDownloaded,
		}

		// Perform the download (with resume & retries)
		f.downloadWithRetries(ctx, response, offset, file, pw)

		sum := hasher.Sum(nil)
		response.Hash = hex.EncodeToString(sum)

		// A failed download that wrote nothing leaves an empty file behind, because the file is created
		// before the first request is even sent; clean it up instead of littering the output directory.
		if response.err != nil && response.BytesDownloaded() == 0 && offset == 0 {
			_ = os.Remove(request.FilePath)
		}
	}()

	return response
}

// DownloadFiles downloads multiple files concurrently.
//
// Parameters:
//   - requests: a slice of *Request objects representing the files to download.
//   - parallel: the maximum number of concurrent downloads.
//
// Returns:
//   - A channel of *Response objects, where each response corresponds to a file download.
//   - A function that can be called to cancel all downloads.
func (f *Fetch) DownloadFiles(requests []*Request, parallel int) (<-chan *Response, func()) {
	result := make(chan *Response)
	done := make(chan struct{})

	var (
		wg      sync.WaitGroup
		sem     = make(chan struct{}, parallel)
		mu      sync.Mutex
		cancels []func()
	)

	// cancelAll cancels all ongoing downloads
	cancelAll := func() {
		select {
		case <-done:
			// already canceled
		default:
			close(done)
		}

		mu.Lock()
		defer mu.Unlock()
		for _, cancelFn := range cancels {
			cancelFn()
		}
	}

	go func() {
		defer close(result)

		for _, req := range requests {
			wg.Add(1)

			go func(r *Request) {
				defer wg.Done()

				// Either grab a slot or exit if canceled
				select {
				case sem <- struct{}{}:
					// slot acquired
				case <-done:
					return
				}

				defer func() { <-sem }()

				// Start the download
				resp := f.DownloadFile(r)

				// Capture the Cancel() function. A cancelAll racing with this send would otherwise miss
				// the download entirely, so check afterwards whether that already happened.
				mu.Lock()
				cancels = append(cancels, resp.cancel)
				mu.Unlock()

				select {
				case <-done:
					resp.cancel()
				default:
				}

				select {
				case result <- resp:
				case <-done:
					// The consumer has stopped reading and everything was cancelled. Handing this
					// response over would block forever, taking the sem slot and the close of result
					// with it.
					resp.cancel()
					_ = resp.Error()
					return
				}

				// Waiting for the download the complete before continuing
				_ = resp.Error()
			}(req)
		}

		wg.Wait()
	}()

	return result, cancelAll
}

// region - Private functions

// downloadWithRetries drives one download to completion, resuming where it can and retrying what is worth retrying.
//
// writer is the concrete *progressWriter rather than an io.Writer because a resumed download that the server answers
// without honouring Range has to rewind the digest as well as the file.
func (f *Fetch) downloadWithRetries(
	ctx context.Context,
	response *Response,
	offset int64,
	file *os.File,
	writer *progressWriter,
) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= f.retries; attempt++ {
		// Before each attempt, see if we've been canceled
		select {
		case <-ctx.Done():
			response.err = ctx.Err()
			return
		default:
		}

		if attempt > 0 {
			backoff := min(time.Duration(fibonacci(attempt+1))*time.Second, maxBackoff)

			log.WithFields(log.Fields{
				"attempt": attempt,
				"error":   err,
				"url":     response.Request.Url,
			}).Warn("failed to download file; retrying in ", backoff)

			// A plain Sleep here would keep a cancelled download alive for the whole backoff
			timer := time.NewTimer(backoff)

			select {
			case <-ctx.Done():
				timer.Stop()
				response.err = ctx.Err()
				return
			case <-timer.C:
			}
		}

		isRangeReq := offset > 0
		if isRangeReq {
			response.Request.httpReq.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		} else {
			response.Request.httpReq.Header.Del("Range")
		}

		// Send it
		resp, err = f.httpClient.Do(response.Request.httpReq)
		if err != nil {
			response.err = fmt.Errorf("request error: %w", err)

			if ctx.Err() != nil {
				response.err = ctx.Err()
				return
			}

			continue
		}

		// Fallback if server doesn’t support Range
		if isRangeReq && resp.StatusCode == http.StatusOK {
			// Truncate file and reset offset
			if tErr := file.Truncate(0); tErr != nil {
				response.StatusCode = resp.StatusCode
				response.setSize(0)
				response.err = fmt.Errorf("truncate failed: %w", tErr)
			}

			// The bytes that were already on disk have been discarded, so they must be discarded from the
			// digest too - otherwise the final hash covers the stale prefix as well as the full body.
			writer.hasher.Reset()
			response.resetProgress()

			offset = 0

			if _, sErr := file.Seek(0, io.SeekStart); sErr != nil {
				response.StatusCode = resp.StatusCode
				response.setSize(0)
				response.err = fmt.Errorf("seek after truncate failed: %w", sErr)
			}

			attempt-- // retry same attempt count with fresh download
			resp.Body.Close()
			continue
		}

		response.downloaded.Store(offset)
		response.Downloaded = offset

		// Handle '416 Range Not Satisfiable' (already complete)
		if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
			response.StatusCode = resp.StatusCode
			response.setSize(offset)

			// offset can be 0 here when the server answers 416 to a request that carried no Range header,
			// and 0/0 is NaN rather than a progress figure.
			if offset > 0 {
				response.Progress = 1
			}

			resp.Body.Close()
			break
		}

		// Anything that is not 2xx is a failure. Retrying only makes sense when the server told us the
		// problem is temporary; a 403 or a 404 will answer the same way ten times in a row, and retrying
		// it just burns minutes of backoff before failing anyway.
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			response.StatusCode = resp.StatusCode
			response.err = fmt.Errorf("unexpected status: %d", resp.StatusCode)
			resp.Body.Close()

			if !isRetryableStatus(resp.StatusCode) {
				return
			}

			continue
		}

		// Compute total size from Content-Range or Content-Length
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			// e.g. "bytes 500-999/1234"
			var start, end, total int64
			if _, scanErr := fmt.Sscanf(cr, "bytes %d-%d/%d", &start, &end, &total); scanErr == nil {
				response.setSize(total)
			} else {
				response.setSize(sizeFrom(offset, resp.ContentLength))
			}
		} else {
			response.setSize(sizeFrom(offset, resp.ContentLength))
		}

		// Track where this attempt started
		startOffset := offset

		// Actually copy data
		_, err = io.Copy(writer, resp.Body)
		if err != nil {
			if ctx.Err() != nil {
				response.err = ctx.Err()
				resp.Body.Close()
				break
			}

			// figure out how many bytes actually made it to the disk
			newOffset, seekErr := file.Seek(0, io.SeekEnd)
			if seekErr != nil {
				response.err = fmt.Errorf("seek after partial download failed: %w", seekErr)
				resp.Body.Close()
				break
			}

			// bump offset to resume after what we have
			offset = newOffset
			response.err = fmt.Errorf("download interrupted (wrote %d bytes), will resume: %w",
				newOffset-startOffset, err)
			resp.Body.Close()
			continue
		}

		// Success
		if response.TotalSize() <= 0 {
			response.setSize(response.BytesDownloaded())
			response.Progress = 1
		}

		response.StatusCode = resp.StatusCode
		response.err = nil
		resp.Body.Close()
		break
	}
}

// validateDownloadUrl rejects URLs that could never be downloaded, so the caller finds out immediately
// instead of after a full round of retries. Relative URLs are the common case: some sites hand out
// links such as "/r/subreddit", which http.Client rejects with "no Host in request URL" every time.
func validateDownloadUrl(u *neturl.URL) error {
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid download URL %q: not an absolute http(s) URL", u)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid download URL %q: unsupported scheme %q", u, u.Scheme)
	}

	return nil
}

// isRetryableStatus reports whether it's worth sending the request again. Server-side failures and the
// explicit "slow down / try later" statuses are; every other client error is permanent.
func isRetryableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooEarly, http.StatusTooManyRequests:
		return true
	}

	return statusCode >= 500
}

func fibonacci(n int) int {
	if n <= 1 {
		return n
	}

	a, b := 0, 1

	for i := 1; i < n; i++ {
		a, b = b, a+b
	}

	return b
}

// endregion

// sizeFrom combines the bytes already on disk with the length the server reported. A chunked response has no length
// and reports -1, which must be surfaced as "unknown" rather than quietly turned into offset-1.
func sizeFrom(offset, contentLength int64) int64 {
	if contentLength < 0 {
		return -1
	}

	return offset + contentLength
}
