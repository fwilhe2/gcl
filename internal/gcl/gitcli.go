package gcl

import (
	"encoding/base64"
	"io"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
)

var (
	// gitAvailable reports whether a working git binary is installed.
	// Running it (rather than only looking it up in PATH) also rules out
	// stubs such as macOS's /usr/bin/git without the developer tools.
	gitAvailable = sync.OnceValue(func() bool {
		return exec.Command("git", "--version").Run() == nil
	})
	runGit           = func(cmd *exec.Cmd) error { return cmd.Run() }
	lookupCredential = func(gitUrl string) bool { return credentialFromHelper(gitUrl) != nil }
)

// gitCloneCLI clones with the git binary. Unlike go-git, git resolves
// deltas with bounded memory, which gitMemoryConfig scales to the
// available memory, and reports progress while receiving objects and
// resolving deltas.
func gitCloneCLI(gitUrl, clonePath string, mirror bool, progress io.Writer) error {
	args := []string{"clone", "--progress"}
	if mirror {
		args = append(args, "--mirror")
	} else {
		args = append(args, "--recurse-submodules")
	}
	args = append(args, "--", gitUrl, clonePath)
	return runGit(gitCommand(gitUrl, progress, args...))
}

// gitUpdateMirrorCLI fetches every ref into an existing mirror, overwriting
// and pruning local refs so it stays an exact copy of the remote.
func gitUpdateMirrorCLI(gitUrl, clonePath string, progress io.Writer) error {
	return runGit(gitCommand(gitUrl, progress,
		"-C", clonePath, "fetch", "--progress", "--prune", "--force", "origin", "+refs/*:refs/*"))
}

func gitCommand(gitUrl string, progress io.Writer, args ...string) *exec.Cmd {
	var gitArgs []string
	for _, setting := range gitMemoryConfig(memoryBudget()) {
		gitArgs = append(gitArgs, "-c", setting)
	}
	gitArgs = append(gitArgs, args...)

	cmd := exec.Command("git", gitArgs...)
	// git's own output goes to stderr too: stdout is reserved for the
	// clone path.
	cmd.Stdout = progress
	cmd.Stderr = progress
	cmd.Env = gitEnv(os.Environ(), gitUrl)
	return cmd
}

// gitEnv prepares the environment for git: it never prompts for
// credentials on the terminal (a prompt would stall a run over many
// repositories), and if no credential helper knows the URL but a forge
// token is configured for its host, the token is sent as an HTTP header
// scoped to that host. The token goes through the environment rather than
// the command line, where other users could see it.
func gitEnv(environ []string, gitUrl string) []string {
	env := append(slices.Clone(environ), "GIT_TERMINAL_PROMPT=0")

	username, token := tokenForURL(gitUrl)
	if token == "" || lookupCredential(gitUrl) {
		return env
	}
	u, err := url.Parse(gitUrl)
	if err != nil {
		return env
	}

	key := "http." + u.Scheme + "://" + u.Host + "/.extraHeader"
	value := "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+token))
	return appendGitConfigEnv(env, key, value)
}

// appendGitConfigEnv adds a config entry via GIT_CONFIG_COUNT and
// GIT_CONFIG_KEY_<n>/GIT_CONFIG_VALUE_<n>, keeping entries the
// environment already carries.
func appendGitConfigEnv(env []string, key, value string) []string {
	count := 0
	env = slices.DeleteFunc(env, func(e string) bool {
		v, ok := strings.CutPrefix(e, "GIT_CONFIG_COUNT=")
		if ok {
			count, _ = strconv.Atoi(v)
		}
		return ok
	})
	n := strconv.Itoa(count)
	return append(env,
		"GIT_CONFIG_KEY_"+n+"="+key,
		"GIT_CONFIG_VALUE_"+n+"="+value,
		"GIT_CONFIG_COUNT="+strconv.Itoa(count+1),
	)
}
