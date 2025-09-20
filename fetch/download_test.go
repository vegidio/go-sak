package fetch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeebo/blake3"
)

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		filePath    string
		headers     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name:     "valid request",
			url:      "https://example.com/file.txt",
			filePath: "/tmp/file.txt",
			headers:  map[string]string{"Authorization": "Bearer token"},
		},
		{
			name:        "invalid URL",
			url:         "://invalid-url",
			filePath:    "/tmp/file.txt",
			expectError: true,
			errorMsg:    "failed to create request",
		},
		{
			name:     "empty headers",
			url:      "https://example.com/file.txt",
			filePath: "/tmp/file.txt",
			headers:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New(tt.headers, 3)

			req, err := f.NewRequest(tt.url, tt.filePath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, req)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
				assert.Equal(t, tt.url, req.Url)
				assert.Equal(t, tt.filePath, req.FilePath)
				assert.NotNil(t, req.httpReq)
				assert.Equal(t, "GET", req.httpReq.Method)
				assert.Equal(t, userAgent, req.httpReq.Header.Get("User-Agent"))

				// Check custom headers are set
				if tt.headers != nil {
					for key, value := range tt.headers {
						if key != "User-Agent" { // User-Agent is overridden
							assert.Equal(t, value, req.httpReq.Header.Get(key))
						}
					}
				}
			}
		})
	}
}

func TestDownloadFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "download_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name           string
		serverContent  string
		serverStatus   int
		serverHeaders  map[string]string
		existingFile   string
		expectedError  bool
		expectedStatus int
	}{
		{
			name:           "successful download",
			serverContent:  "Hello, World!",
			serverStatus:   http.StatusOK,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "download with content-length",
			serverContent:  "Test content with length",
			serverStatus:   http.StatusOK,
			serverHeaders:  map[string]string{"Content-Length": "24"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "resume download",
			serverContent:  "Hello, World! This is a long file for testing resume functionality.",
			serverStatus:   http.StatusPartialContent,
			serverHeaders:  map[string]string{"Content-Range": "bytes 6-66/67"},
			existingFile:   "Hello,",
			expectedStatus: http.StatusPartialContent,
		},
		{
			name:           "already complete download",
			serverContent:  "",
			serverStatus:   http.StatusRequestedRangeNotSatisfiable,
			existingFile:   "Complete file",
			expectedStatus: http.StatusRequestedRangeNotSatisfiable,
		},
		{
			name:          "server error",
			serverContent: "",
			serverStatus:  http.StatusInternalServerError,
			expectedError: true,
		},
		{
			name:           "not found",
			serverContent:  "",
			serverStatus:   http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate expected hash for the test
			var expectedHash string
			if !tt.expectedError && tt.expectedStatus != http.StatusNotFound {
				hasher := blake3.New()
				if tt.existingFile != "" && tt.serverStatus == http.StatusPartialContent {
					// For resume test, hash the complete content
					completeContent := tt.existingFile + tt.serverContent[len(tt.existingFile):]
					hasher.Write([]byte(completeContent))
				} else if tt.existingFile != "" && tt.serverStatus == http.StatusRequestedRangeNotSatisfiable {
					// For already complete download, hash the existing content
					hasher.Write([]byte(tt.existingFile))
				} else if tt.serverStatus == http.StatusOK || tt.serverStatus == http.StatusPartialContent {
					hasher.Write([]byte(tt.serverContent))
				}
				expectedHash = fmt.Sprintf("%x", hasher.Sum(nil))
			}

			// Setup test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Set custom headers
				for key, value := range tt.serverHeaders {
					w.Header().Set(key, value)
				}

				w.WriteHeader(tt.serverStatus)

				if tt.serverStatus == http.StatusPartialContent && tt.existingFile != "" {
					// For resume test, return content after the existing part
					w.Write([]byte(tt.serverContent[len(tt.existingFile):]))
				} else if tt.serverStatus == http.StatusOK || tt.serverStatus == http.StatusPartialContent {
					w.Write([]byte(tt.serverContent))
				}
			}))
			defer server.Close()

			// Setup file path
			filePath := filepath.Join(tempDir, fmt.Sprintf("test_%d.txt", time.Now().UnixNano()))

			// Create an existing file if specified
			if tt.existingFile != "" {
				err := os.WriteFile(filePath, []byte(tt.existingFile), 0644)
				require.NoError(t, err)
			}

			f := New(nil, 1)
			req, err := f.NewRequest(server.URL, filePath)
			require.NoError(t, err)

			response := f.DownloadFile(req)
			require.NotNil(t, response)
			assert.Equal(t, req, response.Request)
			assert.NotNil(t, response.Done)

			// Wait for download to complete
			err = response.Error()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, response.StatusCode)
				assert.True(t, response.IsComplete())

				// Verify hash matches expected value
				if expectedHash != "" {
					assert.Equal(t, expectedHash, response.Hash, "Hash should match expected Blake3 hash")
				}

				if tt.serverStatus == http.StatusOK || tt.serverStatus == http.StatusPartialContent {
					// Verify file content
					content, readErr := os.ReadFile(filePath)
					assert.NoError(t, readErr)

					expectedContent := tt.serverContent
					if tt.existingFile != "" && tt.serverStatus == http.StatusPartialContent {
						expectedContent = tt.existingFile + tt.serverContent[len(tt.existingFile):]
					}
					assert.Equal(t, expectedContent, string(content))

					// Check progress is complete
					if response.Size > 0 {
						assert.Equal(t, 1.0, response.Progress)
						assert.Equal(t, response.Size, response.Downloaded)
					}
				}
			}
		})
	}
}

