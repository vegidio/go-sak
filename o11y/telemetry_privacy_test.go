package o11y

import (
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingTransport records how many requests were attempted and fails them all, so a test can assert on outbound
// traffic without depending on the network.
type countingTransport struct{ calls atomic.Int64 }

func (c *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	c.calls.Add(1)
	return nil, assert.AnError
}

func withCountingGeoClient(t *testing.T) *countingTransport {
	t.Helper()

	counter := &countingTransport{}
	original := geoClient.Transport
	geoClient.Transport = counter
	t.Cleanup(func() { geoClient.Transport = original })

	return counter
}

func TestNewTelemetry_DoesNotPhoneHomeWhenDisabled(t *testing.T) {
	// Geolocation discloses the caller's public IP to a third party. Opting out of telemetry has to mean opting
	// out of that lookup too, not merely out of exporting the result.
	counter := withCountingGeoClient(t)

	telemetry, err := NewTelemetry("localhost:4318", "test-service", "1.0.0", nil, EnvDevelopment, false)
	require.NoError(t, err)
	defer telemetry.Close()

	assert.Equal(t, int64(0), counter.calls.Load(), "no request may be made when telemetry is disabled")

	fields := telemetry.prefilled()
	assert.NotContains(t, fields, "location.country")
	assert.NotContains(t, fields, "location.city")

	// The locally-read enrichment is still gathered, since it never leaves the process.
	assert.Equal(t, "1.0.0", fields["version"])
	assert.Contains(t, fields, "machine.os")
}

func TestNewTelemetry_MachineIDIsScopedToTheService(t *testing.T) {
	withCountingGeoClient(t)

	a, err := NewTelemetry("localhost:4318", "service-a", "1.0.0", nil, EnvDevelopment, false)
	require.NoError(t, err)
	defer a.Close()

	b, err := NewTelemetry("localhost:4318", "service-b", "1.0.0", nil, EnvDevelopment, false)
	require.NoError(t, err)
	defer b.Close()

	idA, _ := a.prefilled()["machine.id"].(string)
	idB, _ := b.prefilled()["machine.id"].(string)

	require.NotEmpty(t, idA)
	require.NotEmpty(t, idB)

	// A raw host id would be identical across applications, letting a backend correlate this machine's telemetry
	// with that of every other program on it.
	assert.NotEqual(t, idA, idB, "machine.id must be derived per service, not the raw host id")
}

func TestNewTelemetry_SurfacesInitErrorAndStaysUsable(t *testing.T) {
	withCountingGeoClient(t)

	// A space makes url.Parse fail, which is the error path that used to hand back a nil cleanup func.
	telemetry, err := NewTelemetry("ht tp://bad endpoint", "test-service", "1.0.0", nil, EnvDevelopment, true)

	require.Error(t, err, "a failed exporter init must be reported, not silently swallowed")
	require.NotNil(t, telemetry)

	assert.NotPanics(t, func() {
		telemetry.LogInfo("still works", nil)
	})
	assert.NotPanics(t, func() {
		assert.NoError(t, telemetry.Close())
	}, "Close must not panic after a failed init")
}

func TestTelemetry_RenewSessionIsConcurrencySafe(t *testing.T) {
	// RenewSession used to write the enrichment map in place while the Log methods ranged over it, which is a
	// fatal "concurrent map read and map write" rather than merely a -race finding.
	telemetry := newTestTelemetry()
	defer telemetry.Close()

	var wg sync.WaitGroup

	for range 4 {
		wg.Go(func() {
			for range 200 {
				telemetry.RenewSession()
			}
		})
		wg.Go(func() {
			for range 200 {
				telemetry.LogInfo("concurrent", map[string]any{"n": 1})
			}
		})
	}

	wg.Wait()

	sessionID, ok := telemetry.prefilled()["session.id"].(string)
	require.True(t, ok)
	assert.Len(t, sessionID, 36)
}
