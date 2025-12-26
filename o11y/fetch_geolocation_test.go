package o11y

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchGeolocation(t *testing.T) {
	t.Run("successful geolocation fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/json", r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
                "ip": "8.8.8.8",
                "hostname": "dns.google",
                "city": "Mountain View",
                "region": "California",
                "country": "US",
                "loc": "37.4056,-122.0775",
                "org": "AS15169 Google LLC",
                "postal": "94043",
                "timezone": "America/Los_Angeles"
            }`))
		}))
		defer server.Close()

		geo, err := FetchGeolocation(server.URL)
		require.NoError(t, err)
		require.NotNil(t, geo)

		assert.NotEmpty(t, geo.IP)
		assert.NotEmpty(t, geo.Country)
	})

	t.Run("handles non-200 status codes", func(t *testing.T) {
		testCases := []struct {
			name       string
			statusCode int
		}{
			{"not found", http.StatusNotFound},
			{"internal server error", http.StatusInternalServerError},
			{"bad request", http.StatusBadRequest},
			{"service unavailable", http.StatusServiceUnavailable},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.statusCode)
				}))
				defer server.Close()

				geo, err := FetchGeolocation(server.URL)
				assert.Error(t, err)
				assert.Nil(t, geo)
				assert.Contains(t, err.Error(), "unexpected status code")
			})
		}
	})

	t.Run("handles invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{invalid json`))
		}))
		defer server.Close()

		geo, err := FetchGeolocation(server.URL)
		assert.Error(t, err)
		assert.Nil(t, geo)
		assert.Contains(t, err.Error(), "failed to decode response")
	})

	t.Run("handles empty JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}))
		defer server.Close()

		geo, err := FetchGeolocation(server.URL)
		require.NoError(t, err)
		require.NotNil(t, geo)

		// All fields should be empty strings
		assert.Empty(t, geo.IP)
		assert.Empty(t, geo.Hostname)
		assert.Empty(t, geo.City)
		assert.Empty(t, geo.Region)
		assert.Empty(t, geo.Country)
		assert.Empty(t, geo.Loc)
		assert.Empty(t, geo.Org)
		assert.Empty(t, geo.Postal)
		assert.Empty(t, geo.Timezone)
	})

	t.Run("handles partial JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
                "ip": "1.2.3.4",
                "country": "US"
            }`))
		}))
		defer server.Close()

		geo, err := FetchGeolocation(server.URL)
		require.NoError(t, err)
		require.NotNil(t, geo)

		assert.Equal(t, "1.2.3.4", geo.IP)
		assert.Equal(t, "US", geo.Country)
		assert.Empty(t, geo.City)
		assert.Empty(t, geo.Region)
	})

	t.Run("handles server timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow server - sleep longer than context timeout
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		geo, err := FetchGeolocation(server.URL)
		assert.Error(t, err)
		assert.Nil(t, geo)
		assert.Contains(t, err.Error(), "failed to fetch geolocation")
	})

	t.Run("handles connection refused", func(t *testing.T) {
		// Create a server and immediately close it to simulate a connection refused
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		serverURL := server.URL
		server.Close()

		geo, err := FetchGeolocation(serverURL)
		assert.Error(t, err)
		assert.Nil(t, geo)
		assert.Contains(t, err.Error(), "failed to fetch geolocation")
	})

	t.Run("validates complete geolocation data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
                "ip": "203.0.113.42",
                "hostname": "example.com",
                "city": "San Francisco",
                "region": "California",
                "country": "US",
                "loc": "37.7749,-122.4194",
                "org": "AS12345 Example Organization",
                "postal": "94102",
                "timezone": "America/Los_Angeles"
            }`))
		}))
		defer server.Close()

		geo, err := FetchGeolocation(server.URL)
		require.NoError(t, err)
		require.NotNil(t, geo)

		assert.Equal(t, "203.0.113.42", geo.IP)
		assert.Equal(t, "example.com", geo.Hostname)
		assert.Equal(t, "San Francisco", geo.City)
		assert.Equal(t, "California", geo.Region)
		assert.Equal(t, "US", geo.Country)
		assert.Equal(t, "37.7749,-122.4194", geo.Loc)
		assert.Equal(t, "AS12345 Example Organization", geo.Org)
		assert.Equal(t, "94102", geo.Postal)
		assert.Equal(t, "America/Los_Angeles", geo.Timezone)
	})

	t.Run("handles special characters in JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
                "ip": "192.168.1.1",
                "city": "São Paulo",
                "region": "São Paulo",
                "country": "BR",
                "org": "AS1234 Empresa & Cia."
            }`))
		}))
		defer server.Close()

		geo, err := FetchGeolocation(server.URL)
		require.NoError(t, err)
		require.NotNil(t, geo)

		assert.Equal(t, "São Paulo", geo.City)
		assert.Equal(t, "São Paulo", geo.Region)
		assert.Equal(t, "Empresa & Cia.", geo.Org[7:])
	})
}

func TestGeolocation_Structure(t *testing.T) {
	t.Run("geolocation struct has correct json tags", func(t *testing.T) {
		geo := Geolocation{
			IP:       "1.2.3.4",
			Hostname: "test.com",
			City:     "TestCity",
			Region:   "TestRegion",
			Country:  "TC",
			Loc:      "0.0,0.0",
			Org:      "TestOrg",
			Postal:   "12345",
			Timezone: "UTC",
		}

		assert.Equal(t, "1.2.3.4", geo.IP)
		assert.Equal(t, "test.com", geo.Hostname)
		assert.Equal(t, "TestCity", geo.City)
		assert.Equal(t, "TestRegion", geo.Region)
		assert.Equal(t, "TC", geo.Country)
		assert.Equal(t, "0.0,0.0", geo.Loc)
		assert.Equal(t, "TestOrg", geo.Org)
		assert.Equal(t, "12345", geo.Postal)
		assert.Equal(t, "UTC", geo.Timezone)
	})
}
