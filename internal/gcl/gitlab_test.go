package gcl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func writeProjectPage(t *testing.T, w http.ResponseWriter, paths ...string) {
	t.Helper()
	projects := make([]gitLabProject, 0, len(paths))
	for _, path := range paths {
		projects = append(projects, gitLabProject{HTTPURLToRepo: fmt.Sprintf("https://gitlab.com/%s.git", path)})
	}
	if err := json.NewEncoder(w).Encode(projects); err != nil {
		t.Fatal(err)
	}
}

func TestGitLabForgeListsGroupProjectsAcrossPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/groups/mygroup/projects" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("include_subgroups") != "true" {
			t.Errorf("include_subgroups not requested: %s", r.URL.RawQuery)
		}
		switch r.URL.Query().Get("page") {
		case "1":
			writeProjectPage(t, w, "mygroup/one", "mygroup/sub/two")
		case "2":
			writeProjectPage(t, w, "mygroup/three")
		default:
			t.Errorf("unexpected page: %s", r.URL.RawQuery)
		}
	}))
	t.Cleanup(server.Close)

	forge := newGitLabForge(server.URL)
	forge.perPage = 2

	got, err := forge.ListCloneURLs("mygroup")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"https://gitlab.com/mygroup/one.git",
		"https://gitlab.com/mygroup/sub/two.git",
		"https://gitlab.com/mygroup/three.git",
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

func TestGitLabForgeEscapesSubgroupPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/groups/mygroup%2Fsub/projects" {
			t.Errorf("unexpected request path: %s", r.URL.EscapedPath())
			http.NotFound(w, r)
			return
		}
		writeProjectPage(t, w, "mygroup/sub/one")
	}))
	t.Cleanup(server.Close)

	forge := newGitLabForge(server.URL)

	got, err := forge.ListCloneURLs("mygroup/sub")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0] != "https://gitlab.com/mygroup/sub/one.git" {
		t.Fatalf("ListCloneURLs() = %v", got)
	}
}

func TestGitLabForgeFallsBackToUserProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/groups/someuser/projects":
			http.NotFound(w, r)
		case "/users/someuser/projects":
			writeProjectPage(t, w, "someuser/dotfiles")
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	forge := newGitLabForge(server.URL)

	got, err := forge.ListCloneURLs("someuser")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0] != "https://gitlab.com/someuser/dotfiles.git" {
		t.Fatalf("ListCloneURLs() = %v", got)
	}
}

func TestForgeForHostKnowsGitLabCom(t *testing.T) {
	if _, ok := forgeForHost("gitlab.com"); !ok {
		t.Fatal("forgeForHost(gitlab.com) not found")
	}
}

func TestForgeForHostSelfHostedGitLab(t *testing.T) {
	t.Setenv("GCL_GITLAB_HOSTS", "git.example.com, gitlab.internal")

	for _, host := range []string{"git.example.com", "gitlab.internal"} {
		if _, ok := forgeForHost(host); !ok {
			t.Fatalf("forgeForHost(%s) not found", host)
		}
	}
	if _, ok := forgeForHost("other.example.com"); ok {
		t.Fatal("forgeForHost(other.example.com) unexpectedly found")
	}
}

func TestForgeForHostSelfHostedGitHub(t *testing.T) {
	t.Setenv("GCL_GITHUB_HOSTS", "github.example.com=https://github.example.com/api/v3, git.internal = https://git.internal/api/v3 ")

	for _, host := range []string{"github.example.com", "git.internal"} {
		if _, ok := forgeForHost(host); !ok {
			t.Fatalf("forgeForHost(%s) not found", host)
		}
	}
	if _, ok := forgeForHost("other.example.com"); ok {
		t.Fatal("forgeForHost(other.example.com) unexpectedly found")
	}
}
