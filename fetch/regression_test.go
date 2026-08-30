package fetch

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeebo/blake3"
)

func TestGetText_UnparsableURLWithRetriesDoesNotPanic(t *testing.T) {
	// resty hands a retry condition a nil *Response when the failure happened before a request could be sent. The
	// old condition read r.Request.Attempt on the right-hand side of an && - which is still evaluated when the
	// left-hand side short-circuits - and panicked. Every existing test missed it by using retries == 0, which
	// takes resty's non-Backoff fast path.
	f := New(nil, 3, false)

	assert.NotPanics(t, func() {
		_, err := f.GetText(context.Background(), "://not-a-url")
		assert.Error(t, err)
	})
}

func TestNew_DoesNotMutateTheCallersHeaders(t *testing.T) {
	headers := map[string]string{"X-Custom": "1"}
	New(headers, 0, false)

	assert.Equal(t, map[string]string{"X-Custom": "1"}, headers,
		"New must not default User-Agent and Content-Type into the caller's own map")
}

func TestDownloadFile_ResumeWhenServerIgnoresRange(t *testing.T) {
	// A resumed download pre-hashes the bytes already on disk. When the server answers a ranged request with 200
	// the file is truncated and restarted - and the digest has to be rewound too, or the reported hash covers the
	// stale prefix as well as the full body.
	body := strings.Repeat("payload-", 4096)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Deliberately ignore the Range header and answer with the whole thing.
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "resumed.bin")

	// Leave a partial file behind so the download issues a Range request.
	require.NoError(t, os.WriteFile(filePath, []byte("STALE-PREFIX-THAT-MUST-NOT-BE-HASHED"), 0o644))

	f := New(nil, 0, false)
	req, err := f.NewRequest(server.URL, filePath, nil)
	require.NoError(t, err)

	resp := f.DownloadFile(req)
	require.NoError(t, resp.Error())

	onDisk, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, body, string(onDisk), "the stale prefix must have been truncated away")

	sum := blake3.Sum256([]byte(body))
	want := hex.EncodeToString(sum[:])
	assert.Equal(t, want, resp.Hash, "the hash must cover the downloaded body only")
	assert.EqualValues(t, len(body), resp.BytesDownloaded())
}

func TestDownloadFiles_CancelDoesNotLeakWorkers(t *testing.T) {
	// A worker parked on the unbuffered result send never noticed cancellation, so it leaked along with its
	// semaphore slot and kept the result channel from ever being closed.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()

	dir := t.TempDir()
	f := New(nil, 0, false)

	requests := make([]*Request, 0, 8)
	for i := range 8 {
		req, err := f.NewRequest(server.URL, filepath.Join(dir, "f"+strconv.Itoa(i)), nil)
		require.NoError(t, err)
		requests = append(requests, req)
	}

	results, cancelAll := f.DownloadFiles(requests, 2)

	// Take one response and then walk away, exactly as a caller that hit an error would.
	<-results
	cancelAll()

	// The channel must still close, which it cannot do while a worker is stuck on a send nobody will receive.
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		for range results {
		}
	}()

	select {
	case <-drained:
	case <-time.After(10 * time.Second):
		t.Fatal("DownloadFiles leaked a worker: the result channel was never closed after cancelling")
	}
}

func TestGetFileCookies_LongLinesAndHttpOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.txt")

	// A session token comfortably past bufio's default 64 KiB line limit, which used to end the scan and return a
	// truncated jar with a nil error.
	huge := strings.Repeat("j", 100*1024)
	content := strings.Join([]string{
		"# Netscape HTTP Cookie File",
		"#HttpOnly_example.com\tTRUE\t/\tTRUE\t0\tsession\tsecret",
		"example.com\tTRUE\t/\tFALSE\t0\thuge\t" + huge,
		"example.com\tTRUE\t/\tFALSE\t0\tafter\ttail",
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	cookies, err := GetFileCookies(path)
	require.NoError(t, err)

	byName := make(map[string]string, len(cookies))
	for _, c := range cookies {
		byName[c.Name] = c.Value
	}

	assert.Equal(t, "secret", byName["session"], "an #HttpOnly_ record is a cookie, not a comment")
	assert.Equal(t, huge, byName["huge"])
	assert.Equal(t, "tail", byName["after"], "a long line must not silently truncate the rest of the jar")
}

func TestGetFileCookies_ReportsOverlongLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.txt")

	// Past even the raised ceiling: this has to be an error rather than a silently short list.
	oversized := strings.Repeat("x", maxCookieLine+1)
	require.NoError(t, os.WriteFile(path, []byte("example.com\tTRUE\t/\tFALSE\t0\tbig\t"+oversized), 0o644))

	_, err := GetFileCookies(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read cookie file")
}
