package gcl

import (
	"errors"
	"fmt"
	"net/http"
	"os"
)

type gitHubForge struct {
	apiBase string
	token   string
	client  *http.Client
	perPage int
}

// newGitHubForge builds a forge for apiBase. token, if set, is used as-is
// (a per-host token from config); otherwise it falls back to GITHUB_TOKEN
// and then the file config's global github_token.
func newGitHubForge(apiBase, token string) *gitHubForge {
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		token = loadFileConfig().GitHubToken
	}
	return &gitHubForge{
		apiBase: apiBase,
		token:   token,
		client:  http.DefaultClient,
		perPage: 100,
	}
}

type gitHubRepo struct {
	CloneURL string `json:"clone_url"`
}

func (g *gitHubForge) ListCloneURLs(owner string) ([]string, error) {
	urls, err := g.listRepos("orgs", owner)
	if errors.Is(err, errOwnerNotFound) {
		return g.listRepos("users", owner)
	}
	return urls, err
}

func (g *gitHubForge) listRepos(ownerKind, owner string) ([]string, error) {
	header := http.Header{}
	header.Set("Accept", "application/vnd.github+json")
	if g.token != "" {
		header.Set("Authorization", "Bearer "+g.token)
	}

	var urls []string
	for page := 1; ; page++ {
		requestURL := fmt.Sprintf("%s/%s/%s/repos?per_page=%d&page=%d&sort=full_name", g.apiBase, ownerKind, owner, g.perPage, page)

		repos, err := getJSONList[gitHubRepo](g.client, requestURL, header)
		if err != nil {
			return nil, fmt.Errorf("listing repositories of %s: %w", owner, err)
		}
		for _, repo := range repos {
			urls = append(urls, repo.CloneURL)
		}
		if len(repos) < g.perPage {
			return urls, nil
		}
	}
}
