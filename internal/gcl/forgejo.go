package gcl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ForgejoClient implements ForgeClient for Forgejo (Gitea-based)
type ForgejoClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewForgejoClient creates a new Forgejo client
func NewForgejoClient(host string, token string) *ForgejoClient {
	baseURL := fmt.Sprintf("https://%s/api/v1", host)

	return &ForgejoClient{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{},
	}
}

// ListRepositories lists all repositories for a Forgejo user or organization
func (cc *ForgejoClient) ListRepositories(ctx context.Context, pathStr string) ([]Repository, error) {
	// Remove leading/trailing slashes
	pathStr = strings.Trim(pathStr, "/")

	if pathStr == "" {
		return nil, fmt.Errorf("Forgejo path must contain user/org name")
	}

	var allRepos []Repository
	page := 1

	for {
		url := fmt.Sprintf("%s/users/%s/repos?page=%d&limit=50", cc.baseURL, pathStr, page)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		if cc.token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("token %s", cc.token))
		}

		resp, err := cc.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch repositories: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		var repos []struct {
			Name    string `json:"name"`
			HTMLURL string `json:"html_url"`
			Owner   struct {
				Login string `json:"login"`
			} `json:"owner"`
		}

		err = json.NewDecoder(resp.Body).Decode(&repos)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if len(repos) == 0 {
			break
		}

		for _, repo := range repos {
			allRepos = append(allRepos, Repository{
				Name:  repo.Name,
				URL:   repo.HTMLURL,
				Owner: repo.Owner.Login,
			})
		}

		page++
	}

	return allRepos, nil
}