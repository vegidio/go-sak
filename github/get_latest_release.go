package github

import (
	"context"

	"github.com/google/go-github/v74/github"
)

// GetLatestRelease retrieves the latest published release for the specified GitHub repository. It takes the repository
// owner and repository name as parameters and returns the latest release information or an error if the request fails.
//
// # Parameters:
//   - owner: The GitHub username or organization name that owns the repository
//   - repo: The name of the repository
//
// # Returns:
//   - *github.RepositoryRelease: The latest release information including tag name, body, assets, etc.
//   - error: An error if the API request fails or if no releases are found
//
// # Example:
//
//	release, err := GetLatestRelease("microsoft", "vscode")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Latest release: %s\n", release.GetTagName())
func GetLatestRelease(owner, repo string) (*github.RepositoryRelease, error) {
	client := github.NewClient(nil)

	release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
	if err != nil {
		return nil, err
	}

	return release, nil
}
