package gcl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/client"
)

// gitOutput runs git with args in dir and returns its trimmed stdout.
var gitOutput = func(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	return strings.TrimSpace(string(out)), err
}

// errSkipUpdate marks a working copy that is left alone because it is not
// safe to fast-forward it.
var errSkipUpdate = errors.New("skipped")

// updateWorkingCopy fast-forwards an existing working copy, but only if
// it is clean and has its default branch checked out. Anything else is
// reported and left untouched.
func updateWorkingCopy(gitUrl, clonePath string, opts CloneOptions) error {
	progress := opts.Progress
	if progress == nil {
		progress = os.Stderr
	}

	var err error
	if gitAvailable() {
		err = gitUpdateWorkingCopyCLI(gitUrl, clonePath, progress)
	} else {
		err = goGitUpdateWorkingCopy(gitUrl, clonePath, progress)
	}
	if errors.Is(err, errSkipUpdate) {
		fmt.Fprintf(os.Stderr, "Not updating %s: %v\n", clonePath, err)
		fmt.Println(clonePath)
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Println(clonePath)
	return nil
}

func gitUpdateWorkingCopyCLI(gitUrl, clonePath string, progress io.Writer) error {
	status, err := gitOutput(clonePath, "status", "--porcelain")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("%w: working tree has local changes", errSkipUpdate)
	}
	branch, err := gitOutput(clonePath, "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil || branch == "" {
		return fmt.Errorf("%w: HEAD is detached", errSkipUpdate)
	}
	if def := gitDefaultBranch(clonePath); branch != def {
		return fmt.Errorf("%w: on branch %s, not %s", errSkipUpdate, branch, def)
	}

	fmt.Fprintf(os.Stderr, "Updating %s\n", clonePath)
	return runGit(gitCommand(gitUrl, progress,
		"-C", clonePath, "pull", "--ff-only", "--progress", "--recurse-submodules"))
}

// gitDefaultBranch returns the branch origin/HEAD points to, falling back
// to main or master.
func gitDefaultBranch(clonePath string) string {
	if ref, err := gitOutput(clonePath, "symbolic-ref", "--short", "-q", "refs/remotes/origin/HEAD"); err == nil && ref != "" {
		return strings.TrimPrefix(ref, "origin/")
	}
	if _, err := gitOutput(clonePath, "rev-parse", "--verify", "-q", "refs/remotes/origin/main"); err == nil {
		return "main"
	}
	return "master"
}

func goGitUpdateWorkingCopy(gitUrl, clonePath string, progress io.Writer) error {
	repo, err := plainOpen(clonePath)
	if err != nil {
		return err
	}
	head, err := repo.Head()
	if err != nil {
		return err
	}
	if !head.Name().IsBranch() {
		return fmt.Errorf("%w: HEAD is detached", errSkipUpdate)
	}
	branch := head.Name().Short()
	if def := goGitDefaultBranch(repo); branch != def {
		return fmt.Errorf("%w: on branch %s, not %s", errSkipUpdate, branch, def)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	status, err := wt.Status()
	if err != nil {
		return err
	}
	if !status.IsClean() {
		return fmt.Errorf("%w: working tree has local changes", errSkipUpdate)
	}

	fmt.Fprintf(os.Stderr, "Updating %s\n", clonePath)
	pullOptions := &git.PullOptions{
		RemoteName:        "origin",
		ReferenceName:     head.Name(),
		SingleBranch:      true,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Progress:          progress,
	}
	if auth := httpAuthFor(gitUrl); auth != nil {
		pullOptions.ClientOptions = append(pullOptions.ClientOptions, client.WithHTTPAuth(auth))
	}
	err = wt.Pull(pullOptions)
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}
	return nil
}

func goGitDefaultBranch(repo *git.Repository) string {
	if ref, err := repo.Reference(plumbing.NewRemoteHEADReferenceName("origin"), false); err == nil && ref.Type() == plumbing.SymbolicReference {
		return strings.TrimPrefix(ref.Target().Short(), "origin/")
	}
	if _, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", "main"), false); err == nil {
		return "main"
	}
	return "master"
}