func TestDownloadFileWithCancel(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "download_cancel_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("This should not complete"))
	}))
	defer server.Close()

	filePath := filepath.Join(tempDir, "cancel_test.txt")
	f := New(nil, 0)
	req, err := f.NewRequest(server.URL, filePath)
	require.NoError(t, err)

	response := f.DownloadFile(req)

	// Cancel the download quickly
	go func() {
		time.Sleep(50 * time.Millisecond)
		response.Cancel()
	}()

	err = response.Error()
	assert.Error(t, err)
	assert.True(t, response.IsComplete())
}

func TestDownloadFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "download_files_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test servers
	servers := make([]*httptest.Server, 3)
	contents := []string{"Content 1", "Content 2", "Content 3"}
	expectedHashes := make([]string, len(contents))

	// Calculate expected hashes
	for i, content := range contents {
		hasher := blake3.New()
		hasher.Write([]byte(content))
		expectedHashes[i] = fmt.Sprintf("%x", hasher.Sum(nil))
	}

	for i := range servers {
		content := contents[i]
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond) // Small delay to test concurrency
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		}))
		defer servers[i].Close()
	}

	// Create requests
	requests := make([]*Request, len(servers))
	f := New(nil, 1)

	for i, server := range servers {
		filePath := filepath.Join(tempDir, fmt.Sprintf("file_%d.txt", i))
		req, reqErr := f.NewRequest(server.URL, filePath)
		require.NoError(t, reqErr)
		requests[i] = req
	}

	// Test concurrent downloads
	responsesChan, cancelAll := f.DownloadFiles(requests, 2)

	responses := make([]*Response, 0, len(requests))
	for response := range responsesChan {
		responses = append(responses, response)
		err := response.Error()
		assert.NoError(t, err)
	}

	assert.Len(t, responses, len(requests))

	// Verify all files were downloaded correctly
	for _, response := range responses {
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.True(t, response.IsComplete())

		content, readErr := os.ReadFile(response.Request.FilePath)
		assert.NoError(t, readErr)
		assert.Contains(t, contents, string(content))

		// Find which content this response corresponds to and verify hash
		contentStr := string(content)
		for i, expectedContent := range contents {
			if contentStr == expectedContent {
				assert.Equal(t, expectedHashes[i], response.Hash,
					"Hash should match expected Blake3 hash for content: %s", expectedContent)
				break
			}
		}
	}

	// Test cancellation functionality
	t.Run("cancel all downloads", func(t *testing.T) {
		responsesChan2, cancelAll2 := f.DownloadFiles(requests, 1)

		// Cancel immediately
		cancelAll2()

		// Channel should close
		responsesReceived := 0
		for range responsesChan2 {
			responsesReceived++
		}

		// We might receive some responses before cancellation takes effect
		assert.True(t, responsesReceived <= len(requests))
	})

	// Test the cancelAll function doesn't panic when called multiple times
	cancelAll()
	cancelAll() // Should not panic
}

