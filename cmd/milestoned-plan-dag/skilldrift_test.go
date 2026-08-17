package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/kentra-io/milestoned-plan-dag/internal/validate"
)

// skillPath is the authoring skill this CLI ships. It is the one document
// that instructs an agent which commands to run, so every invocation in it
// must name a subcommand this binary actually dispatches.
const skillPath = "../../skills/plan-author/SKILL.md"

// inlineSpan matches a markdown inline code span. Invocations appear both
// in fenced blocks and inline in prose, and a drift check that reads only
// one of the two misses half the document.
var inlineSpan = regexp.MustCompile("`([^`\n]+)`")

// invocations returns every line of md that starts with the CLI's name,
// looking inside both fenced code blocks and inline code spans.
func invocations(md, bin string) []string {
	var out []string
	add := func(s string) {
		if strings.HasPrefix(strings.TrimSpace(s), bin+" ") {
			out = append(out, strings.TrimSpace(s))
		}
	}

	inFence := false
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			add(line)
			continue
		}
		for _, m := range inlineSpan.FindAllStringSubmatch(line, -1) {
			add(m[1])
		}
	}
	return out
}

func TestSkillInvocationsNameRealCommands(t *testing.T) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("reading %s: %v", skillPath, err)
	}

	found := invocations(string(data), "milestoned-plan-dag")
	if len(found) == 0 {
		t.Fatalf("no milestoned-plan-dag invocations found in %s — the extractor is broken, not the skill", skillPath)
	}

	for _, inv := range found {
		fields := strings.Fields(inv)
		sub := fields[1]
		if strings.HasPrefix(sub, "-") {
			continue // `milestoned-plan-dag --help` and friends
		}
		if _, ok := commands[sub]; !ok {
			t.Errorf("%s: %q names subcommand %q, which this CLI does not dispatch (have %v)",
				skillPath, inv, sub, commandNames())
		}
	}
}

func TestExtractorFindsBothSpanKinds(t *testing.T) {
	md := "prose with `milestoned-plan-dag bogus plan.yaml` inline\n" +
		"```\nmilestoned-plan-dag validate plan.yaml\n```\n"
	got := invocations(md, "milestoned-plan-dag")
	if len(got) != 2 {
		t.Fatalf("invocations() = %v, want 2 (one inline, one fenced)", got)
	}
}

func commandNames() []string {
	names := make([]string, 0, len(commands))
	for n := range commands {
		names = append(names, n)
	}
	return names
}

// TestExampleMeetsTheBar holds the worked example to the standard the skill
// teaches: it is the artifact agents copy, so a stale example is a quality
// regression the prose cannot catch.
func TestExampleMeetsTheBar(t *testing.T) {
	data, err := os.ReadFile("../../skills/plan-author/example.yaml")
	if err != nil {
		t.Fatalf("reading example.yaml: %v", err)
	}

	res, err := validate.Validate(data)
	if err != nil {
		t.Fatalf("example.yaml failed to parse: %v", err)
	}
	if !res.OK() {
		t.Fatalf("example.yaml is invalid: %v", res.Errors)
	}

	var doc struct {
		Milestones []struct {
			Slug     string `yaml:"slug"`
			Contract struct {
				Check    string      `yaml:"check"`
				Criteria interface{} `yaml:"criteria"`
			} `yaml:"contract"`
		} `yaml:"milestones"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parsing example.yaml: %v", err)
	}

	named := 0
	for _, m := range doc.Milestones {
		if list, ok := m.Contract.Criteria.([]interface{}); ok && len(list) > 0 {
			named++
		}
		if m.Contract.Check == "" {
			t.Errorf("milestone %q: empty check", m.Slug)
		}
	}
	if named < 2 {
		t.Errorf("example.yaml demonstrates structured named criteria on %d milestone(s), want at least 2 — "+
			"the example must show the shape the skill asks for", named)
	}
}
