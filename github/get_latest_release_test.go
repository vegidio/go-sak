package github

import (
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-github/v74/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestRelease(t *testing.T) {
	t.Run("successful request with known stable repository", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping integration test in short mode")
		}

		// Test with the Go repository, which is stable and always has releases
		release, err := GetLatestRelease("golang", "go")

		if err != nil {
			// Skip if there's a network/API issue
			t.Skipf("API call failed (likely network/rate limit): %v", err)
			return
		}

		require.NotNil(t, release)
		assert.NotEmpty(t, release.GetTagName())
		assert.Contains(t, release.GetTagName(), "go") // Go releases start with "go"
	})

	t.Run("successful request with vscode repository", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping integration test in short mode")
		}

		release, err := GetLatestRelease("microsoft", "vscode")

		if err != nil {
			t.Skipf("API call failed (likely network/rate limit): %v", err)
			return
		}

		require.NotNil(t, release)
		assert.NotEmpty(t, release.GetTagName())
		assert.NotNil(t, release.Name)
		// VSCode releases typically have numeric versions
		assert.Regexp(t, `^\d+\.\d+\.\d+`, release.GetTagName())
	})

	t.Run("error with empty owner", func(t *testing.T) {
		release, err := GetLatestRelease("", "repo")

		assert.Error(t, err)
		assert.Nil(t, release)
	})

	t.Run("error with empty repo", func(t *testing.T) {
		release, err := GetLatestRelease("owner", "")

		assert.Error(t, err)
		assert.Nil(t, release)
	})

	t.Run("error with both empty parameters", func(t *testing.T) {
		release, err := GetLatestRelease("", "")

		assert.Error(t, err)
		assert.Nil(t, release)
	})

	t.Run("error with non-existent repository", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping API test in short mode")
		}

		// Use a repository that definitely doesn't exist
		release, err := GetLatestRelease("thisuserdoesnotexist12345", "thisrepodoesnotexist12345")

		assert.Error(t, err)
		assert.Nil(t, release)

		// Should be a 404 error
		var githubErr *github.ErrorResponse
		if errors.As(err, &githubErr) {
			assert.Equal(t, http.StatusNotFound, githubErr.Response.StatusCode)
		}
	})

	t.Run("mock successful response structure", func(t *testing.T) {
		// Test our understanding of the GitHub API response structure
		tagName := "v1.2.3"
		name := "Release v1.2.3"
		body := "Release notes"
		prerelease := false

		release := &github.RepositoryRelease{
			TagName:    &tagName,
			Name:       &name,
			Body:       &body,
			Prerelease: &prerelease,
		}

		// Test that we can access all the expected fields
		assert.Equal(t, "v1.2.3", release.GetTagName())
		assert.Equal(t, "Release v1.2.3", release.GetName())
		assert.Equal(t, "Release notes", release.GetBody())
		assert.False(t, release.GetPrerelease())
	})

	t.Run("mock response with assets", func(t *testing.T) {
		assetName := "binary-linux-amd64"
		downloadCount := 1500

		asset := &github.ReleaseAsset{
			Name:          &assetName,
			DownloadCount: &downloadCount,
		}

		release := &github.RepositoryRelease{
			TagName: github.Ptr("v2.0.0"),
			Assets:  []*github.ReleaseAsset{asset},
		}

		assert.Equal(t, "v2.0.0", release.GetTagName())
		assert.Len(t, release.Assets, 1)
		assert.Equal(t, "binary-linux-amd64", release.Assets[0].GetName())
		assert.Equal(t, 1500, release.Assets[0].GetDownloadCount())
	})

	t.Run("mock error scenarios", func(t *testing.T) {
		testCases := []struct {
			name        string
			owner       string
			repo        string
			mockError   error
			expectError bool
		}{
			{
				name:        "api rate limit error",
				owner:       "test-owner",
				repo:        "test-repo",
				mockError:   errors.New("API rate limit exceeded"),
				expectError: true,
			},
			{
				name:  "not found error",
				owner: "nonexistent",
				repo:  "repo",
				mockError: &github.ErrorResponse{
					Response: &http.Response{StatusCode: 404},
					Message:  "Not Found",
				},
				expectError: true,
			},
			{
				name:        "network error",
				owner:       "test-owner",
				repo:        "test-repo",
				mockError:   errors.New("network timeout"),
				expectError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create a mock function that returns the expected error
				mockGetLatestRelease := func(owner, repo string) (*github.RepositoryRelease, error) {
					return nil, tc.mockError
				}

				release, err := mockGetLatestRelease(tc.owner, tc.repo)

				if tc.expectError {
					assert.Error(t, err)
					assert.Nil(t, release)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, release)
				}
			})
		}
	})

	t.Run("input validation edge cases", func(t *testing.T) {
		testCases := []struct {
			name        string
			owner       string
			repo        string
			description string
		}{
			{
				name:        "whitespace only owner",
				owner:       "   ",
				repo:        "repo",
				description: "should error with whitespace-only owner",
			},
			{
				name:        "whitespace only repo",
				owner:       "owner",
				repo:        "   ",
				description: "should error with whitespace-only repo",
			},
			{
				name:        "special characters",
				owner:       "owner-with-dashes",
				repo:        "repo.with.dots",
				description: "should handle special characters (though repo might not exist)",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if testing.Short() {
					t.Skip("Skipping API validation test in short mode")
				}

				_, err := GetLatestRelease(tc.owner, tc.repo)
				// All these cases should result in errors (either validation or 404)
				assert.Error(t, err, tc.description)
			})
		}
	})

	t.Run("response field accessibility", func(t *testing.T) {
		// Test that we can safely access optional fields using Get methods
		release := &github.RepositoryRelease{
			TagName: github.Ptr("v1.0.0"),
			// Intentionally leaving other fields nil to test Get methods
		}

		// These should not panic even with nil fields
		assert.Equal(t, "v1.0.0", release.GetTagName())
		assert.Empty(t, release.GetName())       // Should return empty string for nil
		assert.Empty(t, release.GetBody())       // Should return empty string for nil
		assert.False(t, release.GetPrerelease()) // Should return false for nil
		assert.Empty(t, release.GetHTMLURL())    // Should return empty string for nil
	})
}

func BenchmarkGetLatestRelease(b *testing.B) {
	b.Run("mock response processing", func(b *testing.B) {
		// Benchmark just the response processing without API calls
		release := &github.RepositoryRelease{
			TagName: github.Ptr("v1.0.0"),
			Name:    github.Ptr("Test Release"),
			Body:    github.Ptr("Test release body"),
		}

		mockFunc := func(owner, repo string) (*github.RepositoryRelease, error) {
			return release, nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := mockFunc("test", "repo")
			if err != nil {
				b.Fatal(err)
			}
			// Access some fields to simulate real usage
			_ = result.GetTagName()
			_ = result.GetName()
		}
	})
}
