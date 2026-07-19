package gcl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func writeRepoPage(t *testing.T, w http.ResponseWriter, names ...string) {
	t.Helper()
	repos := make([]gitHubRepo, 0, len(names))
	for _, name := range names {
		repos = append(repos, gitHubRepo{CloneURL: fmt.Sprintf("https://github.com/%s.git", name)})
	}
	if err := json.NewEncoder(w).Encode(repos); err != nil {
		t.Fatal(err)
	}
}

func TestGitHubForgeListsOrgReposAcrossPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs/myorg/repos" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		switch r.URL.Query().Get("page") {
		case "1":
			writeRepoPage(t, w, "myorg/one", "myorg/two")
		case "2":
			writeRepoPage(t, w, "myorg/three")
		default:
			t.Errorf("unexpected page: %s", r.URL.RawQuery)
		}
	}))
	t.Cleanup(server.Close)

	forge := newGitHubForge(server.URL)
	forge.perPage = 2

	got, err := forge.ListCloneURLs("myorg")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"https://github.com/myorg/one.git",
		"https://github.com/myorg/two.git",
		"https://github.com/myorg/three.git",
	}
	if len(got) != len(want) {
		t.Fatalf("ListCloneURLs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ListCloneURLs() = %v, want %v", got, want)
		}
	}
}

func TestGitHubForgeFallsBackToUserRepos(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/someuser/repos":
			http.NotFound(w, r)
		case "/users/someuser/repos":
			writeRepoPage(t, w, "someuser/dotfiles")
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	forge := newGitHubForge(server.URL)

	got, err := forge.ListCloneURLs("someuser")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0] != "https://github.com/someuser/dotfiles.git" {
		t.Fatalf("ListCloneURLs() = %v", got)
	}
}

func TestGitHubForgeReportsAPIErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limit exceeded", http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	forge := newGitHubForge(server.URL)

	_, err := forge.ListCloneURLs("myorg")
	if err == nil {
		t.Fatal("ListCloneURLs() succeeded, want error")
	}
}
