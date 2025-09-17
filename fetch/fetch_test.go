package fetch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("with nil headers", func(t *testing.T) {
		f := New(nil, 3)

		assert.NotNil(t, f)
		assert.NotNil(t, f.restClient)
		assert.NotNil(t, f.httpClient)
		assert.NotNil(t, f.headers)
		assert.Equal(t, 3, f.retries)
		assert.Equal(t, userAgent, f.headers["User-Agent"])
	})

	t.Run("with empty headers", func(t *testing.T) {
		headers := make(map[string]string)
		f := New(headers, 2)

		assert.NotNil(t, f)
		assert.Equal(t, 2, f.retries)
		assert.Equal(t, userAgent, f.headers["User-Agent"])
	})

	t.Run("with custom headers", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer token",
			"Content-Type":  "application/json",
		}
		f := New(headers, 1)

		assert.NotNil(t, f)
		assert.Equal(t, 1, f.retries)
		assert.Equal(t, "Bearer token", f.headers["Authorization"])
		assert.Equal(t, "application/json", f.headers["Content-Type"])
		assert.Equal(t, userAgent, f.headers["User-Agent"])
	})

	t.Run("with custom user agent", func(t *testing.T) {
		customUA := "MyBot/1.0"
		headers := map[string]string{
			"User-Agent": customUA,
		}
		f := New(headers, 0)

		assert.NotNil(t, f)
		assert.Equal(t, 0, f.retries)
		assert.Equal(t, customUA, f.headers["User-Agent"])
	})

	t.Run("with zero retries", func(t *testing.T) {
		f := New(nil, 0)

		assert.NotNil(t, f)
		assert.Equal(t, 0, f.retries)
	})
}

func TestFetch_GetText(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		expectedBody := "Hello, World!"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedBody))
		}))
		defer server.Close()

		f := New(nil, 0)
		result, err := f.GetText(server.URL)

		assert.NoError(t, err)
		assert.Equal(t, expectedBody, result)
	})

	t.Run("empty response body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		f := New(nil, 0)
		result, err := f.GetText(server.URL)

		assert.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("server returns error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		}))
		defer server.Close()

		f := New(nil, 0)
		result, err := f.GetText(server.URL)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "404")
		assert.Empty(t, result)
	})

	t.Run("server returns 500 error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		f := New(nil, 0)
		result, err := f.GetText(server.URL)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500")
		assert.Empty(t, result)
	})

	t.Run("invalid URL", func(t *testing.T) {
		f := New(nil, 0)
		result, err := f.GetText("invalid://url")

		assert.Error(t, err)
		assert.Empty(t, result)
	})

	t.Run("connection refused", func(t *testing.T) {
		f := New(nil, 0)
		result, err := f.GetText("http://localhost:99999")

		assert.Error(t, err)
		assert.Empty(t, result)
	})

	t.Run("with custom headers", func(t *testing.T) {
		expectedBody := "Custom headers work!"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedBody))
		}))
		defer server.Close()

		headers := map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer token",
		}
		f := New(headers, 0)
		result, err := f.GetText(server.URL)

		assert.NoError(t, err)
		assert.Equal(t, expectedBody, result)
	})
}

func TestFetch_GetResult(t *testing.T) {
	type TestResponse struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	}

	t.Run("successful JSON response", func(t *testing.T) {
		expectedResponse := TestResponse{
			Message: "Success",
			Code:    200,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(expectedResponse)
		}))
		defer server.Close()

		f := New(nil, 0)
		var result TestResponse
		resp, err := f.GetResult(server.URL, nil, &result)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Equal(t, expectedResponse.Message, result.Message)
		assert.Equal(t, expectedResponse.Code, result.Code)
	})

	t.Run("with additional headers", func(t *testing.T) {
		expectedResponse := TestResponse{Message: "With headers", Code: 201}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(expectedResponse)
		}))
		defer server.Close()

		headers := map[string]string{
			"Authorization":   "Bearer test-token",
			"X-Custom-Header": "custom-value",
		}

		f := New(nil, 0)
		var result TestResponse
		resp, err := f.GetResult(server.URL, headers, &result)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusCreated, resp.StatusCode())
		assert.Equal(t, expectedResponse.Message, result.Message)
		assert.Equal(t, expectedResponse.Code, result.Code)
	})

	t.Run("server returns error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Bad Request"}`))
		}))
		defer server.Close()

		f := New(nil, 0)
		var result TestResponse
		resp, err := f.GetResult(server.URL, nil, &result)

		assert.Error(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, err.Error(), "400")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode())
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"invalid": json}`))
		}))
		defer server.Close()

		f := New(nil, 0)
		var result TestResponse
		resp, _ := f.GetResult(server.URL, nil, &result)

		// Should not error on invalid JSON when using resty with SetResult
		// Resty will attempt to unmarshal and may partially succeed or fail silently
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode())
	})

	t.Run("connection error", func(t *testing.T) {
		f := New(nil, 0)
		var result TestResponse
		resp, err := f.GetResult("http://localhost:99999", nil, &result)

		assert.Error(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("nil headers map", func(t *testing.T) {
		expectedResponse := TestResponse{Message: "No extra headers", Code: 200}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(expectedResponse)
		}))
		defer server.Close()

		f := New(nil, 0)
		var result TestResponse
		resp, err := f.GetResult(server.URL, nil, &result)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Equal(t, expectedResponse.Message, result.Message)
		assert.Equal(t, expectedResponse.Code, result.Code)
	})
}

