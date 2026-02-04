package repository

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-github/v82/github"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RefLister lists branches and tags for a repository. Used for testing so
// ResolveBranchAndPath can be tested without calling real GitHub/GitLab APIs.
type RefLister interface {
	ListBranchesAndTags(repoInfo *RepositoryInfo, token string) ([]string, error)
}

// testRefLister is set by tests to mock branch/tag listing. When non-nil,
// ResolveBranchAndPath uses it instead of the real API clients.
var testRefLister RefLister

// SetTestRefLister sets the RefLister used by ResolveBranchAndPath. Only for
// tests; call with nil to restore real API behavior.
func SetTestRefLister(l RefLister) {
	testRefLister = l
}

// ResolveBranchAndPath resolves ambiguous URLs by listing branches
// Returns: (branch/ref, path, error)
func ResolveBranchAndPath(repoInfo *RepositoryInfo, urlPath string, token string) (string, string, error) {
	if testRefLister != nil {
		branches, err := testRefLister.ListBranchesAndTags(repoInfo, token)
		if err != nil {
			return "", "", err
		}
		return findLongestMatch(branches, urlPath)
	}
	switch repoInfo.Type {
	case GitHub:
		return resolveGitHubBranchAndPath(repoInfo, urlPath, token)
	case GitLab:
		return resolveGitLabBranchAndPath(repoInfo, urlPath, token)
	default:
		return "", "", fmt.Errorf("unsupported repository type: %s", repoInfo.Type)
	}
}

// resolveGitHubBranchAndPath resolves GitHub branch and path
func resolveGitHubBranchAndPath(repoInfo *RepositoryInfo, urlPath string, token string) (string, string, error) {
	ctx := context.Background()
	var client *github.Client
	if token != "" {
		client = github.NewClient(nil).WithAuthToken(token)
	} else {
		client = github.NewClient(nil)
	}

	// List all branches
	opts := &github.BranchListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allBranches []string
	for {
		branches, resp, err := client.Repositories.ListBranches(
			ctx,
			repoInfo.Owner,
			repoInfo.Repo,
			opts,
		)
		if err != nil {
			return "", "", fmt.Errorf("failed to list branches: %w", err)
		}

		for _, branch := range branches {
			allBranches = append(allBranches, branch.GetName())
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// Also list tags
	tagOpts := &github.ListOptions{PerPage: 100}
	tags, _, err := client.Repositories.ListTags(ctx, repoInfo.Owner, repoInfo.Repo, tagOpts)
	if err == nil {
		for _, tag := range tags {
			allBranches = append(allBranches, tag.GetName())
		}
	}

	// Find longest matching branch/tag in the path
	return findLongestMatch(allBranches, urlPath)
}

// resolveGitLabBranchAndPath resolves GitLab branch and path
func resolveGitLabBranchAndPath(repoInfo *RepositoryInfo, urlPath string, token string) (string, string, error) {
	var client *gitlab.Client
	var err error

	if token != "" {
		client, err = gitlab.NewClient(token, gitlab.WithBaseURL(repoInfo.BaseURL+"/api/v4"))
	} else {
		client, err = gitlab.NewClient("", gitlab.WithBaseURL(repoInfo.BaseURL+"/api/v4"))
	}

	if err != nil {
		return "", "", fmt.Errorf("failed to create GitLab client: %w", err)
	}

	projectID := fmt.Sprintf("%s/%s", repoInfo.Owner, repoInfo.Repo)

	// List all branches
	opts := &gitlab.ListBranchesOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}

	var allBranches []string
	for {
		branches, resp, err := client.Branches.ListBranches(projectID, opts)
		if err != nil {
			return "", "", fmt.Errorf("failed to list branches: %w", err)
		}

		for _, branch := range branches {
			allBranches = append(allBranches, branch.Name)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// Also list tags
	tagOpts := &gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}
	tags, _, err := client.Tags.ListTags(projectID, tagOpts)
	if err == nil {
		for _, tag := range tags {
			allBranches = append(allBranches, tag.Name)
		}
	}

	log.Printf("Found %d branches/tags for %s", len(allBranches), projectID)

	// Find longest matching branch/tag in the path
	return findLongestMatch(allBranches, urlPath)
}

// findLongestMatch finds the longest branch name that matches the beginning of the path
// Returns: (matched branch, remaining path, error)
func findLongestMatch(branches []string, urlPath string) (string, string, error) {
	urlPath = strings.Trim(urlPath, "/")

	var longestMatch string
	var longestMatchLen int

	for _, branch := range branches {
		// Check if path starts with this branch
		if strings.HasPrefix(urlPath, branch) {
			if len(branch) > longestMatchLen {
				longestMatch = branch
				longestMatchLen = len(branch)
			}
		}
	}

	if longestMatch == "" {
		return "", "", fmt.Errorf("no matching branch found in path: %s", urlPath)
	}

	// Extract remaining path after the branch
	remainingPath := strings.TrimPrefix(urlPath, longestMatch)
	remainingPath = strings.TrimPrefix(remainingPath, "/")

	log.Printf("Resolved: branch=%s, path=%s", longestMatch, remainingPath)

	return longestMatch, remainingPath, nil
}