func TestDownloadWithRetries(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "retry_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testContent := "Success after retries"
	hasher := blake3.New()
	hasher.Write([]byte(testContent))
	expectedHash := fmt.Sprintf("%x", hasher.Sum(nil))

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testContent))
		}
	}))
	defer server.Close()

	filePath := filepath.Join(tempDir, "retry_test.txt")
	f := New(nil, 3) // 3 retries
	req, err := f.NewRequest(server.URL, filePath)
	require.NoError(t, err)

	response := f.DownloadFile(req)
	err = response.Error()

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, 3, attempts) // Should have tried 3 times
	assert.Equal(t, expectedHash, response.Hash, "Hash should match expected Blake3 hash")

	content, readErr := os.ReadFile(filePath)
	assert.NoError(t, readErr)
	assert.Equal(t, testContent, string(content))
}

func TestDownloadWithRangeRequest(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "range_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fullContent := "0123456789abcdefghijklmnopqrstuvwxyz"
	hasher := blake3.New()
	hasher.Write([]byte(fullContent))
	expectedHash := fmt.Sprintf("%x", hasher.Sum(nil))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			// Parse range header: "bytes=10-"
			var start int
			if n, _ := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); n == 1 {
				if start < len(fullContent) {
					w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(fullContent)-1, len(fullContent)))
					w.WriteHeader(http.StatusPartialContent)
					w.Write([]byte(fullContent[start:]))
					return
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fullContent))
	}))
	defer server.Close()

	filePath := filepath.Join(tempDir, "range_test.txt")

	// Create a partial file
	partialContent := fullContent[:10]
	err = os.WriteFile(filePath, []byte(partialContent), 0644)
	require.NoError(t, err)

	f := New(nil, 1)
	req, err := f.NewRequest(server.URL, filePath)
	require.NoError(t, err)

	response := f.DownloadFile(req)
	err = response.Error()
	assert.NoError(t, err)
	assert.Equal(t, expectedHash, response.Hash, "Hash should match expected Blake3 hash of complete file")

	// Verify the complete file
	content, readErr := os.ReadFile(filePath)
	assert.NoError(t, readErr)
	assert.Equal(t, fullContent, string(content))
}

func TestProgressWriter(t *testing.T) {
	var downloadedBytes int64
	var callCount int

	callback := func(downloaded int64) {
		downloadedBytes += downloaded
		callCount++
	}

	buffer := &strings.Builder{}
	pw := &progressWriter{
		file:     buffer,
		callback: callback,
	}

	// Write some data
	testData := []byte("Hello, World!")
	n, err := pw.Write(testData)

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, "Hello, World!", buffer.String())
	assert.Equal(t, int64(len(testData)), downloadedBytes)
	assert.Equal(t, 1, callCount)

	// Write more data
	moreData := []byte(" More data")
	n, err = pw.Write(moreData)

	assert.NoError(t, err)
	assert.Equal(t, len(moreData), n)
	assert.Equal(t, "Hello, World! More data", buffer.String())
	assert.Equal(t, int64(len(testData)+len(moreData)), downloadedBytes)
	assert.Equal(t, 2, callCount)
}

func TestProgressWriterWithNilCallback(t *testing.T) {
	buffer := &strings.Builder{}
	pw := &progressWriter{
		file:     buffer,
		callback: nil,
	}

	testData := []byte("Test data")
	n, err := pw.Write(testData)

	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, "Test data", buffer.String())
}

