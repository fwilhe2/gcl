package gcl

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestMemAvailable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "meminfo")
	writeFile(t, path, "MemTotal:       16069872 kB\nMemFree:          481412 kB\nMemAvailable:    7596032 kB\n")
	if got, want := memAvailable(path), uint64(7596032*1024); got != want {
		t.Fatalf("memAvailable() = %d, want %d", got, want)
	}
	if got := memAvailable(filepath.Join(t.TempDir(), "missing")); got != 0 {
		t.Fatalf("memAvailable(missing) = %d, want 0", got)
	}
}

func TestCgroupHeadroomTakesTightestAncestor(t *testing.T) {
	dir := t.TempDir()
	procCgroup := filepath.Join(dir, "cgroup")
	root := filepath.Join(dir, "fs")
	writeFile(t, procCgroup, "0::/user.slice/app.scope\n")
	// The scope itself is unlimited, its parent slice is not.
	writeFile(t, filepath.Join(root, "user.slice", "app.scope", "memory.max"), "max\n")
	writeFile(t, filepath.Join(root, "user.slice", "app.scope", "memory.current"), "100\n")
	writeFile(t, filepath.Join(root, "user.slice", "memory.max"), "1000\n")
	writeFile(t, filepath.Join(root, "user.slice", "memory.current"), "400\n")

	got, ok := cgroupHeadroom(procCgroup, root)
	if !ok || got != 600 {
		t.Fatalf("cgroupHeadroom() = %d, %v, want 600, true", got, ok)
	}
}

func TestCgroupHeadroomWithoutLimit(t *testing.T) {
	dir := t.TempDir()
	procCgroup := filepath.Join(dir, "cgroup")
	writeFile(t, procCgroup, "0::/\n")
	writeFile(t, filepath.Join(dir, "fs", "memory.max"), "max\n")
	if _, ok := cgroupHeadroom(procCgroup, filepath.Join(dir, "fs")); ok {
		t.Fatal("cgroupHeadroom() reported a limit")
	}
}

func TestSystemAvailableMemoryIsPositive(t *testing.T) {
	if systemAvailableMemory() == 0 {
		t.Fatal("systemAvailableMemory() = 0")
	}
}
