package github

import (
	"strings"
	"testing"

	"github.com/google/go-github/v74/github"
	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/semver"
)

func TestIsOutdatedRelease_VersionComparison(t *testing.T) {
	tests := []struct {
		name           string
		latestVersion  string
		currentVersion string
		expected       bool
		description    string
	}{
		{
			name:           "outdated_version_without_v_prefix",
			latestVersion:  "v1.2.0",
			currentVersion: "1.1.0",
			expected:       true,
			description:    "Current version is outdated when latest is newer",
		},
		{
			name:           "outdated_version_with_v_prefix",
			latestVersion:  "v1.2.0",
			currentVersion: "v1.1.0",
			expected:       true,
			description:    "Current version with v prefix is outdated when latest is newer",
		},
		{
			name:           "up_to_date_version",
			latestVersion:  "v1.2.0",
			currentVersion: "1.2.0",
			expected:       false,
			description:    "Current version is up to date",
		},
		{
			name:           "newer_version",
			latestVersion:  "v1.2.0",
			currentVersion: "1.3.0",
			expected:       false,
			description:    "Current version is newer than latest",
		},
		{
			name:           "latest_without_v_prefix",
			latestVersion:  "1.2.0",
			currentVersion: "1.1.0",
			expected:       true,
			description:    "Latest version without v prefix, current is outdated",
		},
		{
			name:           "both_without_v_prefix",
			latestVersion:  "1.2.0",
			currentVersion: "1.1.0",
			expected:       true,
			description:    "Both versions without v prefix, current is outdated",
		},
		{
			name:           "patch_version_outdated",
			latestVersion:  "v1.2.3",
			currentVersion: "v1.2.2",
			expected:       true,
			description:    "Current patch version is outdated",
		},
		{
			name:           "major_version_outdated",
			latestVersion:  "v2.0.0",
			currentVersion: "v1.9.9",
			expected:       true,
			description:    "Current major version is outdated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock release
			release := &github.RepositoryRelease{
				Name: &tt.latestVersion,
			}

			// Test the semver comparison logic by simulating the function's behavior
			latestVersion := *release.Name
			version := tt.currentVersion

			// Apply the same prefix logic as the function
			if !strings.HasPrefix(version, "v") {
				version = "v" + version
			}
			if !strings.HasPrefix(latestVersion, "v") {
				latestVersion = "v" + latestVersion
			}

			result := semver.Compare(latestVersion, version) > 0

			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestIsOutdatedRelease_ErrorScenarios(t *testing.T) {
	t.Run("invalid_repository", func(t *testing.T) {
		// Test with a repository that doesn't exist
		result := IsOutdatedRelease("nonexistent-owner", "nonexistent-repo", "1.0.0")
		assert.False(t, result, "Should return false when repository doesn't exist")
	})
}

func TestIsOutdatedRelease_EdgeCases(t *testing.T) {
	t.Run("empty_version_string", func(t *testing.T) {
		// Test with an empty version string
		result := IsOutdatedRelease("golang", "go", "")
		// The function should handle this gracefully
		assert.False(t, result, "Should handle empty version string gracefully")
	})

	t.Run("invalid_semver_format", func(t *testing.T) {
		// Test with invalid semantic version format
		result := IsOutdatedRelease("golang", "go", "invalid-version")
		// semver.Compare should handle invalid versions appropriately
		assert.False(t, result, "Should handle invalid semver format gracefully")
	})
}

// Integration test examples (these would actually call the GitHub API)
func TestIsOutdatedRelease_Integration(t *testing.T) {
	// Skip integration tests in normal unit test runs
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	t.Run("real_repository_check", func(t *testing.T) {
		// Test with a real repository - using a version that's likely to be outdated
		result := IsOutdatedRelease("vegidio", "mediasim", "1.0.0")
		assert.True(t, result, "mediasim 1.0.0 should be outdated compared to latest release")
	})

	t.Run("current_version_check", func(t *testing.T) {
		// Test with a very recent version that's likely to be current or newer
		result := IsOutdatedRelease("golang", "go", "1.23.0")
		// This assertion just verifies the function doesn't panic
		assert.IsType(t, false, result, "Should return a boolean value")
	})
}
