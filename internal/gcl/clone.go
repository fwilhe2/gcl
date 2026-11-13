package gcl

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/client"
)

var (
	scpLikeURLPattern = regexp.MustCompile(`^(?:[^@]+@)?([^:]+):/?(.+)$`)
	plainClone        = git.PlainClone
	plainOpen         = git.PlainOpen
)

type CloneOptions struct {
	BaseDir  string
	Progress io.Writer
	// All treats the URL as an owner (organization, user, or group)
	// and clones all of its repositories, even if the URL has more
	// than one path segment (e.g. a GitLab subgroup).
	All bool
	// Backup creates a bare mirror clone (all branches, tags and other
	// refs) in <repo>.git instead of a working copy. Running it again on
	// an existing mirror updates it instead of skipping it.
	Backup bool
	// Update fast-forwards existing working copies that are clean and
	// have their default branch checked out.
	Update bool
}

func dirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

func Clone(gitUrl string) error {
	return CloneWithOptions(gitUrl, CloneOptions{})
}

func CloneWithOptions(gitUrl string, opts CloneOptions) error {
	host, repoPath, err := parseCloneURL(gitUrl)
	if err != nil {
		return err
	}

	if opts.All || !strings.Contains(repoPath, "/") {
		return cloneOwner(host, repoPath, opts)
	}

	clonePath, err := clonePathFor(gitUrl, opts.BaseDir)
	if err != nil {
		return err
	}
	if opts.Backup {
		clonePath += ".git"
	}

	exists, err := dirExists(clonePath)
	if err != nil {
		return err
	}
	if exists {
		if opts.Backup {
			return updateMirror(gitUrl, clonePath, opts)
		}
		if opts.Update {
			return updateWorkingCopy(gitUrl, clonePath, opts)
		}
		fmt.Fprintf(os.Stderr, "Directory already exists: %s\n", clonePath)
		fmt.Println(clonePath)
		return nil
	}

	fmt.Fprintf(os.Stderr, "Cloning into %s\n", clonePath)

	err = os.MkdirAll(filepath.Dir(clonePath), 0o750)
	if err != nil {
		return err
	}

	progress := opts.Progress
	if progress == nil {
		progress = os.Stderr
	}

	if gitAvailable() {
		err = gitCloneCLI(gitUrl, clonePath, opts.Backup, progress)
	} else {
		err = goGitClone(gitUrl, clonePath, opts.Backup, progress)
	}
	if err != nil {
		cleanupErr := os.RemoveAll(clonePath)
		if cleanupErr != nil {
			return errors.Join(err, cleanupErr)
		}
		return err
	}

	fmt.Println(clonePath)
	return nil
}

// goGitClone clones in-process with go-git, the fallback when git is not
// installed. go-git's memory use grows with the size of the repository,
// so it runs under a memory guard that stops gcl, rather than letting the
// system run out of memory, once the budget is exhausted.
func goGitClone(gitUrl, clonePath string, mirror bool, progress io.Writer) error {
	cloneOptions := &git.CloneOptions{
		URL:               gitUrl,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Progress:          progress,
	}
	objectsDir := filepath.Join(clonePath, git.GitDirName, "objects")
	if mirror {
		cloneOptions.Mirror = true
		cloneOptions.Tags = plumbing.AllTags
		cloneOptions.RecurseSubmodules = git.NoRecurseSubmodules
		objectsDir = filepath.Join(clonePath, "objects")
	}
	if auth := httpAuthFor(gitUrl); auth != nil {
		cloneOptions.ClientOptions = append(cloneOptions.ClientOptions, client.WithHTTPAuth(auth))
	}

	budget := memoryBudget()
	stopGuard := guardMemory(budget, abortOnMemoryLimit("clone of "+gitUrl, budget, func() {
		_ = os.RemoveAll(clonePath)
	}))
	defer stopGuard()
	stopProgress := reportProgress(objectsDir, progress)
	defer stopProgress()

	_, err := plainClone(clonePath, cloneOptions)
	return err
}

// updateMirror refreshes an existing backup mirror: it fetches every ref
// from the remote, overwriting and pruning local ones so the mirror stays
// an exact copy.
func updateMirror(gitUrl, clonePath string, opts CloneOptions) error {
	fmt.Fprintf(os.Stderr, "Updating mirror %s\n", clonePath)

	progress := opts.Progress
	if progress == nil {
		progress = os.Stderr
	}

	var err error
	if gitAvailable() {
		err = gitUpdateMirrorCLI(gitUrl, clonePath, progress)
	} else {
		err = goGitUpdateMirror(gitUrl, clonePath, progress)
	}
	if err != nil {
		return err
	}

	fmt.Println(clonePath)
	return nil
}

