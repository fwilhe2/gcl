package gcl

import (
	"encoding/base64"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// captureGit records git invocations instead of running them; fake, if
// set, simulates the command's effect.
func captureGit(t *testing.T, fake func(cmd *exec.Cmd)) *[]*exec.Cmd {
	t.Helper()
	setGitAvailable(t, true)
	previous := runGit
	t.Cleanup(func() { runGit = previous })
	var cmds []*exec.Cmd
	runGit = func(cmd *exec.Cmd) error {
		cmds = append(cmds, cmd)
		if fake != nil {
			fake(cmd)
		}
		return nil
	}
	return &cmds
}

func stubCredential(t *testing.T, has bool) {
	t.Helper()
	previous := lookupCredential
	t.Cleanup(func() { lookupCredential = previous })
	lookupCredential = func(string) bool { return has }
}

// gitSubcommand returns the arguments after the leading -c settings.
func gitSubcommand(cmd *exec.Cmd) []string {
	args := cmd.Args[1:]
	for len(args) >= 2 && args[0] == "-c" {
		args = args[2:]
	}
	return args
}

func envValue(env []string, key string) (string, bool) {
	for i := len(env) - 1; i >= 0; i-- {
		if v, ok := strings.CutPrefix(env[i], key+"="); ok {
			return v, true
		}
	}
	return "", false
}

func TestCloneUsesGitCLI(t *testing.T) {
	isolateTokens(t, "")
	stubCredential(t, false)
	cmds := captureGit(t, func(cmd *exec.Cmd) {
		args := gitSubcommand(cmd)
		if err := os.MkdirAll(args[len(args)-1], 0o750); err != nil {
			t.Fatal(err)
		}
	})
	baseDir := t.TempDir()

	err := CloneWithOptions("https://github.com/fwilhe2/gcl", CloneOptions{BaseDir: baseDir, Progress: io.Discard})
	if err != nil {
		t.Fatal(err)
	}

	if len(*cmds) != 1 {
		t.Fatalf("ran %d git commands, want 1", len(*cmds))
	}
	cmd := (*cmds)[0]
	want := []string{
		"clone", "--progress", "--recurse-submodules", "--", "https://github.com/fwilhe2/gcl",
		filepath.Join(baseDir, "github.com", "fwilhe2", "gcl"),
	}
	if got := gitSubcommand(cmd); !slices.Equal(got, want) {
		t.Fatalf("git args = %q, want %q", got, want)
	}
	for _, setting := range []string{"pack.threads=", "core.deltaBaseCacheLimit=", "core.bigFileThreshold="} {
		if !slices.ContainsFunc(cmd.Args, func(a string) bool { return strings.HasPrefix(a, setting) }) {
			t.Errorf("git args %q lack %s", cmd.Args, setting)
		}
	}
	if v, _ := envValue(cmd.Env, "GIT_TERMINAL_PROMPT"); v != "0" {
		t.Errorf("GIT_TERMINAL_PROMPT = %q, want 0", v)
	}
	if cmd.Stdout != io.Discard || cmd.Stderr != io.Discard {
		t.Error("git output is not sent to the progress writer")
	}
}

func TestCloneBackupUsesGitMirror(t *testing.T) {
	isolateTokens(t, "")
	stubCredential(t, false)
	cmds := captureGit(t, nil)
	baseDir := t.TempDir()

	if err := CloneWithOptions("https://github.com/fwilhe2/gcl", CloneOptions{BaseDir: baseDir, Backup: true, Progress: io.Discard}); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"clone", "--progress", "--mirror", "--", "https://github.com/fwilhe2/gcl",
		filepath.Join(baseDir, "github.com", "fwilhe2", "gcl.git"),
	}
	if got := gitSubcommand((*cmds)[0]); !slices.Equal(got, want) {
		t.Fatalf("git args = %q, want %q", got, want)
	}
}

