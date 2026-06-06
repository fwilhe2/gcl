package gcl

import (
	"path/filepath"
	"testing"
)

func TestCloneBaseDirDefault(t *testing.T) {
	t.Setenv("GCL_BASE_DIR", "")

	homedir := t.TempDir()
	t.Setenv("HOME", homedir)

	got, err := cloneBaseDir()
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

	got, err := cloneBaseDir()
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

	got, err := cloneBaseDir()
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(homedir, "src")
	if got != want {
		t.Fatalf("cloneBaseDir() = %q, want %q", got, want)
	}
}
