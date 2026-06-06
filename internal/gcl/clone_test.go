package gcl

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v6"
)

func TestCloneBaseDirDefault(t *testing.T) {
	t.Setenv("GCL_BASE_DIR", "")

	homedir := t.TempDir()
	t.Setenv("HOME", homedir)

	got, err := cloneBaseDir("")
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(homedir, "code")
	if got != want {
		t.Fatalf("cloneBaseDir() = %q, want %q", got, want)
	}
}

func TestCloneBaseDirFromEnv(t *testing.T) {
	want := t.TempDir()
	t.Setenv("GCL_BASE_DIR", want)

	got, err := cloneBaseDir("")
	if err != nil {
		t.Fatal(err)
	}

	if got != want {
		t.Fatalf("cloneBaseDir() = %q, want %q", got, want)
	}
}

func TestCloneBaseDirExpandsHome(t *testing.T) {
	homedir := t.TempDir()
	t.Setenv("HOME", homedir)
	t.Setenv("GCL_BASE_DIR", "~/src")

	got, err := cloneBaseDir("")
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(homedir, "src")
	if got != want {
		t.Fatalf("cloneBaseDir() = %q, want %q", got, want)
	}
}

func TestCloneBaseDirOverrideWinsOverEnv(t *testing.T) {
	t.Setenv("GCL_BASE_DIR", t.TempDir())

	want := t.TempDir()
	got, err := cloneBaseDir(want)
	if err != nil {
		t.Fatal(err)
	}

	if got != want {
		t.Fatalf("cloneBaseDir() = %q, want %q", got, want)
	}
}

func TestClonePathForHTTPSURL(t *testing.T) {
	baseDir := t.TempDir()

	got, err := clonePathFor("https://github.com/fwilhe2/gcl", baseDir)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(baseDir, "github.com", "fwilhe2", "gcl")
	if got != want {
		t.Fatalf("clonePathFor() = %q, want %q", got, want)
	}
}

func TestClonePathForSSHURL(t *testing.T) {
	baseDir := t.TempDir()

	got, err := clonePathFor("ssh://git@github.com/fwilhe2/gcl.git", baseDir)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(baseDir, "github.com", "fwilhe2", "gcl.git")
	if got != want {
		t.Fatalf("clonePathFor() = %q, want %q", got, want)
	}
}

func TestClonePathForScpLikeURL(t *testing.T) {
	baseDir := t.TempDir()

	got, err := clonePathFor("git@github.com:fwilhe2/gcl.git", baseDir)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(baseDir, "github.com", "fwilhe2", "gcl.git")
	if got != want {
		t.Fatalf("clonePathFor() = %q, want %q", got, want)
	}
}

func TestClonePathRejectsUnsupportedURL(t *testing.T) {
	_, err := clonePathFor("not-a-git-url", t.TempDir())
	if err == nil {
		t.Fatal("clonePathFor() succeeded, want error")
	}
}

func TestClonePathRejectsRepoPathTraversal(t *testing.T) {
	_, err := clonePathFor("https://github.com/fwilhe2/../gcl", t.TempDir())
	if err == nil {
		t.Fatal("clonePathFor() succeeded, want error")
	}
}

func TestCloneRemovesTargetDirectoryAfterFailedClone(t *testing.T) {
	baseDir := t.TempDir()
	cloneErr := errors.New("clone failed")
	var tempPath string
	previousPlainClone := plainClone
	t.Cleanup(func() {
		plainClone = previousPlainClone
	})
	plainClone = func(ctx context.Context, path string, opts *git.CloneOptions) (*git.Repository, error) {
		tempPath = path
		if err := os.MkdirAll(path, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "partial"), []byte("partial"), 0o600); err != nil {
			t.Fatal(err)
		}
		return nil, cloneErr
	}

	err := CloneWithOptions("https://github.com/fwilhe2/gcl", CloneOptions{
		BaseDir:  baseDir,
		Progress: io.Discard,
	})
	if !errors.Is(err, cloneErr) {
		t.Fatalf("CloneWithOptions() error = %v, want %v", err, cloneErr)
	}

	clonePath := filepath.Join(baseDir, "github.com", "fwilhe2", "gcl")
	if _, err := os.Stat(clonePath); !os.IsNotExist(err) {
		t.Fatalf("clone path still exists after failure: %v", err)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("temporary clone path still exists after failure: %v", err)
	}
}

