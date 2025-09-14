package github

import (
	"context"
	"strings"

	"github.com/google/go-github/v74/github"
	"golang.org/x/mod/semver"
)

// IsOutdatedRelease checks if a given version is outdated compared to the latest release of a GitHub repository. It
// fetches the latest tag from the specified repository and performs semantic version comparison.
//
// # Parameters:
//   - owner: The owner (username or organization) of the GitHub repository
//   - repo: The name of the GitHub repository
//   - version: The version to check (can be with or without 'v' prefix)
//
// # Returns:
//   - true if the given version is older than the latest release
//   - false if the given version is up-to-date, newer, or if an error occurs
//     (e.g., repository not found, no releases, network error)
//
// The function automatically handles version prefixes by ensuring both versions have the 'v' prefix before performing
// semantic version comparison using golang.org/x/mod/semver.
//
// # Example:
//
//	outdated := IsOutdatedRelease("golang", "go", "1.20.0")
//	if outdated {
//	    fmt.Println("Your Go version is outdated")
//	}
func IsOutdatedRelease(owner, repo, version string) bool {
	client := github.NewClient(nil)

	tags, _, err := client.Repositories.ListTags(context.Background(), owner, repo, nil)
	if err != nil || len(tags) == 0 {
		return false
	}

	latestVersion := tags[0].GetName()
	if latestVersion == "" {
		return false
	}

	// Ensure both versions have the 'v' prefix for semver comparison
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !strings.HasPrefix(latestVersion, "v") {
		latestVersion = "v" + latestVersion
	}

	// Use semantic version comparison
	return semver.Compare(latestVersion, version) > 0
}
