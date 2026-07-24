package plan

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load parses a plan from YAML bytes into the typed model.
//
// A leading `# yaml-language-server: $schema=...` editor directive is a YAML
// comment and is tolerated transparently by the parser. After parsing, any
// milestone with no explicit slug gets a kebab-slug derived from its goal
// (design D5), so the returned model always carries a slug.
func Load(data []byte) (*Plan, error) {
	var p Plan
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse plan yaml: %w", err)
	}
	for i := range p.Milestones {
		if p.Milestones[i].Slug == "" {
			p.Milestones[i].Slug = DeriveSlug(p.Milestones[i].Goal)
		}
	}
	return &p, nil
}

// LoadFile reads and parses a plan file.
func LoadFile(path string) (*Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Load(data)
}

// DeriveSlug produces a stable, kebab-case slug from a milestone's goal: it
// lowercases, replaces every run of non-alphanumeric characters with a single
// hyphen, and trims leading/trailing hyphens.
func DeriveSlug(goal string) string {
	var b strings.Builder
	pendingHyphen := false
	for _, r := range strings.ToLower(goal) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingHyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingHyphen = false
			b.WriteRune(r)
		} else {
			pendingHyphen = true
		}
	}
	return b.String()
}
