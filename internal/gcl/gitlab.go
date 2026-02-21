package gcl

import (
	"context"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/api/client-go"
)

// GitLabClient implements ForgeClient for GitLab
type GitLabClient struct {
	client *gitlab.Client
	host   string
}

// NewGitLabClient creates a new GitLab client
func NewGitLabClient(host string, token string) *GitLabClient {
	baseURL := fmt.Sprintf("https://%s/api/v4", host)

	var client *gitlab.Client
	var err error

	if token != "" {
		client, err = gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	} else {
		client, err = gitlab.NewClient("", gitlab.WithBaseURL(baseURL))
	}

	if err != nil {
		// Return client even if there's an error, it may still work
		client, _ = gitlab.NewClient("", gitlab.WithBaseURL(baseURL))
	}

	return &GitLabClient{
		client: client,
		host:   host,
	}
}

// ListRepositories lists all repositories for a GitLab group or user (including nested groups)
func (gc *GitLabClient) ListRepositories(ctx context.Context, pathStr string) ([]Repository, error) {
	// Remove leading/trailing slashes
	pathStr = strings.Trim(pathStr, "/")

	if pathStr == "" {
		return nil, fmt.Errorf("GitLab path must contain group/user name")
	}

	// Try to get as a group first (groups support nested structures like foo/bar/baz)
	groupRepos, err := gc.listGroupRepositories(ctx, pathStr)
	if err == nil && len(groupRepos) > 0 {
		return groupRepos, nil
	}

	// If group fails, try as user
	userRepos, err := gc.listUserRepositories(ctx, pathStr)
	if err != nil {
		return nil, fmt.Errorf("failed to list repos for %s as group or user: %w", pathStr, err)
	}

	return userRepos, nil
}

// listGroupRepositories lists repositories in a GitLab group (with nested group support)
func (gc *GitLabClient) listGroupRepositories(ctx context.Context, groupPath string) ([]Repository, error) {
	var allRepos []Repository

	// List projects directly owned by the group
	listOpts := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
		Archived:    gitlab.Ptr(false),
	}

	for {
		projects, resp, err := gc.client.Groups.ListGroupProjects(groupPath, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list group projects: %w", err)
		}

		for _, project := range projects {
			allRepos = append(allRepos, Repository{
				Name:  project.Name,
				URL:   project.WebURL,
				Owner: groupPath,
			})
		}

		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	// List subgroups and recursively get their repositories
	subgroupOpts := &gitlab.ListSubGroupsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}

	for {
		subgroups, resp, err := gc.client.Groups.ListSubGroups(groupPath, subgroupOpts)
		if err != nil {
			// Subgroups might not be available, continue with what we have
			break
		}

		for _, subgroup := range subgroups {
			subgroupRepos, err := gc.listGroupRepositories(ctx, subgroup.FullPath)
			if err == nil {
				allRepos = append(allRepos, subgroupRepos...)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		subgroupOpts.Page = resp.NextPage
	}

	return allRepos, nil
}

// listUserRepositories lists repositories for a GitLab user
func (gc *GitLabClient) listUserRepositories(ctx context.Context, username string) ([]Repository, error) {
	var allRepos []Repository

	listOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
		Archived:    gitlab.Ptr(false),
	}

	for {
		projects, resp, err := gc.client.Projects.ListUserProjects(username, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list user projects: %w", err)
		}

		for _, project := range projects {
			allRepos = append(allRepos, Repository{
				Name:  project.Name,
				URL:   project.WebURL,
				Owner: username,
			})
		}

		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	return allRepos, nil
}