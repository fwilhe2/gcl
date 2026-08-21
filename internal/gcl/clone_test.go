package gcl

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

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

	want := filepath.Join(baseDir, "github.com", "fwilhe2", "gcl")
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

	want := filepath.Join(baseDir, "github.com", "fwilhe2", "gcl")
	if got != want {
		t.Fatalf("clonePathFor() = %q, want %q", got, want)
	}
}

func TestClonePathStripsGitSuffix(t *testing.T) {
	baseDir := t.TempDir()

	got, err := clonePathFor("https://github.com/fwilhe2/gcl.git", baseDir)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(baseDir, "github.com", "fwilhe2", "gcl")
	if got != want {
		t.Fatalf("clonePathFor() = %q, want %q", got, want)
	}
}

func TestClonePathTrimsTrailingSlash(t *testing.T) {
	baseDir := t.TempDir()

	got, err := clonePathFor("https://github.com/fwilhe2/gcl/", baseDir)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(baseDir, "github.com", "fwilhe2", "gcl")
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

type stubForge struct {
	cloneURLs    []string
	listedOwners *[]string
}

func (s *stubForge) ListCloneURLs(owner string) ([]string, error) {
	if s.listedOwners != nil {
		*s.listedOwners = append(*s.listedOwners, owner)
	}
	return s.cloneURLs, nil
}

func TestCloneOwnerClonesAllRepositories(t *testing.T) {
	baseDir := t.TempDir()

	previousForgeForHost := forgeForHost
	previousPlainClone := plainClone
	t.Cleanup(func() {
		forgeForHost = previousForgeForHost
		plainClone = previousPlainClone
	})

	forgeForHost = func(host string) (Forge, bool) {
		if host != "example.com" {
			t.Fatalf("unexpected host: %s", host)
		}
		return &stubForge{cloneURLs: []string{
			"https://example.com/myorg/one.git",
			"https://example.com/myorg/two.git",
		}}, true
	}

	var clonedPaths []string
	plainClone = func(path string, opts *git.CloneOptions) (*git.Repository, error) {
		clonedPaths = append(clonedPaths, path)
		if err := os.MkdirAll(path, 0o750); err != nil {
			t.Fatal(err)
		}
		return nil, nil
	}

	err := CloneWithOptions("https://example.com/myorg", CloneOptions{
		BaseDir:  baseDir,
		Progress: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		filepath.Join(baseDir, "example.com", "myorg", "one"),
		filepath.Join(baseDir, "example.com", "myorg", "two"),
	}
	if len(clonedPaths) != len(want) {
		t.Fatalf("cloned paths = %v, want %v", clonedPaths, want)
	}
	for i := range want {
		if clonedPaths[i] != want[i] {
			t.Fatalf("cloned paths = %v, want %v", clonedPaths, want)
		}
	}
}

func TestCloneAllTreatsMultiSegmentPathAsOwner(t *testing.T) {
	baseDir := t.TempDir()

	previousForgeForHost := forgeForHost
	previousPlainClone := plainClone
	t.Cleanup(func() {
		forgeForHost = previousForgeForHost
		plainClone = previousPlainClone
	})

	var listedOwners []string
	forgeForHost = func(host string) (Forge, bool) {
		return &stubForge{cloneURLs: []string{
			"https://example.com/mygroup/sub/one.git",
		}, listedOwners: &listedOwners}, true
	}

	var clonedPaths []string
	plainClone = func(path string, opts *git.CloneOptions) (*git.Repository, error) {
		clonedPaths = append(clonedPaths, path)
		if err := os.MkdirAll(path, 0o750); err != nil {
			t.Fatal(err)
		}
		return nil, nil
	}

	err := CloneWithOptions("https://example.com/mygroup/sub", CloneOptions{
		BaseDir:  baseDir,
		Progress: io.Discard,
		All:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(listedOwners) != 1 || listedOwners[0] != "mygroup/sub" {
		t.Fatalf("listed owners = %v, want [mygroup/sub]", listedOwners)
	}

	want := []string{filepath.Join(baseDir, "example.com", "mygroup", "sub", "one")}
	if len(clonedPaths) != 1 || clonedPaths[0] != want[0] {
		t.Fatalf("cloned paths = %v, want %v", clonedPaths, want)
	}
}

func TestCloneOwnerUnsupportedHost(t *testing.T) {
	err := CloneWithOptions("https://example.org/myorg", CloneOptions{
		BaseDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("CloneWithOptions() succeeded, want error")
	}
}

func TestCloneRemovesTargetDirectoryAfterFailedClone(t *testing.T) {
	baseDir := t.TempDir()
	cloneErr := errors.New("clone failed")
	previousPlainClone := plainClone
	t.Cleanup(func() {
		plainClone = previousPlainClone
	})
	plainClone = func(path string, opts *git.CloneOptions) (*git.Repository, error) {
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
}

func TestCloneBackupMirrorsIntoGitSuffixedDir(t *testing.T) {
	baseDir := t.TempDir()
	previousPlainClone := plainClone
	t.Cleanup(func() {
		plainClone = previousPlainClone
	})

	var gotPath string
	var gotMirror bool
	plainClone = func(path string, opts *git.CloneOptions) (*git.Repository, error) {
		gotPath = path
		gotMirror = opts.Mirror
		if err := os.MkdirAll(path, 0o750); err != nil {
			t.Fatal(err)
		}
		return nil, nil
	}

	err := CloneWithOptions("https://github.com/fwilhe2/gcl", CloneOptions{
		BaseDir:  baseDir,
		Backup:   true,
		Progress: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(baseDir, "github.com", "fwilhe2", "gcl.git")
	if gotPath != want {
		t.Fatalf("clone path = %q, want %q", gotPath, want)
	}
	if !gotMirror {
		t.Fatal("clone was not a mirror")
	}
}

func TestCloneBackupUpdatesExistingMirror(t *testing.T) {
	baseDir := t.TempDir()
	mirrorPath := filepath.Join(baseDir, "github.com", "fwilhe2", "gcl.git")
	if err := os.MkdirAll(mirrorPath, 0o750); err != nil {
		t.Fatal(err)
	}

	openErr := errors.New("open failed")
	previousPlainOpen := plainOpen
	t.Cleanup(func() {
		plainOpen = previousPlainOpen
	})
	var gotPath string
	plainOpen = func(path string) (*git.Repository, error) {
		gotPath = path
		return nil, openErr
	}

	err := CloneWithOptions("https://github.com/fwilhe2/gcl", CloneOptions{
		BaseDir:  baseDir,
		Backup:   true,
		Progress: io.Discard,
	})
	if !errors.Is(err, openErr) {
		t.Fatalf("CloneWithOptions() error = %v, want %v", err, openErr)
	}
	if gotPath != mirrorPath {
		t.Fatalf("opened %q, want %q", gotPath, mirrorPath)
	}
}
