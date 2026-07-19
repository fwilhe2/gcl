package gcl

import "strings"

// Forge lists the repositories of an owner (organization or user) on a
// code hosting platform.
type Forge interface {
	ListCloneURLs(owner string) ([]string, error)
}

var forgeForHost = func(host string) (Forge, bool) {
	switch strings.ToLower(host) {
	case "github.com":
		return newGitHubForge("https://api.github.com"), true
	default:
		return nil, false
	}
}
