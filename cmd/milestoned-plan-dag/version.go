package main

import (
	"fmt"
	"runtime/debug"
)

// version, commit, and date are injected at build time via -ldflags by
// GoReleaser (see .goreleaser.yaml). They stay at their zero values for
// `go build`/`go install`, in which case buildVersion falls back to
// runtime/debug.ReadBuildInfo so a plain
// `go install .../cmd/milestoned-plan-dag@<ref>` still reports a
// meaningful version (module pseudo-version + VCS revision).
//
// Mirrors cmd/lifecycle/version.go in the spec-lifecycle sibling primitive.
var (
	version = ""
	commit  = ""
	date    = ""
)

// readBuildInfo is swappable in tests to exercise the not-ok branch, which
// is unreachable in a test binary (the toolchain always embeds build info).
var readBuildInfo = debug.ReadBuildInfo

// versionLine is what `--version` prints. The leading "milestoned-plan-dag
// version " prefix is load-bearing, not decoration: spec-lifecycle's
// internal/plandag.Version parses this binary's output with the regexp
// `^milestoned-plan-dag version\s+(.*)$` to run its companion-primitive
// preflight. Changing the prefix breaks that seam.
func versionLine() string {
	return fmt.Sprintf("%s version %s", binName, buildVersion())
}

// buildVersion returns a human-readable version string for `--version`.
// It prefers the ldflags-injected values (release builds); when those are
// absent it derives an equivalent string from the Go module build info
// embedded by the toolchain in every `go build`/`go install` binary.
func buildVersion() string {
	if version != "" {
		v := version
		if commit != "" {
			v += fmt.Sprintf(" (%s)", commit)
		}
		if date != "" {
			v += fmt.Sprintf(" built %s", date)
		}
		return v
	}

	info, ok := readBuildInfo()
	if !ok {
		return "unknown"
	}
	return buildInfoVersion(info)
}

// buildInfoVersion formats a version string from Go module build info:
// the module (pseudo-)version, plus the VCS revision (truncated to 12
// chars, "-dirty" suffix when the working tree was modified) when present.
func buildInfoVersion(info *debug.BuildInfo) string {
	v := info.Main.Version // e.g. "(devel)" or a pseudo-version
	if v == "" {
		v = "unknown"
	}

	var rev string
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}

	if rev != "" {
		if len(rev) > 12 {
			rev = rev[:12]
		}
		v += fmt.Sprintf(" (%s", rev)
		if dirty {
			v += "-dirty"
		}
		v += ")"
	}

	return v
}
