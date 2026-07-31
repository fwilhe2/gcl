package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// Values injected at build time via -ldflags, see .goreleaser.yaml and the
// Makefile. Anything left empty is recovered from the build info that the Go
// toolchain embeds into the binary.
var (
	version   = "dev"
	commit    = ""
	date      = ""
	treeState = ""
)

// versionInfo renders the multi-line output of `gcl --version`.
func versionInfo() string {
	v, c, d, tree := version, commit, date, treeState

	// Only the release pipeline stamps a version via -ldflags, everything else
	// is a development build even if the Go toolchain can derive a version
	// from the git tag it was built at.
	released := !isDevVersion(version)

	if bi, ok := debug.ReadBuildInfo(); ok {
		if isDevVersion(v) && !isDevVersion(bi.Main.Version) {
			v = bi.Main.Version
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if c == "" {
					c = s.Value
				}
			case "vcs.time":
				if d == "" {
					d = s.Value
				}
			case "vcs.modified":
				if tree == "" {
					if s.Value == "true" {
						tree = "dirty"
					} else {
						tree = "clean"
					}
				}
			}
		}
	}

	kind := "release build"
	if !released {
		kind = "development build, not a release"
	}
	if isDevVersion(v) {
		v = "dev"
	}

	source := "unknown"
	switch {
	case c == "":
		// binary was built without any VCS information
	case tree == "dirty":
		source = fmt.Sprintf("%s (dirty git tree)", c)
	case tree == "clean":
		source = fmt.Sprintf("%s (clean git tree)", c)
	default:
		source = fmt.Sprintf("%s (unknown git tree state)", c)
	}

	if d == "" {
		d = "unknown"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "gcl %s (%s)\n", displayVersion(v), kind)
	fmt.Fprintf(&b, "commit: %s\n", source)
	fmt.Fprintf(&b, "built:  %s\n", d)
	fmt.Fprintf(&b, "go:     %s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	return b.String()
}

// isDevVersion reports whether v is one of the placeholders used for builds
// that do not come from a released tag.
func isDevVersion(v string) bool {
	switch v {
	case "", "dev", "(devel)", "unknown":
		return true
	}
	return false
}

// displayVersion prefixes plain semver versions with a "v", since GoReleaser
// strips it from the tag name.
func displayVersion(v string) string {
	if v != "" && v[0] >= '0' && v[0] <= '9' {
		return "v" + v
	}
	return v
}
