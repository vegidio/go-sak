package o11y

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Geolocation represents the response from ipinfo.io
type Geolocation struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
}

// FetchGeolocation retrieves geolocation information for the current public IP address.
//
// It makes an HTTP GET request to the ipinfo.io service to get location details including IP address, city, region,
// country, coordinates, organization, postal code, and timezone information.
//
// Returns a pointer to a Geolocation struct containing the location data, or an error if the request fails, returns a
// non-200 status code, or the response cannot be decoded.
func FetchGeolocation(baseURL ...string) (*Geolocation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	url := "https://ipinfo.io/json"
	if len(baseURL) > 0 && baseURL[0] != "" {
		url = baseURL[0] + "/json"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch geolocation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var geo Geolocation
	if err = json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &geo, nil
}
