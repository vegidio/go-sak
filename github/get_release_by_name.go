package github

import (
	"context"

	"github.com/google/go-github/v74/github"
)

// GetReleaseByName retrieves a specific release by its tag name for the specified GitHub repository.
// It takes the repository owner, repository name, and the release tag name as parameters.
//
// # Parameters:
//   - ctx: Context for cancellation and timeouts
//   - owner: The GitHub username or organization name that owns the repository
//   - repo: The name of the repository
//   - tagName: The tag name of the release (e.g., "v1.0.0")
//
// # Returns:
//   - *github.RepositoryRelease: The release information including tag name, body, assets, etc.
//   - error: An error if the API request fails or if the release is not found
//
// # Example:
//
//	release, err := GetReleaseByName(ctx, "microsoft", "vscode", "1.85.0")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Release: %s\n", release.GetTagName())
func GetReleaseByName(ctx context.Context, owner, repo, tagName string) (*github.RepositoryRelease, error) {
	client := github.NewClient(nil)

	release, _, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, tagName)
	if err != nil {
		return nil, err
	}

	return release, nil
}