func TestCloneMovesTemporaryDirectoryIntoPlaceAfterSuccess(t *testing.T) {
	baseDir := t.TempDir()
	previousPlainClone := plainClone
	t.Cleanup(func() {
		plainClone = previousPlainClone
	})
	plainClone = func(ctx context.Context, path string, opts *git.CloneOptions) (*git.Repository, error) {
		if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("ok"), 0o600); err != nil {
			t.Fatal(err)
		}
		return nil, nil
	}

	err := CloneWithOptions("https://github.com/fwilhe2/gcl", CloneOptions{
		BaseDir:  baseDir,
		Progress: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}

	clonePath := filepath.Join(baseDir, "github.com", "fwilhe2", "gcl")
	got, err := os.ReadFile(filepath.Join(clonePath, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ok" {
		t.Fatalf("README.md = %q, want %q", got, "ok")
	}
}

func TestClonePassesLargeRepoOptionsToGoGit(t *testing.T) {
	baseDir := t.TempDir()
	previousPlainClone := plainClone
	t.Cleanup(func() {
		plainClone = previousPlainClone
	})

	var gotOptions git.CloneOptions
	var output bytes.Buffer
	plainClone = func(ctx context.Context, path string, opts *git.CloneOptions) (*git.Repository, error) {
		gotOptions = *opts
		_, err := opts.Progress.Write([]byte("counting objects: 42\rreceiving objects: 100%\n"))
		return nil, err
	}

	err := CloneWithOptions("https://github.com/fwilhe2/gcl", CloneOptions{
		BaseDir:        baseDir,
		Depth:          1,
		Progress:       &output,
		SkipSubmodules: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if gotOptions.Depth != 1 {
		t.Fatalf("Depth = %d, want 1", gotOptions.Depth)
	}
	if gotOptions.RecurseSubmodules != git.NoRecurseSubmodules {
		t.Fatalf("RecurseSubmodules = %v, want %v", gotOptions.RecurseSubmodules, git.NoRecurseSubmodules)
	}

	for _, want := range []string{
		"Cloning https://github.com/fwilhe2/gcl",
		"Target:",
		"Depth: 1",
		"Submodules: skipped",
		"remote: counting objects: 42",
		"remote: receiving objects: 100%",
		"Done in",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("output does not contain %q:\n%s", want, output.String())
		}
	}
}

func TestCloneReportsStatusDuringQuietLongClone(t *testing.T) {
	baseDir := t.TempDir()
	previousPlainClone := plainClone
	previousStatusInterval := statusInterval
	t.Cleanup(func() {
		plainClone = previousPlainClone
		statusInterval = previousStatusInterval
	})

	statusInterval = time.Millisecond
	var output bytes.Buffer
	plainClone = func(ctx context.Context, path string, opts *git.CloneOptions) (*git.Repository, error) {
		time.Sleep(5 * time.Millisecond)
		if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("ok"), 0o600); err != nil {
			t.Fatal(err)
		}
		return nil, nil
	}

	err := CloneWithOptions("https://github.com/fwilhe2/gcl", CloneOptions{
		BaseDir:  baseDir,
		Progress: &output,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output.String(), "Still cloning after") {
		t.Fatalf("output does not contain quiet-clone status:\n%s", output.String())
	}
}

func TestCloneCleansUpAfterContextCancellation(t *testing.T) {
	baseDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	var tempPath string
	previousPlainClone := plainClone
	t.Cleanup(func() {
		plainClone = previousPlainClone
	})
	plainClone = func(ctx context.Context, path string, opts *git.CloneOptions) (*git.Repository, error) {
		tempPath = path
		if err := os.WriteFile(filepath.Join(path, "partial"), []byte("partial"), 0o600); err != nil {
			t.Fatal(err)
		}
		cancel()
		return nil, ctx.Err()
	}

	err := CloneWithOptions("https://github.com/fwilhe2/gcl", CloneOptions{
		BaseDir:  baseDir,
		Context:  ctx,
		Progress: io.Discard,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("CloneWithOptions() error = %v, want %v", err, context.Canceled)
	}

	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("temporary clone path still exists after cancellation: %v", err)
	}
}
