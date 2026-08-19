package gcl

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v6/plumbing/transport/http"
)

// credentialFromHelper invokes "git credential fill" to retrieve stored
// credentials for the given repository URL. It returns nil if the URL is
// not HTTP(S) or if no credentials could be obtained.
func credentialFromHelper(gitUrl string) *http.BasicAuth {
	u, err := url.Parse(gitUrl)
	if err != nil {
		return nil
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil
	}

	input := fmt.Sprintf("protocol=%s\nhost=%s\npath=%s\n\n", u.Scheme, u.Host, strings.TrimPrefix(u.Path, "/"))

	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader(input)
	// Without this, git falls back to prompting on the terminal when no helper
	// has a matching credential, which would block the clone indefinitely.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var username, password string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if k, v, ok := strings.Cut(line, "="); ok {
			switch k {
			case "username":
				username = v
			case "password":
				password = v
			}
		}
	}

	if username == "" && password == "" {
		return nil
	}

	return &http.BasicAuth{
		Username: username,
		Password: password,
	}
}