func TestFetch_Integration_WithRetry(t *testing.T) {
	t.Run("retry on server error", func(t *testing.T) {
		var requestCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Server Error"))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Success after retry"))
		}))
		defer server.Close()

		f := New(nil, 3)
		result, err := f.GetText(server.URL)

		assert.NoError(t, err)
		assert.Equal(t, "Success after retry", result)
		assert.Equal(t, 3, requestCount)
	})

	t.Run("exhaust all retries", func(t *testing.T) {
		var requestCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Persistent Error"))
		}))
		defer server.Close()

		f := New(nil, 2)
		result, err := f.GetText(server.URL)

		assert.Error(t, err)
		assert.Empty(t, result)
		// With retry count 2, should make 1 initial + 2 retry attempts = 3 total
		assert.Equal(t, 3, requestCount)
	})
}

func TestFetch_UserAgent(t *testing.T) {
	t.Run("default user agent is set", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, userAgent, r.Header.Get("User-Agent"))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		defer server.Close()

		f := New(nil, 0)
		_, err := f.GetText(server.URL)

		assert.NoError(t, err)
	})

	t.Run("custom user agent is used", func(t *testing.T) {
		customUA := "TestBot/1.0"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, customUA, r.Header.Get("User-Agent"))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		defer server.Close()

		headers := map[string]string{"User-Agent": customUA}
		f := New(headers, 0)
		_, err := f.GetText(server.URL)

		assert.NoError(t, err)
	})
}

func TestFetch_Concurrent(t *testing.T) {
	t.Run("concurrent requests", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("Response from %s", r.URL.Path)))
		}))
		defer server.Close()

		f := New(nil, 0)

		// Start multiple concurrent requests
		numRequests := 10
		results := make(chan string, numRequests)
		errors := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(id int) {
				url := fmt.Sprintf("%s/request-%d", server.URL, id)
				result, err := f.GetText(url)
				if err != nil {
					errors <- err
				} else {
					results <- result
				}
			}(i)
		}

		// Collect results
		successCount := 0
		errorCount := 0
		timeout := time.After(5 * time.Second)

		for i := 0; i < numRequests; i++ {
			select {
			case result := <-results:
				assert.Contains(t, result, "Response from /request-")
				successCount++
			case err := <-errors:
				t.Logf("Request error: %v", err)
				errorCount++
			case <-timeout:
				t.Fatal("Test timed out waiting for concurrent requests")
			}
		}

		assert.Equal(t, numRequests, successCount)
		assert.Equal(t, 0, errorCount)
	})
}

func TestFetch_EdgeCases(t *testing.T) {
	t.Run("empty URL", func(t *testing.T) {
		f := New(nil, 0)
		result, err := f.GetText("")

		assert.Error(t, err)
		assert.Empty(t, result)
	})

	t.Run("very large response", func(t *testing.T) {
		// Create a large response (1MB)
		largeBody := make([]byte, 1024*1024)
		for i := range largeBody {
			largeBody[i] = byte('A' + (i % 26))
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(largeBody)
		}))
		defer server.Close()

		f := New(nil, 0)
		result, err := f.GetText(server.URL)

		assert.NoError(t, err)
		assert.Len(t, result, len(largeBody))
		assert.Equal(t, string(largeBody), result)
	})

	t.Run("special characters in response", func(t *testing.T) {
		specialBody := "Hello, 世界! 🌍 Special chars: àáâãäåæçèéêë"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(specialBody))
		}))
		defer server.Close()

		f := New(nil, 0)
		result, err := f.GetText(server.URL)

		assert.NoError(t, err)
		assert.Equal(t, specialBody, result)
	})
}

// Benchmark tests
func BenchmarkNew(b *testing.B) {
	headers := map[string]string{
		"Authorization": "Bearer token",
		"Content-Type":  "application/json",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New(headers, 3)
	}
}

func BenchmarkGetText(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Benchmark response"))
	}))
	defer server.Close()

	f := New(nil, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.GetText(server.URL)
	}
}
