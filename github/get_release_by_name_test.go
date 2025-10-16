package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v74/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetReleaseByName_Success(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/microsoft/vscode/releases/tags/1.85.0", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		tagName := "1.85.0"
		name := "Version 1.85.0"
		body := "Release notes for version 1.85.0"

		release := github.RepositoryRelease{
			TagName: &tagName,
			Name:    &name,
			Body:    &body,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	// Override the GitHub API base URL for testing
	client := github.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(server.URL + "/")

	// We need to temporarily modify the function to accept a client
	// For this test, we'll test the actual function which makes real API calls
	release, err := GetReleaseByName("microsoft", "vscode", "1.85.0")

	// Note: This will make a real API call. For true unit testing,
	// you'd want to refactor GetReleaseByName to accept a client parameter
	require.NoError(t, err)
	assert.NotNil(t, release)
	assert.Equal(t, "1.85.0", release.GetTagName())
}

func TestGetReleaseByName_ReleaseNotFound(t *testing.T) {
	// This test will make a real API call that should fail
	release, err := GetReleaseByName("microsoft", "vscode", "nonexistent-tag-12345")

	assert.Error(t, err)
	assert.Nil(t, release)
}

func TestGetReleaseByName_InvalidOwner(t *testing.T) {
	release, err := GetReleaseByName("", "vscode", "1.85.0")

	assert.Error(t, err)
	assert.Nil(t, release)
}

func TestGetReleaseByName_InvalidRepo(t *testing.T) {
	release, err := GetReleaseByName("microsoft", "", "1.85.0")

	assert.Error(t, err)
	assert.Nil(t, release)
}

func TestGetReleaseByName_InvalidTagName(t *testing.T) {
	release, err := GetReleaseByName("microsoft", "vscode", "")

	assert.Error(t, err)
	assert.Nil(t, release)
}

func TestGetReleaseByName_RepositoryNotFound(t *testing.T) {
	release, err := GetReleaseByName("nonexistent-owner-12345", "nonexistent-repo-12345", "v1.0.0")

	assert.Error(t, err)
	assert.Nil(t, release)
}
