package gcl

import (
	"net/url"
	"os"
	"strings"
)

// tokenForURL returns the configured API token for the forge hosting an
// HTTP(S) clone URL, together with the username that forge expects for
// token authentication over HTTPS. It is the fallback when no git
// credential helper has a credential for the URL, so a token that is
// already set up for listing an owner's repositories also works for
// cloning private ones. Empty if the URL is not HTTP(S) or no token is
// configured for its host.
func tokenForURL(gitUrl string) (username, token string) {
	u, err := url.Parse(gitUrl)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", ""
	}
	host := strings.ToLower(u.Hostname())

	switch {
	case host == "github.com":
		return "x-access-token", gitHubToken("")
	case host == "gitlab.com" || isGitLabHost(host):
		return "oauth2", gitLabToken()
	default:
		if cfg, ok := gitHubHostConfigFor(host); ok {
			return "x-access-token", gitHubToken(cfg.Token)
		}
		return "", ""
	}
}

// gitHubToken returns hostToken if set, otherwise GITHUB_TOKEN, otherwise
// the config file's github_token.
func gitHubToken(hostToken string) string {
	if hostToken != "" {
		return hostToken
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	return loadFileConfig().GitHubToken
}

// gitLabToken returns GITLAB_TOKEN, otherwise the config file's
// gitlab_token.
func gitLabToken() string {
	if token := os.Getenv("GITLAB_TOKEN"); token != "" {
		return token
	}
	return loadFileConfig().GitLabToken
}
