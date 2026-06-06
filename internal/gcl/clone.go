package gcl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v6"
)

var (
	scpLikeURLPattern = regexp.MustCompile(`^(?:[^@]+@)?([^:]+):/?(.+)$`)
	plainClone        = git.PlainCloneContext
	statusInterval    = 30 * time.Second
)

type CloneOptions struct {
	BaseDir        string
	Context        context.Context
	Depth          int
	Progress       io.Writer
	SkipSubmodules bool
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
	start := time.Now()
	clonePath, err := clonePathFor(gitUrl, opts.BaseDir)
	if err != nil {
		return err
	}
	out := newLockedWriter(outputWriter(opts.Progress))

	exists, err := dirExists(clonePath)
	if err != nil {
		return err
	}
	if exists {
		fmt.Fprintf(out, "Directory already exists: %s\n", clonePath)
		return nil
	}

	fmt.Fprintf(out, "Cloning %s\n", gitUrl)
	fmt.Fprintf(out, "Target: %s\n", clonePath)
	if opts.Depth > 0 {
		fmt.Fprintf(out, "Depth: %d\n", opts.Depth)
	}
	if opts.SkipSubmodules {
		fmt.Fprintln(out, "Submodules: skipped")
	}

	ctx := opts.Context
	stop := func() {}
	if ctx == nil {
		ctx, stop = signal.NotifyContext(context.Background(), os.Interrupt)
	}
	defer stop()

	err = os.MkdirAll(filepath.Dir(clonePath), 0o750)
	if err != nil {
		return err
	}

	tempPath, err := os.MkdirTemp(filepath.Dir(clonePath), ".gcl-"+filepath.Base(clonePath)+"-*")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Preparing temporary clone: %s\n", tempPath)

	recurseSubmodules := git.DefaultSubmoduleRecursionDepth
	if opts.SkipSubmodules {
		recurseSubmodules = git.NoRecurseSubmodules
	}

	err = runCloneWithStatus(ctx, out, start, tempPath, &git.CloneOptions{
		URL:               gitUrl,
		Depth:             opts.Depth,
		RecurseSubmodules: recurseSubmodules,
		Progress:          newProgressWriter(out),
	})
	if err != nil {
		fmt.Fprintf(out, "Clone failed after %s; cleaning up temporary directory\n", elapsed(start))
		cleanupErr := os.RemoveAll(tempPath)
		if cleanupErr != nil {
			return errors.Join(err, cleanupErr)
		}
		return err
	}

	err = os.Rename(tempPath, clonePath)
	if err != nil {
		cleanupErr := os.RemoveAll(tempPath)
		if cleanupErr != nil {
			return errors.Join(err, cleanupErr)
		}
		return err
	}

	fmt.Fprintf(out, "Done in %s\n", elapsed(start))

	return nil
}

func runCloneWithStatus(ctx context.Context, out io.Writer, start time.Time, tempPath string, opts *git.CloneOptions) error {
	done := make(chan error, 1)
	go func() {
		_, err := plainClone(ctx, tempPath, opts)
		done <- err
	}()

	var ticker *time.Ticker
	var ticks <-chan time.Time
	if statusInterval > 0 {
		ticker = time.NewTicker(statusInterval)
		defer ticker.Stop()
		ticks = ticker.C
	}

	cancelReported := false
	ctxDone := ctx.Done()
	for {
		select {
		case err := <-done:
			return err
		case <-ticks:
			fmt.Fprintf(out, "Still cloning after %s; large repositories can be quiet while packfiles download\n", elapsed(start))
		case <-ctxDone:
			if !cancelReported {
				fmt.Fprintf(out, "Cancel requested; waiting for clone operation to stop\n")
				cancelReported = true
			}
			ctxDone = nil
		}
	}
}

func outputWriter(progress io.Writer) io.Writer {
	if progress != nil {
		return progress
	}
	return os.Stdout
}

func elapsed(start time.Time) time.Duration {
	return time.Since(start).Round(time.Second)
}

type progressWriter struct {
	out       io.Writer
	lineStart bool
}

func newProgressWriter(out io.Writer) io.Writer {
	return &progressWriter{
		out:       out,
		lineStart: true,
	}
}

func (w *progressWriter) Write(p []byte) (int, error) {
	written := len(p)
	for len(p) > 0 {
		if w.lineStart {
			if _, err := fmt.Fprint(w.out, "remote: "); err != nil {
				return 0, err
			}
			w.lineStart = false
		}

		nextBreak, separator := nextProgressBreak(p)
		if nextBreak == -1 {
			if _, err := w.out.Write(p); err != nil {
				return 0, err
			}
			return written, nil
		}

		if _, err := w.out.Write(p[:nextBreak]); err != nil {
			return 0, err
		}
		if separator == '\r' {
			if _, err := fmt.Fprintln(w.out); err != nil {
				return 0, err
			}
		} else if _, err := w.out.Write([]byte{separator}); err != nil {
			return 0, err
		}
		w.lineStart = true
		p = p[nextBreak+1:]
	}
	return written, nil
}

func nextProgressBreak(p []byte) (int, byte) {
	nextNewline := bytes.IndexByte(p, '\n')
	nextCarriageReturn := bytes.IndexByte(p, '\r')
	switch {
	case nextNewline == -1:
		return nextCarriageReturn, '\r'
	case nextCarriageReturn == -1:
		return nextNewline, '\n'
	case nextCarriageReturn < nextNewline:
		return nextCarriageReturn, '\r'
	default:
		return nextNewline, '\n'
	}
}

type lockedWriter struct {
	mu  sync.Mutex
	out io.Writer
}

func newLockedWriter(out io.Writer) io.Writer {
	return &lockedWriter{out: out}
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.out.Write(p)
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
			repoPath := strings.TrimPrefix(matches[2], "/")
			if err := validateRepoPath(repoPath); err != nil {
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
	repoPath := strings.TrimPrefix(urlComponents.Path, "/")
	if host == "" || repoPath == "" {
		return "", "", fmt.Errorf("unsupported git URL: %s", gitUrl)
	}
	if err := validateRepoPath(repoPath); err != nil {
		return "", "", err
	}

	return host, repoPath, nil
}

func validateRepoPath(repoPath string) error {
	for _, part := range strings.Split(repoPath, "/") {
		if part == "" || part == ".." {
			return fmt.Errorf("unsupported repository path: %s", repoPath)
		}
	}
	return nil
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
