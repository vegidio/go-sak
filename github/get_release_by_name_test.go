package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetReleaseByName_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This makes a real API call to GitHub. GetReleaseByName does not accept a client, so we cannot
	// point it at a mock server (see the TODO in history). Skip gracefully on transient API issues.
	release, err := GetReleaseByName(context.Background(), "microsoft", "vscode", "1.85.0")
	if err != nil {
		t.Skipf("API call failed (likely network/rate limit): %v", err)
	}

	assert.NotNil(t, release)
	assert.Equal(t, "1.85.0", release.GetTagName())
}

func TestGetReleaseByName_ReleaseNotFound(t *testing.T) {
	// This test will make a real API call that should fail
	release, err := GetReleaseByName(context.Background(), "microsoft", "vscode", "nonexistent-tag-12345")

	assert.Error(t, err)
	assert.Nil(t, release)
}

func TestGetReleaseByName_InvalidOwner(t *testing.T) {
	release, err := GetReleaseByName(context.Background(), "", "vscode", "1.85.0")

	assert.Error(t, err)
	assert.Nil(t, release)
}

func TestGetReleaseByName_InvalidRepo(t *testing.T) {
	release, err := GetReleaseByName(context.Background(), "microsoft", "", "1.85.0")

	assert.Error(t, err)
	assert.Nil(t, release)
}

func TestGetReleaseByName_InvalidTagName(t *testing.T) {
	release, err := GetReleaseByName(context.Background(), "microsoft", "vscode", "")

	assert.Error(t, err)
	assert.Nil(t, release)
}

func TestGetReleaseByName_RepositoryNotFound(t *testing.T) {
	release, err := GetReleaseByName(context.Background(), "nonexistent-owner-12345", "nonexistent-repo-12345", "v1.0.0")

	assert.Error(t, err)
	assert.Nil(t, release)
}
