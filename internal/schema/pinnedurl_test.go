package schema

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The schema identifies itself, and every document that points an editor at
// it, with one tag-pinned raw GitHub URL. Pinning is the point: an editor
// validating against `main` can go green on a plan the installed binary
// rejects, because the binary carries the schema as it stood at its own
// release. These tests keep the pin honest — every mention agrees on one tag —
// so cutting a release is a single find-and-replace, and a half-done bump
// fails CI rather than shipping.
//
// The tag itself is bumped to the version about to be released *before*
// tagging (docs/releasing.md); .github/workflows/release.yml refuses to
// publish a tag that disagrees with the pin.
const repoRoot = "../.."

var pinnedSchemaURL = regexp.MustCompile(
	`https://raw\.githubusercontent\.com/kentra-io/milestoned-plan-dag/([^/]+)/schema/plan\.schema\.json`)

// The published schema's $id is a tag-pinned raw GitHub URL.
func TestSchemaIDIsThePinnedPublishedURL(t *testing.T) {
	var doc struct {
		ID string `json:"$id"`
	}
	if err := json.Unmarshal(PlanSchemaJSON(), &doc); err != nil {
		t.Fatalf("parse embedded schema: %v", err)
	}
	m := pinnedSchemaURL.FindStringSubmatch(doc.ID)
	if m == nil || m[0] != doc.ID {
		t.Fatalf("$id = %q, want a tag-pinned %s URL", doc.ID, pinnedSchemaURL)
	}
	if !regexp.MustCompile(`^v\d+\.\d+\.\d+$`).MatchString(m[1]) {
		t.Errorf("$id is pinned to %q, want a release tag like v0.1.0", m[1])
	}
}

// Every pinned schema URL in the repository names the same tag.
func TestPinnedSchemaURLsAgree(t *testing.T) {
	refs := map[string][]string{} // tag -> files mentioning it

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "dist" {
				return fs.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".json", ".md", ".yaml", ".yml", ".go":
		default:
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range pinnedSchemaURL.FindAllStringSubmatch(string(b), -1) {
			rel, _ := filepath.Rel(repoRoot, path)
			refs[m[1]] = append(refs[m[1]], rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo: %v", err)
	}

	// "main" is the deliberate unpinned fallback in schema.go; it is not a
	// competing pin. Everything else must agree on one tag.
	delete(refs, "main")

	if len(refs) == 0 {
		t.Fatal("no pinned schema URLs found; the pin guard is not guarding anything")
	}
	if len(refs) > 1 {
		var tags []string
		for tag, files := range refs {
			sort.Strings(files)
			tags = append(tags, tag+" ("+strings.Join(dedupe(files), ", ")+")")
		}
		sort.Strings(tags)
		t.Fatalf("pinned schema URLs disagree on the tag:\n  %s", strings.Join(tags, "\n  "))
	}
}

func dedupe(in []string) []string {
	out := in[:0:0]
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}
