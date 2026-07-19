package gcl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"strings"
)

var errOwnerNotFound = errors.New("owner not found")

// Forge lists the repositories of an owner (organization, user, or
// group) on a code hosting platform.
type Forge interface {
	ListCloneURLs(owner string) ([]string, error)
}

var forgeForHost = func(host string) (Forge, bool) {
	host = strings.ToLower(host)
	switch {
	case host == "github.com":
		return newGitHubForge("https://api.github.com"), true
	case host == "gitlab.com" || isGitLabHost(host):
		return newGitLabForge("https://" + host + "/api/v4"), true
	default:
		return nil, false
	}
}

// isGitLabHost reports whether host is listed in GCL_GITLAB_HOSTS, a
// comma-separated list of self-hosted GitLab instances.
func isGitLabHost(host string) bool {
	for extraHost := range strings.SplitSeq(os.Getenv("GCL_GITLAB_HOSTS"), ",") {
		if strings.EqualFold(strings.TrimSpace(extraHost), host) {
			return true
		}
	}
	return false
}

func getJSONList[T any](client *http.Client, requestURL string, header http.Header) ([]T, error) {
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	maps.Copy(request.Header, header)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, errOwnerNotFound
	}
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return nil, fmt.Errorf("%s: %s", response.Status, body)
	}

	var items []T
	if err := json.NewDecoder(response.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}