func TestCloneBackupUpdatesExistingMirrorWithGit(t *testing.T) {
	isolateTokens(t, "")
	stubCredential(t, false)
	cmds := captureGit(t, nil)
	baseDir := t.TempDir()
	mirrorPath := filepath.Join(baseDir, "github.com", "fwilhe2", "gcl.git")
	if err := os.MkdirAll(mirrorPath, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := CloneWithOptions("https://github.com/fwilhe2/gcl", CloneOptions{BaseDir: baseDir, Backup: true, Progress: io.Discard}); err != nil {
		t.Fatal(err)
	}

	want := []string{"-C", mirrorPath, "fetch", "--progress", "--prune", "--force", "origin", "+refs/*:refs/*"}
	if got := gitSubcommand((*cmds)[0]); !slices.Equal(got, want) {
		t.Fatalf("git args = %q, want %q", got, want)
	}
}

func TestGitEnvSendsTokenWhenNoCredentialHelperMatches(t *testing.T) {
	isolateTokens(t, `{"github_hosts": {"ghe.example.com": {"api_base": "https://ghe.example.com/api/v3", "token": "ghe-token"}}}`)
	stubCredential(t, false)

	env := gitEnv([]string{"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=user.name", "GIT_CONFIG_VALUE_0=x"},
		"https://ghe.example.com/o/r.git")

	if v, _ := envValue(env, "GIT_CONFIG_COUNT"); v != "2" {
		t.Fatalf("GIT_CONFIG_COUNT = %q, want 2", v)
	}
	if v, _ := envValue(env, "GIT_CONFIG_KEY_0"); v != "user.name" {
		t.Fatalf("existing config entry overwritten: GIT_CONFIG_KEY_0 = %q", v)
	}
	if v, _ := envValue(env, "GIT_CONFIG_KEY_1"); v != "http.https://ghe.example.com/.extraHeader" {
		t.Fatalf("GIT_CONFIG_KEY_1 = %q", v)
	}
	wantHeader := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:ghe-token"))
	if v, _ := envValue(env, "GIT_CONFIG_VALUE_1"); v != wantHeader {
		t.Fatalf("GIT_CONFIG_VALUE_1 = %q, want %q", v, wantHeader)
	}
}

func TestGitEnvLeavesAuthToCredentialHelper(t *testing.T) {
	isolateTokens(t, "")
	t.Setenv("GITHUB_TOKEN", "gh-env")
	stubCredential(t, true)

	env := gitEnv(nil, "https://github.com/o/r.git")
	if _, ok := envValue(env, "GIT_CONFIG_COUNT"); ok {
		t.Fatalf("token header set although the credential helper has a credential: %q", env)
	}
}

func TestGitEnvWithoutTokenOrForSSH(t *testing.T) {
	isolateTokens(t, "")
	stubCredential(t, false)
	if _, ok := envValue(gitEnv(nil, "https://github.com/o/r.git"), "GIT_CONFIG_COUNT"); ok {
		t.Fatal("header set without a token")
	}
	t.Setenv("GITHUB_TOKEN", "gh-env")
	if _, ok := envValue(gitEnv(nil, "git@github.com:o/r.git"), "GIT_CONFIG_COUNT"); ok {
		t.Fatal("header set for an SSH URL")
	}
}

// TestGitCLIClonesLocalRepository runs the real git binary end to end.
func TestGitCLIClonesLocalRepository(t *testing.T) {
	requireGit(t)
	useStoredCredentials(t)
	isolateTokens(t, "")

	source := filepath.Join(t.TempDir(), "source")
	for _, args := range [][]string{
		{"init", "-q", "-b", "main", source},
		{"-C", source, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "init"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	target := filepath.Join(t.TempDir(), "mirror.git")
	if err := gitCloneCLI("file://"+filepath.ToSlash(source), target, true, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := gitUpdateMirrorCLI("file://"+filepath.ToSlash(source), target, io.Discard); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", target, "rev-parse", "refs/heads/main").CombinedOutput(); err != nil {
		t.Fatalf("mirror lacks main: %v\n%s", err, out)
	}
}
