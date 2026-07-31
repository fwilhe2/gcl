package cmd

import (
	"strings"
	"testing"
)

func withBuildVars(t *testing.T, v, c, d, tree string) {
	t.Helper()
	oldV, oldC, oldD, oldTree := version, commit, date, treeState
	version, commit, date, treeState = v, c, d, tree
	t.Cleanup(func() {
		version, commit, date, treeState = oldV, oldC, oldD, oldTree
	})
}

func TestVersionInfoRelease(t *testing.T) {
	withBuildVars(t, "1.2.3", "abc123", "2026-07-31T16:00:00Z", "clean")

	got := versionInfo()
	for _, want := range []string{"gcl v1.2.3", "release build", "abc123 (clean git tree)", "2026-07-31T16:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("version info %q does not contain %q", got, want)
		}
	}
	if strings.Contains(got, "development build") {
		t.Errorf("release build reported as development build: %q", got)
	}
}

func TestVersionInfoDirtyRelease(t *testing.T) {
	withBuildVars(t, "1.2.3", "abc123", "2026-07-31T16:00:00Z", "dirty")

	if got := versionInfo(); !strings.Contains(got, "abc123 (dirty git tree)") {
		t.Errorf("version info %q does not report a dirty git tree", got)
	}
}

func TestVersionInfoDevelopment(t *testing.T) {
	withBuildVars(t, "dev", "", "", "")

	got := versionInfo()
	if !strings.Contains(got, "development build, not a release") {
		t.Errorf("version info %q is not marked as a development build", got)
	}
	// The build info of the test binary may or may not carry VCS data, so only
	// assert that the version itself is not presented as a release.
	if strings.Contains(got, "(release build)") {
		t.Errorf("development build reported as release: %q", got)
	}
}
