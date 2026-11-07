package gcl

import (
	"os"
	"path/filepath"
	"testing"
)

// isolateTokens clears every token and host setting and points the config
// file at path (a nonexistent file when empty).
func isolateTokens(t *testing.T, configJSON string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if configJSON != "" {
		if err := os.WriteFile(path, []byte(configJSON), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GCL_CONFIG", path)
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GCL_GITHUB_HOSTS", "")
	t.Setenv("GCL_GITLAB_HOSTS", "")
}

func TestTokenForURL(t *testing.T) {
	isolateTokens(t, `{
		"github_token": "gh-global",
		"gitlab_token": "gl-global",
		"github_hosts": {
			"ghe.example.com": {"api_base": "https://ghe.example.com/api/v3", "token": "ghe-host"},
			"ghe2.example.com": {"api_base": "https://ghe2.example.com/api/v3"}
		},
		"gitlab_hosts": ["git.example.com"]
	}`)

	tests := []struct {
		url, username, token string
	}{
		{"https://github.com/o/r.git", "x-access-token", "gh-global"},
		{"https://GitHub.com/o/r", "x-access-token", "gh-global"},
		{"https://ghe.example.com/o/r.git", "x-access-token", "ghe-host"},
		{"https://ghe2.example.com/o/r.git", "x-access-token", "gh-global"},
		{"https://ghe.example.com:8443/o/r.git", "x-access-token", "ghe-host"},
		{"https://gitlab.com/g/r.git", "oauth2", "gl-global"},
		{"https://git.example.com/g/sub/r.git", "oauth2", "gl-global"},
		{"https://unknown.example.com/o/r.git", "", ""},
		{"ssh://git@github.com/o/r.git", "", ""},
		{"git@github.com:o/r.git", "", ""},
	}
	for _, tt := range tests {
		username, token := tokenForURL(tt.url)
		if username != tt.username || token != tt.token {
			t.Errorf("tokenForURL(%q) = %q, %q, want %q, %q", tt.url, username, token, tt.username, tt.token)
		}
	}
}

func TestTokenForURLEnvWinsOverConfig(t *testing.T) {
	isolateTokens(t, `{"github_token": "from-config", "gitlab_token": "from-config"}`)
	t.Setenv("GITHUB_TOKEN", "gh-env")
	t.Setenv("GITLAB_TOKEN", "gl-env")
	t.Setenv("GCL_GITHUB_HOSTS", "ghe.example.com=https://ghe.example.com/api/v3")

	if _, token := tokenForURL("https://github.com/o/r"); token != "gh-env" {
		t.Errorf("github.com token = %q, want gh-env", token)
	}
	if _, token := tokenForURL("https://ghe.example.com/o/r"); token != "gh-env" {
		t.Errorf("GitHub Enterprise token = %q, want gh-env", token)
	}
	if _, token := tokenForURL("https://gitlab.com/g/r"); token != "gl-env" {
		t.Errorf("gitlab.com token = %q, want gl-env", token)
	}
}

func TestHTTPAuthForFallsBackToToken(t *testing.T) {
	requireGit(t)
	useStoredCredentials(t)
	isolateTokens(t, "")
	t.Setenv("GITHUB_TOKEN", "gh-env")

	auth := httpAuthFor("https://github.com/o/r.git")
	if auth == nil || auth.Username != "x-access-token" || auth.Password != "gh-env" {
		t.Fatalf("httpAuthFor() = %+v, want token credentials", auth)
	}
}

func TestHTTPAuthForPrefersCredentialHelper(t *testing.T) {
	requireGit(t)
	useStoredCredentials(t, "https://alice:s3cret@github.com")
	isolateTokens(t, "")
	t.Setenv("GITHUB_TOKEN", "gh-env")

	auth := httpAuthFor("https://github.com/o/r.git")
	if auth == nil || auth.Username != "alice" || auth.Password != "s3cret" {
		t.Fatalf("httpAuthFor() = %+v, want helper credentials", auth)
	}
}
