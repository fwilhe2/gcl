package gcl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v6/plumbing/transport"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
)

// requireGit skips the test when no git binary is available, since
// credentialFromHelper shells out to "git credential fill".
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

// useGitCredentialHelper points git at a throwaway global config using the
// given credential helper, so tests never touch the real user configuration.
func useGitCredentialHelper(t *testing.T, helper string) {
	t.Helper()

	dir := t.TempDir()
	gitconfig := filepath.Join(dir, "gitconfig")
	// Quote the value: git config treats ";" and "#" in a bare value as the
	// start of a comment, which would truncate a shell helper.
	quoted := `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(helper) + `"`
	content := fmt.Sprintf("[credential]\n\thelper = %s\n", quoted)
	if err := os.WriteFile(gitconfig, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", dir)
	t.Setenv("GIT_CONFIG_GLOBAL", gitconfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

// useStoredCredentials configures the "store" helper backed by a temporary
// file holding the given entries, e.g. "https://alice:s3cret@git.example.com".
func useStoredCredentials(t *testing.T, entries ...string) {
	t.Helper()

	credentials := filepath.Join(t.TempDir(), "credentials")
	var content string
	if len(entries) > 0 {
		content = strings.Join(entries, "\n") + "\n"
	}
	if err := os.WriteFile(credentials, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	useGitCredentialHelper(t, "store --file="+filepath.ToSlash(credentials))
}

func basicAuth(t *testing.T, auth transport.AuthMethod) *http.BasicAuth {
	t.Helper()
	got, ok := auth.(*http.BasicAuth)
	if !ok {
		t.Fatalf("credentialFromHelper() = %T, want *http.BasicAuth", auth)
	}
	return got
}

func TestCredentialFromHelperReturnsStoredCredentials(t *testing.T) {
	requireGit(t)
	useStoredCredentials(t, "https://alice:s3cret@git.example.com")

	got := basicAuth(t, credentialFromHelper("https://git.example.com/myorg/myrepo.git"))
	if got.Username != "alice" || got.Password != "s3cret" {
		t.Fatalf("credentialFromHelper() = %q:%q, want %q:%q", got.Username, got.Password, "alice", "s3cret")
	}
}

func TestCredentialFromHelperMatchesHostIncludingPort(t *testing.T) {
	requireGit(t)
	useStoredCredentials(t,
		"https://wrong:wrong@git.example.com",
		"https://alice:s3cret@git.example.com:8443",
	)

	got := basicAuth(t, credentialFromHelper("https://git.example.com:8443/myorg/myrepo.git"))
	if got.Username != "alice" || got.Password != "s3cret" {
		t.Fatalf("credentialFromHelper() = %q:%q, want %q:%q", got.Username, got.Password, "alice", "s3cret")
	}
}

func TestCredentialFromHelperIgnoresOtherProtocol(t *testing.T) {
	requireGit(t)
	useStoredCredentials(t, "http://alice:s3cret@git.example.com")

	if auth := credentialFromHelper("https://git.example.com/myorg/myrepo.git"); auth != nil {
		t.Fatalf("credentialFromHelper() = %v, want nil", auth)
	}
}

func TestCredentialFromHelperReturnsNilForNonHTTPURL(t *testing.T) {
	requireGit(t)
	useStoredCredentials(t, "https://alice:s3cret@git.example.com")

	urls := []string{
		"ssh://git@git.example.com/myorg/myrepo.git",
		"git@git.example.com:myorg/myrepo.git",
		"/local/path/to/repo",
	}
	for _, gitUrl := range urls {
		if auth := credentialFromHelper(gitUrl); auth != nil {
			t.Fatalf("credentialFromHelper(%q) = %v, want nil", gitUrl, auth)
		}
	}
}

// The helper must fail fast rather than block on a terminal prompt when no
// credential is stored for the host, which is what GIT_TERMINAL_PROMPT=0 buys.
func TestCredentialFromHelperDoesNotPromptForUnknownHost(t *testing.T) {
	requireGit(t)
	useStoredCredentials(t)

	done := make(chan transport.AuthMethod, 1)
	go func() {
		done <- credentialFromHelper("https://unknown.example.com/myorg/myrepo.git")
	}()

	select {
	case auth := <-done:
		if auth != nil {
			t.Fatalf("credentialFromHelper() = %v, want nil", auth)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("credentialFromHelper() blocked, want it to fail without prompting")
	}
}

func TestCredentialFromHelperReturnsNilWhenGitIsMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if auth := credentialFromHelper("https://git.example.com/myorg/myrepo.git"); auth != nil {
		t.Fatalf("credentialFromHelper() = %v, want nil", auth)
	}
}

// When a helper answers with a password but no username, git insists on
// filling the username in itself and fails with prompts disabled, so the
// caller never sees a half-populated BasicAuth.
func TestCredentialFromHelperReturnsNilForPasswordWithoutUsername(t *testing.T) {
	requireGit(t)
	if runtime.GOOS == "windows" {
		t.Skip("shell-based credential helper is not portable to windows")
	}
	useGitCredentialHelper(t, "!f() { echo password=tok3n; }; f")

	if auth := credentialFromHelper("https://git.example.com/myorg/myrepo.git"); auth != nil {
		t.Fatalf("credentialFromHelper() = %v, want nil", auth)
	}
}

func TestCredentialFromHelperUsesShellHelperOutput(t *testing.T) {
	requireGit(t)
	if runtime.GOOS == "windows" {
		t.Skip("shell-based credential helper is not portable to windows")
	}
	useGitCredentialHelper(t, "!f() { echo username=bot; echo password=tok3n; }; f")

	got := basicAuth(t, credentialFromHelper("https://git.example.com/myorg/myrepo.git"))
	if got.Username != "bot" || got.Password != "tok3n" {
		t.Fatalf("credentialFromHelper() = %q:%q, want %q:%q", got.Username, got.Password, "bot", "tok3n")
	}
}

// recordCredentialRequest captures what git forwards to the helper for the
// given repository URL.
func recordCredentialRequest(t *testing.T, gitUrl string) string {
	t.Helper()

	request := filepath.Join(t.TempDir(), "request")
	useGitCredentialHelper(t, fmt.Sprintf("!f() { cat > %s; }; f", request))

	credentialFromHelper(gitUrl)

	out, err := os.ReadFile(request)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// The path is sent to git, but git drops it before asking a helper unless
// credential.useHttpPath is set, so credentials match per host by default.
func TestCredentialFromHelperOmitsRepositoryPathByDefault(t *testing.T) {
	requireGit(t)
	if runtime.GOOS == "windows" {
		t.Skip("shell-based credential helper is not portable to windows")
	}

	got := recordCredentialRequest(t, "https://git.example.com/myorg/myrepo.git")

	for _, want := range []string{"protocol=https", "host=git.example.com"} {
		if !strings.Contains(got, want) {
			t.Fatalf("credential request = %q, want it to contain %q", got, want)
		}
	}
	if strings.Contains(got, "path=") {
		t.Fatalf("credential request = %q, want no path", got)
	}
}

func TestCredentialFromHelperSendsRepositoryPathWhenConfigured(t *testing.T) {
	requireGit(t)
	if runtime.GOOS == "windows" {
		t.Skip("shell-based credential helper is not portable to windows")
	}

	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "credential.useHttpPath")
	t.Setenv("GIT_CONFIG_VALUE_0", "true")

	got := recordCredentialRequest(t, "https://git.example.com/myorg/myrepo.git")

	if !strings.Contains(got, "path=myorg/myrepo.git") {
		t.Fatalf("credential request = %q, want it to contain %q", got, "path=myorg/myrepo.git")
	}
}