func TestFibonacci(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 0},
		{1, 1},
		{2, 1},
		{3, 2},
		{4, 3},
		{5, 5},
		{6, 8},
		{7, 13},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("fibonacci(%d)", tt.input), func(t *testing.T) {
			result := fibonacci(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDownloadFilePermissionError(t *testing.T) {
	// Skip this test on Windows as it handles permissions differently
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	// Try to write to a directory that doesn't exist or is not writable
	filePath := "/root/nonexistent/file.txt"

	f := New(nil, 1)
	req, err := f.NewRequest(server.URL, filePath)
	require.NoError(t, err)

	response := f.DownloadFile(req)
	err = response.Error()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not open file")
}

func TestDownloadFilesParallelism(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "parallel_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Calculate expected hash for "content"
	hasher := blake3.New()
	hasher.Write([]byte("content"))
	expectedHash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Track concurrent requests
	var activeRequests int32
	var maxConcurrent int32
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		activeRequests++
		if activeRequests > maxConcurrent {
			maxConcurrent = activeRequests
		}
		mu.Unlock()

		// Simulate some work
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		activeRequests--
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("content"))
	}))
	defer server.Close()

	// Create 5 requests but limit to 2 concurrent
	requests := make([]*Request, 5)
	f := New(nil, 1)

	for i := range requests {
		filePath := filepath.Join(tempDir, fmt.Sprintf("parallel_%d.txt", i))
		req, reqErr := f.NewRequest(server.URL, filePath)
		require.NoError(t, reqErr)
		requests[i] = req
	}

	responsesChan, cancelAll := f.DownloadFiles(requests, 2) // Max 2 parallel
	defer cancelAll()

	responses := make([]*Response, 0, len(requests))
	for response := range responsesChan {
		responses = append(responses, response)
		err := response.Error()
		assert.NoError(t, err)
	}

	assert.Len(t, responses, 5)
	assert.LessOrEqual(t, maxConcurrent, int32(2)) // Should not exceed parallelism limit

	// Verify all responses have the correct hash
	for _, response := range responses {
		assert.Equal(t, expectedHash, response.Hash, "All responses should have the same Blake3 hash")
	}
}

func TestResponseBytes(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "response_bytes_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testContent := "This is test content for the Bytes method"
	hasher := blake3.New()
	hasher.Write([]byte(testContent))
	expectedHash := fmt.Sprintf("%x", hasher.Sum(nil))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	filePath := filepath.Join(tempDir, "bytes_test.txt")
	f := New(nil, 1)
	req, err := f.NewRequest(server.URL, filePath)
	require.NoError(t, err)

	response := f.DownloadFile(req)
	err = response.Error()
	require.NoError(t, err)
	assert.Equal(t, expectedHash, response.Hash, "Hash should match expected Blake3 hash")

	// Test Bytes method
	content, err := response.Bytes()
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestResponseBytesFileNotFound(t *testing.T) {
	req := &Request{
		Url:      "http://example.com",
		FilePath: "/nonexistent/path/file.txt",
	}

	response := &Response{
		Request: req,
		Done:    make(chan struct{}),
	}
	close(response.Done) // Mark as complete

	content, err := response.Bytes()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
	assert.Nil(t, content)
}

func TestBlake3FileHash(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blake3_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a file with known content
	testContent := "Hello, World!"
	expectedHash := "288a86a79f20a3d6dccdca7713beaed178798296bdfa7913fa2a62d9727bf8f8"
	filePath := filepath.Join(tempDir, "test_file.txt")

	err = os.WriteFile(filePath, []byte(testContent), 0644)
	require.NoError(t, err)

	// Calculate hash using Blake3
	hasher := blake3.New()
	fileContent, err := os.ReadFile(filePath)
	require.NoError(t, err)

	_, err = hasher.Write(fileContent)
	require.NoError(t, err)

	actualHash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Verify the hash matches the expected value
	assert.Equal(t, expectedHash, actualHash)
	assert.Equal(t, testContent, string(fileContent))
}
