package gcl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

var errOwnerNotFound = errors.New("owner not found")

type gitHubForge struct {
	apiBase string
	token   string
	client  *http.Client
	perPage int
}

func newGitHubForge(apiBase string) *gitHubForge {
	return &gitHubForge{
		apiBase: apiBase,
		token:   os.Getenv("GITHUB_TOKEN"),
		client:  http.DefaultClient,
		perPage: 100,
	}
}

func (g *gitHubForge) ListCloneURLs(owner string) ([]string, error) {
	urls, err := g.listRepos("orgs", owner)
	if errors.Is(err, errOwnerNotFound) {
		return g.listRepos("users", owner)
	}
	return urls, err
}

func (g *gitHubForge) listRepos(ownerKind, owner string) ([]string, error) {
	var urls []string
	for page := 1; ; page++ {
		repos, err := g.listReposPage(ownerKind, owner, page)
		if err != nil {
			return nil, err
		}
		for _, repo := range repos {
			urls = append(urls, repo.CloneURL)
		}
		if len(repos) < g.perPage {
			return urls, nil
		}
	}
}

type gitHubRepo struct {
	CloneURL string `json:"clone_url"`
}

func (g *gitHubForge) listReposPage(ownerKind, owner string, page int) ([]gitHubRepo, error) {
	url := fmt.Sprintf("%s/%s/%s/repos?per_page=%d&page=%d&sort=full_name", g.apiBase, ownerKind, owner, g.perPage, page)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	if g.token != "" {
		request.Header.Set("Authorization", "Bearer "+g.token)
	}

	response, err := g.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s", errOwnerNotFound, owner)
	}
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return nil, fmt.Errorf("listing repositories of %s failed: %s: %s", owner, response.Status, body)
	}

	var repos []gitHubRepo
	if err := json.NewDecoder(response.Body).Decode(&repos); err != nil {
		return nil, err
	}
	return repos, nil
}
