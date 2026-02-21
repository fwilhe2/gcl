package gcl

import (
	"context"
	"fmt"
	"net/url"
)

// Repository represents a repository from any forge
type Repository struct {
	Name  string
	URL   string
	Owner string
}

// Config holds authentication tokens for various forges
type Config struct {
	GitHubToken   string
	GitLabToken   string
	CodebergToken string
}

// ListRepositories lists all repositories for a given forge URL
func ListRepositories(ctx context.Context, forgeURL string, config *Config) ([]Repository, error) {
	parsedURL, err := url.Parse(forgeURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	host := parsedURL.Host
	path := parsedURL.Path

	// Detect forge and create appropriate client
	var client ForgeClient

	platform, err := DetectPlatform(forgeURL)
	if err != nil {
		return nil, fmt.Errorf("cant infer platform: %w", err)
	}

	switch {
	case isGitHub(host) || platform == GitHubEnterprise:
		client = NewGitHubClient(config.GitHubToken)
	case platform == GitLab:
		client = NewGitLabClient(host, config.GitLabToken)
	case platform == Forgejo:
		client = NewForgejoClient(host, config.CodebergToken)
	default:
		return nil, fmt.Errorf("unsupported forge: %s", host)
	}

	return client.ListRepositories(ctx, path)
}

// ForgeClient is the interface that all forge clients must implement
type ForgeClient interface {
	ListRepositories(ctx context.Context, path string) ([]Repository, error)
}

func isGitHub(host string) bool {
	return host == "github.com" || host == "www.github.com"
}
