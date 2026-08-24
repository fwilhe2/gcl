package gcl

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type gitLabForge struct {
	apiBase string
	token   string
	client  *http.Client
	perPage int
}

func newGitLabForge(apiBase string) *gitLabForge {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		token = loadFileConfig().GitLabToken
	}
	return &gitLabForge{
		apiBase: apiBase,
		token:   token,
		client:  &http.Client{Timeout: 30 * time.Second},
		perPage: 100,
	}
}

type gitLabProject struct {
	HTTPURLToRepo string `json:"http_url_to_repo"`
}

func (g *gitLabForge) ListCloneURLs(owner string) ([]string, error) {
	// A group listing includes the projects of all nested subgroups.
	urls, err := g.listProjects(owner, "groups/"+url.PathEscape(owner)+"/projects", url.Values{"include_subgroups": {"true"}})
	if errors.Is(err, errOwnerNotFound) {
		return g.listProjects(owner, "users/"+url.PathEscape(owner)+"/projects", url.Values{})
	}
	return urls, err
}

func (g *gitLabForge) listProjects(owner, resource string, query url.Values) ([]string, error) {
	header := http.Header{}
	if g.token != "" {
		header.Set("PRIVATE-TOKEN", g.token)
	}

	query.Set("order_by", "path")
	query.Set("sort", "asc")
	query.Set("per_page", strconv.Itoa(g.perPage))

	var urls []string
	for page := 1; ; page++ {
		query.Set("page", strconv.Itoa(page))
		requestURL := fmt.Sprintf("%s/%s?%s", g.apiBase, resource, query.Encode())

		projects, err := getJSONList[gitLabProject](g.client, requestURL, header)
		if err != nil {
			return nil, fmt.Errorf("listing repositories of %s: %w", owner, err)
		}
		for _, project := range projects {
			urls = append(urls, project.HTTPURLToRepo)
		}
		if len(projects) < g.perPage {
			return urls, nil
		}
	}
}
