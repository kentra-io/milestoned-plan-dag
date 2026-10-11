package main

import (
	"regexp"
	"runtime/debug"
	"strings"
	"testing"
)

// planDAGVersionRe is the format callers parse; changing the output breaks them.
var planDAGVersionRe = regexp.MustCompile(`(?i)^milestoned-plan-dag version\s+(.*)$`)

func TestVersionLineFormat(t *testing.T) {
	line := versionLine()

	m := planDAGVersionRe.FindStringSubmatch(line)
	if m == nil {
		t.Fatalf("versionLine() = %q does not match %q", line, planDAGVersionRe)
	}
	if strings.TrimSpace(m[1]) == "" {
		t.Errorf("versionLine() = %q — parses, but the captured version is empty", line)
	}
}

// TestVersionFlagsExitZero pins the dispatch wiring: `--version`, `-v`, and
// the bare `version` word must all print the version and succeed. A missing
// flag here is a defect that made a released binary unusable to version checks.
func TestVersionFlagsExitZero(t *testing.T) {
	for _, arg := range []string{"--version", "-v", "version"} {
		if code := run([]string{arg}); code != 0 {
			t.Errorf("run([%q]) = %d, want 0", arg, code)
		}
	}
}

// TestBuildVersionPrefersLdflags covers the release-build path, where
// GoReleaser injects version/commit/date.
func TestBuildVersionPrefersLdflags(t *testing.T) {
	defer func(v, c, d string) { version, commit, date = v, c, d }(version, commit, date)
	version, commit, date = "1.2.3", "abcdef", "2026-01-01"

	got := buildVersion()
	want := "1.2.3 (abcdef) built 2026-01-01"
	if got != want {
		t.Errorf("buildVersion() = %q, want %q", got, want)
	}
}

// TestBuildVersionFallsBackToBuildInfo covers the `go install` path, where
// no ldflags are injected and the version comes from module build info.
func TestBuildVersionFallsBackToBuildInfo(t *testing.T) {
	defer func(v string) { version = v }(version)
	version = ""

	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.1.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "0123456789abcdef0123"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	got := buildInfoVersion(info)
	want := "v0.1.0 (0123456789ab-dirty)"
	if got != want {
		t.Errorf("buildInfoVersion() = %q, want %q", got, want)
	}
}

// TestBuildVersionUnknownWhenNoBuildInfo covers the defensive branch that is
// unreachable in a normal test binary.
func TestBuildVersionUnknownWhenNoBuildInfo(t *testing.T) {
	defer func(v string, r func() (*debug.BuildInfo, bool)) { version, readBuildInfo = v, r }(version, readBuildInfo)
	version = ""
	readBuildInfo = func() (*debug.BuildInfo, bool) { return nil, false }

	if got := buildVersion(); got != "unknown" {
		t.Errorf("buildVersion() = %q, want %q", got, "unknown")
	}
}
