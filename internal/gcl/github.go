package gcl

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
)

// GitHubClient implements ForgeClient for GitHub
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient creates a new GitHub client
func NewGitHubClient(token string) *GitHubClient {
	var httpClient *github.Client

	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		tc := oauth2.NewClient(context.Background(), ts)
		httpClient = github.NewClient(tc)
	} else {
		httpClient = github.NewClient(nil)
	}

	return &GitHubClient{client: httpClient}
}

// ListRepositories lists all repositories for a GitHub organization or user
func (gc *GitHubClient) ListRepositories(ctx context.Context, pathStr string) ([]Repository, error) {
	// Remove leading/trailing slashes
	owner := strings.Trim(pathStr, "/")

	if owner == "" {
		return nil, fmt.Errorf("GitHub path must contain owner/org name")
	}

	var allRepos []Repository
	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	// Try to list as organization first
	for {
		repos, resp, err := gc.client.Repositories.ListByOrg(ctx, owner, opts)
		if err == nil {
			for _, repo := range repos {
				allRepos = append(allRepos, Repository{
					Name:  *repo.Name,
					URL:   *repo.HTMLURL,
					Owner: owner,
				})
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
			continue
		}

		// If org fails, try user
		userOpts := &github.RepositoryListOptions{
			ListOptions: github.ListOptions{PerPage: 100},
		}

		for {
			repos, resp, errUser := gc.client.Repositories.List(ctx, owner, userOpts)
			if errUser != nil {
				return nil, fmt.Errorf("failed to list repos for %s: %w", owner, err)
			}

			for _, repo := range repos {
				allRepos = append(allRepos, Repository{
					Name:  *repo.Name,
					URL:   *repo.HTMLURL,
					Owner: owner,
				})
			}

			if resp.NextPage == 0 {
				break
			}
			userOpts.Page = resp.NextPage
		}
		break
	}

	return allRepos, nil
}