func goGitUpdateMirror(gitUrl, clonePath string, progress io.Writer) error {
	repo, err := plainOpen(clonePath)
	if err != nil {
		return err
	}

	fetchOptions := &git.FetchOptions{
		RefSpecs: []config.RefSpec{"+refs/*:refs/*"},
		Tags:     plumbing.AllTags,
		Force:    true,
		Prune:    true,
		Progress: progress,
	}
	if auth := httpAuthFor(gitUrl); auth != nil {
		fetchOptions.ClientOptions = append(fetchOptions.ClientOptions, client.WithHTTPAuth(auth))
	}

	budget := memoryBudget()
	// The existing mirror stays; a partial fetch leaves at most a
	// temporary pack behind.
	stopGuard := guardMemory(budget, abortOnMemoryLimit("update of "+clonePath, budget, nil))
	defer stopGuard()
	stopProgress := reportProgress(filepath.Join(clonePath, "objects"), progress)
	defer stopProgress()

	err = repo.Fetch(fetchOptions)
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}
	return nil
}

func cloneOwner(host, owner string, opts CloneOptions) error {
	forge, ok := forgeForHost(host)
	if !ok {
		return fmt.Errorf("cloning all repositories of an owner is not supported for host %s", host)
	}

	cloneURLs, err := forge.ListCloneURLs(owner)
	if err != nil {
		return err
	}
	if len(cloneURLs) == 0 {
		return fmt.Errorf("no repositories found for %s on %s", owner, host)
	}

	fmt.Fprintf(os.Stderr, "Found %d repositories for %s\n", len(cloneURLs), owner)

	repoOpts := opts
	repoOpts.All = false

	var errs []error
	for _, cloneURL := range cloneURLs {
		if err := CloneWithOptions(cloneURL, repoOpts); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to clone %s: %v\n", cloneURL, err)
			errs = append(errs, fmt.Errorf("%s: %w", cloneURL, err))
		}
	}
	return errors.Join(errs...)
}

func clonePathFor(gitUrl string, baseDirOverride string) (string, error) {
	baseDir, err := cloneBaseDir(baseDirOverride)
	if err != nil {
		return "", err
	}

	host, repoPath, err := parseCloneURL(gitUrl)
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, host, filepath.FromSlash(repoPath)), nil
}

func parseCloneURL(gitUrl string) (string, string, error) {
	if !strings.Contains(gitUrl, "://") {
		if matches := scpLikeURLPattern.FindStringSubmatch(gitUrl); matches != nil && !strings.Contains(matches[1], "/") {
			repoPath, err := normalizeRepoPath(matches[2])
			if err != nil {
				return "", "", err
			}
			return matches[1], repoPath, nil
		}
	}

	urlComponents, err := url.Parse(gitUrl)
	if err != nil {
		return "", "", err
	}

	host := urlComponents.Hostname()
	if host == "" {
		host = urlComponents.Host
	}
	if host == "" || strings.Trim(urlComponents.Path, "/") == "" {
		return "", "", fmt.Errorf("unsupported git URL: %s", gitUrl)
	}
	repoPath, err := normalizeRepoPath(urlComponents.Path)
	if err != nil {
		return "", "", err
	}

	return host, repoPath, nil
}

func normalizeRepoPath(repoPath string) (string, error) {
	normalized := strings.Trim(repoPath, "/")
	normalized = strings.TrimSuffix(normalized, ".git")
	if normalized == "" {
		return "", fmt.Errorf("unsupported repository path: %s", repoPath)
	}
	for part := range strings.SplitSeq(normalized, "/") {
		if part == "" || part == ".." {
			return "", fmt.Errorf("unsupported repository path: %s", repoPath)
		}
	}
	return normalized, nil
}

func cloneBaseDir(baseDir string) (string, error) {
	if baseDir == "" {
		baseDir = os.Getenv("GCL_BASE_DIR")
	}
	if baseDir == "" {
		baseDir = loadFileConfig().BaseDir
	}
	if baseDir == "" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homedir, "code"), nil
	}

	return expandHome(baseDir)
}

func expandHome(dir string) (string, error) {
	if dir != "~" && !strings.HasPrefix(dir, "~/") {
		return dir, nil
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if dir == "~" {
		return homedir, nil
	}
	return filepath.Join(homedir, strings.TrimPrefix(dir, "~/")), nil
}
