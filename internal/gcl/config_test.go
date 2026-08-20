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
	writeConfig(t, `{"github_hosts": {"github.example.com": {"api_base": "https://github.example.com/api/v3"}}}`)

	if _, ok := forgeForHost("github.example.com"); !ok {
		t.Fatal("forgeForHost(github.example.com) not found")
	}
	if _, ok := forgeForHost("other.example.com"); ok {
		t.Fatal("forgeForHost(other.example.com) unexpectedly found")
	}
}

func TestGitHubHostTokenFromConfigFile(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	writeConfig(t, `{"github_token": "global-tok", "github_hosts": {"github.example.com": {"api_base": "https://github.example.com/api/v3", "token": "host-tok"}}}`)

	forge, ok := forgeForHost("github.example.com")
	if !ok {
		t.Fatal("forgeForHost(github.example.com) not found")
	}
	if got := forge.(*gitHubForge).token; got != "host-tok" {
		t.Fatalf("token = %q, want %q", got, "host-tok")
	}
}

func TestCloneBaseDirFromConfigFile(t *testing.T) {
	t.Setenv("GCL_BASE_DIR", "")
	writeConfig(t, `{"base_dir": "/from/config"}`)

	got, err := cloneBaseDir("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/from/config" {
		t.Fatalf("cloneBaseDir() = %q, want %q", got, "/from/config")
	}
}

func TestCloneBaseDirEnvWinsOverConfigFile(t *testing.T) {
	writeConfig(t, `{"base_dir": "/from/config"}`)
	t.Setenv("GCL_BASE_DIR", "/from/env")

	got, err := cloneBaseDir("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/from/env" {
		t.Fatalf("cloneBaseDir() = %q, want %q", got, "/from/env")
	}
}

func TestGitHubTokenFromConfigFile(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	writeConfig(t, `{"github_token": "tok3n"}`)

	forge := newGitHubForge("https://api.github.com", "")
	if forge.token != "tok3n" {
		t.Fatalf("token = %q, want %q", forge.token, "tok3n")
	}
}

func TestGitHubTokenEnvWinsOverConfigFile(t *testing.T) {
	writeConfig(t, `{"github_token": "from-config"}`)
	t.Setenv("GITHUB_TOKEN", "from-env")

	forge := newGitHubForge("https://api.github.com", "")
	if forge.token != "from-env" {
		t.Fatalf("token = %q, want %q", forge.token, "from-env")
	}
}

func TestGitLabTokenFromConfigFile(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	writeConfig(t, `{"gitlab_token": "tok3n"}`)

	forge := newGitLabForge("https://gitlab.com/api/v4")
	if forge.token != "tok3n" {
		t.Fatalf("token = %q, want %q", forge.token, "tok3n")
	}
}

func TestGitLabTokenEnvWinsOverConfigFile(t *testing.T) {
	writeConfig(t, `{"gitlab_token": "from-config"}`)
	t.Setenv("GITLAB_TOKEN", "from-env")

	forge := newGitLabForge("https://gitlab.com/api/v4")
	if forge.token != "from-env" {
		t.Fatalf("token = %q, want %q", forge.token, "from-env")
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
