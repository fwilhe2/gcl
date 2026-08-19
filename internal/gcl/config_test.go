package gcl

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, contents string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GCL_CONFIG", path)
}

func TestForgeForHostSelfHostedGitLabFromConfigFile(t *testing.T) {
	writeConfig(t, `{"gitlab_hosts": ["git.example.com", "gitlab.internal"]}`)

	for _, host := range []string{"git.example.com", "gitlab.internal"} {
		if _, ok := forgeForHost(host); !ok {
			t.Fatalf("forgeForHost(%s) not found", host)
		}
	}
	if _, ok := forgeForHost("other.example.com"); ok {
		t.Fatal("forgeForHost(other.example.com) unexpectedly found")
	}
}

func TestForgeForHostSelfHostedGitHubFromConfigFile(t *testing.T) {
	writeConfig(t, `{"github_hosts": {"github.example.com": "https://github.example.com/api/v3"}}`)

	if _, ok := forgeForHost("github.example.com"); !ok {
		t.Fatal("forgeForHost(github.example.com) not found")
	}
	if _, ok := forgeForHost("other.example.com"); ok {
		t.Fatal("forgeForHost(other.example.com) unexpectedly found")
	}
}

func TestLoadFileConfigIgnoresMissingFile(t *testing.T) {
	t.Setenv("GCL_CONFIG", filepath.Join(t.TempDir(), "does-not-exist.json"))

	cfg := loadFileConfig()
	if len(cfg.GitHubHosts) != 0 || len(cfg.GitLabHosts) != 0 {
		t.Fatalf("loadFileConfig() = %+v, want zero value", cfg)
	}
}

func TestLoadFileConfigIgnoresInvalidJSON(t *testing.T) {
	writeConfig(t, `not json`)

	cfg := loadFileConfig()
	if len(cfg.GitHubHosts) != 0 || len(cfg.GitLabHosts) != 0 {
		t.Fatalf("loadFileConfig() = %+v, want zero value", cfg)
	}
}
