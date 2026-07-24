package main

import (
	"flag"
	"fmt"
	"os"

	"milestoned-plan-dag/internal/plan"
	"milestoned-plan-dag/internal/resolve"
	"milestoned-plan-dag/internal/validate"
)

// resolveCmd runs `resolve <plan.yaml>`: it validates the plan, then emits
// the machine-readable YAML projection (edges + topological order) to
// stdout. YAML is the sole projection format — resolve defines no flags at
// all, so any flag (e.g. --format json) is unknown and rejected with a
// non-zero exit (plan-projection: "there is no --format json flag").
func resolveCmd(args []string) int {
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "usage: milestoned-plan-dag resolve <plan.yaml>")
		return 2
	}
	path := rest[0]

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve: %v\n", err)
		return 1
	}

	// resolve only ever runs on a validated plan (plan-projection: "resolve
	// reads a validated plan").
	res, _ := validate.Validate(data)
	if !res.OK() {
		for _, e := range res.Errors {
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		fmt.Fprintf(os.Stderr, "resolve: %s is invalid (%d error(s))\n", path, len(res.Errors))
		return 1
	}

	p, err := plan.Load(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve: %v\n", err)
		return 1
	}

	proj, err := resolve.Resolve(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve: %v\n", err)
		return 1
	}

	out, err := proj.Marshal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve: %v\n", err)
		return 1
	}

	os.Stdout.Write(out)
	return 0
}
