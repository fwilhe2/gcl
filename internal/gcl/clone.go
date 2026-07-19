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
)

var (
	scpLikeURLPattern = regexp.MustCompile(`^(?:[^@]+@)?([^:]+):/?(.+)$`)
	plainClone        = git.PlainClone
)

type CloneOptions struct {
	BaseDir  string
	Progress io.Writer
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
	clonePath, err := clonePathFor(gitUrl, opts.BaseDir)
	if err != nil {
		return err
	}

	exists, err := dirExists(clonePath)
	if err != nil {
		return err
	}
	if exists {
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

	_, err = plainClone(clonePath, &git.CloneOptions{
		URL:               gitUrl,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Progress:          progress,
	})
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
